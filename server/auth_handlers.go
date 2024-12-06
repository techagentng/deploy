package server

import (
	// "bytes"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/errors"
	errs "github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
	jwtPackage "github.com/techagentng/citizenx/services/jwt"
)

func createS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return s3.NewFromConfig(cfg), nil
}

func uploadFileToS3(client *s3.Client, file multipart.File, bucketName, key string) (string, error) {
	defer file.Close()

	// Read the file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file content: %v\n", err) // Log the error
		return "", fmt.Errorf("failed to read file content: %v", err)
	}

	//     // Log bucket and key information
	fmt.Printf("Uploading to bucket: %s\n", bucketName)
	fmt.Printf("Uploading with key: %s\n", key)

	// Upload the file to S3
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileContent),
		ACL:    types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		fmt.Printf("Error uploading file to S3: %v\n", err) // Log the error
		return "", fmt.Errorf("failed to upload file to S3x: %v", err)
	}

	// Log successful upload
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, os.Getenv("AWS_REGION"), key)
	fmt.Printf("File uploaded successfully, URL: %s\n", fileURL) // Log the URL

	return fileURL, nil
}

// Define allowed MIME types and max file size
const (
	MaxFileSize      = 5 * 1024 * 1024 // 5 MB
	AllowedMimeTypes = "image/jpeg,image/png,image/gif"
)

// validateFile checks the file type and size
func validateFile(file *multipart.FileHeader) error {
	// Check file size
	if file.Size > MaxFileSize {
		return fmt.Errorf("file size exceeds limit of %d bytes", MaxFileSize)
	}

	// Check file MIME type
	mimeType := file.Header.Get("Content-Type")
	if !isValidMimeType(mimeType) {
		return fmt.Errorf("invalid file type: %s", mimeType)
	}

	return nil
}

// isValidMimeType checks if the MIME type is allowed
func isValidMimeType(mimeType string) bool {
	allowedTypes := strings.Split(AllowedMimeTypes, ",")
	for _, allowedType := range allowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

func (s *Server) handleUpdateUserImageUrl() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle file upload
		file, fileHeader, err := c.Request.FormFile("profileImage")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid file"})
			return
		}

		// Validate file type and size
		if err := validateFile(fileHeader); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the access token from the authorization header
		accessToken := getTokenFromHeader(c)
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Validate and decode the access token to get the userID
		secret := s.Config.JWTSecret
		accessClaims, err := jwtPackage.ValidateAndGetClaims(accessToken, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var userID uint
		switch userIDValue := accessClaims["id"].(type) {
		case float64:
			userID = uint(userIDValue)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID format"})
			return
		}

		// Create S3 client
		s3Client, err := createS3Client()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create S3 client"})
			return
		}

		userIDString := strconv.FormatUint(uint64(userID), 10)

		// Generate unique filename
		filename := userIDString + "_" + fileHeader.Filename

		// Upload file to S3
		filepath, err := uploadFileToS3(s3Client, file, os.Getenv("AWS_BUCKET"), filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to S3xx"})
			return
		}

		// Retrieve user from service
		user, err := s.AuthRepository.FindUserByID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
			return
		}

		// Update user image URL
		user.ThumbNailURL = filepath
		if err := s.AuthRepository.UpsertUserImage(user.ID, filepath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
			return
		}

		log.Println("Filepath:", filepath)
		c.JSON(http.StatusOK, gin.H{
			"message": "File uploaded and user profile updated successfully",
			"url":     filepath,
		})
	}
}

func init() {
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		err = os.Mkdir("uploads", os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating uploads directory: %v", err)
		}
	}
}

