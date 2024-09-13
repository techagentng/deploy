package db

import (
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

// LikeRepository interface
type PostRepository interface {
    CreatePost(post *models.Post) error
}

// likeRepo struct
type postRepo struct {
	DB *gorm.DB
}

// NewLikeRepo creates a new instance of LikeRepository
func NewPostRepo(db *GormDB) PostRepository {
	return &postRepo{db.DB}
}

func (r *postRepo) CreatePost(post *models.Post) error {
    if err := r.DB.Create(post).Error; err != nil {
        return err
    }
    return nil
}
