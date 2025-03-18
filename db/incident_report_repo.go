package db

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

const (
	DefaultPageSize = 20
	DefaultPage     = 1
)

type IncidentReportRepository interface {
	SaveIncidentReport(report *models.IncidentReport) (*models.IncidentReport, error)
	HasPreviousReports(userID uint) (bool, error)
	UpdateReward(userID uint, reward *models.Reward) error
	FindUserByID(id uint) (*models.UserResponse, error)
	GetReportByID(report_id string) (*models.IncidentReport, error)
	GetAllReports(page int) ([]map[string]interface{}, error)
	GetAllReportsByState(state string, page int) ([]models.IncidentReport, error)
	GetAllReportsByLGA(lga string, page int) ([]models.IncidentReport, error)
	GetAllReportsByReportType(lga string, page int) ([]models.IncidentReport, error)
	GetReportPercentageByState() ([]models.StateReportPercentage, error)
	Save(report *models.IncidentReport) error
	GetReportStatusByID(reportID string) (string, error)
	UpdateIncidentReport(report *models.IncidentReport) error
	GetReportsPostedTodayCount() (int64, error)
	GetTotalUserCount() (int64, error)
	GetRegisteredUsersCountByLGA(lga string) (int64, error)
	GetAllReportsByStateByTime(state string, startTime, endTime time.Time, page int) ([]models.IncidentReport, error)
	GetReportsByTypeAndLGA(reportType string, lga string) ([]models.SubReport, error)
	GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, []models.StateReportCount, error)
	SaveStateLgaReportType(lga *models.LGA, state *models.State) error
	GetIncidentMarkers() ([]Marker, error)
	DeleteByID(id string) error
	GetStateReportCounts() ([]models.StateReportCount, error)
	GetVariadicStateReportCounts(reportTypes []string, states []string, startDate, endDate *time.Time) ([]models.StateReportCount, error)
	GetAllCategories() ([]string, error)
	GetAllStates() ([]string, error)
	GetRatingPercentages(reportType, state string) (*models.RatingPercentage, error)
	GetReportCountsByStateAndLGA() ([]models.ReportCount, error)
	ListAllStatesWithReportCounts() ([]models.StateReportCount, error)
	GetTotalReportCount() (int64, error)
	GetNamesByCategory(stateName string, lgaID string, reportTypeCategory string) ([]string, error)
	UploadMediaToS3(file multipart.File, fileHeader *multipart.FileHeader, bucketName, folderName string) (string, error)
	SaveReportType(reportType *models.ReportType) (*models.ReportType, error)
	SaveSubReport(subReport *models.SubReport) (*models.SubReport, error)
	GetSubReportsByCategory(category string) ([]models.SubReport, error)
	IsBookmarked(userID uint, reportID uuid.UUID, bookmark *models.Bookmark) error
	SaveBookmark(bookmark *models.Bookmark) error
	GetBookmarkedReports(userID uint) ([]models.IncidentReport, error)
	GetReportsByUserID(userID uint) ([]models.ReportType, error)
	GetReportTypeCountsByLGA(lga string) (map[string]interface{}, error)
	GetReportCountsByState(state string) ([]string, []int, error)
	GetTopCategories() ([]string, []int, error)
	GetReportsByCategoryAndReportID(category string, reportID string) ([]models.ReportType, error)
	GetReportsByCategory(category string) ([]models.ReportType, error)
	GetFilteredIncidentReports(category, state, lga string) ([]models.IncidentReport, []string, error)
	GetIncidentReportByID(reportID string) (*models.IncidentReport, error)
	UpdateReportTypeWithIncidentReport(report *models.IncidentReport) error
	FindReportTypeByCategory(category string, reportType *models.ReportType) error
	GetReportTypeByCategory(category string) (*models.ReportType, error)
	GetIncidentReportByReportTypeID(reportTypeID string) (*models.IncidentReport, error)
	FindIncidentReportByReportTypeID(reportTypeID string) (*models.IncidentReport, error)
	SaveMedia(media *models.Media) error
	GetReportIDByUser(ctx context.Context, userID uint) (uuid.UUID, error)
	GetReportTypeeByID(reportTypeID string) (*models.ReportType, error)
	// GetLastReportIDByUserID(userID uint) (string, error)
	GetAllIncidentReportsByUser(userID uint, page int) ([]map[string]interface{}, error)
	ReportExists(reportID uuid.UUID) (bool, error)
	UpdateBlockRequest(ctx context.Context, reportID uuid.UUID) error
	BlockUser(ctx context.Context, userID uint) error
	ReportUser(ctx context.Context, userID uint) error
	CreateFollow(follow models.Follow) error
	GetFollowersByReport(reportID uuid.UUID) ([]models.User, error)
	GetOAuthState(state string) (*models.OAuthState, error)
	SaveOAuthState(oauthState *models.OAuthState) error
	GetReportCountByLGA(lga string) (int, error)
	GetReportCountByState(state string) (int, error)
	GetOverallReportCount() (int, error)
	CreateReportType(reportType *models.ReportType) error
	GetLastReportIDByUserID(userID uint) (string, error)
	GetGovernorDetails(stateName string) (*models.State, error)
	CreateState(state *models.State) error
	GetAllStatesRatingPercentages(reportType string) (map[string]*models.RatingPercentage, error)
	FetchStates() ([]models.State, error)
	FetchLGAs() ([]models.LGA, error)
	FetchStateByName(stateName string) (*models.State, error)
}

type incidentReportRepo struct {
	DB *gorm.DB
}

func NewIncidentReportRepo(db *GormDB) IncidentReportRepository {
	return &incidentReportRepo{db.DB}
}
var (
    ErrStateNotFound = errors.New("state not found")
    ErrDatabase      = errors.New("database error")
)

// GetLastReportIDByUserID fetches the last report ID created by a given user.
func (i *incidentReportRepo) GetLastReportIDByUserID(userID uint) (string, error) {
	var reportID string

	// Query to get the last report ID for the given user
	err := i.DB.Raw(`
		SELECT id FROM incident_reports 
		WHERE user_id = ? 
		ORDER BY created_at DESC 
		LIMIT 1
	`, userID).Scan(&reportID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("No reports found for user ID: %d", userID)
			return "", fmt.Errorf("no reports found for user ID: %d", userID)
		}
		log.Printf("Error fetching last report ID: %v", err)
		return "", err
	}

	return reportID, nil
}