func (s *Server) handleSignup() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse multipart form data
		if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max size
			response.JSON(c, "", http.StatusBadRequest, nil, err)
			return
		}

		var filePath string // This will hold the S3 URL

		// Get the profile image from the form
		file, handler, err := c.Request.FormFile("profile_image")
		if err == nil {
			defer file.Close()

			// Create S3 client
			s3Client, err := createS3Client()
			if err != nil {
				response.JSON(c, "", http.StatusInternalServerError, nil, err)
				return
			}

			// Generate unique filename
			userID := c.PostForm("user_id")
			filename := fmt.Sprintf("%s_%s", userID, handler.Filename)

			// Upload file to S3
			filePath, err = uploadFileToS3(s3Client, file, os.Getenv("AWS_BUCKET"), filename)
			if err != nil {
				response.JSON(c, "", http.StatusInternalServerError, nil, err)
				return
			}
		} else if err == http.ErrMissingFile {
			filePath = "uploads/default-profile.png" // Adjust this to a default S3 URL if necessary
		} else {
			response.JSON(c, "", http.StatusBadRequest, nil, err)
			return
		}

		// Decode the other form data into the user struct
		var user models.User
		user.Fullname = c.PostForm("fullname")
		user.Username = c.PostForm("username")
		user.Telephone = c.PostForm("telephone")
		user.Email = c.PostForm("email")
		user.Password = c.PostForm("password")
		user.ThumbNailURL = filePath // Set the S3 URL in the user struct

		// Fetch the UUID for the role
		role, err := s.AuthService.GetRoleByName("User") // Use a service method to fetch the role by name
		if err != nil {
			response.JSON(c, "", http.StatusInternalServerError, nil, err)
			return
		}
		log.Printf("Fetched role ID for 'User': %s", role.ID.String())

		// Assign the role UUID directly to RoleID
		user.RoleID = role.ID

		// Validate the user data using the validator package
		validate := validator.New()
		if err := validate.Struct(user); err != nil {
			response.JSON(c, "", http.StatusBadRequest, nil, err)
			return
		}

		// Signup the user using the service
		userResponse, err := s.AuthService.SignupUser(&user)
		if err != nil {
			response.HandleErrors(c, err) // Use HandleErrors to handle different error types
			return
		}

		// Send welcome email
		subject := "Welcome to Our Platform!"

		_, err = s.Mail.SendWelcomeMessage(user.Email, subject)
		if err != nil {
			log.Printf("Error sending welcome email: %v", err)
			// Log the error but do not interrupt the signup flow
		}

		response.JSON(c, "Signup successful, check your email for verification", http.StatusCreated, userResponse, nil)
	}
}


// Middleware to redirect non-credential users to sign-in page for certain actions
func (s *Server) handleNonCredentialLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// macAddressToken := c.Request.Header.Get("MAC-Address-Token")
		macAddressToken := "676765454646466"
		// If MAC address token is not provided, proceed with the request
		if macAddressToken == "" {
			c.Next()
			return
		}

		// Find user by MAC address using AuthRepository
		user, err := s.AuthRepository.FindUserByMacAddress(macAddressToken)
		if err != nil {
			// If user not found, create a new user
			user = &models.LoginRequestMacAddress{
				MacAddress: macAddressToken,
			}

			// Save the new user to the database
			if _, err := s.AuthRepository.CreateUserWithMacAddress(user); err != nil {
				response.JSON(c, "Failed to create user", http.StatusInternalServerError, nil, errs.New("Failed to create user", http.StatusInternalServerError))
				return
			}

			// Generate MAC address token and return it in the login response
			macAddressTokenResponse, err := s.AuthService.LoginMacAddressUser(user)
			if err != nil {
				response.JSON(c, "Failed to generate MAC address token", http.StatusInternalServerError, nil, errs.New("Failed to generate MAC address token", http.StatusInternalServerError))
				return
			}

			// Respond with the MAC address token
			response.JSON(c, "Login successful", http.StatusOK, macAddressTokenResponse, nil)
			return
		}

		// Respond with the user details or an access token if needed
		response.JSON(c, "Login successful", http.StatusOK, user, nil)
	}
}

func (s *Server) handleLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginRequest models.LoginRequest
		if err := decode(c, &loginRequest); err != nil {
			response.JSON(c, "", errors.ErrBadRequest.Status, nil, err)
			return
		}
		userResponse, err := s.AuthService.LoginUser(&loginRequest)
		if err != nil {
			response.JSON(c, "", err.Status, nil, err)
			return
		}
		response.JSON(c, "login successful", http.StatusOK, userResponse, nil)
	}
}

