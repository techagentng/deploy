package db

import (
	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

// LikeRepository interface
type LikeRepository interface {
	LikePost(userID uint, reportID string, like models.Like) error
	GetUserPoints(userID uint) (int, error)
	UpdateUserPoints(userID uint, points int) error
	RecordVote(userID uint, reportID string, voteType string) error
	BeginTransaction() *gorm.DB
	DislikeReport(userID uint, reportID string) error
}

// likeRepo struct
type likeRepo struct {
	DB *gorm.DB
}

// NewLikeRepo creates a new instance of LikeRepository
func NewLikeRepo(db *GormDB) LikeRepository {
	return &likeRepo{db.DB}
}


func (r *likeRepo) LikePost(userID uint, reportID string, like models.Like) error {
	return r.DB.Create(&like).Error
}

func (r *likeRepo) GetUserPoints(userID uint) (int, error) {
	var userPoints models.UserPoints
	if err := r.DB.Where("user_id = ?", userID).First(&userPoints).Error; err != nil {
		return 0, err
	}
	return userPoints.Points, nil
}

func (r *likeRepo) UpdateUserPoints(userID uint, points int) error {
	return r.DB.Model(&models.UserPoints{}).Where("user_id = ?", userID).Update("points", points).Error
}

func (r *likeRepo) RecordVote(userID uint, reportID string, voteType string) error {
	vote := models.Votes{UserID: userID, ReportID: reportID, VoteType: voteType}
	return r.DB.Create(&vote).Error
}

func (r *likeRepo) BeginTransaction() *gorm.DB {
	return r.DB.Begin()
}

// DislikeReport handles the logic for disliking a report with transaction management
func (r *likeRepo) DislikeReport(userID uint, reportID string) error {
	tx := r.BeginTransaction()
	if tx == nil {
		return errors.New("failed to start transaction")
	}
	defer tx.Rollback() // Ensure rollback on failure

	// Check if the user has already voted
	var existingVote models.Votes
	if err := tx.Where("user_id = ? AND report_id = ?", userID, reportID).First(&existingVote).Error; err == nil {
		return errors.New("user has already voted")
	}

	// Increment downvote count for the report
	if err := tx.Model(&models.IncidentReport{}).Where("id = ?", reportID).UpdateColumn("downvote_count", gorm.Expr("downvote_count + 1")).Error; err != nil {
		return err
	}

	// Update user points
	userPoints, err := r.GetUserPoints(userID)
	if err != nil {
		return err
	}

	newPoints := userPoints - 2
	if newPoints < 0 {
		newPoints = 0
	}

	if err := r.UpdateUserPoints(userID, newPoints); err != nil {
		return err
	}

	// Record the downvote
	if err := r.RecordVote(userID, reportID, "downvote"); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}