func (i *incidentReportRepo) UpdateReward(userID uint, reward *models.Reward) error {
	// Find the existing reward for the user
	existingReward := &models.Reward{}

	// Retrieve the existing reward from the database
	if err := i.DB.Where("user_id = ?", userID).First(existingReward).Error; err != nil {
		// Check if the error is due to record not found
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If record not found, create a new reward with the provided details
			// and save it to the database
			if err := i.DB.Create(reward).Error; err != nil {
				return err
			}
			return nil
		}
		// Return other errors
		return err
	}

	// Use COALESCE to handle NULL sums
	var totalBalance sql.NullInt64
	err := i.DB.Table("rewards").Select("COALESCE(SUM(balance), 0)").Where("user_id = ?", userID).Scan(&totalBalance).Error
	if err != nil {
		return fmt.Errorf("failed to retrieve total balance: %w", err)
	}

	// Log or use the balance
	if totalBalance.Valid {
		fmt.Println("Total balance:", totalBalance.Int64)
	} else {
		fmt.Println("Total balance is 0")
	}

	// Update the existing reward with the new values
	existingReward.RewardType = reward.RewardType
	existingReward.Point = reward.Point
	existingReward.IncidentReportID = reward.IncidentReportID

	// Update the balance if provided in the reward parameter
	if reward.Balance != 0 {
		existingReward.Balance = reward.Balance
	}

	// Save the updated reward to the database
	if err := i.DB.Save(existingReward).Error; err != nil {
		return fmt.Errorf("failed to update reward: %w", err)
	}

	return nil
}

func (i *incidentReportRepo) SaveIncidentReport(report *models.IncidentReport) (*models.IncidentReport, error) {
	// Save the new report to the database
	if err := i.DB.Create(&report).Error; err != nil {
		return nil, fmt.Errorf("failed to save report: %v", err)
	}

	return report, nil
}

func (i *incidentReportRepo) HasPreviousReports(userID uint) (bool, error) {
	// Retrieve the database connection from the GormDB struct
	db := i.DB

	// Initialize a reward object to store the query result
	var reward models.Reward

	// Query the database to find a reward for the given user ID where the amount is greater than 0
	err := db.Where("user_id = ? AND balance > ?", userID, 0).First(&reward).Error
	if err != nil {
		// If the error is "record not found", return false indicating no previous reports
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		// Return the error if it's not a "record not found" error
		return false, fmt.Errorf("could not find reward for user: %v", err)
	}

	// If the reward amount is greater than 0, return true indicating previous reports
	return true, nil
}

func (i *incidentReportRepo) FindUserByID(id uint) (*models.UserResponse, error) {
	var user models.UserResponse
	err := i.DB.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (i *incidentReportRepo) GetReportByID(report_id string) (*models.IncidentReport, error) {
	var report models.IncidentReport
	err := i.DB.Where("id = ?", report_id).First(&report).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &report, nil
}

func (repo *incidentReportRepo) GetAllReports(page int) ([]map[string]interface{}, error) {
	var reports []map[string]interface{}

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * 20

	err := repo.DB.
		Table("incident_reports").
		Select(`
			incident_reports.*, 
			users.thumb_nail_url AS thumbnail_urls,  -- Changed thumbnail_url to thumbnail_urls
			users.profile_image AS profile_image, 
			incident_reports.feed_urls
		`).
		Joins("JOIN users ON users.id = incident_reports.user_id").
		Order("incident_reports.created_at DESC").
		Limit(20).
		Offset(offset).
		Scan(&reports).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no incident reports found")
		}
		return nil, err
	}

	for _, report := range reports {
		// Profile image logic remains unchanged
		if profileImage, exists := report["profile_image"]; exists && profileImage != "" {
			report["profile_image"] = profileImage
		} else if thumbnailUrl, exists := report["thumbnail_urls"]; exists && thumbnailUrl != "" {  // Updated to thumbnail_urls
			report["profile_image"] = thumbnailUrl
		} else {
			report["profile_image"] = nil
		}

		// Ensure feed_urls is properly handled
		if feedUrls, exists := report["feed_urls"]; exists {
			report["feed_urls"] = feedUrls
		}
	}

	return reports, nil
}


func (repo *incidentReportRepo) GetAllReportsByState(state string, page int) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport
	offset := (page - 1) * DefaultPageSize

	err := repo.DB.Where("state = ?", state).
		Order("timeof_incidence DESC").
		Limit(DefaultPageSize).
		Offset(offset).
		Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

// GetAllReportsByState returns incident reports filtered by state and time range, with pagination
func (repo *incidentReportRepo) GetAllReportsByStateByTime(state string, startTime, endTime time.Time, page int) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport
	offset := (page - 1) * DefaultPageSize

	err := repo.DB.Where("state = ? AND timeof_incidence BETWEEN ? AND ?", state, startTime, endTime).
		Order("timeof_incidence DESC").
		Limit(DefaultPageSize).
		Offset(offset).
		Find(&reports).Error

	if err != nil {
		return nil, err
	}
	return reports, nil
}