func generateJWTState(secret string) (string, error) {
    // Use a more specific claim structure
    claims := jwt.MapClaims{
        "exp": time.Now().Add(10 * time.Minute).Unix(),
        "iat": time.Now().Unix(),
        "state": uuid.New().String(), // Add a unique state identifier
    }

    // Create a new JWT token with a specific key ID or audience
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    
    // Optionally set a key ID
    token.Header["kid"] = "your-key-identifier"

    // Sign the token using the provided secret
    signedToken, err := token.SignedString([]byte(secret))
    if err != nil {
        return "", fmt.Errorf("failed to sign JWT token: %w", err)
    }
    return signedToken, nil
}

func (s *Server) HandleGoogleLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Google OAuth2 configuration
		config := &oauth2.Config{
			ClientID:     s.Config.GoogleClientID,
			ClientSecret: s.Config.GoogleClientSecret,
			RedirectURL:  s.Config.GoogleRedirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		}

		// Generate the JWT state
		state, err := generateJWTState(s.Config.JWTSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
			return
		}

		// Create the Google Auth URL
		authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

		// Redirect the user to the Google Auth URL
		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

// Thread-safe in-memory state store
var stateStore = sync.Map{}

func removeState(state string) {
	stateStore.Delete(state)
}

func verifyState(state string, secret string) bool {
    token, err := jwt.Parse(state, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    return err == nil && token.Valid
}

func validateAccessToken(token string) (bool, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=" + token)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}


func (s *Server) HandleGoogleCallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the authorization code and state from query parameters
		code := c.Query("code")
		state := c.Query("state")
		log.Printf("Received state: %s", state)
		log.Printf("Received code: %s", code)

		// Step 1: Verify the state parameter to prevent CSRF
		if !verifyState(state, s.Config.JWTSecret) {
			log.Println("Invalid or expired state")
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid or expired state"})
			return
		}

		// Step 2: Remove the state to prevent reuse (optional but recommended)
		removeState(state)

		// Step 3: Exchange the authorization code for an access token
		tokenResponse, err := exchangeCodeForToken(code, s.Config.GoogleClientID, s.Config.GoogleClientSecret, s.Config.GoogleRedirectURL)
		if err != nil {
			log.Printf("Token exchange failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
			return
		}

		// Step 4: Extract and validate the access token
		accessToken, ok := tokenResponse["access_token"].(string)
		if !ok || accessToken == "" {
			log.Println("Access token missing in response")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Access token missing in response"})
			return
		}

		// Optional: Validate the access token with Google
		if valid, err := validateAccessToken(accessToken); !valid || err != nil {
			log.Printf("Invalid access token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid access token"})
			return
		}

		// Step 5: Fetch user data from Google using the access token
		userData, err := s.getUserDataFromGoogle(accessToken)
		if err != nil {
			log.Printf("Failed to fetch user information: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
			return
		}

		email, ok := userData["email"].(string)
		if !ok || email == "" {
			log.Println("Invalid user data: email missing")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user data: email missing"})
			return
		}

		// Step 6: Get or create the user in the database
		user, err := s.getOrCreateUser(email, userData)
		if err != nil {
			log.Printf("Error processing user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user"})
			return
		}

		// Step 7: Generate a JWT token for the authenticated user
		tokenString, err := GenerateJWTTokenForUser(*user, s.Config.JWTSecret)
		if err != nil {
			log.Printf("Error generating JWT token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token"})
			return
		}

		// Step 8: Respond with the JWT token and user details
		c.JSON(http.StatusOK, gin.H{
			"token": tokenString,
			"user": gin.H{
				"email":   user.Email,
				"name":    user.Fullname,
				"picture": user.ThumbNailURL,
			},
		})
	}
}


func exchangeCodeForToken(code, clientID, clientSecret, redirectURI string) (map[string]interface{}, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}
	return tokenResponse, nil
}


// Helper function: Get or create user
func (s *Server) getOrCreateUser(email string, userData map[string]interface{}) (*models.User, error) {
	user, err := s.AuthRepository.GetUserByEmail(email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create a new user if not found
			newUser := &models.User{
				Email:        email,
				Fullname:     userData["name"].(string),
				ThumbNailURL: userData["picture"].(string),
			}
			if _, err := s.AuthRepository.CreateUser(newUser); err != nil {
				return nil, err
			}
			return newUser, nil
		}
		return nil, err
	}
	return user, nil
}

type UserClaims struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.StandardClaims
}

