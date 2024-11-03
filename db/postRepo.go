package db

import (
	"fmt"

	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

// LikeRepository interface
type PostRepository interface {
	CreatePost(post *models.Post) error
	GetPostsByUserID(userID uint) ([]models.Post, error)
	GetAllPosts() ([]models.Post, error)
	GetPostByID(id string) (*models.Post, error)
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

// GetPostsByUserID fetches all posts created by a specific user based on userID
func (r *postRepo) GetPostsByUserID(userID uint) ([]models.Post, error) {
	var posts []models.Post
	// Fetch posts where the userID matches
	err := r.DB.Where("user_id = ?", userID).Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *postRepo) GetAllPosts() ([]models.Post, error) {
	var posts []models.Post

	if err := r.DB.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *postRepo) GetPostByID(id string) (*models.Post, error) {
	var post models.Post
	if err := r.DB.Where("id = ?", id).First(&post).Error; err != nil {
		return nil, fmt.Errorf("error retrieving post with ID %s: %w", id, err)
	}
	return &post, nil
}
