package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleSumAllRewardsBalance() gin.HandlerFunc {
	return func(c *gin.Context) {
		totalBalance, err := s.RewardService.GetAllRewardsBalanceCount()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"total_balance": totalBalance})
	}
}

func (s *Server) handleGetAllRewardsList() gin.HandlerFunc {
	return func(c *gin.Context) {
		rewards, err := s.RewardService.GetAllRewards()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rewards)
	}
}

func (s *Server) handleGetUserRewardBalance() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the context
		userIDCtx, ok := c.Get("userID")
		if !ok {
			// Handle the case where the user ID is not found in the context
			c.JSON(http.StatusInternalServerError, gin.H{"error": "userID not found in context"})
			return
		}

		// Convert userIDCtx to uint type (assuming userID is uint)
		userID, ok := userIDCtx.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid userID type"})
			return
		}

		// Fetch the reward balance for the user using the service
		balance, err := s.RewardRepository.GetUserRewardBalance(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return the balance as an integer in the response
		c.JSON(http.StatusOK, gin.H{
			"balance": balance,
		})
	}
}