func (repo *incidentReportRepo) GetAllReportsByLGA(lga string, page int) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport
	offset := (page - 1) * DefaultPageSize

	err := repo.DB.Where("lga = ?", lga).
		Order("timeof_incidence DESC").
		Limit(DefaultPageSize).
		Offset(offset).
		Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func (repo *incidentReportRepo) GetAllReportsByReportType(reportType string, page int) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport
	offset := (page - 1) * DefaultPageSize

	err := repo.DB.Where("report_type = ?", reportType).
		Order("timeof_incidence DESC").
		Limit(DefaultPageSize).
		Offset(offset).
		Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func (r *incidentReportRepo) GetRewardByUserID(userID uint) (*models.Reward, error) {
	var reward models.Reward
	if err := r.DB.First(&reward, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &reward, nil
}

func (r *incidentReportRepo) Save(report *models.IncidentReport) error {
	return r.DB.Create(report).Error
}

func (r *incidentReportRepo) GetReportPercentageByState() ([]models.StateReportPercentage, error) {
	var results []models.StateReportPercentage

	query := `
        SELECT 
            state_name, 
            COUNT(*) AS count, 
            (COUNT(*) * 100.0 / (SELECT COUNT(*) FROM incident_reports)) AS percentage 
        FROM 
            incident_reports 
        GROUP BY 
            state_name;
    `

	if err := r.DB.Raw(query).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (repo *incidentReportRepo) GetReportStatusByID(reportID string) (string, error) {
	var report models.IncidentReport
	err := repo.DB.Select("report_status").Where("id = ?", reportID).First(&report).Error
	if err != nil {
		return "", err
	}
	return report.ReportStatus, nil
}

func (repo *incidentReportRepo) GetReportsPostedTodayCount() (int64, error) {
	var count int64
	// Get the start of today
	startOfToday := time.Now().Truncate(24 * time.Hour)

	// Count the reports posted today
	err := repo.DB.Model(&models.IncidentReport{}).
		Where("timeof_incidence >= ?", startOfToday).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTotalUserCount returns the total number of users in the database
func (repo *incidentReportRepo) GetTotalUserCount() (int64, error) {
	var count int64

	// Count the total number of users
	err := repo.DB.Model(&models.User{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetRegisteredUsersCountByLGA returns the number of registered users in a specific Local Government Area (LGA)
func (repo *incidentReportRepo) GetRegisteredUsersCountByLGA(lga string) (int64, error) {
	var count int64

	// Count the total number of users in the specified LGA
	err := repo.DB.Model(&models.User{}).
		Where("lga_name = ?", lga).
		Count(&count).Error

	if err != nil {
		log.Printf("Error querying database: %v", err)
		return 0, err
	}

	log.Printf("LGA: %s, User Count: %d", lga, count)
	return count, nil
}

func (repo *incidentReportRepo) GetReportsByTypeAndLGA(reportType string, lga string) ([]models.SubReport, error) {
	var reports []models.SubReport
	err := repo.DB.Joins("JOIN report_types ON report_types.id = sub_reports.report_type_id").
		Joins("JOIN lgas ON lgas.id = sub_reports.lga_id").
		Where("report_types.name = ? AND lgas.name = ?", reportType, lga).
		Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

// GetReportTypeCounts gets the report types and their corresponding incident report counts

func (repo *incidentReportRepo) GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, []models.StateReportCount, error) {
    var reportTypes []string
    var counts []int
    var totalUsers int
    var totalReports int
    var topStates []models.StateReportCount

    // Determine the time range from the database if not provided
    var minTime, maxTime time.Time
    if (startDate == nil || *startDate == "") || (endDate == nil || *endDate == "") {
        // Fetch min and max time_of_incidence for the given state and lga
        err := repo.DB.Raw(`
            SELECT MIN(time_of_incidence), MAX(time_of_incidence)
            FROM incident_reports
            WHERE state_name = ? AND lga_name = ?
        `, state, lga).Row().Scan(&minTime, &maxTime)
        if err != nil {
            return nil, nil, 0, 0, nil, fmt.Errorf("failed to fetch time range from DB: %v", err)
        }
    }

    // Use query params if provided, otherwise fall back to DB-derived range
    var effectiveStartDate, effectiveEndDate time.Time
    if startDate != nil && *startDate != "" {
        var err error
        effectiveStartDate, err = time.Parse("2006-01-02", *startDate)
        if err != nil {
            return nil, nil, 0, 0, nil, errors.New("failed to parse start date: " + err.Error())
        }
    } else {
        effectiveStartDate = minTime
    }

    if endDate != nil && *endDate != "" {
        var err error
        effectiveEndDate, err = time.Parse("2006-01-02", *endDate)
        if err != nil {
            return nil, nil, 0, 0, nil, errors.New("failed to parse end date: " + err.Error())
        }
    } else {
        effectiveEndDate = maxTime
    }

    // Base query for report types and counts from IncidentReport table
    query := `
        SELECT ir.category, COUNT(*) AS count,
               (SELECT COUNT(DISTINCT ir.user_id) FROM incident_reports ir WHERE ir.state_name = ? AND ir.lga_name = ?) AS total_users,
               (SELECT COUNT(*) FROM incident_reports ir WHERE ir.state_name = ? AND ir.lga_name = ?) AS total_reports
        FROM incident_reports ir
        WHERE ir.state_name = ? AND ir.lga_name = ?
        AND ir.time_of_incidence BETWEEN ? AND ?
        GROUP BY ir.category
    `

    // Prepare query arguments with effective time range
    args := []interface{}{
        state, lga, // For total_users subquery
        state, lga, // For total_reports subquery
        state, lga, // For main WHERE clause
        effectiveStartDate, effectiveEndDate, // Time range
    }

    // Execute the query with parameters
    rows, err := repo.DB.Raw(query, args...).Rows()
    if err != nil {
        return nil, nil, 0, 0, nil, err
    }
    defer rows.Close()

    // Process the result rows
    for rows.Next() {
        var reportType string
        var count int
        if err := rows.Scan(&reportType, &count, &totalUsers, &totalReports); err != nil {
            return nil, nil, 0, 0, nil, err
        }
        reportTypes = append(reportTypes, reportType)
        counts = append(counts, count)
    }

    if err := rows.Err(); err != nil {
        return nil, nil, 0, 0, nil, err
    }

    // Query to get all states with report counts from IncidentReport table
    topStatesQuery := `
        SELECT state_name, COUNT(*) AS report_count
        FROM incident_reports
        WHERE lga_name = ?
        AND time_of_incidence BETWEEN ? AND ?
        GROUP BY state_name
        ORDER BY report_count DESC
    `

    topStatesArgs := []interface{}{
        lga,
        effectiveStartDate,
        effectiveEndDate,
    }

    err = repo.DB.Raw(topStatesQuery, topStatesArgs...).Scan(&topStates).Error
    if err != nil {
        return nil, nil, 0, 0, nil, fmt.Errorf("could not fetch top states: %v", err)
    }

    return reportTypes, counts, totalUsers, totalReports, topStates, nil
}

// SaveReportTypeAndSubReport saves both ReportType and SubReport in a transaction
func (repo *incidentReportRepo) SaveStateLgaReportType(lga *models.LGA, state *models.State) error {
	// Start a transaction
	tx := repo.DB.Begin()
	// Commit State to the database
	if err := tx.Create(state).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit LGA to the database
	if err := tx.Create(lga).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	return tx.Commit().Error
}

type Marker struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Popup string  `json:"popup"`
}

func (repo *incidentReportRepo) GetIncidentMarkers() ([]Marker, error) {
	var markers []Marker

	query := `
        SELECT
            latitude AS lat,
            longitude AS lng,
            incident_reports.state_name AS popup,
            COALESCE(report_counts.count, 0) AS count
        FROM
            incident_reports
        LEFT JOIN (
            SELECT
                state_name,
                COUNT(*) AS count
            FROM
                incident_reports
            GROUP BY
                state_name
        ) AS report_counts ON incident_reports.state_name = report_counts.state_name
        GROUP BY
            incident_reports.state_name, latitude, longitude, report_counts.count
    `

	if err := repo.DB.Raw(query).Scan(&markers).Error; err != nil {
		return nil, err
	}

	return markers, nil
}

func (repo *incidentReportRepo) DeleteByID(id string) error {
	var report models.SubReport
	if err := repo.DB.Where("id = ?", id).First(&report).Error; err != nil {
		return err
	}

	if err := repo.DB.Delete(&report).Error; err != nil {
		return err
	}

	return nil
}

func (repo *incidentReportRepo) GetStateReportCounts() ([]models.StateReportCount, error) {
	var stateReportCounts []models.StateReportCount

	err := repo.DB.Table("incident_reports").
		Select("state_name, COUNT(id) as report_count").
		Group("state_name").
		Scan(&stateReportCounts).Error

	if err != nil {
		return nil, err
	}

	return stateReportCounts, nil
}

func (repo *incidentReportRepo) GetVariadicStateReportCounts(reportTypes []string, states []string, startDate, endDate *time.Time) ([]models.StateReportCount, error) {
	var stateReportCounts []models.StateReportCount

	// Initialize the query on ReportType model
	db := repo.DB.Model(&models.ReportType{})

	// Select state_name, category, and count the reports, grouping by state_name and category
	query := db.Select("state_name, category, COUNT(id) as report_count").Group("state_name, category")

	// Add report type filter if provided
	if len(reportTypes) > 0 {
		query = query.Where("category IN (?)", reportTypes)
	}

	// Add state filters if provided
	if len(states) > 0 {
		query = query.Where("state_name IN (?)", states)
	}

	// Add date range filter if both dates are provided
	if startDate != nil && endDate != nil {
		query = query.Where("date_of_incidence BETWEEN ? AND ?", startDate, endDate)
	} else if startDate != nil {
		query = query.Where("date_of_incidence >= ?", startDate)
	} else if endDate != nil {
		query = query.Where("date_of_incidence <= ?", endDate)
	}

	// Add filter to exclude empty state names
	query = query.Where("state_name <> ''")

	// Debugging: Log the final query
	sql, args := query.Statement.SQL.String(), query.Statement.Vars
	log.Printf("Final query: %s with args: %v", sql, args)

	// Execute the query and scan the results into stateReportCounts
	err := query.Scan(&stateReportCounts).Error
	if err != nil {
		return nil, err
	}

	return stateReportCounts, nil
}

func (i *incidentReportRepo) GetAllCategories() ([]string, error) {
	var categories []string
	if err := i.DB.Model(&models.ReportType{}).Distinct().Pluck("category", &categories).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %v", err)
	}
	return categories, nil
}

func (i *incidentReportRepo) GetAllStates() ([]string, error) {
	var states []string
	if err := i.DB.Model(&models.ReportType{}).Distinct().Pluck("state_name", &states).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %v", err)
	}
	return states, nil
}

func (i *incidentReportRepo) GetRatingPercentages(reportType, state string) (*models.RatingPercentage, error) {
    var totalCount int64
    var goodCount int64
    var badCount int64

    // Count total reports
    if err := i.DB.Model(&models.IncidentReport{}). 
        Where("category = ? AND state_name = ?", reportType, state).
        Count(&totalCount).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch total count: %v", err)
    }

    // Count good ratings
    if err := i.DB.Model(&models.IncidentReport{}). 
        Where("category = ? AND state_name = ? AND rating = ?", reportType, state, "good"). 
        Count(&goodCount).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch good count: %v", err)
    }

    // Count bad ratings
    if err := i.DB.Model(&models.IncidentReport{}). // Fixed to use IncidentReport
        Where("category = ? AND state_name = ? AND rating = ?", reportType, state, "bad"). // Fixed field name to 'rating'
        Count(&badCount).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch bad count: %v", err)
    }

    // Calculate percentages with zero division protection
    var goodPercentage, badPercentage float64
    if totalCount > 0 { // Added to prevent division by zero
        goodPercentage = float64(goodCount) / float64(totalCount) * 100
        badPercentage = float64(badCount) / float64(totalCount) * 100
    }

    return &models.RatingPercentage{
        GoodPercentage: goodPercentage,
        BadPercentage:  badPercentage,
    }, nil
}

