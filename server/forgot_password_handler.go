package server

import (
	"fmt"
	"net/http"
	"os"
	"time"
    "golang.org/x/crypto/bcrypt"
	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/errors"
	"github.com/golang-jwt/jwt"
	"github.com/techagentng/citizenx/server/response"
	"github.com/techagentng/citizenx/services/utils"
	// jwtPackage "github.com/techagentng/citizenx/services/jwt"
)

func (s *Server) HandleForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the email from the request
		email := struct {
			Email string `json:"email" binding:"required,email"`
		}{}
		errs := decode(c, &email)
		if errs != nil {
			response.JSON(c, "", http.StatusBadRequest, nil, errs)
			return
		}

		// Find user by email
		user, err := s.AuthRepository.FindUserByEmail(email.Email)
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

		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:3002"
		}
		// Create reset link
		resetLink := fmt.Sprintf("%s/reset-password/%s", baseURL, resetToken)

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
        // Step 1: Parse and validate the request
        var req ResetPasswordRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        // Step 2: Validate the reset token
        claims, err := utils.VerifyResetToken(req.Token)
        if err != nil || time.Now().After(time.Unix(claims.ExpiresAt, 0)) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
            return
        }

        // Step 3: Find the user associated with the token
        user, err := s.AuthRepository.FindUserByEmail(claims.Email)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
            return
        }

        // Step 4: Hash the new password
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
            return
        }

		// Step 5: Update the user's password in the database
		if err := s.AuthRepository.UpdateUserPassword(user, string(hashedPassword)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update password"})
			return
		}

        // Step 6: Respond with success
        c.JSON(http.StatusOK, gin.H{"message": "Password reset successful"})
    }
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
			return nil, errors.New("invalid token type", http.StatusAccepted)
		}
		return claims, nil
	}

	return nil, err
}