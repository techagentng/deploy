package services

import (
	"fmt"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

type RewardService interface {
	ApproveReportPoints(reportID string, userID uint) error
	RejectReportPoints(reportID string, userID uint) error
	AcceptReportPoints(reportID string, userID uint) error
	SaveReward(reward *models.Reward) error
	GetAllRewardsBalanceCount() (int, error)
	GetAllRewards() ([]models.Reward, error)
}

type rewardService struct {
	Config       *config.Config
	rewardRepo   db.RewardRepository
	incidentRepo db.IncidentReportRepository
}

func NewRewardService(rewardRepo db.RewardRepository, incidentRepo db.IncidentReportRepository, conf *config.Config) RewardService {
	return &rewardService{
		Config:       conf,
		rewardRepo:   rewardRepo,
		incidentRepo: incidentRepo,
	}
}

func (s *rewardService) ApproveReportPoints(reportID string, userID uint) error {
	report, err := s.incidentRepo.GetReportByID(reportID)
	var reward models.Reward
	if err != nil {
		return fmt.Errorf("error fetching report: %v", err)
	}

	//get user reward points
	points, err := s.rewardRepo.GetRewardPointByReportID(reportID)
	if err != nil {
		return err
	}
	//update reward balance with the points value
	report.ReportStatus = "approved"
	if _, err := s.incidentRepo.UpdateIncidentReport(report); err != nil {
		return fmt.Errorf("error updating report status: %v", err)
	}

	newBalance := reward.Balance + points
	reward = models.Reward{
		Model:            models.Model{},
		IncidentReportID: reportID,
		UserID:           userID,
		RewardType:       "Another Entry",
		Point:            points,
		Balance:          newBalance,
		AccountNumber:    "",
	}

	if err := s.rewardRepo.SaveReward(&reward); err != nil {
		return fmt.Errorf("error saving reward: %v", err)
	}

	return nil
}

func (s *rewardService) RejectReportPoints(reportID string, userID uint) error {
	report, err := s.incidentRepo.GetReportByID(reportID)
	if err != nil {
		return fmt.Errorf("error fetching report: %v", err)
	}

	//update reward balance with the points value
	report.ReportStatus = "rejected"
	if _, err := s.incidentRepo.UpdateIncidentReport(report); err != nil {
		return fmt.Errorf("error updating report status: %v", err)
	}

	return nil
}

func (s *rewardService) AcceptReportPoints(reportID string, userID uint) error {
	report, err := s.incidentRepo.GetReportByID(reportID)
	if err != nil {
		return fmt.Errorf("error fetching report: %v", err)
	}

	//update reward balance with the points value
	report.ReportStatus = "accepted"
	if _, err := s.incidentRepo.UpdateIncidentReport(report); err != nil {
		return fmt.Errorf("error updating report status: %v", err)
	}

	return nil
}

func (s *rewardService) SaveReward(reward *models.Reward) error {
	return s.rewardRepo.SaveReward(reward)
}

func (s *rewardService) GetAllRewardsBalanceCount() (int, error) {
	totalBalance, err := s.rewardRepo.SumAllRewardsBalance()
	if err != nil {
		return 0, fmt.Errorf("error getting total rewards balance: %w", err)
	}
	return totalBalance, nil
}

func (s *rewardService) GetAllRewards() ([]models.Reward, error) {
	rewards, err := s.rewardRepo.GetAllRewards()
	if err != nil {
		return nil, fmt.Errorf("error getting all rewards: %w", err)
	}
	return rewards, nil
}