func (i *incidentReportRepo) GetReportCountsByStateAndLGA() ([]models.ReportCount, error) {
	var results []models.ReportCount

	err := i.DB.Model(&models.IncidentReport{}). // Query the 'incident_reports' table
		Select("state_name, lga_name, COUNT(*) as count"). // Select state, LGA, and count of reports
		Group("state_name, lga_name"). // Group results by state and LGA
		Scan(&results).Error // Store results in 'results' slice

	if err != nil {
		return nil, err // Return error if query fails
	}

	return results, nil // Return results
}


func (repo *incidentReportRepo) ListAllStatesWithReportCounts() ([]models.StateReportCount, error) {
	var topStates []models.StateReportCount

	// Query to get the top 6 states with their report counts
	query := `
        SELECT state_name, COUNT(*) AS report_count
        FROM report_types
        GROUP BY state_name
        ORDER BY report_count DESC
        LIMIT 6
    `

	// Execute the query
	err := repo.DB.Raw(query).Scan(&topStates).Error
	if err != nil {
		return nil, fmt.Errorf("could not fetch states with report counts: %v", err)
	}

	return topStates, nil
}

func (i *incidentReportRepo) GetTotalReportCount() (int64, error) {
	var count int64

	err := i.DB.Model(&models.ReportType{}).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (i *incidentReportRepo) uploadFileToS3(file multipart.File, bucketName, key string) (string, error) {
	defer file.Close()

	// Step 1: Load the AWS config with environment credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		),
	)
	if err != nil {
		return "", fmt.Errorf("unable to load AWS config: %v", err)
	}

	// Step 2: Create an S3 client with the configured credentials
	svc := s3.NewFromConfig(cfg)

	// Step 3: Read the file content into memory
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %v", err)
	}

	// Step 4: Prepare the S3 PutObjectInput
	putObjectInput := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileContent),
		ACL:    "public-read", // Set the ACL to public-read
	}

	// Step 5: Upload the file to S3
	_, err = svc.PutObject(context.TODO(), putObjectInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3a: %v", err)
	}

	// Step 6: Construct and return the file URL
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, os.Getenv("AWS_REGION"), key)
	return fileURL, nil
}

func (i *incidentReportRepo) GetNamesByCategory(stateName string, lgaID string, reportTypeCategory string) ([]string, error) {
	var names []string

	err := i.DB.Model(&models.SubReport{}).
		Where("state_name = ? AND lga_id = ? AND report_type_category = ?", stateName, lgaID, reportTypeCategory).
		Pluck("sub_report_type", &names).Error

	if err != nil {
		return nil, err
	}

	return names, nil
}

func createS3Client() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return s3.NewFromConfig(cfg), nil
}

func (i *incidentReportRepo) UploadMediaToS3(file multipart.File, fileHeader *multipart.FileHeader, bucketName, folderName string) (string, error) {
	defer file.Close()

	// Create an S3 client
	client, err := createS3Client()
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %v", err)
	}

	// Generate a unique key for the file
	key := fmt.Sprintf("%s/%s", folderName, fileHeader.Filename)

	// Upload the file to S3
	fileURL, err := uploadFileToS3(client, file, bucketName, key)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3b: %v", err)
	}

	return fileURL, nil
}

// Function to upload a file to S3
func uploadFileToS3(client *s3.Client, file multipart.File, bucketName, key string) (string, error) {
	defer file.Close()

	// Step 1: Read the file content into memory
	fileContent, err := io.ReadAll(file)
	if err != nil {
		// Log and return an error if reading the file fails
		fmt.Printf("Error reading file content: %v\n", err)
		return "", fmt.Errorf("failed to read file content: %v", err)
	}

	// Step 2: Log information about the bucket and key
	fmt.Printf("Uploading to bucket: %s\n", bucketName)
	fmt.Printf("Uploading with key: %s\n", key)

	// Step 3: Prepare the S3 PutObjectInput
	putObjectInput := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),          // Specify the S3 bucket name
		Key:    aws.String(key),                 // Specify the object key (file name)
		Body:   bytes.NewReader(fileContent),    // Use the file content as the body
		ACL:    types.ObjectCannedACLPublicRead, // Directly use the ObjectCannedACL enum
	}

	// Step 4: Attempt to upload the file to S3
	_, err = client.PutObject(context.TODO(), putObjectInput)
	if err != nil {
		// Log and return an error if the upload fails
		fmt.Printf("Error uploading file to S3: %v\n", err)
		return "", fmt.Errorf("failed to upload file to S3c: %v", err)
	}

	// Step 5: Construct the public URL for the uploaded file
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, os.Getenv("AWS_REGION"), key)

	// Step 6: Log the successful upload and the URL
	fmt.Printf("File uploaded successfully, URL: %s\n", fileURL)

	// Step 7: Return the file URL
	return fileURL, nil
}

