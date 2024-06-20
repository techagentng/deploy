package db

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

// LikeRepository interface
type LikeRepository interface {
	// Define methods for handling likes
	// Example:
	LikePost(userID uint, reportID string, like models.Like) error
	// UnlikePost(userID uint, postID uint) error
	// IsPostLikedByUser(userID uint, postID uint) (bool, error)
	// Add other methods as per your requirements
}

// likeRepo struct
type likeRepo struct {
	DB *gorm.DB
}

// NewLikeRepo creates a new instance of LikeRepository
func NewLikeRepo(db *GormDB) LikeRepository {
	return &likeRepo{db.DB}
}

func (repo *likeRepo) LikePost(userID uint, reportID string, like models.Like) error {
	// Begin transaction
	tx := repo.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if the report exists
	var report models.IncidentReport
	if err := tx.First(&report, "id = ?", reportID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Check if the user has already liked the report
	var existingLike models.Like
	if err := tx.Where("user_id = ? AND incident_report_id = ?", userID, reportID).First(&existingLike).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return err
	}

	// Increment or decrement the report's like count based on like action
	if existingLike.ID == "" {
		// User hasn't liked the report, create a new like
		newLike := models.Like{
			ID:     uuid.New().String(),
			UserID: userID,
			// IncidentReportID: reportID,
			Count: 1,
		}

		if err := tx.Create(&newLike).Error; err != nil {
			tx.Rollback()
			return err
		}

		report.LikeCount++
	} else {
		// User has already liked the report, toggle the like (unlike)
		existingLike.Count = 1 - existingLike.Count

		if err := tx.Save(&existingLike).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Update the report's like count based on the like toggle
		report.LikeCount += existingLike.Count*2 - 1
	}

	// Save the updated report with like count
	if err := tx.Save(&report).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	return tx.Commit().Error
}

// Implement other methods similarly
