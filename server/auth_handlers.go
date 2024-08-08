package server

import (
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
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/errors"
	errs "github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
	jwtPackage "github.com/techagentng/citizenx/services/jwt"
)

func createS3Client() (*s3.Client, error) {
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
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
        return "", fmt.Errorf("failed to read file content: %v", err)
    }

    // Upload the file to S3
    _, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(key),
        Body:   bytes.NewReader(fileContent),
		ACL: "public-read",
    })
    if err != nil {
        return "", fmt.Errorf("failed to upload file to S3: %v", err)
    }

    // Return the S3 URL of the uploaded file
    fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, os.Getenv("AWS_REGION"), key)
    return fileURL, nil
}

// Define allowed MIME types and max file size
const (
    MaxFileSize = 5 * 1024 * 1024 // 5 MB
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
        secret := s.Config.JWTSecret // Adjust this based on your application's configuration
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
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to S3xxx"})
            return
        }

        // Retrieve user from service
        user, err := s.AuthRepository.FindUserByID(userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
            return
        }

        // Create new image record for the user
        user.ThumbNailURL = filepath
        if err := s.AuthRepository.CreateUserImage(user); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"message": "File uploaded and user profile updated successfully"})
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

        // Initialize the file path variable
        var filePath string

        // Get the profile image from the form
        file, handler, err := c.Request.FormFile("profile_image")
        if err == nil {
            // If file is provided, handle it
            defer file.Close()
            
            // Save the image to the specified directory
            filePath = fmt.Sprintf("uploads/%s", handler.Filename)
            out, err := os.Create(filePath)
            if err != nil {
                response.JSON(c, "", http.StatusInternalServerError, nil, err)
                return
            }
            defer out.Close()

            _, err = io.Copy(out, file)
            if err != nil {
                response.JSON(c, "", http.StatusInternalServerError, nil, err)
                return
            }
        } else if err == http.ErrMissingFile {
            // If no file is provided, set a default image path or URL
            filePath = "uploads/default-profile.png" // Adjust this path as necessary
        } else {
            // Handle other errors
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
        user.ThumbNailURL = filePath // Set the file path in the user struct

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

        response.JSON(c, "Signup successful, check your email for verification", http.StatusCreated, userResponse, nil)
    }
}


// Function to extract MAC address from a token
func extractMACAddressFromToken(macAddressToken string) (string, error) {
	// Split the token string to extract the MAC address part
	parts := strings.Split(macAddressToken, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format", errors.ErrInternalServerError.Status)
	}
	// Decode the MAC address part of the token
	macAddressBytes, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode token segment: %v", err)
	}
	// Convert the decoded bytes to string
	macAddress := string(macAddressBytes)
	return macAddress, nil
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

func (s *Server) HandleGoogleLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		state, err := generateJWTToken(s.Config.JWTSecret)
		if err != nil {
			response.JSON(c, "", errors.ErrInternalServerError.Status, nil, err)
			return
		}

		url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		c.Header("Access-Control-Allow-Origin", os.Getenv("ACCESS_CONTROL_ALLOW_ORIGIN"))
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Redirect(http.StatusTemporaryRedirect, url)

	}
}

func (s *Server) HandleGoogleCallback() gin.HandlerFunc {
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

		oauth2Config := &oauth2.Config{
			ClientID:     s.Config.GoogleClientID,
			ClientSecret: s.Config.GoogleClientSecret,
			RedirectURL:  s.Config.GoogleRedirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		}

		token, err := oauth2Config.Exchange(context.Background(), code)

		if err != nil || token == nil {
			fmt.Println("Token exchange error:", err.Error())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		authPayload, errr := s.googleSignInUser(c, token.AccessToken)
		log.Println("Google code:", authPayload)
		if errr != nil {
			log.Println("printed", errr)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authTokenxxx",
			})
			return
		}
		c.Header("Access-Control-Allow-Origin", "https://citizenx-dashboard-sbqx.onrender.com")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		log.Println("authpay data", authPayload.Data)
		response.JSON(c, "google sign in successful", http.StatusOK, authPayload, err)
	}
}

// generateJWTToken generates a jwt token to manage the state between calls to google
func generateJWTToken(secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("empty secret")
	}

	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
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

// AccessTokenDuration represents the default duration for access tokens.
var AccessTokenDuration = 15 * time.Minute

// RefreshTokenDuration represents the default duration for refresh tokens.
var RefreshTokenDuration = 7 * 24 * time.Hour