func (repo *incidentReportRepo) SaveReportType(reportType *models.ReportType) (*models.ReportType, error) {
	if err := repo.DB.Create(reportType).Error; err != nil {
		return nil, err // Return nil and the error
	}
	return reportType, nil // Return the created reportType and nil error
}

// SaveSubReport saves the sub report to the database.
func (i *incidentReportRepo) SaveSubReport(subReport *models.SubReport) (*models.SubReport, error) {
	// Save the new SubReport to the database
	if err := i.DB.Create(&subReport).Error; err != nil {
		return nil, fmt.Errorf("failed to save sub report: %v", err)
	}

	return subReport, nil
}

func (repo *incidentReportRepo) GetSubReportsByCategory(category string) ([]models.SubReport, error) {
	var subReports []models.SubReport

	// Query to get sub-reports for the specified report type category
	query := `
        SELECT sr.*
        FROM sub_reports sr
        JOIN report_types rt ON sr.report_type_id = rt.id
        WHERE rt.category = ?
    `

	// Execute the query with the category parameter
	err := repo.DB.Raw(query, category).Scan(&subReports).Error
	if err != nil {
		return nil, fmt.Errorf("could not fetch sub-reports: %v", err)
	}

	return subReports, nil
}

func (repo *incidentReportRepo) GetAllIncidentReportsByUser(userID uint, page int) ([]map[string]interface{}, error) {
	var reports []map[string]interface{} // For dynamic fields

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * 20

	// Query the incident_reports table by user_id, join with users table to get profile image
	err := repo.DB.
		Table("incident_reports").
		Select(`
			incident_reports.*, 
			users.thumb_nail_url AS thumbnail_urls,  -- Align naming to match GetAllReports
			users.profile_image AS profile_image 
		`).
		Joins("JOIN users ON users.id = incident_reports.user_id").
		Where("incident_reports.user_id = ?", userID).
		Order("incident_reports.created_at DESC"). // Use the same ordering as GetAllReports
		Limit(20).
		Offset(offset).
		Scan(&reports).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no incident reports found for the user")
		}
		return nil, err
	}

	// If no reports are found, return an empty slice instead of an error
	if len(reports) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Process the reports to set profile_image and handle feed_urls
	for _, report := range reports {
		if profileImage, exists := report["profile_image"]; exists && profileImage != "" {
			report["profile_image"] = profileImage
		} else if thumbnailUrl, exists := report["thumbnail_urls"]; exists && thumbnailUrl != "" {
			report["profile_image"] = thumbnailUrl
		} else {
			report["profile_image"] = nil
		}

		// Ensure feed_urls is properly handled
		if feedUrls, exists := report["feed_urls"]; exists {
			report["feed_urls"] = feedUrls
		}
	}

	return reports, nil
}



func (repo *incidentReportRepo) IsBookmarked(userID uint, reportID uuid.UUID, bookmark *models.Bookmark) error {
	return repo.DB.Where("user_id = ? AND report_id = ?", userID, reportID).
		First(bookmark).Error
}

func (repo *incidentReportRepo) SaveBookmark(bookmark *models.Bookmark) error {
	return repo.DB.Create(bookmark).Error
}

func (repo *incidentReportRepo) GetBookmarkedReports(userID uint) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport

	log.Printf("Retrieving bookmarked reports for userID: %d", userID)

	// Perform the query with a join on bookmarks and preload the associated ReportType
	err := repo.DB.
		Joins("JOIN bookmarks ON bookmarks.report_id = incident_reports.id").
		Where("bookmarks.user_id = ?", userID).
		Preload("ReportType"). // This preloads the related ReportType data
		Find(&reports).Error

	if err != nil {
		log.Printf("Error retrieving reports: %v", err)
		return nil, err
	}

	log.Printf("Found reports: %v", reports)

	return reports, nil
}

func (repo *incidentReportRepo) GetReportsByUserID(userID uint) ([]models.ReportType, error) {
	var reports []models.ReportType

	err := repo.DB.Preload("SubReports").Where("user_id = ?", userID).Find(&reports).Error
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (repo *incidentReportRepo) GetReportTypeCountsByLGA(lga string) (map[string]interface{}, error) {
	var reportTypes []string
	var counts []int
	var totalCount int

	// Updated SQL query referencing the correct table: incident_reports
	query := `
        SELECT rt.category AS report_type, COUNT(ir.id) AS report_count
        FROM incident_reports ir
        JOIN report_types rt ON ir.report_type_id = rt.id
        WHERE ir.lga_name = ?
        GROUP BY rt.category
        ORDER BY report_count DESC;
    `

	// Execute the query
	rows, err := repo.DB.Raw(query, lga).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process the result rows
	for rows.Next() {
		var reportType string
		var count int
		if err := rows.Scan(&reportType, &count); err != nil {
			return nil, err
		}
		reportTypes = append(reportTypes, reportType)
		counts = append(counts, count)
		totalCount += count // Sum up the report counts
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Prepare the final response structure
	response := map[string]interface{}{
		"report_types":  reportTypes,
		"report_counts": counts,
		"total_count":   totalCount,
	}

	return response, nil
}


// Repository function
func (repo *incidentReportRepo) GetReportCountsByState(state string) ([]string, []int, error) {
	var lgas []string
	var counts []int

	// SQL query to get LGAs and their report counts for the selected state
	query := `
        SELECT lga_name, COUNT(*) AS report_count
        FROM incident_reports
        WHERE state_name = ?
        GROUP BY lga_name
        ORDER BY report_count DESC;
    `

	// Execute the query with the state parameter
	rows, err := repo.DB.Raw(query, state).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process the result rows
	for rows.Next() {
		var lgaName string
		var count int
		if err := rows.Scan(&lgaName, &count); err != nil {
			return nil, nil, err
		}
		lgas = append(lgas, lgaName)
		counts = append(counts, count)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return lgas, counts, nil
}

func (repo *incidentReportRepo) GetTopCategories() ([]string, []int, error) {
	var categories []string
	var counts []int

	// SQL query to get top 10 categories and their report counts
	query := `
        SELECT category, COUNT(*) AS report_count
        FROM incident_reports
        GROUP BY category
        ORDER BY report_count DESC
        LIMIT 10;
    `

	// Execute the query
	rows, err := repo.DB.Raw(query).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	// Process the result rows
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, nil, err
		}
		categories = append(categories, category)
		counts = append(counts, count)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return categories, counts, nil
}

func (repo *incidentReportRepo) GetReportsByCategoryAndReportID(category string, reportID string) ([]models.ReportType, error) {
	var reports []models.ReportType

	// GORM query to fetch reports by category and report_id
	err := repo.DB.
		Where("category = ? AND report_id = ?", category, reportID).
		Order("date_of_incidence DESC").
		Find(&reports).Error

	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (repo *incidentReportRepo) GetReportsByCategory(category string) ([]models.ReportType, error) {
	var reports []models.ReportType

	// GORM query to fetch reports by category
	err := repo.DB.
		Where("category = ?", category).
		Order("date_of_incidence DESC").
		Find(&reports).Error

	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (i *incidentReportRepo) GetFilteredIncidentReports(category, state, lga string) ([]models.IncidentReport, []string, error) {
	var reports []models.IncidentReport
	var filters []string

	// Start building the query
	query := i.DB.Model(&models.IncidentReport{})

	// Apply the filters only if they are provided
	if category != "" {
		query = query.Where("category = ?", category)
		filters = append(filters, category) // Append the category value, not the name
	}
	if state != "" {
		query = query.Where("state_name = ?", state)
		filters = append(filters, state) // Append the state value
	}
	if lga != "" {
		query = query.Where("lga_name = ?", lga)
		filters = append(filters, lga) // Append the LGA value
	}

	// Execute the query and get the results
	if err := query.Find(&reports).Error; err != nil {
		return nil, nil, err
	}

	// Return the incident reports and the filters that were applied
	return reports, filters, nil
}

// GetIncidentReportByID retrieves an incident report by its ID from the database.
func (i *incidentReportRepo) GetIncidentReportByID(reportID string) (*models.IncidentReport, error) {
	var report models.IncidentReport
	if err := i.DB.Where("id = ?", reportID).First(&report).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no report found with ID %s: %w", reportID, err)
		}
		return nil, fmt.Errorf("error retrieving report with ID %s: %w", reportID, err)
	}
	return &report, nil
}

func (i *incidentReportRepo) GetReportTypeeByID(reportTypeID string) (*models.ReportType, error) {
	var reportType models.ReportType
	if err := i.DB.Where("id = ?", reportTypeID).First(&reportType).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no report type found with ID %s: %w", reportTypeID, err)
		}
		return nil, fmt.Errorf("error retrieving report type with ID %s: %w", reportTypeID, err)
	}
	return &reportType, nil
}

