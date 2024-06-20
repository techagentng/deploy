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