func GenerateJWTTokenForUser(user models.User, secretKey string) (string, error) {
	// Define the token expiration time
	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours

	// Create the claims
	claims := UserClaims{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Fullname,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Create the token with the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token using the secret key
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *Server) exchangeCodeForToken(code string) (*oauth2.Token, error) {
	config := &oauth2.Config{
		ClientID:     s.Config.GoogleClientID,
		ClientSecret: s.Config.GoogleClientSecret,
		RedirectURL:  s.Config.GoogleRedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	// Exchange the authorization code for an access token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %v", err)
	}

	return token, nil
}

// getUserDataFromGoogle retrieves the user data from Google using the access token
func (s *Server) getUserDataFromGoogle(accessToken string) (map[string]interface{}, error) {
    // Google API endpoint to retrieve user profile information
    userInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo"
    req, err := http.NewRequest("GET", userInfoURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request to fetch user data: %v", err)
    }

    // Set the authorization header with the access token
    req.Header.Set("Authorization", "Bearer "+accessToken)

    // Make the request to Google API
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve user data from Google: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("received non-200 status code from Google API: %d", resp.StatusCode)
    }

    // Decode the user data response
    var userData map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
        return nil, fmt.Errorf("failed to decode user data: %v", err)
    }

    return userData, nil
}

// generateJWTToken generates a jwt token to manage the state between calls to google
func generateJWTToken(secret string) (string, error) {
	// Define claims for the JWT
	claims := jwt.MapClaims{
		"timestamp": time.Now().Unix(),      // Current timestamp
		"nonce":     generateRandomString(), // Random string for uniqueness
	}

	// Create the JWT with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret
	return token.SignedString([]byte(secret))
}

// Utility function to generate a random string for the nonce
func generateRandomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 16) // Generate a 16-character string
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}


type GoogleUser struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

// AuthPayload represents the authentication payload structure.
type AuthPayload struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int    `json:"expires_in"`
	AccessTokenExpiration time.Time
	Data                  interface{} `json:"data"`
}

// AuthRequest represents the authentication request structure.
type AuthRequest struct {
	email string `json:"email"`
}

var AccessTokenDuration = 15 * time.Minute

var RefreshTokenDuration = 7 * 24 * time.Hour

func generateToken() string {
	tokenBytes := make([]byte, 32)

	// Read random bytes into the slice
	_, err := rand.Read(tokenBytes)
	if err != nil {
		// Handle error
		return ""
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	return token
}

func AddAccessToken(duration time.Duration) func(*AuthPayload) {
	return func(payload *AuthPayload) {
		payload.AccessTokenExpiration = time.Now().Add(duration)
		payload.AccessToken = generateToken()
	}
}

func AddRefreshTokenSessionEntry(c context.Context, duration time.Duration) func(*AuthPayload) error {
	return func(payload *AuthPayload) error {
		refreshToken := generateToken()
		payload.RefreshToken = refreshToken
		return nil
	}
}

func (s *Server) googleSignInUser(c *gin.Context, token string) (*AuthPayload, error) {
	googleUserDetails, err := s.getUserInfoFromGoogle(token)
	if err != nil {
		return nil, fmt.Errorf("unable to get user details from google: %v", err)
	}

	// // Fetch user by email to get the role ID
	// user, err := s.AuthRepository.FindUserByEmail(googleUserDetails.Email)
	// if err != nil {
	//     return nil, fmt.Errorf("unable to fetch user by email: %v", err)
	// }

	// Fetch the role by ID
	// role, err := s.AuthRepository.FindRoleByID(user.RoleID)
	// if err != nil {
	//     return nil, fmt.Errorf("unable to fetch role for user: %v", err)
	// }

	// Call GetGoogleSignInToken with the googleUserDetails and the found role
	authPayload, err := s.GetGoogleSignInToken(c, googleUserDetails)
	if err != nil {
		return nil, fmt.Errorf("unable to sign in user: %v", err)
	}

	// Log the Google user details and the authentication payload for debugging
	fmt.Println("Google user details:", googleUserDetails)
	fmt.Printf("Auth Payload: %+v\n", authPayload)

	return authPayload, nil
}

// getUserInfoFromGoogle will return information of user which is fetched from Google
func (srv *Server) getUserInfoFromGoogle(token string) (*GoogleUser, error) {
	var googleUserDetails GoogleUser

	url := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token
	googleUserDetailsRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error occurred while getting information from Google: %+v", err)
	}

	googleUserDetailsResponse, googleDetailsResponseError := http.DefaultClient.Do(googleUserDetailsRequest)
	if googleDetailsResponseError != nil {
		return nil, fmt.Errorf("error occurred while getting information from Google: %+v", googleDetailsResponseError)
	}

	body, err := io.ReadAll(googleUserDetailsResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("error occurred while getting information from Google: %+v", err)
	}
	defer googleUserDetailsResponse.Body.Close()

	err = json.Unmarshal(body, &googleUserDetails)
	if err != nil {
		return nil, fmt.Errorf("error occurred while getting information from Google: %+v", err)
	}

	return &googleUserDetails, nil
}