// Generate a token (access or refresh)
func generateToken() string {
	// Create a byte slice to hold the random bytes
	tokenBytes := make([]byte, 32) // You can adjust the token length as needed

	// Read random bytes into the slice
	_, err := rand.Read(tokenBytes)
	if err != nil {
		// Handle error
		return ""
	}

	// Encode the random bytes to base64 to generate a token string
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	return token
}

// AddAccessToken is a functional option to add an access token to the authentication payload.
func AddAccessToken(duration time.Duration) func(*AuthPayload) {
	return func(payload *AuthPayload) {
		payload.AccessTokenExpiration = time.Now().Add(duration)
		// Generate the access token here and set it in the payload
		payload.AccessToken = generateToken()
	}
}

// AddRefreshTokenSessionEntry is a functional option to add a refresh token session entry.
func AddRefreshTokenSessionEntry(c context.Context, duration time.Duration) func(*AuthPayload) error {
	return func(payload *AuthPayload) error {
		refreshToken := generateToken()
		// Store the refresh token session entry
		// err := store.StoreRefreshTokenSession(c, refreshToken, duration)
		// if err != nil {
		//     return err
		// }
		// Set the refresh token in the payload
		payload.RefreshToken = refreshToken
		return nil
	}
}

func (s *Server) googleSignInUser(c *gin.Context, token string) (*AuthPayload, error) {
	googleUserDetails, err := s.getUserInfoFromGoogle(token)
	if err != nil {
		return nil, fmt.Errorf("unable to get user details from google: %v", err)
	}

	authPayload, err := s.GetGoogleSignInToken(c, googleUserDetails)
	if err != nil {
		return nil, fmt.Errorf("unable to sign in user: %v", err)
	}
	fmt.Println("Google user details:", googleUserDetails)
	// Log authentication payload for debugging

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

    if googleUserDetails == nil {
        log.Println("Google user details are nil")
        return nil, fmt.Errorf("error: google user details can't be empty")
    }
    if googleUserDetails.Email == "" {
        log.Println("Google user email is empty")
        return nil, fmt.Errorf("error: email can't be empty")
    }
    if googleUserDetails.Name == "" {
        log.Println("Google user name is empty")
        return nil, fmt.Errorf("error: name can't be empty")
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

    log.Printf("Generating token pair for user: %s", googleUserDetails.Email)
    accessToken, refreshToken, err := jwtPackage.GenerateTokenPair(user.Email, s.Config.JWTSecret, user.AdminStatus, user.ID)
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

	newUser := &models.User{
		Email:    googleUserDetails.Email,
		IsSocial: true,
		Fullname: googleUserDetails.Name,
		// Add other fields as necessary
	}

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

	userIDString, ok := userID.(uint)
	if !ok {
		// Handle the case where userID is not a string
		log.Println("userID is not a string")
		return nil, nil
	}
	// Get email, isAdmin, and id from authRequest or other sources
	email := authRequest.email
	isAdmin := false

	// Call GenerateTokenPair with the obtained values
	accessToken, refreshToken, err := jwtPackage.GenerateTokenPair(email, s.Config.GoogleClientSecret, isAdmin, userIDString)
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

// validateState checks the state string with the system jwt secret while also validating the state validity
func validateState(state, secret string) error {
	token, err := jwt.Parse(state, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if !token.Valid {
		return fmt.Errorf("invalid state")
	}
	return err
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

func (s *Server) handleEditUserProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the access token from the authorization header
		accessToken := getTokenFromHeader(c)
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Validate and decode the access token to get the userID
		secret := s.Config.JWTSecret // Adjust this based on your application's configuration
		accessClaims, err := jwtPackage.ValidateAndGetClaims(accessToken, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Extract userID from accessClaims
		userIDValue, ok := accessClaims["id"]
		if !ok {
			respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("UserID not found in claims", http.StatusBadRequest))
			return
		}

		// Convert userIDValue to uint
		var userID uint
		switch v := userIDValue.(type) {
		case float64:
			userID = uint(v)
		default:
			respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("Invalid userID format", http.StatusBadRequest))
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

		response.JSON(c, "User details updated successfully", http.StatusOK, nil, nil)
	}
}

func (s *Server) handleShowProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// If user's thumbnail URL is empty, set it to the default thumbnail URL
		if user.ThumbNailURL == "" {
			user.ThumbNailURL = "default_thumbnail_url"
		}

		user, err = s.AuthService.GetUserProfile(userIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile"})
			return
		}

		// Prepare response data
		responseData := gin.H{
			"name":         user.Fullname,
			"username":     user.Username,
			"email":        user.Email,
			"profileImage": user.ThumbNailURL,
		}

		// Return the response
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
