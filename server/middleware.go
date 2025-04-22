package server

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	// ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"

	// ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"
	errs "github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
	"github.com/techagentng/citizenx/services/jwt"
)

func (s *Server) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from header
		accessToken := getTokenFromHeader(c)
		if accessToken == "" {
			respondAndAbort(c, "", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Check if the token is blacklisted
		if s.AuthRepository.IsTokenInBlacklist(accessToken) {
			respondAndAbort(c, "Access token is blacklisted", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Validate token and get claims
		secret := s.Config.JWTSecret
		accessClaims, err := jwt.ValidateAndGetClaims(accessToken, secret)
		if err != nil {
			respondAndAbort(c, "", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Extract userID from claims
		userIDValue, ok := accessClaims["id"]
		if !ok {
			respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("User ID not found in token", http.StatusBadRequest))
			return
		}

		// Convert userID to uint, typically claims store numbers as float64
		var userID uint
		switch v := userIDValue.(type) {
		case float64:
			userID = uint(v)
		case int:
			userID = uint(v)
		case string:
			// If userID is provided as a string, try to convert it to uint
			parsedID, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("Invalid User ID format", http.StatusBadRequest))
				return
			}
			userID = uint(parsedID)
		default:
			respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("Invalid User ID type", http.StatusBadRequest))
			return
		}

		// Fetch the user from the database by ID
		user, err := s.AuthRepository.FindUserByID(userID)
		if err != nil {
			switch {
			case errors.Is(err, errs.InActiveUserError):
				respondAndAbort(c, "Inactive user", http.StatusUnauthorized, nil, errs.New(err.Error(), http.StatusUnauthorized))
				return
			case errors.Is(err, gorm.ErrRecordNotFound):
				respondAndAbort(c, "User not found", http.StatusUnauthorized, nil, errs.New(err.Error(), http.StatusUnauthorized))
				return
			default:
				respondAndAbort(c, "Unable to find entity", http.StatusInternalServerError, nil, errs.New("Internal server error", http.StatusInternalServerError))
				return
			}
		}

		// Set user-related values in the context for further handlers
		c.Set("user", user)
		c.Set("userID", userID)
		c.Set("access_token", accessToken)
		c.Set("fullName", user.Fullname)
		c.Set("username", user.Username)
		c.Set("profile_image", user.ThumbNailURL)
		c.Set("user_role", accessClaims["role"].(string))
		fmt.Println("Username set in context:", user.Username)

		// Ensure that the user is correctly set as a pointer to User
		c.Set("currentUser", &user)

		// Continue to the next middleware or handler
		c.Next()
	}
}


// Function to check if the user has exceeded the rate limit
func isRateLimitExceeded(userID uint, lat float64, lng float64) bool {
	var mu sync.Mutex
	// Lock access to userReports map
	mu.Lock()
	defer mu.Unlock()

	var userReports = make(map[uint][]models.IncidentReport)
	reports, ok := userReports[userID]
	if !ok {
		return false
	}

	currentTime := time.Now()
	count := 0
	for _, report := range reports {
		if report.Latitude == lat && report.Longitude == lng && currentTime.Sub(report.TimeofIncidence) <= 2*time.Minute {
			count++
		}
	}

	return count >= 5
}

func rateLimitAndSpamDetection() gin.HandlerFunc {
	return func(c *gin.Context) {
		var mu sync.Mutex
		// Retrieve the userID from the context
		userIDInterface, exists := c.Get("userID")
		if !exists {
			// Handle case where userID is not found
			response.JSON(c, "User ID not found", http.StatusInternalServerError, nil, errors.New("user ID not found"))
			return
		}
		userID, ok := userIDInterface.(uint)
		if !ok {
			// Handle case where userID is not of expected type
			response.JSON(c, "Invalid type for userID", http.StatusInternalServerError, nil, errors.New("invalid type for userID"))
			return
		}

		latStr := c.PostForm("latitude")
		lngStr := c.PostForm("longitude")
		lat, _ := strconv.ParseFloat(latStr, 64)
		lng, _ := strconv.ParseFloat(lngStr, 64)

		if isRateLimitExceeded(userID, lat, lng) {
			// Inform the user that their account is under review for spam
			response.JSON(c, "Your account is under review for spam. Please wait for 3 hours.", http.StatusTooManyRequests, nil, errs.ErrInternalServerError)
			return
		}

		// Add the current report to the user's reports
		report := models.IncidentReport{
			Latitude:        lat,
			Longitude:       lng,
			TimeofIncidence: time.Now(),
		}

		// Lock access to userReports map
		mu.Lock()
		defer mu.Unlock()
		var userReports = make(map[uint][]models.IncidentReport)

		userReports[userID] = append(userReports[userID], report)

		// Proceed with the request
		c.Next()
	}
}

func keyFunc(c *gin.Context) string {
	//TODO Handle when email isn't sent successfully in any of the three tries
	//b1, err := c.Request.GetBody()
	buf, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		response.JSON(c, "", http.StatusBadRequest, nil, err)
		return ""
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	var foundUser models.ForgotPassword
	err = decode(c, &foundUser)
	if err != nil {
		response.JSON(c, "", http.StatusBadRequest, nil, err)
		return ""
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return foundUser.Email
}

func keyFuncMacAddress(c *gin.Context) string {
	// Extract MAC address from the request
	macAddress := c.PostForm("mac_address")
	return macAddress
}

// respondAndAbort calls response.JSON and aborts the Context
func respondAndAbort(c *gin.Context, message string, status int, data interface{}, e *errs.Error) {
	response.JSON(c, message, status, data, e)
	c.Abort()
}

func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

// getTokenFromHeader returns the token string in the authorization header
func getTokenFromHeader(c *gin.Context) string {
	authHeader := c.Request.Header.Get("Authorization")
	if len(authHeader) > 8 {
		return authHeader[7:]
	}
	return ""
}

// Middleware for Restricting Access to Protected Routes
func restrictAccessToProtectedRoutes() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is non-credential
		_, exists := c.Get("user")
		if !exists {
			// Check if the user is a MAC address user
			accessToken := getTokenFromHeader(c)
			if accessToken != "" {
				// Validate and decode the access token to get the claims
				secret := os.Getenv("JWT_SECRET")
				accessClaims, err := jwt.ValidateAndGetClaims(accessToken, secret)
				if err == nil {
					_, isMACAddressUser := accessClaims["mac_address"]
					if isMACAddressUser {
						// Handle the case for MAC address user
						response.JSON(c, "", http.StatusForbidden, nil, errs.New("Forbidden: Access restricted for MAC address users", http.StatusForbidden))
						c.Abort()
						return
					}
				}
			}

			// User is non-credential and not a MAC address user, restrict access to protected routes
			restrictedRoutes := []string{"/user/:reportID/like", "/user/:reportID/bookmark"}
			if containsString(restrictedRoutes, c.Request.URL.Path) {
				response.JSON(c, "", http.StatusForbidden, nil, errs.New("Forbidden: Access restricted for non-credential users", http.StatusForbidden))
				c.Abort()
				return
			}
		}

		// User is authenticated or not accessing a protected route, continue with the request
		c.Next()
	}
}

// Function to check if a string exists in a slice of strings
func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