// GetGoogleSignInToken returns the signin access token and refresh token pair to the social user
func (s *Server) GetGoogleSignInToken(c *gin.Context, googleUserDetails *GoogleUser) (*AuthPayload, error) {
	log.Println("Starting Google sign-in process")

	if googleUserDetails == nil || googleUserDetails.Email == "" || googleUserDetails.Name == "" {
		return nil, fmt.Errorf("error: google user details are incomplete")
	}

	log.Printf("Looking for existing user with email: %s", googleUserDetails.Email)
	user, err := s.AuthRepository.FindUserByEmail(googleUserDetails.Email)
	if err != nil {
		if err.Error() == "user not found" {
			log.Printf("No existing user found with email: %s. Proceeding to sign-up.", googleUserDetails.Email)
			user, err = s.signUpAndCreateUser(c, googleUserDetails)
			if err != nil {
				log.Printf("Error during sign-up for email %s: %v", googleUserDetails.Email, err)
				return nil, fmt.Errorf("error signing up user: %v", err)
			}
		} else {
			log.Printf("Error occurred while checking if email exists: %v", err)
			return nil, fmt.Errorf("error checking if email exists: %v", err)
		}
	} else {
		log.Printf("Existing user found: %+v", user)
	}

	// Fetch the role by ID
	role, err := s.AuthRepository.FindRoleByID(user.RoleID)
	if err != nil {
		log.Printf("Error fetching role for user: %v", err)
		return nil, fmt.Errorf("unable to fetch role for user: %v", err)
	}

	log.Printf("Generating token pair for user: %s", googleUserDetails.Email)

	// Generate the token pair
	accessToken, refreshToken, err := jwtPackage.GenerateTokenPair(
		user.Email,         // Use the user's email
		s.Config.JWTSecret, // JWT secret from the server config
		user.AdminStatus,   // Admin status from the user model
		user.ID,            // Use the correct user ID
		role.Name,          // Pass the role name
	)

	if err != nil {
		log.Printf("Error generating token pair for email %s: %v", googleUserDetails.Email, err)
		return nil, fmt.Errorf("error generating token pair: %v", err)
	}

	payload := &AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenDuration.Seconds()),
	}
	log.Printf("Auth payload generated for user %s: %+v", googleUserDetails.Email, payload)
	return payload, nil
}

func (s *Server) signUpAndCreateUser(c *gin.Context, googleUserDetails *GoogleUser) (*models.User, error) {
	log.Printf("Attempting to sign up user with email: %s", googleUserDetails.Email)

	// Fetch the role you want to assign (e.g., "User" role) using the repository
	roleName := "User"
	role, err := s.AuthRepository.FindRoleByName(roleName)
	if err != nil {
		log.Printf("Error fetching role: %v", err)
		return nil, fmt.Errorf("error fetching role: %v", err)
	}

	// Create a new user with the fetched role's ID
	newUser := &models.User{
		Email:    googleUserDetails.Email,
		IsSocial: true,
		Fullname: googleUserDetails.Name,
		RoleID:   role.ID,
	}

	// Create the user using the repository
	createdUser, err := s.AuthRepository.CreateUser(newUser)
	if err != nil {
		log.Printf("Error creating user for email %s: %v", googleUserDetails.Email, err)
		return nil, fmt.Errorf("error creating user: %v", err)
	}

	log.Printf("User created successfully: %+v", createdUser)
	return createdUser, nil
}

