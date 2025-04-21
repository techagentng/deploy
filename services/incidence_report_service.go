package services

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

type IncidentReportService interface {
	SaveReport(userID uint, lat float64, lng float64, report *models.IncidentReport, reportID string, totalPoints int) (*models.IncidentReport, error)
	GetAllReports(currentUserState string) ([]map[string]interface{}, error)
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
	BookmarkReport(userID uint, reportID uuid.UUID) error
	GetBookmarkedReports(userID uint) ([]models.IncidentReport, error)
	GetUserReports(userID uint) ([]models.ReportType, error)
	GetReportTypeCountsByLGA(lga string) (map[string]interface{}, error)
	AddMediaToReport(reportTypeID string, feedURLs, thumbnailURLs, fullsizeURLs []string) error
	GetReportCountByLGA(lga string) (int, error)
	GetReportCountByState(state string) (int, error)
	GetOverallReportCount() (int, error)
	GetGovernorDetails(stateName string) (*models.State, error)
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
    fmt.Println("Report ID:", reportID)

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

    // Check if user has previous reports
    hasRewardPoints, err := s.incidentRepo.HasPreviousReports(userID)
    if err != nil {
        return nil, fmt.Errorf("error checking previous reports: %v", err)
    }

    // Set reward details based on the user's previous reports
    if !hasRewardPoints {
        reward = &models.Reward{
            UserID:           userID,
            RewardType:       "New entry",
            Point:            reportPoints,
            IncidentReportID: reportID,
            Balance:          10, // Assuming new users get a balance of 10
        }
    } else {
        reward = &models.Reward{
            UserID:           userID,
            RewardType:       "Another entry",
            Point:            reportPoints,
            IncidentReportID: reportID,
        }
    }

    // Update the reward record in the repository
    if err := s.incidentRepo.UpdateReward(userID, reward); err != nil {
        return nil, fmt.Errorf("error creating reward: %v", err)
    }

    // Set the reward points on the report
    report.RewardPoint = reportPoints

    log.Printf("Fetching report type for category: %s", report.Category)

    // Fetch the ReportType based on category
    reportType, err := s.incidentRepo.GetReportTypeByCategory(report.Category)
    if err != nil {
        return nil, fmt.Errorf("error fetching report type: %v", err)
    }

    // Check if the reportType was found
    if reportType == nil {
        // If no report type found, we should create a new one
        newReportType := &models.ReportType{
            ID:       uuid.New(), // Ensure a new UUID is created for the ReportType
            Category: report.Category,
            Name:     report.Category + " Report", // Default name based on category
        }

        // Create a new report type
        if err := s.incidentRepo.CreateReportType(newReportType); err != nil {
            return nil, fmt.Errorf("error creating new report type: %v", err)
        }

        // Fetch the newly created ReportTypeID
        reportType = newReportType
    }

    // Assign the fetched (or newly created) ReportTypeID
    report.ReportTypeID = reportType.ID

    // Save the incident report
    savedReport, err := s.incidentRepo.SaveIncidentReport(report)
    if err != nil {
        return nil, fmt.Errorf("error saving report: %v", err)
    }

    // Construct the response object with saved report data
    reportResponse := &models.IncidentReport{
        DateOfIncidence:      savedReport.DateOfIncidence,
        Description:          savedReport.Description,
        FeedURLs:             savedReport.FeedURLs,
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
        ThumbnailURLs:        savedReport.ThumbnailURLs,
        StateName:            savedReport.StateName,
        LGAName:              savedReport.LGAName,
		IsAnonymous:    savedReport.IsAnonymous,
    }

    return reportResponse, nil
}


// Service function signature
func (s *IncidentService) GetAllReports(currentUserState string) ([]map[string]interface{}, error) {
	// Call the repository method to fetch reports, passing the current user state for filtering
	return s.incidentRepo.GetAllReports(currentUserState)
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

func (s *IncidentService) BookmarkReport(userID uint, reportID uuid.UUID) error {
	// First check if the report exists
	exists, err := s.incidentRepo.ReportExists(reportID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("report not found")
	}

	// Check if the report is already bookmarked by the user
	var bookmark models.Bookmark
	err = s.incidentRepo.IsBookmarked(userID, reportID, &bookmark)
	if err == nil {
		// Report is already bookmarked
		return errors.New("report already bookmarked")
	}

	// Only proceed if the error was "record not found"
	if err != gorm.ErrRecordNotFound {
		return err // Return any other unexpected errors
	}

	// Create a new bookmark
	newBookmark := models.Bookmark{
		UserID:   userID,
		ReportID: reportID,
	}

	// Save the bookmark
	return s.incidentRepo.SaveBookmark(&newBookmark)
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
func (s *IncidentService) AddMediaToReport(reportTypeID string, feedURLs, thumbnailURLs, fullsizeURLs []string) error {
	// Fetch the incident report by reportTypeID using the repository function
	incidentReport, err := s.incidentRepo.FindIncidentReportByReportTypeID(reportTypeID)
	if err != nil {
		return fmt.Errorf("failed to find incident report by report type ID: %v", err)
	}

	// Check if the incident report is valid
	if incidentReport == nil {
		return fmt.Errorf("invalid incident report ID: %s", reportTypeID)
	}

	// Iterate through media URLs and create media records associated with the incident report
	for i := 0; i < len(feedURLs); i++ {
		media := models.Media{
			IncidentReportID: incidentReport.ID,
			FeedURL:          feedURLs[i],
			ThumbnailURL:     thumbnailURLs[i],
			FullSizeURL:      fullsizeURLs[i],
		}

		// Use the repository to save the media record
		if err := s.incidentRepo.SaveMedia(&media); err != nil {
			return fmt.Errorf("error saving media: %v", err)
		}
	}

	return nil
}

// appendURLs ensures that the new URLs are appended correctly to the existing string of URLs
func appendURLs(existingURLs string, newURLs []string) string {
	if existingURLs != "" {
		return existingURLs + "," + strings.Join(newURLs, ",")
	}
	return strings.Join(newURLs, ",")
}

func (s *IncidentService) GetReportCountByLGA(lga string) (int, error) {
    return s.incidentRepo.GetReportCountByLGA(lga)
}

func (s *IncidentService) GetReportCountByState(state string) (int, error) {
    return s.incidentRepo.GetReportCountByState(state)
}

func (s *IncidentService) GetOverallReportCount() (int, error) {
    return s.incidentRepo.GetOverallReportCount()
}

func (s *IncidentService) GetGovernorDetails(stateName string) (*models.State, error) {
    return s.incidentRepo.GetGovernorDetails(stateName)
}

