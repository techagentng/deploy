package services

import (
	"errors"
	"fmt"
	"math"

	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

type IncidentReportService interface {
	SaveReport(userID uint, lat float64, lng float64, report *models.IncidentReport, reportID string, totalPoints int) (*models.IncidentReport, error)
	GetAllReports(page int) ([]models.IncidentReport, error)
	GetAllReportsByState(state string, page int) ([]models.IncidentReport, error)
	GetAllReportsByLGA(lga string, page int) ([]models.IncidentReport, error)
	GetAllReportsByReportType(reportType string, page int) ([]models.IncidentReport, error)
	GetReportPercentageByState() ([]models.StateReportPercentage, error)
	GetTotalUserCount() (int64, error)
	GetRegisteredUsersCountByLGA(lga string) (int64, error)
	GetReportsByTypeAndLGA(reportType string, lga string) ([]models.SubReport, error)
	GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, []models.StateReportCount, error)
	ListAllStatesWithReportCounts() ([]models.StateReportCount, error)
	GetTotalReportCount() (int64, error)
	GetNamesByCategory(stateName string, lgaID string, reportTypeCategory string) ([]string, error)
	BookmarkReport(userID uint, reportID string) error
	GetBookmarkedReports(userID uint) ([]models.IncidentReport, error)
	GetUserReports(userID uint) ([]models.ReportType, error)
	GetReportTypeCountsByLGA(lga string) (map[string]interface{}, error)
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
		UserFullname: report.UserFullname,
		UserUsername: report.UserUsername,
	}

	// Save the report to the database
	_, err = s.incidentRepo.SaveIncidentReport(report)
	if err != nil {
		return nil, fmt.Errorf("error saving report: %v", err)
	}

	return reportResponse, nil
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

func (s *IncidentService) GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, []models.StateReportCount, error) {
    reportTypes, counts, totalUsers, totalReports, topStates, err := s.incidentRepo.GetReportTypeCounts(state, lga, startDate, endDate)
    if err != nil {
        return nil, nil, 0, 0, nil, err
    }
    return reportTypes, counts, totalUsers, totalReports, topStates, nil
}

func (s *IncidentService) ListAllStatesWithReportCounts() ([]models.StateReportCount, error) {
    return s.incidentRepo.ListAllStatesWithReportCounts()
}

func (s *IncidentService) GetTotalReportCount() (int64, error) {
    return s.incidentRepo.GetTotalReportCount()
}

func (s *IncidentService) GetNamesByCategory(stateName string, lgaID string, reportTypeCategory string) ([]string, error) {
    // Call the repository method with the correct parameters
    names, err := s.incidentRepo.GetNamesByCategory(stateName, lgaID, reportTypeCategory)
    if err != nil {
        return nil, fmt.Errorf("error getting names by category: %v", err)
    }
    return names, nil
}

func (s *IncidentService) BookmarkReport(userID uint, reportID string) error {
	// Check if the report is already bookmarked by the user
	var bookmark models.Bookmark
	err := s.incidentRepo.IsBookmarked(userID, reportID, &bookmark)
	if err == nil {
		// Report is already bookmarked, return an appropriate response
		return errors.New("report already bookmarked")
	}

	// Create a new bookmark
	bookmark = models.Bookmark{
		UserID:   userID,
		ReportID: reportID,
	}

	// Save the bookmark
	return s.incidentRepo.SaveBookmark(&bookmark)
}

func (s *IncidentService) GetBookmarkedReports(userID uint) ([]models.IncidentReport, error) {
    // Call the repository method to get the bookmarked reports
    return s.incidentRepo.GetBookmarkedReports(userID)
}

func (s *IncidentService) GetUserReports(userID uint) ([]models.ReportType, error) {
    return s.incidentRepo.GetReportsByUserID(userID)
}

func (s *IncidentService) GetReportTypeCountsByLGA(lga string) (map[string]interface{}, error) {
    return s.incidentRepo.GetReportTypeCountsByLGA(lga)
}