func (s *Server) SocialAuthenticate(authRequest *AuthRequest, authPayloadOption func(*AuthPayload), c *gin.Context) (*AuthPayload, error) {
	// Get the user ID from the context
	userID, ok := c.Get("userID")
	if !ok {
		return nil, fmt.Errorf("userID not found in context")
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		log.Println("userID is not a uint")
		return nil, fmt.Errorf("userID is not a valid uint")
	}

	// Get email from authRequest
	email := authRequest.email

	// Fetch the role from the repository based on userID
	userRole, err := s.AuthRepository.GetUserRoleByUserID(userIDUint)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve role for user: %v", err)
	}

	// Determine if the user is an admin
	isAdmin := userRole.Name == "admin"

	// Pass the role name to GenerateTokenPair
	accessToken, refreshToken, err := jwtPackage.GenerateTokenPair(email, s.Config.GoogleClientSecret, isAdmin, userIDUint, userRole.Name)
	if err != nil {
		return nil, err
	}

	// Construct AuthPayload and return
	payload := &AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenDuration.Seconds()),
	}

	authPayloadOption(payload)

	return payload, nil
}

func validateState(state, secret string) error {
    // Step 1: Basic validation
    if state == "" {
        return fmt.Errorf("empty state token")
    }
    log.Printf("State received: %s", state)

    // Step 2: Decode state (in case it's URL-encoded)
    decodedState, err := url.QueryUnescape(state)
    if err != nil {
        return fmt.Errorf("failed to decode state: %v", err)
    }
    log.Printf("Decoded state: %s", decodedState)

    // Step 3: Parse the JWT token
    token, err := jwt.ParseWithClaims(decodedState, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
        // Verify the signing method is HMAC
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })

    // Handle token parsing errors
    if err != nil {
        return fmt.Errorf("token parse error: %v", err)
    }

    // Step 4: Check token validity and claims
    if token == nil || !token.Valid {
        return fmt.Errorf("token is invalid or nil")
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return fmt.Errorf("failed to parse claims")
    }

    log.Printf("Token claims: %v", claims)

    // Optional: Add custom claim validation if needed
    return nil
}


func GetValuesFromContext(c *gin.Context) (string, *models.User, *errors.Error) {
	var tokenI, userI interface{}
	var tokenExists, userExists bool

	if tokenI, tokenExists = c.Get("access_token"); !tokenExists {

		fmt.Println("called 404")

		return "", nil, errors.New("forbidden", http.StatusForbidden)
	}
	if userI, userExists = c.Get("user"); !userExists {
		return "", nil, errors.New("forbidden", http.StatusForbidden)
	}
	log.Println("got herr")
	token, ok := tokenI.(string)
	if !ok {
		return "", nil, errors.New("internal server error", http.StatusInternalServerError)
	}
	user, ok := userI.(*models.User)
	if !ok {
		return "", nil, errors.New("internal server error", http.StatusInternalServerError)
	}
	return token, user, nil
}

// Logout invalidates the access token and adds it to the blacklist
func (s *Server) handleLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the access token from the context
		token, exists := c.Get("access_token")
		if !exists {
			log.Println("Access token not found in context")
			respondAndAbort(c, "Access token not found in context", http.StatusInternalServerError, nil, errs.New("Internal server error", http.StatusInternalServerError))
			return
		}

		accessToken, ok := token.(string)
		if !ok {
			log.Println("Access token is not a string")
			respondAndAbort(c, "Access token is not a string", http.StatusInternalServerError, nil, errs.New("Internal server error", http.StatusInternalServerError))
			return
		}

		blackListEntry := &models.Blacklist{
			Token: accessToken,
		}

		// Add the access token to the blacklist
		if err := s.AuthRepository.AddToBlackList(blackListEntry); err != nil {
			log.Printf("Error adding access token to blacklist: %v", err)
			respondAndAbort(c, "Logout failed", http.StatusInternalServerError, nil, errs.New("Internal server error", http.StatusInternalServerError))
			return
		}

		// Retrieve the user from the context
		user, exists := c.Get("user")
		if !exists {
			log.Println("User not found in context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
			return
		}

		// Type assert user to *models.User
		u, ok := user.(*models.User)
		if !ok {
			log.Println("User data is corrupted")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User data is corrupted"})
			return
		}

		// Update user's online status in the database
		if err := s.AuthRepository.SetUserOffline(u); err != nil {
			log.Printf("Failed to set user offline: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set user offline"})
			return
		}

		// Respond with a success message
		response.JSON(c, "Logout successful", http.StatusOK, nil, nil)
	}
}

