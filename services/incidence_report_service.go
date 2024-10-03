package services

import (
	"errors"
	"fmt"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
	"math"
	"strings"
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
	AddMediaToReport(reportID string, feedURLs []string, thumbnailURLs []string, fullsizeURLs []string) error
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

	// Initialize reward and points calculation
	var reward *models.Reward
	mediaPoints := totalPoints * 10
	var descPoint, locationPoint int

	if report.Description != "" {
		descPoint += 10
	}
	if !math.IsNaN(lat) && !math.IsNaN(lng) {
		locationPoint += 10
	}

	reportPoints := locationPoint + descPoint + mediaPoints

	// Check if the user has previous reports for rewards
	hasRewardPoints, err := s.incidentRepo.HasPreviousReports(userID)
	if err != nil {
		return nil, err
	}

	if !hasRewardPoints {
		reward = &models.Reward{
			UserID:           userID,
			RewardType:       "New entry",
			Point:            reportPoints,
			IncidentReportID: reportID,
			Balance:          10,
		}
	} else {
		reward = &models.Reward{
			UserID:           userID,
			RewardType:       "Another entry",
			Point:            reportPoints,
			IncidentReportID: reportID,
		}
	}

	// Update the user's rewards
	if err := s.incidentRepo.UpdateReward(userID, reward); err != nil {
		return nil, fmt.Errorf("error creating reward: %v", err)
	}

	// Set the report points for the incident report
	report.RewardPoint = reportPoints

	savedReport, err := s.incidentRepo.SaveIncidentReport(report)
	if err != nil {
		return nil, fmt.Errorf("error saving report: %v", err)
	}

	// Prepare the report response
	reportResponse := &models.IncidentReport{
		ID:                   savedReport.ID,
		DateOfIncidence:      savedReport.DateOfIncidence,
		Description:          savedReport.Description,
		FeedURLs:             savedReport.FeedURLs,
		ThumbnailURLs:        savedReport.ThumbnailURLs,
		FullSizeURLs:         savedReport.FullSizeURLs,
		ProductName:          savedReport.ProductName,
		StateName:            savedReport.StateName,
		LGAName:              savedReport.LGAName,
		Latitude:             lat,
		Longitude:            lng,
		UserIsAnonymous:      false,
		Address:              savedReport.Address,
		IsVerified:           false,
		UserID:               savedReport.UserID,
		ReportTypeID:         savedReport.ReportTypeID,
		AdminID:              0,
		Landmark:             savedReport.Landmark,
		LikeCount:            savedReport.LikeCount,
		BookmarkedReports:    []*models.User{},
		IsResponse:           savedReport.IsResponse,
		TimeofIncidence:      savedReport.TimeofIncidence,
		ReportStatus:         savedReport.ReportStatus,
		RewardPoint:          savedReport.RewardPoint,
		ActionTypeName:       savedReport.ActionTypeName,
		Rating:               savedReport.Rating,
		HospitalName:         savedReport.HospitalName,
		Department:           savedReport.Department,
		DepartmentHeadName:   savedReport.DepartmentHeadName,
		AccidentCause:        savedReport.AccidentCause,
		SchoolName:           savedReport.SchoolName,
		VicePrincipal:        savedReport.VicePrincipal,
		OutageLength:         savedReport.OutageLength,
		AirportName:          savedReport.AirportName,
		Country:              savedReport.Country,
		StateEmbassyLocation: savedReport.StateEmbassyLocation,
		NoWater:              savedReport.NoWater,
		AmbassedorsName:      savedReport.AmbassedorsName,
		HospitalAddress:      savedReport.HospitalAddress,
		RoadName:             savedReport.RoadName,
		AirlineName:          savedReport.AirlineName,
		Category:             savedReport.Category,
		UserFullname:         savedReport.UserFullname,
		UserUsername:         savedReport.UserUsername,
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

// AddMediaToReport associates media URLs (feed, thumbnail, fullsize) with an incident report.
func (s *IncidentService) AddMediaToReport(reportID string, feedURLs []string, thumbnailURLs []string, fullsizeURLs []string) error {
	// Fetch the existing report from the database using the report ID.
	report, err := s.incidentRepo.GetIncidentReportByID(reportID)
	if err != nil {
		return fmt.Errorf("error retrieving report with ID %s: %v", reportID, err)
	}

	// Append the new media URLs to the report's existing URLs
	if report.FeedURLs != "" {
		report.FeedURLs = report.FeedURLs + "," + strings.Join(feedURLs, ",")
	} else {
		report.FeedURLs = strings.Join(feedURLs, ",")
	}

	if report.ThumbnailURLs != "" {
		report.ThumbnailURLs = report.ThumbnailURLs + "," + strings.Join(thumbnailURLs, ",")
	} else {
		report.ThumbnailURLs = strings.Join(thumbnailURLs, ",")
	}

	if report.FullSizeURLs != "" {
		report.FullSizeURLs = report.FullSizeURLs + "," + strings.Join(fullsizeURLs, ",")
	} else {
		report.FullSizeURLs = strings.Join(fullsizeURLs, ",")
	}

	// Update the incident report in the database with the new media URLs.
	err = s.incidentRepo.UpdateIncidentReport(report)
	if err != nil {
		return fmt.Errorf("error updating incident report with media URLs: %v", err)
	}

	// Also update the related report type information.
	err = s.incidentRepo.UpdateReportTypeWithIncidentReport(report)
	if err != nil {
		return fmt.Errorf("error updating report type for incident report ID %s: %v", reportID, err)
	}

	return nil
}
