package db

import (
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
	DislikeReport(userID uint, reportID string) error
	UpvoteReport(userID uint, reportID string) error
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
    // Check if the user has already upvoted this report
    var existingVote models.Votes
    if err := lk.DB.Where("user_id = ? AND report_id = ? AND vote_type = ?", userID, reportID, "upvote").First(&existingVote).Error; err == nil {
        // User has already upvoted, do nothing
        return nil
    }

    // If no existing upvote, proceed to increment the upvote count
    if err := lk.DB.Model(&models.IncidentReport{}).Where("id = ?", reportID).UpdateColumn("upvote_count", gorm.Expr("upvote_count + 1")).Error; err != nil {
        return err
    }

    // Record the upvote in the votes table
    vote := models.Votes{
        UserID:   userID,
        ReportID: reportID,
        VoteType: "upvote",
    }
    if err := lk.DB.Create(&vote).Error; err != nil {
        return err
    }

    return nil
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
    log.Printf("DislikeReport called: userID = %d, reportID = %s", userID, reportID)
    
    tx := r.BeginTransaction()
    if tx == nil {
        return errors.New("failed to start transaction")
    }
    defer tx.Rollback()

    var existingVote models.Votes
    if err := tx.Where("user_id = ? AND report_id = ?", userID, reportID).First(&existingVote).Error; err == nil {
        return errors.New("user has already downvoted")
    }

    if err := tx.Model(&models.IncidentReport{}).Where("id = ?", reportID).UpdateColumn("downvote_count", gorm.Expr("downvote_count + 1")).Error; err != nil {
        return err
    }

    if err := r.RecordVote(userID, reportID, "downvote"); err != nil {
        return err
    }

    return tx.Commit().Error
}

