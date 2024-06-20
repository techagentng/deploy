package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
)

// Handler function for liking a post

func (s *Server) handleLikeReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID and report ID from the request
		userIDCtx, ok := c.Get("userID")
		if !ok {
			// Handle the case where userID is not found in context
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID not found in context", http.StatusInternalServerError))
			return
		}

		// Assert the type of userID as uint
		userID, ok := userIDCtx.(uint)
		if !ok {
			// Handle the case where userID is not of type uint
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID is not of type uint", http.StatusInternalServerError))
			return
		}
		reportIDStr := c.Param("reportID")
		var likes models.Like
		LikeID := uuid.New()
		likeIDString := LikeID.String()
		likes.ID = likeIDString
		// Call the like service to like the report
		err := s.LikeService.LikeReport(userID, reportIDStr)
		if err != nil {
			response.JSON(c, "", http.StatusInternalServerError, nil, err)
			return
		}

		// Respond with success message
		response.JSON(c, "Report liked successfully", http.StatusOK, nil, nil)
	}
}
