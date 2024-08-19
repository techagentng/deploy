package services

import (
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

// LikeService interface
type LikeService interface {
	LikeReport(userID uint, reportID string) error
	DislikeReport(userID uint, reportID string) error
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

// LikeReport handles the logic for liking a report
func (lk *likeService) LikeReport(userID uint, reportID string) error {
	var like models.Like
	like.UserID = userID
	return lk.likeRepo.LikePost(userID, reportID, like)
}

// DislikeReport handles the logic for disliking a report
func (lk *likeService) DislikeReport(userID uint, reportID string) error {
	return lk.likeRepo.DislikeReport(userID, reportID)
}

