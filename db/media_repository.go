package db

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/techagentng/citizenx/models"
	"gorm.io/gorm"
)

type MediaRepository interface {
	SaveMedia(media models.Media, reportID string, userID uint) error
	RewardAndSavePoints(mediaCount int, report *models.IncidentReport) error
	GetMediaCountByByUserID(userID uint) (int, error)
	CreateMediaCount(mediaCount *models.MediaCount) error
	UploadMediaToS3(file multipart.File, fileHeader *multipart.FileHeader, bucketName, folderName string) (string, error)
}

type mediaRepo struct {
	DB *gorm.DB
}

// RewardAndSavePoints implements MediaRepository.
func (m *mediaRepo) RewardAndSavePoints(mediaCount int, report *models.IncidentReport) error {
	panic("unimplemented")
}

func NewMediaRepo(db *GormDB) MediaRepository {
	return &mediaRepo{db.DB}
}

func (m *mediaRepo) SaveMedia(media models.Media, reportID string, userID uint) error {
	ID := uuid.New()
	media.ID = ID.String()
	media.UserID = userID

	// Call the reward calculation and saving function
	// err := m.RewardAndSavePoints(mediaCount, report)
	// if err != nil {
	// 	return err
	// }

	if err := m.DB.Create(media).Error; err != nil {
		return err
	}
	return nil
}

// func (m *mediaRepo) RewardAndSavePoints(mediaCount int, report *models.IncidentReport) error {
// 	const pointsPerMedia = 10
// 	// Calculate total points for media
// 	mediaPoints := mediaCount * pointsPerMedia
// 	// Calculate other points based on report details
// 	otherPoints := calculatePoints(mediaCount, report)
// 	// Sum all points
// 	totalPoints := mediaPoints + otherPoints
// 	report.RewardPoint = totalPoints

// 	return nil
// }

// func calculatePoints(mediaCount int, report *models.IncidentReport) int {
// 	points := 0
// 	if report.Description != "" {
// 		points += 10
// 	}
// 	if report.Latitude != 0 && report.Longitude != 0 {
// 		points += 10
// 	}
// 	if report.MediaCount > 0 {
// 		points += report.MediaCount * 10
// 	}
// 	return points
// }

// GetMediaCountByByUserID fetches the current media count for a given user
func (repo *mediaRepo) GetMediaCountByByUserID(userID uint) (int, error) {
	var media models.Media
	err := repo.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&media).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Handle the case where no rewards exist for the user
			return 0, nil
		}
		return 0, err
	}
	return media.Count, nil
}

func (repo *mediaRepo) CreateMediaCount(mediaCount *models.MediaCount) error {
	if err := repo.DB.Create(mediaCount).Error; err != nil {
		return err
	}
	return nil
}

//	func (repo *mediaRepo) GetMediaCount() (models.MediaCount, error) {
//	    // Fetch media count from the database
//	    count, err := repo.mediaRepo.GetMediaCount()
//	    if err != nil {
//	        return MediaCount{}, err
//	    }
//	    return count, nil
//	}
func (repo *mediaRepo) UploadMediaToS3(file multipart.File, fileHeader *multipart.FileHeader, bucketName, folderName string) (string, error) {
	defer file.Close()

	// Sanitize the filename by replacing spaces with underscores
	sanitizedFilename := strings.ReplaceAll(fileHeader.Filename, " ", "_")

	// Generate a unique key for the file
	key := fmt.Sprintf("%s/%s", folderName, sanitizedFilename)

	// Create an S3 client
	client, err := createS3Client()
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %v", err)
	}

	// Upload the file to S3
	fileURL, err := uploadFileToS3(client, file, bucketName, key)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return fileURL, nil
}
