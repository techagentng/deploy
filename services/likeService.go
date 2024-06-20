package services

import (
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

// LikeService interface
type LikeService interface {
	// Define methods for handling likes
	// Example:
	LikeReport(userID uint, postID string) error
	// UnlikeReport(userID uint, reportID uint64) error
	// UnlikePost(userID uint, postID uint) error
	// IsPostLikedByUser(userID uint, postID uint) (bool, error)
	// Add other methods as per your requirements
}

// likeService struct
type likeService struct {
	Config   *config.Config
	likeRepo db.LikeRepository
}

// NewLikeService creates a new instance of LikeService
func NewLikeService(likeRepo db.LikeRepository, conf *config.Config) LikeService {
	return &likeService{
		likeRepo: likeRepo,
		Config:   conf,
	}
}

func (lk *likeService) LikeReport(userID uint, reportID string) error {
	var like models.Like
	like.UserID = userID
	return lk.likeRepo.LikePost(userID, reportID, like)
}