// Handler for updating user profile
func (s *Server) handleEditUserProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the access token from the authorization header
		accessToken := getTokenFromHeader(c)
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Validate and decode the access token to get the userID
		secret := s.Config.JWTSecret
		accessClaims, err := jwtPackage.ValidateAndGetClaims(accessToken, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Extract userID from accessClaims
		userIDValue, ok := accessClaims["id"]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "UserID not found in claims"})
			return
		}

		// Convert userIDValue to uint
		var userID uint
		switch v := userIDValue.(type) {
		case float64:
			userID = uint(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID format"})
			return
		}

		// Parse request body into userDetails
		var userDetails models.EditProfileResponse
		if err := c.ShouldBindJSON(&userDetails); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Call service method to update user details
		if err := s.AuthService.EditUserProfile(userID, &userDetails); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user details"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User details updated successfully"})
	}
}

func (s *Server) handleShowProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve user ID from context
		userID, ok := c.Get("userID")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
			return
		}

		userIDStr, ok := userID.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid type for user ID"})
			return
		}

		// Retrieve user from the database
		user, err := s.AuthRepository.FindUserByID(userIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile"})
			return
		}

		// Prepare response data with the necessary fields
		responseData := gin.H{
			"email":        user.Email,
			"name":         user.Fullname,
			"profileImage": user.ThumbNailURL,
			"username":     user.Username,
		}

		// Return the response with the user's profile data
		response.JSON(c, "User profile retrieved successfully", http.StatusOK, responseData, nil)
	}
}

// Assuming you have imported necessary packages and defined your server and repository

func (s *Server) handleGetOnlineUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call repository function to get online users
		onlineUsers, err := s.AuthRepository.GetOnlineUserCount()
		if err != nil {
			// Handle error
			response.JSON(c, "Error fetching online users", http.StatusInternalServerError, nil, err)
			return
		}

		// Respond with online users
		response.JSON(c, "Successfully fetched online users", http.StatusOK, onlineUsers, nil)
	}
}

func (s *Server) handleGetAllUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call service function to get all users
		users, err := s.AuthService.GetAllUsers()
		if err != nil {
			// Handle error
			response.JSON(c, "Error fetching all users", http.StatusInternalServerError, nil, err)
			return
		}

		// Respond with users
		response.JSON(c, "Successfully fetched all users", http.StatusOK, users, nil)
	}
}

func (s *Server) handleDeleteUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the userID from the context
		userID, exists := c.Get("userID")
		if !exists {
			response.JSON(c, "User ID not found in context", http.StatusUnauthorized, nil, nil)
			return
		}

		// Type assert to uint
		userIDUint, ok := userID.(uint)
		if !ok {
			response.JSON(c, "Invalid user ID type", http.StatusInternalServerError, nil, nil)
			return
		}

		// Perform the user deletion
		if err := s.AuthService.DeleteUser(userIDUint); err != nil {
			response.JSON(c, "Failed to delete user", http.StatusInternalServerError, nil, err)
			return
		}

		response.JSON(c, "User deleted successfully", http.StatusOK, nil, nil)
	}
}

// func (s *Server) SendPasswordResetEmail(token, email string) *apiError.Error {
// 	link := fmt.Sprintf("%s/verifyEmail/%s", s.Config.BaseUrl, token)
// 	value := map[string]interface{}{}
// 	value["link"] = link
// 	subject := "Verify your email"
// 	body := "Please Click the link below to verify your email"
// 	templateName := "emailverification"
// 	err := SendMail(email, subject, body, templateName, value)
// 	if err != nil {
// 		log.Printf("Error: %v", err.Error())
// 		return apiError.New("Internal server error: check email config", http.StatusInternalServerError)
// 	}
// 	return nil
// }
// func generateUniqueToken() string {
// 	return uuid.New().String()
// }
func (s *Server) GenerateGoogleState() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Generate a JWT state
        state, err := generateJWTToken(s.Config.JWTSecret)
        if err != nil {
            log.Println("Error generating state:", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
            return
        }

        // Send the JWT state to the frontend
        c.JSON(http.StatusOK, gin.H{"state": state})
    }
}