func (i *incidentReportRepo) UpdateIncidentReport(report *models.IncidentReport) error {
	// Use a transaction to ensure atomicity of the operation.
	return i.DB.Transaction(func(tx *gorm.DB) error {
		// Fetch the existing incident report from the database.
		var existingReport models.IncidentReport
		if err := tx.Where("id = ?", report.ID).First(&existingReport).Error; err != nil {
			return fmt.Errorf("error retrieving existing report with ID %s: %w", report.ID, err)
		}

		// Perform any necessary validation or data preparation on the new report data.
		if err := validateIncidentReport(report); err != nil {
			return fmt.Errorf("invalid incident report data: %w", err)
		}

		// Update the existing report's fields with the new data.
		existingReport.Description = report.Description
		existingReport.FeedURLs = report.FeedURLs
		existingReport.ThumbnailURLs = report.ThumbnailURLs
		existingReport.FullSizeURLs = report.FullSizeURLs
		existingReport.StateName = report.StateName
		existingReport.LGAName = report.LGAName
		existingReport.Latitude = report.Latitude
		existingReport.Longitude = report.Longitude
		existingReport.UserIsAnonymous = report.UserIsAnonymous
		existingReport.Address = report.Address
		existingReport.UserUsername = report.UserUsername
		existingReport.Email = report.Email
		existingReport.View = report.View
		existingReport.IsVerified = report.IsVerified
		existingReport.UserID = report.UserID
		existingReport.AdminID = report.AdminID
		existingReport.Landmark = report.Landmark
		existingReport.LikeCount = report.LikeCount
		existingReport.IsResponse = report.IsResponse
		existingReport.TimeofIncidence = report.TimeofIncidence
		existingReport.ReportStatus = report.ReportStatus
		existingReport.RewardPoint = report.RewardPoint
		existingReport.RewardAccountNumber = report.RewardAccountNumber
		existingReport.ActionTypeName = report.ActionTypeName
		existingReport.IsState = report.IsState
		existingReport.Rating = report.Rating
		existingReport.HospitalName = report.HospitalName
		existingReport.Department = report.Department
		existingReport.DepartmentHeadName = report.DepartmentHeadName
		existingReport.AccidentCause = report.AccidentCause
		existingReport.SchoolName = report.SchoolName
		existingReport.VicePrincipal = report.VicePrincipal
		existingReport.OutageLength = report.OutageLength
		existingReport.AirportName = report.AirportName
		existingReport.Country = report.Country
		existingReport.StateEmbassyLocation = report.StateEmbassyLocation
		existingReport.NoWater = report.NoWater
		existingReport.AmbassedorsName = report.AmbassedorsName
		existingReport.HospitalAddress = report.HospitalAddress
		existingReport.RoadName = report.RoadName
		existingReport.AirlineName = report.AirlineName
		existingReport.Category = report.Category
		existingReport.Terminal = report.Terminal
		existingReport.QueueTime = report.QueueTime
		existingReport.SubReportType = report.SubReportType
		existingReport.UpvoteCount = report.UpvoteCount
		existingReport.DownvoteCount = report.DownvoteCount

		// Save the updated report to the database.
		if err := tx.Save(&existingReport).Error; err != nil {
			return fmt.Errorf("error updating incident report in the database: %w", err)
		}

		// Check and update related entities like ReportType if needed.
		if err := updateRelatedReportType(tx, &existingReport, report.ReportTypeID); err != nil {
			return fmt.Errorf("error updating related report type: %w", err)
		}

		return nil
	})
}

// updateRelatedReportType updates or creates a ReportType if needed.
func updateRelatedReportType(tx *gorm.DB, report *models.IncidentReport, newReportTypeID uuid.UUID) error {
	// Check if the report already has an associated ReportType and if it's the same as the new one.
	if report.ReportTypeID == newReportTypeID {
		// If the ReportType is already associated with the report, no update is needed.
		return nil
	}

	// If the ReportType is different, update it to the new one.
	var reportType models.ReportType
	if err := tx.Where("id = ?", newReportTypeID).First(&reportType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Handle case where ReportType is not found, or create a new one as needed.
			return fmt.Errorf("report type with ID %v not found", newReportTypeID)
		}
		return fmt.Errorf("error retrieving report type: %w", err)
	}

	// Associate the new ReportType with the report.
	report.ReportTypeID = newReportTypeID

	// Save the updated report with the new ReportType.
	if err := tx.Save(report).Error; err != nil {
		return fmt.Errorf("error saving report with updated ReportType: %w", err)
	}

	return nil
}

