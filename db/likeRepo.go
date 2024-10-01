package db

import (
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

// LikeRepository interface
type LikeRepository interface {
	GetUserPoints(userID uint) (int, error)
	UpdateUserPoints(userID uint, points int) error
	RecordVote(userID uint, reportID string, voteType string) error
	BeginTransaction() *gorm.DB
	DownVoteReport(userID uint, reportID string) error
	UpvoteReport(userID uint, reportID string) error
	GetUpvoteAndDownvoteCounts(reportID string) (int, int, error)
}

// likeRepo struct
type likeRepo struct {
	DB *gorm.DB
}

// NewLikeRepo creates a new instance of LikeRepository
func NewLikeRepo(db *GormDB) LikeRepository {
	return &likeRepo{db.DB}
}

func (lk *likeRepo) UpvoteReport(userID uint, reportID string) error {
	// Start a transaction to ensure consistency
	tx := lk.DB.Begin()

	// Check if the user has already upvoted this report
	var existingVote models.Votes
	if err := tx.Where("user_id = ? AND report_id = ? AND vote_type = ?", userID, reportID, "upvote").First(&existingVote).Error; err == nil {
		// User has already upvoted, rollback and return
		log.Println("User has already upvoted, rolling back transaction")
		tx.Rollback()
		return nil
	}

	// Record the upvote in the votes table
	vote := models.Votes{
		UserID:   userID,
		ReportID: reportID,
		VoteType: "upvote",
	}
	if err := tx.Create(&vote).Error; err != nil {
		tx.Rollback() // Rollback transaction on error
		return err
	}

	// Update the upvote count by querying the Votes table
	var upvoteCount int64
	if err := tx.Model(&models.Votes{}).Where("report_id = ? AND vote_type = ?", reportID, "upvote").Count(&upvoteCount).Error; err != nil {
		log.Println("Failed to count upvotes, rolling back")
		tx.Rollback()
		return fmt.Errorf("failed to count upvotes: %w", err)
	}

	if err := tx.Model(&models.IncidentReport{}).Where("id = ?", reportID).Update("upvote_count", upvoteCount).Error; err != nil {
		log.Println("Failed to update upvote count, rolling back")
		tx.Rollback()
		return fmt.Errorf("failed to update upvote count: %w", err)
	}

	log.Println("Committing transaction")
	// Commit the transaction
	return tx.Commit().Error
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

// DownVoteReport handles the logic for disliking a report with transaction management
func (r *likeRepo) DownVoteReport(userID uint, reportID string) error {
	log.Printf("DownVoteReport called: userID = %d, reportID = %s", userID, reportID)

	// Start a transaction to ensure consistency
	tx := r.DB.Begin()

	// Check if the user has already downvoted this report
	var existingVote models.Votes
	if err := tx.Where("user_id = ? AND report_id = ? AND vote_type = ?", userID, reportID, "downvote").First(&existingVote).Error; err == nil {
		log.Println("User has already downvoted, rolling back transaction")
		tx.Rollback()
		return errors.New("user has already downvoted")
	}

	// Record the downvote in the votes table
	vote := models.Votes{
		UserID:   userID,
		ReportID: reportID,
		VoteType: "downvote",
	}
	if err := tx.Create(&vote).Error; err != nil {
		log.Println("Failed to record downvote, rolling back")
		tx.Rollback() // Rollback transaction on error
		return err
	}

	// Update the downvote count by querying the Votes table
	var downvoteCount int64
	if err := tx.Model(&models.Votes{}).Where("report_id = ? AND vote_type = ?", reportID, "downvote").Count(&downvoteCount).Error; err != nil {
		log.Println("Failed to count downvotes, rolling back")
		tx.Rollback()
		return fmt.Errorf("failed to count downvotes: %w", err)
	}

	if err := tx.Model(&models.IncidentReport{}).Where("id = ?", reportID).Update("downvote_count", downvoteCount).Error; err != nil {
		log.Println("Failed to update downvote count, rolling back")
		tx.Rollback()
		return fmt.Errorf("failed to update downvote count: %w", err)
	}

	log.Println("Committing transaction")
	// Commit the transaction
	return tx.Commit().Error
}

// GetUpvoteAndDownvoteCounts retrieves the upvote and downvote counts for a report.
func (r *likeRepo) GetUpvoteAndDownvoteCounts(reportID string) (int, int, error) {
	var report models.IncidentReport
	if err := r.DB.Select("upvote_count, downvote_count").Where("id = ?", reportID).First(&report).Error; err != nil {
		return 0, 0, err
	}
	return report.UpvoteCount, report.DownvoteCount, nil
}
