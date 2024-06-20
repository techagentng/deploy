package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

const (
	DefaultPageSize = 20
	DefaultPage     = 1
)

type IncidentReportRepository interface {
	SaveBookmarkReport(bookmark *models.BookmarkReport) error
	SaveIncidentReport(report *models.IncidentReport) (*models.IncidentReport, error)
	HasPreviousReports(userID uint) (bool, error)
	UpdateReward(userID uint, reward *models.Reward) error
	FindUserByID(id uint) (*models.UserResponse, error)
	GetReportByID(report_id string) (*models.IncidentReport, error)
	CheckReportInBookmarkedReport(userID uint, reportID string) (bool, error)
	GetAllReports(page int) ([]models.IncidentReport, error)
	GetAllReportsByState(state string, page int) ([]models.IncidentReport, error)
	GetAllReportsByLGA(lga string, page int) ([]models.IncidentReport, error)
	GetAllReportsByReportType(lga string, page int) ([]models.IncidentReport, error)
	GetReportPercentageByState() ([]models.StateReportPercentage, error)
	Save(report *models.IncidentReport) error
	GetReportStatusByID(reportID string) (string, error)
	UpdateIncidentReport(report *models.IncidentReport) (*models.IncidentReport, error)
	GetReportsPostedTodayCount() (int64, error)
	GetTotalUserCount() (int64, error)
	GetRegisteredUsersCountByLGA(lga string) (int64, error)
	GetAllReportsByStateByTime(state string, startTime, endTime time.Time, page int) ([]models.IncidentReport, error)
	GetReportsByTypeAndLGA(reportType string, lga string) ([]models.SubReport, error)
	GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, error)
	SaveStateLgaTime(lga *models.LGA, state *models.State, reportType *models.ReportType, subReport *models.SubReport) error
	GetIncidentMarkers() ([]Marker, error)
	DeleteByID(id string) error
	GetStateReportCounts() ([]models.StateReportCount, error)
}

type incidentReportRepo struct {
	DB *gorm.DB
}

func NewIncidentReportRepo(db *GormDB) IncidentReportRepository {
	return &incidentReportRepo{db.DB}
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
		return err
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

func (repo *incidentReportRepo) CheckReportInBookmarkedReport(userID uint, reportID string) (bool, error) {
	var bookmarkedReport models.BookmarkReport
	if err := repo.DB.Where("user_id = ? AND report_id = ?", userID, reportID).First(&bookmarkedReport).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (repo *incidentReportRepo) SaveBookmarkReport(bookmark *models.BookmarkReport) error {
	tx := repo.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(bookmark).Error; err != nil {
		tx.Rollback()
		log.Printf("Error creating bookmark: %v", err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	log.Printf("Bookmark saved successfully: %+v", bookmark)
	return nil
}

func (repo *incidentReportRepo) GetAllReports(page int) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport
	// Calculate the offset
	offset := (page - 1) * 20

	err := repo.DB.Limit(20).Offset(offset).Order("timeof_incidence DESC").Find(&reports).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no incident reports found")
		}
		return nil, err
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

func (i *incidentReportRepo) UpdateIncidentReport(report *models.IncidentReport) (*models.IncidentReport, error) {
	// Update the existing report in the database
	if err := i.DB.Save(report).Error; err != nil {
		return nil, fmt.Errorf("failed to update report: %v", err)
	}

	return report, nil
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
func (repo *incidentReportRepo) GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, error) {
    var reportTypes []string
    var counts []int

    // Define the query
    query := `
        SELECT rt.report_type, COUNT(*) AS count
        FROM report_types rt
        INNER JOIN incident_reports ir ON rt.report_id = ir.id
        WHERE rt.state_name = ? AND rt.lga_name = ?
    `
    // Append date filtering if provided
    if startDate != nil && endDate != nil && *startDate != "" && *endDate != "" {
        query += ` AND rt.date_of_incidence BETWEEN ? AND ? `
    }

    query += ` GROUP BY rt.report_type `

    // Execute the query
    var rows *sql.Rows
    var err error
    if startDate != nil && endDate != nil && *startDate != "" && *endDate != "" {
        rows, err = repo.DB.Raw(query, state, lga, *startDate, *endDate).Rows()
    } else {
        rows, err = repo.DB.Raw(query, state, lga).Rows()
    }

    if err != nil {
        log.Printf("Error executing query: %v", err)
        return nil, nil, err
    }
    defer func() {
        if rows != nil {
            rows.Close()
        }
    }()

    log.Println("Query executed successfully, processing rows...")

    for rows.Next() {
        var reportType string
        var count int
        if err := rows.Scan(&reportType, &count); err != nil {
            log.Printf("Error scanning row: %v", err)
            return nil, nil, err
        }
        reportTypes = append(reportTypes, reportType)
        counts = append(counts, count)
    }

    if err := rows.Err(); err != nil {
        log.Printf("Rows error: %v", err)
        return nil, nil, err
    }

    log.Printf("Report types: %v", reportTypes)
    log.Printf("Report counts: %v", counts)

    return reportTypes, counts, nil
}

// SaveReportTypeAndSubReport saves both ReportType and SubReport in a transaction
func (repo *incidentReportRepo) SaveStateLgaTime(lga *models.LGA, state *models.State, reportType *models.ReportType, subReport *models.SubReport) error {
	// Start a transaction
	tx := repo.DB.Begin()

	// Commit ReportType to the database
	if err := tx.Create(reportType).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit SubReport to the database
	if err := tx.Create(subReport).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(lga).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(state).Error; err != nil {
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
	
	err := repo.DB.Table("report_types").
		Select("state_name, COUNT(id) as report_count").
		Group("state_name").
		Scan(&stateReportCounts).Error
	
	if err != nil {
		return nil, err
	}
	
	return stateReportCounts, nil
}
