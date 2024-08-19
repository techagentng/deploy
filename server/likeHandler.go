package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/server/response"
)

// HandleUpvoteReport handles the upvoting of a report
func (s *Server) HandleUpvoteReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Upvote handler called")
		// Extract user ID and report ID from the request
		userIDCtx, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID not found in context", http.StatusInternalServerError))
			return
		}

		// Assert the type of userID as uint
		userID, ok := userIDCtx.(uint)
		if !ok {
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID is not of type uint", http.StatusInternalServerError))
			return
		}

		reportID := c.Param("reportID")
		err := s.LikeService.LikeReport(userID, reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Upvoted successfully"})
	}
}

// HandleDownvoteReport handles the downvoting of a report
func (s *Server) HandleDownvoteReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID and report ID from the request
		userIDCtx, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID not found in context", http.StatusInternalServerError))
			return
		}

		// Assert the type of userID as uint
		userID, ok := userIDCtx.(uint)
		if !ok {
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID is not of type uint", http.StatusInternalServerError))
			return
		}

		reportID := c.Param("reportID")
		err := s.LikeService.DislikeReport(userID, reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Downvoted successfully"})
	}
}
