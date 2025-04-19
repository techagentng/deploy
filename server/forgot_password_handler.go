package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"

	// "github.com/techagentng/citizenx/errors"
	// "github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
	"github.com/techagentng/citizenx/services/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	// jwtPackage "github.com/techagentng/citizenx/services/jwt"
)

func (s *Server) HandleForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the email and platform from the request
		req := struct {
			Email    string `json:"email" binding:"required,email"`
			Platform string `json:"platform" binding:"required"` // "web" or "mobile"
		}{}
		errs := decode(c, &req)
		if errs != nil {
			response.JSON(c, "", http.StatusBadRequest, nil, errs)
			return
		}

		// Validate platform
		if req.Platform != "web" && req.Platform != "mobile" {
			response.JSON(c, "", http.StatusBadRequest, nil, fmt.Errorf("invalid platform value"))
			return
		}

		// Find user by email
		user, err := s.AuthRepository.FindUserByEmail(req.Email)
		if err != nil || user == nil {
			response.JSON(c, "", http.StatusNotFound, nil, fmt.Errorf("user not found"))
			return
		}

		// Generate password reset token
		resetToken, err := utils.GeneratePasswordResetToken(user.Email, s.Config.JWTSecret)
		if err != nil {
			response.JSON(c, "failed to generate reset token", http.StatusInternalServerError, nil, err)
			return
		}

		// Save the reset token to the database
		user.ResetToken = resetToken
		if err := s.AuthRepository.UpdateUser(user); err != nil {
			response.JSON(c, "failed to save reset token", http.StatusInternalServerError, nil, err)
			return
		}

		// Create the reset link based on platform
		var resetLink string
		if req.Platform == "mobile" {
			resetLink = fmt.Sprintf("%s/reset-password/%s", resetToken)
		} else {
			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "http://localhost:3002"
			}
			resetLink = fmt.Sprintf("%s/reset-password/%s", baseURL, resetToken)
		}

		// Send password reset email
		_, err = s.Mail.SendResetPassword(user.Email, resetLink)
		if err != nil {
			response.JSON(c, "connection to mail service interrupted", http.StatusInternalServerError, nil, err)
			return
		}

		// Respond with success
		response.JSON(c, "Reset Password Link Sent Successfully", http.StatusOK, nil, nil)
	}
}


type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPasswordHandler handles the reset password request
func (s *Server) ResetPasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Extract the reset token from the URL parameter
		token := c.Param("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Reset token is required"})
			return
		}

		// Step 2: Retrieve the user associated with the token
		user, err := s.AuthRepository.FindUserByResetToken(token)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Invalid or expired reset token"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
			}
			return
		}

		// Step 3: Parse the new password from the request body
		var req struct {
			Password string `json:"newPassword" binding:"required,min=4"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
			return
		}

		// Step 4: Hash the new password
		hashedPassword, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password", "details": err.Error()})
			return
		}

		// Step 5: Update the user's password
		if err := s.AuthRepository.UpdateUserPassword(user, hashedPassword); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password", "details": err.Error()})
			return
		}

		// Step 6: Clear the reset token
		if err := s.AuthRepository.ClearResetToken(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear reset token", "details": err.Error()})
			return
		}

		// Step 7: Return a success response
		c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
	}
}


func hashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashedPassword), nil
}

type TokenClaims struct {
	UserID    uint      `json:"user_id"`
	TokenType string    `json:"type"`
	ExpiresAt time.Time `json:"exp"`
	jwt.StandardClaims
}

// VerifyResetToken verifies the reset password token and extracts the claims
func VerifyResetToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// The secret key used for signing should come from environment variables
		return []byte("your-secret-key"), nil
	})

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		if claims.TokenType != "password_reset_token" {
			return nil, fmt.Errorf("invalid token type", http.StatusAccepted)
		}
		return claims, nil
	}

	return nil, err
}
