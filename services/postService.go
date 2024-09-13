package services

import (
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	// "github.com/techagentng/citizenx/models"
)

// LikeService interface
type PostService interface {
}

// likeService struct
type postService struct {
	Config   *config.Config
	postRepo db.PostRepository
}

// NewLikeService creates a new instance of LikeService
func NewPostService(postRepo db.PostRepository, conf *config.Config) PostService {
	return &postService{
		postRepo: postRepo,
		Config:   conf,
	}
}