// Example validation function for IncidentReport.
func validateIncidentReport(report *models.IncidentReport) error {
	// Validate FeedURLs
	if len(report.FeedURLs) == 0 {
		return fmt.Errorf("feed URLs cannot be empty")
	}

	// Add more validation logic as needed.
	return nil
}

// Inside db.IncidentReportRepository
func (i *incidentReportRepo) UpdateReportTypeWithIncidentReport(report *models.IncidentReport) error {
	// Assuming you have a method to find the report type based on the report ID
	var reportType models.ReportType
	if err := i.DB.First(&reportType, "report_id = ?", report.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("report type not found for report ID %s: %w", report.ID, err)
		}
		return fmt.Errorf("error retrieving report type for report ID %s: %w", report.ID, err)
	}

	// Update fields as necessary
	reportType.IncidentReportRating = report.Rating
	reportType.DateOfIncidence = report.TimeofIncidence

	// Save the updated report type
	if err := i.DB.Save(&reportType).Error; err != nil {
		return fmt.Errorf("error saving updated report type for report ID %s: %w", report.ID, err)
	}

	return nil
}

func (i *incidentReportRepo) FindReportTypeByCategory(category string, reportType *models.ReportType) error {
	// Query the ReportType based on the Category
	err := i.DB.Where("category = ?", category).First(reportType).Error

	if err != nil {
		// If the record is not found, return gorm.ErrRecordNotFound
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		// Return other errors encountered during the query
		return fmt.Errorf("error fetching report type: %w", err)
	}
	return nil
}

func (i *incidentReportRepo) GetReportTypeByCategory(category string) (*models.ReportType, error) {
	var reportType models.ReportType
	err := i.DB.Where("category = ?", category).First(&reportType).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No report type found, return nil without error
		}
		return nil, err // Return any other error
	}
	return &reportType, nil // Return the found report type
}

func (i *incidentReportRepo) CreateReportType(reportType *models.ReportType) error {
    if err := i.DB.Create(reportType).Error; err != nil {
        return fmt.Errorf("error creating report type: %v", err)
    }
    return nil
}

// GetIncidentReportByReportTypeID fetches the incident report associated with the given reportTypeID.
func (i *incidentReportRepo) GetIncidentReportByReportTypeID(reportTypeID string) (*models.IncidentReport, error) {
	// Initialize an empty IncidentReport object
	var incidentReport models.IncidentReport

	// Use GORM to query the database for the incident report with the given reportTypeID
	err := i.DB.
		Where("report_type_id = ?", reportTypeID).
		Preload("ReportType"). // If you want to preload the associated ReportType details
		Find(&incidentReport).Error

	if err != nil {
		return nil, fmt.Errorf("could not find associated incident report with report type ID %s: %v", reportTypeID, err)
	}

	return &incidentReport, nil
}

// Fetch IncidentReport by reportTypeID with preloading ReportType
func (i *incidentReportRepo) FindIncidentReportByReportTypeID(reportTypeID string) (*models.IncidentReport, error) {
	var incidentReport models.IncidentReport
	// Preload ReportType and fetch the IncidentReport by reportTypeID
	err := i.DB.Preload("ReportType").Where("report_type_id = ?", reportTypeID).First(&incidentReport).Error
	if err != nil {
		return nil, fmt.Errorf("could not find associated incident report with report type ID %v: %v", reportTypeID, err)
	}

	return &incidentReport, nil
}

// SaveMedia saves a media record to the database
func (i *incidentReportRepo) SaveMedia(media *models.Media) error {
	// Save the media record to the database
	if err := i.DB.Create(media).Error; err != nil {
		return fmt.Errorf("error saving media: %v", err)
	}
	return nil
}

// Repository method to get reportID based on userID
func (i *incidentReportRepo) GetReportIDByUser(ctx context.Context, userID uint) (uuid.UUID, error) {
	var incidentReport models.IncidentReport

	// Query the database to find the most recent report that matches userID, ordering by creation time or ID
	err := i.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").   // Ensure you get the most recent entry
		Last(&incidentReport).Error // Use Last() to pick the most recent record

	// If record not found or other error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, fmt.Errorf("no report found for user")
		}
		return uuid.Nil, fmt.Errorf("error querying report ID: %w", err)
	}

	fmt.Println("Most recent report ID:", incidentReport.ID)
	// Return the most recent report's ID
	return incidentReport.ID, nil
}

// func (i *incidentReportRepo) GetLastReportIDByUserID(userID uint) (string, error) {
// 	var reportType models.ReportType
// 	result := i.DB.Order("created_at DESC").First(&reportType, "user_id = ?", userID)
// 	if result.Error != nil {
// 		return "", result.Error
// 	}
// 	return reportType.IncidentReportID.String(), nil
// }

func (repo *incidentReportRepo) ReportExists(reportID uuid.UUID) (bool, error) {
	var count int64
	err := repo.DB.Model(&models.IncidentReport{}).
		Where("id = ?", reportID).
		Count(&count).Error
	return count > 0, err
}

func (repo *incidentReportRepo) UpdateBlockRequest(ctx context.Context, reportID uuid.UUID) error {
	// Start a transaction to ensure data integrity
	tx := repo.DB.Begin()
	if tx.Error != nil {
		log.Printf("error starting transaction: %v", tx.Error)
		return errors.New("failed to initiate transaction")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("transaction rolled back due to panic: %v", r)
		}
	}()

	// Step 1: Verify that the incident report exists
	var count int64
	err := tx.Model(&models.IncidentReport{}).Where("id = ?", reportID).Count(&count).Error
	if err != nil {
		tx.Rollback()
		log.Printf("error checking existence of incident report with id %v: %v", reportID, err)
		return errors.New("failed to verify incident report existence")
	}
	if count == 0 {
		tx.Rollback()
		log.Printf("incident report with id %v not found", reportID)
		return errors.New("incident report not found")
	}

	// Step 2: Update the BlockRequest field
	result := tx.Model(&models.IncidentReport{}).
		Where("id = ?", reportID).
		Update("block_request", "true")

	if result.Error != nil {
		tx.Rollback()
		log.Printf("error updating block_request for report id %v: %v", reportID, result.Error)
		return errors.New("failed to update block request")
	}

	// Step 3: Confirm that an update occurred
	if result.RowsAffected == 0 {
		tx.Rollback()
		log.Printf("no rows affected when updating block_request for report id %v", reportID)
		return errors.New("no update occurred, report may not exist")
	}

	// Step 4: Commit the transaction if successful
	if err = tx.Commit().Error; err != nil {
		log.Printf("error committing transaction for report id %v: %v", reportID, err)
		return errors.New("failed to commit transaction")
	}

	// Log success message and return nil for successful update
	log.Printf("successfully updated block_request to true for report id %v", reportID)
	return nil
}

