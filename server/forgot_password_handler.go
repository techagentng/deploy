package server

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/server/response"
	"github.com/techagentng/citizenx/services"
	"github.com/techagentng/citizenx/services/jwt"
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
        resetToken, err := jwt.GeneratePasswordResetToken(user.ID, s.Config.JWTSecret)
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


func (s *Server) ResetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		resetPassword := struct {
			Password string `json:"password" binding:"required"`
		}{}
		errs := decode(c, &resetPassword)
		if errs != nil {
			response.JSON(c, "", http.StatusBadRequest, nil, errs)
			return
		}
		userID := c.Param("userID")
		hashedPassword, err := services.GenerateHashPassword(resetPassword.Password)
		if err != nil {
			log.Printf("Error: %v", err.Error())
			response.JSON(c, "", http.StatusInternalServerError, nil, errs)
			return
		}
		err = s.AuthRepository.ResetPassword(userID, string(hashedPassword))
		if err != nil {
			log.Printf("Error: %v", err.Error())
			response.JSON(c, "", http.StatusInternalServerError, nil, errs)
			return
		}
		response.JSON(c, "Password Reset Successfully", http.StatusOK, nil, nil)
	}
}
