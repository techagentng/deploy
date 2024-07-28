package services

import (
	"fmt"
	"math"

	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

type IncidentReportService interface {
	SaveReport(userID uint, lat float64, lng float64, report *models.IncidentReport, reportID string, totalPoints int) (*models.IncidentReport, error)
	CheckReportInBookmarkedReport(userID uint, reportID string) (bool, error)
	SaveBookmarkReport(bookmark *models.BookmarkReport) error
	GetAllReports(page int) ([]models.IncidentReport, error)
	GetAllReportsByState(state string, page int) ([]models.IncidentReport, error)
	GetAllReportsByLGA(lga string, page int) ([]models.IncidentReport, error)
	GetAllReportsByReportType(reportType string, page int) ([]models.IncidentReport, error)
	GetReportPercentageByState() ([]models.StateReportPercentage, error)
	GetTotalUserCount() (int64, error)
	GetRegisteredUsersCountByLGA(lga string) (int64, error)
	GetReportsByTypeAndLGA(reportType string, lga string) ([]models.SubReport, error)
	GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, error)
	// GetStateReportCounts() ([]models.StateReportCount, error)
}

type IncidentService struct {
	Config       *config.Config
	incidentRepo db.IncidentReportRepository
	rewardRepo   db.RewardRepository
	mediaRepo    db.MediaRepository
}

// NewIncidentReportService instantiates an IncidentReportService
func NewIncidentReportService(incidentReportRepo db.IncidentReportRepository, rewardRepo db.RewardRepository, mediaRepo db.MediaRepository, conf *config.Config) *IncidentService {
	return &IncidentService{
		Config:       conf,
		incidentRepo: incidentReportRepo,
		rewardRepo:   rewardRepo,
		mediaRepo:    mediaRepo,
	}
}

func (s *IncidentService) SaveReport(userID uint, lat float64, lng float64, report *models.IncidentReport, reportID string, totalPoints int) (*models.IncidentReport, error) {
	var reward *models.Reward
	// Calculate points based on media counts and other factors
	mediaPoints := totalPoints * 10
	var descPoint int
	var locationPoint int

	if report.Description != "" {
		descPoint += 10
	}
	if !math.IsNaN(lat) && !math.IsNaN(lng) {
		locationPoint += 10
	}

	reportPoints := locationPoint + descPoint + mediaPoints
	hasRardPoints, err := s.incidentRepo.HasPreviousReports(userID)
	if err != nil {
		return nil, err
	}
	if !hasRardPoints {
		reward = &models.Reward{
			UserID:           userID,
			RewardType:       "New entry",
			Point:            reportPoints,
			IncidentReportID: reportID,
			Balance:          10,
		}
	} else {
		// Award a reward for the new report
		reward = &models.Reward{
			UserID:           userID,
			RewardType:       "Another entry",
			Point:            reportPoints,
			IncidentReportID: reportID,
		}
	}
	// Save the reward to the database
	if err := s.incidentRepo.UpdateReward(userID, reward); err != nil {
		return nil, fmt.Errorf("error creating reward: %v", err)
	}

	report.RewardPoint = reportPoints
	reportResponse := &models.IncidentReport{
		ID:                   report.ID,
		CreatedAt:            0,
		UserFullname:         "",
		DateOfIncidence:      report.DateOfIncidence,
		Description:          report.Description,
		FeedURLs:             report.FeedURLs,
		ThumbnailURLs:        report.ThumbnailURLs,
		FullSizeURLs:         report.FullSizeURLs,
		ProductName:          report.ProductName,
		StateName:            report.StateName,
		LGAName:              report.LGAName,
		Latitude:             lat,
		Longitude:            lng,
		UserIsAnonymous:      false,
		Address:              report.Address,
		UserUsername:         "",
		Telephone:            "",
		Email:                "",
		View:                 0,
		IsVerified:           false,
		UserID:               report.UserID,
		ReportTypeID:         reportID,
		AdminID:              0,
		Landmark:             report.Landmark,
		LikeCount:            report.LikeCount,
		BookmarkedReports:    []*models.User{},
		IsResponse:           report.IsResponse,
		TimeofIncidence:      report.TimeofIncidence,
		ReportStatus:         report.ReportStatus,
		RewardPoint:          report.RewardPoint,
		RewardAccountNumber:  "",
		ActionTypeName:       report.ActionTypeName,
		ReportTypeName:       report.ReportTypeName,
		IsState:              false,
		Rating:               report.Rating,
		HospitalName:         report.HospitalName,
		Department:           report.Department,
		DepartmentHeadName:   report.DepartmentHeadName,
		AccidentCause:        report.AccidentCause,
		SchoolName:           report.SchoolName,
		VicePrincipal:        report.VicePrincipal,
		OutageLength:         report.OutageLength,
		AirportName:          report.AirlineName,
		Country:              report.Country,
		StateEmbassyLocation: report.StateEmbassyLocation,
		NoWater:              report.NoWater,
		AmbassedorsName:      report.AmbassedorsName,
		HospitalAddress:      report.HospitalAddress,
		RoadName:             report.RoadName,
		AirlineName:          report.AirlineName,
		Category: report.Category,
	}

	// Save the report to the database
	_, err = s.incidentRepo.SaveIncidentReport(report)
	if err != nil {
		return nil, fmt.Errorf("error saving report: %v", err)
	}

	return reportResponse, nil
}

func (s *IncidentService) CheckReportInBookmarkedReport(userID uint, reportID string) (bool, error) {
	ok, err := s.incidentRepo.CheckReportInBookmarkedReport(userID, reportID)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *IncidentService) SaveBookmarkReport(bookmark *models.BookmarkReport) error {
	err := s.incidentRepo.SaveBookmarkReport(bookmark)
	if err != nil {
		// Handle error
		return err
	}
	return nil
}

func (s *IncidentService) GetAllReports(page int) ([]models.IncidentReport, error) {
	return s.incidentRepo.GetAllReports(page)
}

func (s *IncidentService) GetAllReportsByState(state string, page int) ([]models.IncidentReport, error) {
	return s.incidentRepo.GetAllReportsByState(state, page)
}

func (s *IncidentService) GetAllReportsByLGA(lga string, page int) ([]models.IncidentReport, error) {
	return s.incidentRepo.GetAllReportsByLGA(lga, page)
}

func (s *IncidentService) GetAllReportsByReportType(lga string, page int) ([]models.IncidentReport, error) {
	return s.incidentRepo.GetAllReportsByReportType(lga, page)
}

func (s *IncidentService) GetReportPercentageByState() ([]models.StateReportPercentage, error) {
	return s.incidentRepo.GetReportPercentageByState()
}

func (s *IncidentService) GetTotalUserCount() (int64, error) {
	return s.incidentRepo.GetTotalUserCount()
}

func (s *IncidentService) GetRegisteredUsersCountByLGA(lga string) (int64, error) {
	return s.incidentRepo.GetRegisteredUsersCountByLGA(lga)
}

func (s *IncidentService) GetReportsByTypeAndLGA(reportType string, lga string) ([]models.SubReport, error) {
	reports, err := s.incidentRepo.GetReportsByTypeAndLGA(reportType, lga)
	if err != nil {
		return nil, fmt.Errorf("error getting reports by type and LGA: %w", err)
	}
	return reports, nil
}

func (s *IncidentService) GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, error) {
	return s.incidentRepo.GetReportTypeCounts(state, lga, startDate, endDate)
}