// ReportUser sets the IsQueried field to true.
func (repo *incidentReportRepo) ReportUser(ctx context.Context, userID uint) error {
	result := repo.DB.Model(&models.User{}).Where("id = ?", userID).
		Update("is_queried", true)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// BlockUser sets the IsBlocked field to true.
func (repo *incidentReportRepo) BlockUser(ctx context.Context, userID uint) error {
	result := repo.DB.Model(&models.User{}).Where("id = ?", userID).
		Update("is_blocked", true)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (repo *incidentReportRepo) CreateFollow(follow models.Follow) error {
	return repo.DB.Create(&follow).Error
}

func (repo *incidentReportRepo) GetFollowersByReport(reportID uuid.UUID) ([]models.User, error) {
	var followers []models.User
	err := repo.DB.Model(&models.IncidentReport{}).Where("id = ?", reportID).Association("Followers").Find(&followers)
	return followers, err
}

func (repo *incidentReportRepo) GetOAuthState(state string) (*models.OAuthState, error) {
    var oauthState models.OAuthState
    if err := repo.DB.Where("state = ?", state).First(&oauthState).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil // State not found
        }
        return nil, err // Other errors
    }
    return &oauthState, nil
}

func (repo *incidentReportRepo) SaveOAuthState(oauthState *models.OAuthState) error {
    return repo.DB.Create(oauthState).Error
}

func (repo *incidentReportRepo) GetReportCountByLGA(lga string) (int, error) {
    var count int64
    err := repo.DB.Model(&models.IncidentReport{}). // Query the report_types table
        Where("lga_name = ?", lga). 
        Count(&count).Error
    if err != nil {
        return 0, err
    }
    return int(count), nil
}

func (repo *incidentReportRepo) GetReportCountByState(state string) (int, error) {
    var count int64
    err := repo.DB.Model(&models.IncidentReport{}). // Query the incident_reports table
        Where("state_name = ?", state). 
        Count(&count).Error
    if err != nil {
        return 0, err
    }
    return int(count), nil
}


func (repo *incidentReportRepo) GetOverallReportCount() (int, error) {
    var count int64
    err := repo.DB.Model(&models.IncidentReport{}).Count(&count).Error
    if err != nil {
        return 0, err
    }
    return int(count), nil
}

func (repo *incidentReportRepo) GetGovernorDetails(stateName string) (*models.State, error) {
    var state models.State
    err := repo.DB.Where("state = ?", stateName).First(&state).Error
    if err != nil {
        return nil, err
    }
    return &state, nil
}

func (repo *incidentReportRepo) CreateState(state *models.State) error {
    existingState := &models.State{}
    err := repo.DB.Where("state = ?", state.State).First(existingState).Error

    if err != nil {
        // If the state doesn't exist, create a new record
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return repo.DB.Create(state).Error
        }
        // Return other errors (e.g., database connection issues)
        return err
    }

    // If the state exists, merge LGAs and update other fields
    // Step 1: Merge LGAs (assuming Lgas is a []string field)
    if len(state.Lgas) > 0 {
        // Get existing LGAs
        existingLgas := existingState.Lgas
        // Create a map to avoid duplicates
        lgaMap := make(map[string]struct{})
        for _, lga := range existingLgas {
            lgaMap[lga] = struct{}{}
        }
        // Add new LGAs
        for _, lga := range state.Lgas {
            lgaMap[lga] = struct{}{}
        }
        // Convert back to slice
        mergedLgas := make([]string, 0, len(lgaMap))
        for lga := range lgaMap {
            mergedLgas = append(mergedLgas, lga)
        }
        existingState.Lgas = mergedLgas
    }

    // Step 2: Update other fields only if provided (non-nil or non-empty)
    if state.State != nil && *state.State != "" {
        existingState.State = state.State
    }
    if state.Governor != nil && *state.Governor != "" {
        existingState.Governor = state.Governor
    }
    if state.DeputyName != nil && *state.DeputyName != "" {
        existingState.DeputyName = state.DeputyName
    }
    if state.LGAC != nil && *state.LGAC != "" {
        existingState.LGAC = state.LGAC
    }
    if state.GovernorImage != nil && *state.GovernorImage != "" {
        existingState.GovernorImage = state.GovernorImage
    }
    if state.DeputyImage != nil && *state.DeputyImage != "" {
        existingState.DeputyImage = state.DeputyImage
    }
    if state.LgacImage != nil && *state.LgacImage != "" {
        existingState.LgacImage = state.LgacImage
    }

    // Step 3: Save the updated state
    return repo.DB.Save(existingState).Error
}

func (i *incidentReportRepo) GetAllStatesRatingPercentages(reportType string) (map[string]*models.RatingPercentage, error) {
    type RatingCount struct {
        StateName    string
        Rating       string
        Count        int64
    }

    var results []RatingCount
    if err := i.DB.Model(&models.IncidentReport{}).
        Select("state_name, rating, COUNT(*) as count").
        Where("category = ?", reportType).
        Group("state_name, rating").
        Scan(&results).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch rating counts: %v", err)
    }

    // Process results
    ratingMap := make(map[string]*models.RatingPercentage)
    totalCounts := make(map[string]int64)

    // First pass: calculate totals
    for _, result := range results {
        totalCounts[result.StateName] += result.Count
    }

    // Second pass: set percentages
    for _, result := range results {
        if _, exists := ratingMap[result.StateName]; !exists {
            ratingMap[result.StateName] = &models.RatingPercentage{}
        }
        total := totalCounts[result.StateName]
        if total > 0 {
            percentage := float64(result.Count) / float64(total) * 100
            if result.Rating == "good" {
                ratingMap[result.StateName].GoodPercentage = percentage
            } else if result.Rating == "bad" {
                ratingMap[result.StateName].BadPercentage = percentage
            }
        }
    }

    return ratingMap, nil
}

// FetchStates retrieves all states from the database
func (repo *incidentReportRepo) FetchStates() ([]models.State, error) {
    var states []models.State
    err := repo.DB.Preload("Lgas").Find(&states).Error
    if err != nil {
        return nil, err
    }

    // Populate the Lgas field for each state
    for i := range states {
        var lgas []models.LGA
        err := repo.DB.Where("state_id = ?", states[i].ID).Find(&lgas).Error
        if err != nil {
            return nil, err
        }

        // Extract LGA names and assign to the Lgas field
        lgaNames := make([]string, len(lgas))
        for j, lga := range lgas {
            if lga.Name != nil {
                lgaNames[j] = *lga.Name
            }
        }
        states[i].Lgas = lgaNames
    }

    return states, nil
}

func (repo *incidentReportRepo) FetchLGAs() ([]models.LGA, error) {
    var lgas []models.LGA
    err := repo.DB.Preload("State").Find(&lgas).Error
    if err != nil {
        return nil, err
    }
    return lgas, nil
}

// Repository method to fetch a state by name
func (repo *incidentReportRepo) FetchStateByName(stateName string) (*models.State, error) {
    var state models.State
    err := repo.DB.Where("state = ?", stateName).First(&state).Error
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, ErrStateNotFound
        }
        return nil, ErrDatabase
    }
    return &state, nil
}