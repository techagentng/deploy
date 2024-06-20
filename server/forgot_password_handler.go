package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/server/response"
	"github.com/techagentng/citizenx/services"
	// jwtPackage "github.com/techagentng/citizenx/services/jwt"
)

func (s *Server) HandleForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := struct {
			Email string `json:"email" binding:"required"`
		}{}
		errs := decode(c, &email)
		if errs != nil {
			response.JSON(c, "", http.StatusBadRequest, nil, errs)
			return
		}

		user, err := s.AuthRepository.FindUserByEmail(email.Email)
		if err != nil {
			response.JSON(c, "", http.StatusNotFound, nil, errors.New("user not found", http.StatusNotFound))
			return
		}

		// Check if user is nil
		if user == nil {
			response.JSON(c, "", http.StatusNotFound, nil, errors.New("user not found", http.StatusNotFound))
			return
		}

		//link := fmt.Sprintf("%s/verifyEmail/%s", s.Config.BaseUrl, token)
		_, err = s.Mail.SendResetPassword(user.Email, fmt.Sprintf("http://localhost:3000/reset-password/%d", user.ID))
		if err != nil {
			response.JSON(c, "connection to mail service interupted", http.StatusInternalServerError, nil, err)
			return
		}
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
