package db

import (
	"bytes"
	"context"
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
	GetAllReportsByUser(userID uint, page int) ([]models.IncidentReport, error)
	IsBookmarked(userID uint, reportID string, bookmark *models.Bookmark) error
	SaveBookmark(bookmark *models.Bookmark) error
	GetBookmarkedReports(userID uint) ([]models.IncidentReport, error)
	GetReportsByUserID(userID uint) ([]models.ReportType, error)
	GetReportTypeCountsByLGA(lga string) ([]string, []int, error)
	GetReportCountsByState(state string) ([]string, []int, error)
	GetTopCategories() ([]string, []int, error)
	GetReportsByCategoryAndReportID(category string, reportID string) ([]models.ReportType, error)
	GetReportsByCategory(category string) ([]models.ReportType, error)
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

func (repo *incidentReportRepo) GetReportTypeCounts(state string, lga string, startDate, endDate *string) ([]string, []int, int, int, []models.StateReportCount, error) {
    var reportTypes []string
    var counts []int
    var totalUsers int
    var totalReports int
    var topStates []models.StateReportCount

    // Base query for report types and counts
    query := `
        SELECT rt.category, COUNT(*) AS count,
               (SELECT COUNT(DISTINCT rt.user_id) FROM report_types rt WHERE rt.state_name = ? AND rt.lga_name = ?) AS total_users,
               (SELECT COUNT(*) FROM report_types rt WHERE rt.state_name = ? AND rt.lga_name = ?) AS total_reports
        FROM report_types rt
        WHERE rt.state_name = ? AND rt.lga_name = ?
    `

    // Prepare query arguments
    var args []interface{}
    args = append(args, state, lga, state, lga, state, lga)

    // Optional date filter
    if startDate != nil && endDate != nil && *startDate != "" && *endDate != "" {
        var err error
        defaultStartDate, err := time.Parse("2006-01-02", *startDate)
        if err != nil {
            return nil, nil, 0, 0, nil, errors.New("failed to parse start date: " + err.Error())
        }

        defaultEndDate, err := time.Parse("2006-01-02", *endDate)
        if err != nil {
            return nil, nil, 0, 0, nil, errors.New("failed to parse end date: " + err.Error())
        }

        query += ` AND rt.date_of_incidence BETWEEN ? AND ?`
        args = append(args, defaultStartDate, defaultEndDate)
    }

    query += ` GROUP BY rt.category`

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

    // Query to get all states with report counts
    topStatesQuery := `
        SELECT state_name, COUNT(*) AS report_count
        FROM report_types
        WHERE lga_name = ?
    `

    // Append date filters if provided
    if startDate != nil && endDate != nil && *startDate != "" && *endDate != "" {
        topStatesQuery += ` AND date_of_incidence BETWEEN ? AND ?`
    }

    topStatesQuery += `
        GROUP BY state_name
        ORDER BY report_count DESC
    `

    topStatesArgs := []interface{}{lga}
    if startDate != nil && endDate != nil && *startDate != "" && *endDate != "" {
        defaultStartDate, err := time.Parse("2006-01-02", *startDate)
        if err != nil {
            return nil, nil, 0, 0, nil, errors.New("failed to parse start date: " + err.Error())
        }

        defaultEndDate, err := time.Parse("2006-01-02", *endDate)
        if err != nil {
            return nil, nil, 0, 0, nil, errors.New("failed to parse end date: " + err.Error())
        }

        topStatesArgs = append(topStatesArgs, defaultStartDate, defaultEndDate)
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

	err := repo.DB.Table("report_types").
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
	if err := i.DB.Model(&models.ReportType{}).
		Where("category = ? AND state_name = ?", reportType, state).
		Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch total count: %v", err)
	}

	// Count good ratings
	if err := i.DB.Model(&models.ReportType{}).
		Where("category = ? AND state_name = ? AND incident_report_rating = ?", reportType, state, "good").
		Count(&goodCount).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch good count: %v", err)
	}

	// Count bad ratings
	if err := i.DB.Model(&models.ReportType{}).
		Where("category = ? AND state_name = ? AND incident_report_rating = ?", reportType, state, "bad").
		Count(&badCount).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch bad count: %v", err)
	}

	// Calculate percentages
	goodPercentage := float64(goodCount) / float64(totalCount) * 100
	badPercentage := float64(badCount) / float64(totalCount) * 100

	return &models.RatingPercentage{
		GoodPercentage: goodPercentage,
		BadPercentage:  badPercentage,
	}, nil
}

func (i *incidentReportRepo) GetReportCountsByStateAndLGA() ([]models.ReportCount, error) {
    var results []models.ReportCount

    err := i.DB.Model(&models.ReportType{}).
        Select("state_name, lga_name, COUNT(*) as count").
        Group("state_name, lga_name").
        Scan(&results).Error

    if err != nil {
        return nil, err
    }

    return results, nil
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
        Bucket: aws.String(bucketName),  // Specify the S3 bucket name
        Key:    aws.String(key),         // Specify the object key (file name)
        Body:   bytes.NewReader(fileContent), // Use the file content as the body
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

func (i *incidentReportRepo) SaveReportType(reportType *models.ReportType) (*models.ReportType, error) {
    // Save the new ReportType to the database
    if err := i.DB.Create(&reportType).Error; err != nil {
        return nil, fmt.Errorf("failed to save report type: %v", err)
    }

    return reportType, nil
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

func (repo *incidentReportRepo) GetAllReportsByUser(userID uint, page int) ([]models.IncidentReport, error) {
	var reports []models.IncidentReport
	// Calculate the offset
	offset := (page - 1) * 20

	// Query the reports by user ID with pagination
	err := repo.DB.Where("user_id = ?", userID).Limit(20).Offset(offset).Order("timeof_incidence DESC").Find(&reports).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no incident reports found for this user")
		}
		return nil, err
	}
	return reports, nil
}

func (repo *incidentReportRepo) IsBookmarked(userID uint, reportID string, bookmark *models.Bookmark) error {
	return repo.DB.Where("user_id = ? AND report_id = ?", userID, reportID).First(bookmark).Error
}

func (repo *incidentReportRepo) SaveBookmark(bookmark *models.Bookmark) error {
	return repo.DB.Create(bookmark).Error
}

func (repo *incidentReportRepo) GetBookmarkedReports(userID uint) ([]models.IncidentReport, error) {
    var reports []models.IncidentReport
    
    log.Printf("Retrieving bookmarked reports for userID: %d", userID)
    
    err := repo.DB.
        Joins("JOIN incident_report_user ON incident_report_user.incident_report_id = incident_reports.id").
        Where("incident_report_user.user_id = ?", userID).
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

func (repo *incidentReportRepo) GetReportTypeCountsByLGA(lga string) ([]string, []int, error) {
    var reportTypes []string
    var counts []int

    // SQL query to get report types and their counts for the specified LGA
    query := `
        SELECT rt.category AS report_type, COUNT(*) AS report_count
        FROM report_types rt
        WHERE rt.lga_name = ?
        GROUP BY rt.category
        ORDER BY report_count DESC;
    `

    // Execute the query
    rows, err := repo.DB.Raw(query, lga).Rows()
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()

    // Process the result rows
    for rows.Next() {
        var reportType string
        var count int
        if err := rows.Scan(&reportType, &count); err != nil {
            return nil, nil, err
        }
        reportTypes = append(reportTypes, reportType)
        counts = append(counts, count)
    }

    if err := rows.Err(); err != nil {
        return nil, nil, err
    }

    return reportTypes, counts, nil
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


