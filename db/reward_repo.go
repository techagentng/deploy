package db

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

type RewardRepository interface {
	// GiftPoints(userID uint, amount float64) error
	// BuyPoints(userID uint, amount float64) error
	// RewardForReport(report models.IncidentReport) error
	GetRewardsByUserID(userID uint) (*models.Reward, error)
	SaveReward(reward *models.Reward) error
	GetReportByID(reportID string) (*models.IncidentReport, error)
	GetCurrentRewardByUserID(userID uint) (int, error)
	GetRewardPointByReportID(reportID string) (int, error)
	GetUserRewardBalance(userID uint) (int, error)
	SumAllRewardsBalance() (int, error)
	GetAllRewards() ([]models.Reward, error)
	GetUserReward(userID uint) (models.Reward, error)
}

type rewardRepo struct {
	DB *gorm.DB
}

func NewRewardRepo(db *GormDB) RewardRepository {
	return &rewardRepo{db.DB}
}

func (r *rewardRepo) GetRewardsByUserID(userID uint) (*models.Reward, error) {
	var reward models.Reward
	if err := r.DB.First(&reward, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &reward, nil
}

func (repo *rewardRepo) SaveReward(reward *models.Reward) error {
	var existingReward models.Reward
	err := repo.DB.Where("user_id = ? AND incident_report_id = ?", reward.UserID, reward.IncidentReportID).First(&existingReward).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new reward entry
			return repo.DB.Create(reward).Error
		}
		return err
	}

	// Update existing reward
	existingReward.Point += reward.Point
	existingReward.Balance = reward.Balance
	return repo.DB.Save(&existingReward).Error
}

func (r *rewardRepo) GetReportByID(reportID string) (*models.IncidentReport, error) {
	var report models.IncidentReport
	if err := r.DB.Where("id = ?", reportID).First(&report).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

// GetCurrentRewardByUserID fetches the current reward balance for a given user
func (repo *rewardRepo) GetCurrentRewardByUserID(userID uint) (int, error) {
	var reward models.Reward
	err := repo.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&reward).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Handle the case where no rewards exist for the user
			return 0, nil
		}
		return 0, err
	}
	return reward.Point, nil
}

func (repo *rewardRepo) GetUserRewardBalance(userID uint) (int, error) {
	var reward models.Reward
	// Query the database to find the reward for the given user ID
	err := repo.DB.Where("user_id = ?", userID).First(&reward).Error
	if err != nil {
		// If the error is "record not found", return 0 balance
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		// Return the error if it's not a "record not found" error
		return 0, fmt.Errorf("could not find reward for user: %v", err)
	}

	// Return the balance of the reward
	return reward.Balance, nil
}

func (r *rewardRepo) GetRewardPointByReportID(reportID string) (int, error) {
	// Initialize a variable to store the reward point
	var rewardPoint int
	// Query the reward table to get the point corresponding to the reportID
	if err := r.DB.Model(&models.Reward{}).
		Where("incident_report_id = ?", reportID).
		Pluck("point", &rewardPoint).
		Error; err != nil {
		return 0, err
	}

	return rewardPoint, nil
}

func (r *rewardRepo) SumAllRewardsBalance() (int, error) {
	var totalBalance int
	err := r.DB.Model(&models.Reward{}).Select("SUM(balance)").Scan(&totalBalance).Error
	if err != nil {
		return 0, err
	}
	return totalBalance, nil
}

func (r *rewardRepo) GetAllRewards() ([]models.Reward, error) {
	var rewards []models.Reward
	err := r.DB.Find(&rewards).Error
	if err != nil {
		return nil, err
	}
	return rewards, nil
}

func (r *rewardRepo) GetUserReward(userID uint) (models.Reward, error) {
    var reward models.Reward
    err := r.DB.Where("user_id = ?", userID).First(&reward).Error

    if err != nil {
        if err == gorm.ErrRecordNotFound {
            // Record not found, initialize a new reward record with default values
            reward = models.Reward{
                UserID:        userID,
                Point:         0,       // Default points
                Balance:       0,       // Default balance
                RewardType:    "default", // Set a default reward type or adjust as needed
                AccountNumber: "",      // Default or empty account number
            }
            // Create a new record
            err = r.DB.Create(&reward).Error
            if err != nil {
                return reward, err
            }
        } else {
            return reward, err
        }
    }

    return reward, nil
}