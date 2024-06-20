package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	facebookOAuth "golang.org/x/oauth2/facebook"
)

type FacebookUser struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

// GetFacebookOAuthConfig will return the config to call facebook Login
func GetFacebookOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("FACEBOOK_APP_ID"),
		ClientSecret: os.Getenv("FACEBOOK_APP_SECRET"),
		Endpoint:     facebookOAuth.Endpoint,
		RedirectURL:  os.Getenv("FACEBOOK_REDIRECT_URL"),
		Scopes:       []string{"email"},
	}
}

// InitFacebookLogin function will initiate the Facebook Login
func (s *Server) handleFBLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var OAuth2Config = GetFacebookOAuthConfig()
		state, err := generateJWTToken(s.Config.JWTSecret)
		if err != nil {
			fmt.Errorf("error generating token state: %v", err)
		}
		url := OAuth2Config.AuthCodeURL(state)
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

func (s *Server) handleFBCallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		state := c.Query("state")
		code := c.Query("code")

		err := validateState(state, s.Config.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid login",
			})
			return
		}

		var OAuth2Config = GetFacebookOAuthConfig()

		token, err := OAuth2Config.Exchange(oauth2.NoContext, code)

		if err != nil || token == nil {
			fmt.Println("Token exchange error:", err.Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}
		fbUserDetails, fbUserDetailsError := GetUserInfoFromFacebook(token.AccessToken)
		log.Println("facebook user", fbUserDetails)
		if fbUserDetailsError != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user details",
			})
			return
		}

		authToken, authTokenError := SignInUser(fbUserDetails)

		if authTokenError != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unable to sign in user",
			})
			return
		}
		c.SetCookie("Authorization", "Bearer "+authToken, 3600, "/", "", false, true)

		c.Redirect(http.StatusTemporaryRedirect, "/profile")
	}
}

func SignInUser(facebookUserDetails FacebookUser) (string, error) {

	if facebookUserDetails == (FacebookUser{}) {
		return "", fmt.Errorf("User details can't be empty")
	}

	if facebookUserDetails.Email == "" {
		return "", fmt.Errorf("Email can't be empty")
	}

	if facebookUserDetails.Name == "" {
		return "", fmt.Errorf("Name can't be empty")
	}

	// Your sign-in logic here...

	tokenString, _ := generateJWTToken(facebookUserDetails.Email)

	if tokenString == "" {
		return "", fmt.Errorf("Unable to generate Auth token")
	}

	return tokenString, nil
}

// GetUserInfoFromFacebook will return information of user which is fetched from facebook
func GetUserInfoFromFacebook(token string) (FacebookUser, error) {
	var fbUserDetails FacebookUser
	facebookUserDetailsRequest, _ := http.NewRequest("GET", "https://graph.facebook.com/me?fields=id,name,email&access_token="+token, nil)
	facebookUserDetailsResponse, facebookUserDetailsResponseError := http.DefaultClient.Do(facebookUserDetailsRequest)

	if facebookUserDetailsResponseError != nil {
		return FacebookUser{}, fmt.Errorf("Error occurred while getting information from Facebook")
	}

	decoder := json.NewDecoder(facebookUserDetailsResponse.Body)
	decoderErr := decoder.Decode(&fbUserDetails)
	defer facebookUserDetailsResponse.Body.Close()

	if decoderErr != nil {
		return FacebookUser{}, fmt.Errorf("Error occurred while getting information from Facebook")
	}

	return fbUserDetails, nil
}
