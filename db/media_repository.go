package db

import (
	"fmt"
	"image"
	"image/jpeg"
	"mime/multipart"
	"os"
	"strings"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
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
	ProcessImageFile(mediaFile *os.File, userIDUint uint, reportIDStr string) (string, string, string, error)
	saveImageToStorage(img image.Image, userIDUint uint, reportIDStr string, sizeType string) (string, error)
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

func (repo *mediaRepo) ProcessImageFile(mediaFile *os.File, userIDUint uint, reportIDStr string) (string, string, string, error) {
	// Step 1: Open the image file
	img, _, err := image.Decode(mediaFile)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode image: %v", err)
	}

	// Step 2: Resize the image for different versions
	// Generate thumbnail (smaller version)
	thumbnail := resize.Resize(150, 0, img, resize.Lanczos3) // Resize to 150px width, maintaining aspect ratio
	// Generate feed-sized image (medium version)
	feedImage := resize.Resize(600, 0, img, resize.Lanczos3) // Resize to 600px width, maintaining aspect ratio
	// Generate full-size image (no resize for full-size)
	fullsizeImage := img // Keep original size for full-size

	// Step 3: Apply filters to enhance the image
	feedImage = repo.applyImageEnhancements(feedImage)
	thumbnail = repo.applyImageEnhancements(thumbnail)
	fullsizeImage = repo.applyImageEnhancements(fullsizeImage)

	// Step 4: Save the images to storage (local or cloud)
	thumbnailURL, err := repo.saveImageToStorage(thumbnail, userIDUint, reportIDStr, "thumbnail")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to save thumbnail: %v", err)
	}

	feedURL, err := repo.saveImageToStorage(feedImage, userIDUint, reportIDStr, "feed")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to save feed image: %v", err)
	}

	fullsizeURL, err := repo.saveImageToStorage(fullsizeImage, userIDUint, reportIDStr, "fullsize")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to save full-size image: %v", err)
	}

	// Step 5: Return the URLs for the different image versions
	return feedURL, thumbnailURL, fullsizeURL, nil
}

// saveImageToStorage saves an image to storage and returns the URL for the saved image
func (repo *mediaRepo) saveImageToStorage(img image.Image, userIDUint uint, reportIDStr string, sizeType string) (string, error) {
	// Generate a filename for the image based on the user ID and report ID
	filename := fmt.Sprintf("%d_%s_%s.jpg", userIDUint, reportIDStr, sizeType)

	// Save the image to a local file or storage system (e.g., S3, GCP, etc.)
	// Here we are saving it locally for simplicity
	filePath := fmt.Sprintf("./uploads/%s", filename)

	// Create the file on the filesystem
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Save the image in JPEG format (you can change this to PNG or other formats if needed)
	err = jpeg.Encode(file, img, nil)
	if err != nil {
		return "", fmt.Errorf("failed to encode image: %v", err)
	}

	// Generate the URL (this would be different if you're using cloud storage)
	// For example, if you're using local storage, you might serve the images via an HTTP server
	url := fmt.Sprintf("https://yourcdn.com/uploads/%s", filename)

	return url, nil
}
// applyImageEnhancements applies blur, sharpness, brightness, and other effects to the image
func (repo *mediaRepo) applyImageEnhancements(img image.Image) image.Image {
	// Apply a slight blur to enhance the image quality and make it look softer
	img = imaging.Blur(img, 2.0) // Apply a slight blur (the higher the number, the more blur)

	// Apply brightness adjustment
	img = imaging.AdjustBrightness(img, 10) // Increase brightness

	// Apply contrast adjustment
	img = imaging.AdjustContrast(img, 10) // Increase contrast

	// Apply saturation adjustment
	img = imaging.AdjustSaturation(img, 0.2) // Slightly increase saturation for a richer look

	return img
}