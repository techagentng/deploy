package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	// "github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"github.com/techagentng/citizenx/config"
	fig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

type MediaService interface {
	ProcessMedia(c *gin.Context, formMedia []*multipart.FileHeader, userID uint, reportID string) ([]string, []string, []string, []string, error)
	SaveMedia(media models.Media, reportID string, userID uint, imageCount int, videoCount int, audioCount int, totalPoints int) error
	ProcessSingleMedia(mediaFile *multipart.FileHeader, userID uint, reportID string) (string, string, string, error)
	downloadVideo(feedURL string) (string, error)
	generateThumbnailFromVideo(videoFilePath string) (string, error)
	UploadThumbnailToStorage(thumbnailPath, bucketName, folderName string) (string, error)
	GenerateThumbnail(feedURL string) (string, error)
	captureThumbnail(videoFilePath string) (string, error)
	ProcessVideoFile(mediaFile *multipart.FileHeader, userID uint, reportIDStr string) (string, string, string, error)
	SaveToStorage(file multipart.File, fileName, bucketName, folderName string) error
	GenerateVideoThumbnail(file multipart.File, outputPath string) error
	// GenerateImageThumbnail(file multipart.File, thumbnailPath string) error
	ProcessImageFile(mediaFile *multipart.FileHeader, userID uint, reportIDStr string) (string, string, string, error)
	GenerateImageThumbnail(mediaFile *multipart.FileHeader, thumbnailPath string) error
	UploadFileToS3(mediaFile *multipart.FileHeader, userID uint, fileType string) (string, error)
}

type mediaService struct {
	Config             *config.Config
	mediaRepo          db.MediaRepository
	rewardRepo         db.RewardRepository
	IncidentReportRepo db.IncidentReportRepository
}

func NewMediaService(mediaRepo db.MediaRepository, rewardRepo db.RewardRepository, reportRepo db.IncidentReportRepository, conf *config.Config) MediaService {
	return &mediaService{
		Config:             conf,
		mediaRepo:          mediaRepo,
		rewardRepo:         rewardRepo,
		IncidentReportRepo: reportRepo,
	}
}

const MaxAudioFileSize = 10 * 1024 * 1024 // 10 MB

func CheckFileSize(fileHeader *multipart.FileHeader) error {
	if fileHeader.Size > MaxAudioFileSize {
		return errors.New("file size exceeds the maximum allowed size")
	}
	return nil
}

func CheckSupportedFile(filename string) (bool, string) {
	supportedFileTypes := map[string]bool{
		".png":  true,
		".jpeg": true,
		".jpg":  true,
		".mp3":  true,
		".wav":  true,
		".ogg":  true,
		".mp4":  true,
	}

	fileExtension := filepath.Ext(filename)
	return supportedFileTypes[fileExtension], fileExtension
}

func generateUniqueFilename(extension string) string {
	timestamp := time.Now().UnixNano()
	randomUUID := uuid.New()
	return fmt.Sprintf("%d_%s%s", timestamp, randomUUID, extension)
}

// Define a struct to hold the results of each goroutine
type ProcessResult struct {
	FeedURL      string
	ThumbnailURL string
	FullSizeURL  string
	FileType     string
	Error        error
}

// Change the parameter type to []*multipart.FileHeader to handle multiple files
func (m *mediaService) ProcessMedia(c *gin.Context, formMedia []*multipart.FileHeader, userID uint, reportID string) ([]string, []string, []string, []string, error) {
	var (
		feedURLs, thumbnailURLs, fullsizeURLs, fileTypes []string
		mu                                               sync.Mutex
		wg                                               sync.WaitGroup
		bucketName                                       = os.Getenv("AWS_BUCKET")
		results                                          = make(chan *ProcessResult, len(formMedia)) // len is valid now as formMedia is a slice
	)

	// Launch a goroutine for each file to process it concurrently
	for _, fileHeader := range formMedia {
		wg.Add(1)
		go func(fileHeader *multipart.FileHeader) {
			defer wg.Done()

			// Open the file
			file, err := fileHeader.Open()
			if err != nil {
				results <- &ProcessResult{Error: fmt.Errorf("failed to open file: %v", err)}
				return
			}
			defer file.Close()

			// Read the file content
			fileBytes, err := ioutil.ReadAll(file)
			if err != nil {
				results <- &ProcessResult{Error: fmt.Errorf("failed to read file: %v", err)}
				return
			}

			// Reset file pointer to the beginning after reading it
			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				results <- &ProcessResult{Error: fmt.Errorf("failed to seek file: %v", err)}
				return
			}

			fileType := getFileType(fileBytes)
			var feedURL, thumbnailURL, fullsizeURL string

			// Define the folder name based on the file type
			folderName := ""
			switch fileType {
			case "image":
				folderName = "images"
				feedURL, thumbnailURL, fullsizeURL, err = processAndStoreImage(fileBytes)
				if err != nil {
					results <- &ProcessResult{Error: fmt.Errorf("failed to process and store image: %v", err)}
					return
				}
			case "video":
				folderName = "videos"
				feedURL, thumbnailURL, fullsizeURL, err = processAndStoreVideo(fileBytes)
				if err != nil {
					results <- &ProcessResult{Error: fmt.Errorf("failed to process and store video: %v", err)}
					return
				}
			case "audio":
				folderName = "audio"
				feedURL, thumbnailURL, err = processAndStoreAudio(fileBytes)
				if err != nil {
					results <- &ProcessResult{Error: fmt.Errorf("failed to process and store audio: %v", err)}
					return
				}
			default:
				results <- &ProcessResult{Error: fmt.Errorf("unsupported file type: %s", fileType)}
				return
			}

			// Upload the processed media to S3
			feedURL, err = m.mediaRepo.UploadMediaToS3(file, fileHeader, bucketName, folderName)
			if err != nil {
				results <- &ProcessResult{Error: fmt.Errorf("failed to upload media to S3: %v", err)}
				return
			}

			// Return the results through the channel
			results <- &ProcessResult{
				FeedURL:      feedURL,
				ThumbnailURL: thumbnailURL,
				FullSizeURL:  fullsizeURL,
				FileType:     fileType,
				Error:        nil,
			}
		}(fileHeader)
	}

	// Close the results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results from the channel
	for result := range results {
		if result.Error != nil {
			return nil, nil, nil, nil, fmt.Errorf("error processing media: %v", result.Error)
		}
		mu.Lock()
		feedURLs = append(feedURLs, result.FeedURL)
		if result.ThumbnailURL != "" {
			thumbnailURLs = append(thumbnailURLs, result.ThumbnailURL)
		}
		if result.FullSizeURL != "" {
			fullsizeURLs = append(fullsizeURLs, result.FullSizeURL)
		}
		fileTypes = append(fileTypes, result.FileType)
		mu.Unlock()
	}

	return feedURLs, thumbnailURLs, fullsizeURLs, fileTypes, nil
}

func getFileType(fileBytes []byte) string {
	// Determine the file type based on the file signature (magic number)
	fileType := http.DetectContentType(fileBytes)

	// Log the detected content type
	log.Printf("Detected content type: %s", fileType)

	switch fileType {
	case "image/jpeg", "image/jpg":
		return "image"
	case "image/png":
		return "image"
	case "image/gif":
		return "image"
	case "video/mp4":
		return "video"
	case "video/avi":
		return "video"
	case "video/quicktime":
		return "video"
	case "audio/mpeg":
		return "audio"
	case "audio/wav":
		return "audio"
	case "audio/ogg":
		return "audio"
	case "audio/flac":
		return "audio"
	case "application/ogg":
		return "audio"
	default:
		// If the file type is not recognized, return "unknown"
		return "unknown"
	}
}

// ImageResult represents the result of processing an image, video, or audio file.
type ImageResult struct {
	FeedURL      string
	ThumbnailURL string
	FullsizeURL  string
	Err          error
}

func (m *mediaService) processFile(fileCh <-chan *multipart.FileHeader, resultCh chan<- ImageResult, userID uint, reportID string) {
	for f := range fileCh {
		go func(f *multipart.FileHeader) {
			// Process and store the file
			file, err := f.Open()
			if err != nil {
				log.Printf("Error opening file: %v\n", err)
				resultCh <- ImageResult{"", "", "", err}
				return
			}
			defer file.Close()

			fileBytes, err := io.ReadAll(file)
			if err != nil {
				log.Printf("Error reading file: %v\n", err)
				resultCh <- ImageResult{"", "", "", err}
				return
			}

			supported, ext := CheckSupportedFile(f.Filename)
			if !supported {
				log.Printf("Unsupported file type: %s\n", f.Filename)
				resultCh <- ImageResult{"", "", "", fmt.Errorf("unsupported file type: %s", f.Filename)}
				return
			}

			var result ImageResult
			if strings.HasPrefix(ext, ".jpg") || strings.HasPrefix(ext, ".jpeg") || strings.HasPrefix(ext, ".png") {
				feedURL, thumbnailURL, fullsizeURL, err := processAndStoreImage(fileBytes)
				result = ImageResult{feedURL, thumbnailURL, fullsizeURL, err}
			} else if strings.HasPrefix(ext, ".mp4") {
				videoURL, thumbnailURL, _, err := processAndStoreVideo(fileBytes)
				result = ImageResult{videoURL, thumbnailURL, "", err}
			} else if strings.HasPrefix(ext, ".mp3") || strings.HasPrefix(ext, ".wav") || strings.HasPrefix(ext, ".ogg") {
				audioURL, _, err := processAndStoreAudio(fileBytes)
				result = ImageResult{audioURL, "", "", err}
			}

			resultCh <- result
		}(f)
	}
}

// Image processing
func processAndStoreImage(fileBytes []byte) (string, string, string, error) {
	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode image: %v", err)
	}

	feedImg := imaging.Fill(img, 1080, 1080, imaging.Center, imaging.Lanczos)
	thumbnailImg := imaging.Resize(img, 161, 161, imaging.Lanczos)
	fullSizeImg := img

	feedFilename := generateUniqueFilename(".jpg")
	thumbnailFilename := generateUniqueFilename(".jpg")
	fullSizeFilename := generateUniqueFilename(".jpg")

	feedDestPath := filepath.Join("media", "feed", feedFilename)
	thumbnailDestPath := filepath.Join("media", "thumbnail", thumbnailFilename)
	fullSizeDestPath := filepath.Join("media", "fullsize", fullSizeFilename)

	log.Printf("Feed directory path: %s", filepath.Dir(feedDestPath))
	log.Printf("Thumbnail directory path: %s", filepath.Dir(thumbnailDestPath))
	log.Printf("Fullsize directory path: %s", filepath.Dir(fullSizeDestPath))

	if err := os.MkdirAll(filepath.Dir(feedDestPath), 0755); err != nil {
		return "", "", "", fmt.Errorf("error creating feed folder: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(thumbnailDestPath), 0755); err != nil {
		return "", "", "", fmt.Errorf("error creating thumbnail folder: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(fullSizeDestPath), 0755); err != nil {
		return "", "", "", fmt.Errorf("error creating fullsize folder: %v", err)
	}

	feedFile, err := os.Create(feedDestPath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create feed file: %v", err)
	}
	defer feedFile.Close()

	thumbnailFile, err := os.Create(thumbnailDestPath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create thumbnail file: %v", err)
	}
	defer thumbnailFile.Close()

	fullSizeFile, err := os.Create(fullSizeDestPath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create fullsize file: %v", err)
	}
	defer fullSizeFile.Close()

	if err := jpeg.Encode(feedFile, feedImg, nil); err != nil {
		return "", "", "", fmt.Errorf("failed to encode feed image: %v", err)
	}
	if err := jpeg.Encode(thumbnailFile, thumbnailImg, nil); err != nil {
		return "", "", "", fmt.Errorf("failed to encode thumbnail image: %v", err)
	}
	if err := jpeg.Encode(fullSizeFile, fullSizeImg, nil); err != nil {
		return "", "", "", fmt.Errorf("failed to encode fullsize image: %v", err)
	}

	log.Printf("Successfully stored images: feed=%s, thumbnail=%s, fullsize=%s", feedDestPath, thumbnailDestPath, fullSizeDestPath)
	return feedDestPath, thumbnailDestPath, fullSizeDestPath, nil
}

func isValidFileType(fileType string) bool {
	switch fileType {
	case "image", "video", "voice_note":
		return true
	default:
		return false
	}
}

func ValidateMedia(file []byte, fileType string, filename string) (models.Media, error) {
	var media models.Media

	if !isValidFileType(fileType) {
		return media, errors.New("invalid file type")
	}

	media.FileType = fileType
	media.FileSize = int64(len(file))
	media.Filename = filename

	if fileType == "image" {
		width, height, err := getImageDimensions(file)
		if err != nil {
			return media, err
		}
		media.Width = width
		media.Height = height
	} else if fileType == "video" {
		width, height, err := getVideoDimensions(file)
		if err != nil {
			return media, err
		}
		media.Width = width
		media.Height = height
	}

	return media, nil
}

func getVideoDimensions(file []byte) (int, int, error) {
	cmd := exec.Command("ffmpeg", "-i", "-", "-hide_banner")
	cmd.Stdin = bytes.NewReader(file)
	var out bytes.Buffer
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return 0, 0, err
	}

	output := out.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Video:") {
			dimensions := strings.Split(line, ",")[2]
			dimensions = strings.TrimSpace(strings.Split(dimensions, " ")[0])
			widthHeight := strings.Split(dimensions, "x")
			if len(widthHeight) != 2 {
				return 0, 0, errors.New("failed to extract dimensions")
			}
			width, err := strconv.Atoi(widthHeight[0])
			if err != nil {
				return 0, 0, err
			}
			height, err := strconv.Atoi(widthHeight[1])
			if err != nil {
				return 0, 0, err
			}
			return width, height, nil
		}
	}

	return 0, 0, errors.New("video dimensions not found")
}

func getImageDimensions(file []byte) (int, int, error) {
	img, _, err := image.DecodeConfig(bytes.NewReader(file))
	if err != nil {
		return 0, 0, err
	}
	return img.Width, img.Height, nil
}

func (m *mediaService) SaveMedia(media models.Media, reportID string, userID uint, imageCount int, videoCount int, audioCount int, totalPoints int) error {
	// Generate a new UUID for the media ID
	ID := uuid.New()
	media.ID = ID.String()
	media.UserID = userID

	// Multiply totalPoints by 10
	rewardPoints := totalPoints * 10

	// Set the points on the media (remove the redundant assignment)
	media.Points = rewardPoints

	// Save the media to the database
	err := m.mediaRepo.SaveMedia(media, reportID, userID)
	if err != nil {
		return err
	}

	// Create and save the media count for the report
	var mcount models.MediaCount
	mcount.Images = imageCount
	mcount.Videos = videoCount
	mcount.Audios = audioCount
	mcount.IncidentReportID = reportID
	mcount.UserID = userID

	// Get the user's reward record
	reward, err := m.rewardRepo.GetRewardByUserID(userID)
	if err != nil {
		return err
	}

	// If no reward record exists, create a new one
	if reward == nil {
		reward = &models.Reward{
			UserID:  userID,
			Balance: 0,
			Point:   0,
		}
	}

	// Update the reward balance and points
	reward.Balance += rewardPoints
	reward.Point += rewardPoints

	// Save the updated reward record
	err = m.rewardRepo.SaveReward(reward)
	if err != nil {
		return err
	}

	return nil
}

func processAndStoreVideo(fileBytes []byte) (string, string, string, error) {
	videoFilename := generateUniqueFilename(".mp4")
	thumbnailFilename := generateUniqueFilename(".jpg")

	videoDestPath := filepath.Join("media", "video", videoFilename)
	thumbnailDestPath := filepath.Join("media", "thumbnail", thumbnailFilename)

	log.Printf("Creating directories for video and thumbnail if they don't exist")
	if err := os.MkdirAll(filepath.Dir(videoDestPath), 0755); err != nil {
		return "", "", "", fmt.Errorf("error creating video folder: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(thumbnailDestPath), 0755); err != nil {
		return "", "", "", fmt.Errorf("error creating thumbnail folder: %v", err)
	}

	log.Printf("Creating temporary file for the uploaded video")
	tempFile, err := os.CreateTemp("", "uploaded_video_*.mp4")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	log.Printf("Temporary file created: %s", tempFile.Name())

	log.Printf("Writing video bytes to the temporary file")
	if _, err := tempFile.Write(fileBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to write temporary video file: %v", err)
	}
	tempFile.Close()

	log.Printf("Executing ffmpeg command to process the video")
	ffmpegCmd := exec.Command("ffmpeg", "-i", tempFile.Name(), "-vf", "scale=1080:-2", "-t", "60", "-c:a", "copy", "-preset", "fast", "-crf", "23", videoDestPath)
	var stderr bytes.Buffer
	ffmpegCmd.Stderr = &stderr
	if err := ffmpegCmd.Run(); err != nil {
		return "", "", "", fmt.Errorf("ffmpeg error: %v, details: %s", err, stderr.String())
	}
	log.Printf("Video processed and saved to: %s", videoDestPath)

	log.Printf("Generating thumbnail for the video")
	ffmpegCmd = exec.Command("ffmpeg", "-i", tempFile.Name(), "-vf", "thumbnail", "-frames:v", "1", thumbnailDestPath)
	stderr.Reset()
	ffmpegCmd.Stderr = &stderr
	if err := ffmpegCmd.Run(); err != nil {
		return "", "", "", fmt.Errorf("ffmpeg thumbnail error: %v, details: %s", err, stderr.String())
	}
	log.Printf("Thumbnail generated and saved to: %s", thumbnailDestPath)

	return videoDestPath, thumbnailDestPath, "video", nil
}

func processAndStoreAudio(fileBytes []byte) (string, string, error) {
	audioFilename := generateUniqueFilename(".mp3")

	audioDestPath := filepath.Join("media", "audio", audioFilename)

	if err := os.MkdirAll(filepath.Dir(audioDestPath), 0755); err != nil {
		return "", "", fmt.Errorf("error creating audio folder: %v", err)
	}

	audioFile, err := os.Create(audioDestPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create audio file: %v", err)
	}
	defer audioFile.Close()

	if _, err := audioFile.Write(fileBytes); err != nil {
		return "", "", fmt.Errorf("failed to write audio file: %v", err)
	}

	return audioDestPath, "audio", nil
}

// ProcessSingleMedia processes a single media file and returns URLs for different formats.
func (m *mediaService) ProcessSingleMedia(mediaFile *multipart.FileHeader, userID uint, reportID string) (string, string, string, error) {
	// Open the media file
	file, err := mediaFile.Open()
	if err != nil {
		return "", "", "", fmt.Errorf("unable to open media file: %w", err)
	}
	defer file.Close()

	// Define buffer size for content type detection
	const bufferSize = 512
	buffer := make([]byte, bufferSize)

	// Read the first 512 bytes of the file for MIME type detection
	_, err = file.Read(buffer)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read file for content type detection: %w", err)
	}

	// Detect content type
	fileType := http.DetectContentType(buffer)
	fileSize := mediaFile.Size

	// Reset file position to the beginning after reading
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to reset file read position: %w", err)
	}

	var feedURL, thumbnailURL, fullsizeURL string

	// Image processing
	if strings.HasPrefix(fileType, "image") {
		if fileSize > 10*1024*1024 { // Example limit of 10MB for images
			return "", "", "", fmt.Errorf("image file size exceeds limit")
		}
		// Retrieve the bucket name from environment variables
		bucketName := os.Getenv("AWS_BUCKET")
		if bucketName == "" {
			return "", "", "", fmt.Errorf("AWS_BUCKET environment variable is not set")
		}

		folderName := "media"

		feedURL, err = m.mediaRepo.UploadMediaToS3(file, mediaFile, bucketName, folderName)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to upload feed image: %w", err)
		}

		// Generate thumbnail for the image
		thumbnailURL, err = m.GenerateThumbnail(feedURL)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to create thumbnail: %w", err)
		}

		fullsizeURL = feedURL // Assuming full-size is the original uploaded image URL

		// Video processing
	} else if strings.HasPrefix(fileType, "video") {
		if fileSize > 80*1024*1024 { // Example limit of 80MB for videos
			return "", "", "", fmt.Errorf("video file size exceeds limit")
		}

		// Retrieve the bucket name from environment variables
		bucketName := os.Getenv("AWS_BUCKET")
		if bucketName == "" {
			return "", "", "", fmt.Errorf("AWS_BUCKET environment variable is not set")
		}

		// Define folder name for video upload (e.g., "videos")
		folderName := "videos"

		// Upload video to S3 and get feed URL
		feedURL, err := m.mediaRepo.UploadMediaToS3(file, mediaFile, bucketName, folderName)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to upload feed video: %w", err)
		}

		// Extract video thumbnail
		thumbnailURL, err = m.ExtractVideoThumbnail(feedURL)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to extract video thumbnail: %w", err)
		}

		fullsizeURL = feedURL // Use feed URL as full-size video URL

	} else {
		return "", "", "", fmt.Errorf("unsupported file type: %s", fileType)
	}

	// Return the generated URLs
	return feedURL, thumbnailURL, fullsizeURL, nil
}

func (m *mediaService) downloadVideo(feedURL string) (string, error) {
	// Create a temporary file for the video
	tempFile, err := os.CreateTemp("", "video-*.mp4")
	if err != nil {
		return "", fmt.Errorf("unable to create temp file for video: %w", err)
	}
	defer tempFile.Close()

	// Download the video
	resp, err := http.Get(feedURL)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save video to file: %w", err)
	}

	return tempFile.Name(), nil
}

func (m *mediaService) generateThumbnailFromVideo(videoFilePath string) (string, error) {
	// Create a temporary file for the thumbnail
	thumbnailPath := strings.Replace(videoFilePath, ".mp4", "_thumbnail.jpg", 1)

	// Execute FFmpeg command to generate the thumbnail
	cmd := exec.Command("ffmpeg", "-i", videoFilePath, "-ss", "00:00:01.000", "-vframes", "1", thumbnailPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate thumbnail with FFmpeg: %w", err)
	}

	return thumbnailPath, nil
}

func (m *mediaService) UploadThumbnailToStorage(thumbnailPath, bucketName, folderName string) (string, error) {
	// Open the thumbnail file
	file, err := os.Open(thumbnailPath)
	if err != nil {
		return "", fmt.Errorf("unable to open thumbnail file: %v", err)
	}
	defer file.Close()

	// Retrieve file info for creating FileHeader
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to get file info: %v", err)
	}

	// Create a FileHeader using the file's information
	fileHeader := &multipart.FileHeader{
		Filename: fileInfo.Name(),
		Size:     fileInfo.Size(),
	}

	// Use UploadMediaToS3 to upload the thumbnail
	thumbnailURL, err := m.mediaRepo.UploadMediaToS3(file, fileHeader, bucketName, folderName)
	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail to S3: %v", err)
	}

	return thumbnailURL, nil
}

func (m *mediaService) ExtractVideoThumbnail(feedURL string) (string, error) {
	// Step 1: Download the video from feedURL to a temporary local file
	videoFilePath, err := m.downloadVideo(feedURL)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer os.Remove(videoFilePath) // Ensure the file is cleaned up afterward

	// Step 2: Generate a thumbnail from the video file
	thumbnailFilePath, err := m.generateThumbnailFromVideo(videoFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to generate thumbnail: %w", err)
	}
	defer os.Remove(thumbnailFilePath) // Clean up the thumbnail file after uploading

	// Step 3: Upload the thumbnail to S3 or another storage service
	thumbnailURL, err := m.uploadThumbnailToStorage(thumbnailFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	return thumbnailURL, nil
}

func (m *mediaService) GenerateThumbnail(feedURL string) (string, error) {
	// Step 1: Download the video from feedURL to a temporary file
	videoFilePath, err := m.downloadVideoFile(feedURL)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer os.Remove(videoFilePath) // Ensure the file is deleted after we're done

	// Step 2: Generate a thumbnail from the video file
	thumbnailFilePath, err := m.captureThumbnail(videoFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to capture thumbnail: %w", err)
	}
	defer os.Remove(thumbnailFilePath) // Ensure the thumbnail file is deleted after upload

	// Step 3: Upload the thumbnail to storage and get its URL
	thumbnailURL, err := m.uploadThumbnailToStorage(thumbnailFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail: %w", err)
	}

	// Return the URL of the uploaded thumbnail
	return thumbnailURL, nil
}

func (m *mediaService) downloadVideoFile(feedURL string) (string, error) {
	// Create a temporary file to store the video
	tempFile, err := os.CreateTemp("", "video-*.mp4")
	if err != nil {
		return "", fmt.Errorf("unable to create temp file for video: %w", err)
	}
	defer tempFile.Close()

	// Download the video content
	resp, err := http.Get(feedURL)
	if err != nil {
		return "", fmt.Errorf("failed to download video from URL: %w", err)
	}
	defer resp.Body.Close()

	// Write the downloaded video to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save video to file: %w", err)
	}

	return tempFile.Name(), nil
}

func (m *mediaService) captureThumbnail(videoFilePath string) (string, error) {
	// Define a path for the generated thumbnail
	thumbnailPath := strings.Replace(videoFilePath, ".mp4", "_thumbnail.jpg", 1)

	// Use FFmpeg to capture a frame at the 1-second mark as a thumbnail
	cmd := exec.Command("ffmpeg", "-i", videoFilePath, "-ss", "00:00:01.000", "-vframes", "1", thumbnailPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("FFmpeg failed to generate thumbnail: %w", err)
	}

	return thumbnailPath, nil
}

// ProcessVideoFile processes a video file, generates a feed URL, a thumbnail, and a full-size URL.
func (m *mediaService) ProcessVideoFile(mediaFile *multipart.FileHeader, userID uint, reportIDStr string) (string, string, string, error) {
    // Open the video file
    file, err := mediaFile.Open()
    if err != nil {
        log.Printf("Error opening video file: %v", err)
        return "", "", "", fmt.Errorf("error opening video file: %v", err)
    }
    defer file.Close()

    // Sanitize and define file paths
    sanitizedFilename := strings.ReplaceAll(mediaFile.Filename, " ", "_")
    videoFileName := fmt.Sprintf("%d_%s", userID, sanitizedFilename)
    thumbnailFileName := fmt.Sprintf("%d_%s_thumbnail.jpg", userID, reportIDStr)

    bucketName := os.Getenv("AWS_BUCKET")
    folderName := "media" // Folder name for S3

    // URLs for S3 storage
    feedURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s", bucketName, os.Getenv("AWS_REGION"), folderName, videoFileName)
    fullSizeURL := feedURL
    thumbnailURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s", bucketName, os.Getenv("AWS_REGION"), folderName, thumbnailFileName)

    // Step 1: Save video to S3
    if err := m.SaveToStorage(file, videoFileName, bucketName, folderName); err != nil {
        log.Printf("Error saving video to S3: %v", err)
        return "", "", "", fmt.Errorf("error saving video to S3: %v", err)
    }

    // // Step 2: Generate a thumbnail using FFmpeg
    // thumbnailPath := fmt.Sprintf("/tmp/%s", thumbnailFileName) // Temporary path
    // if err := m.GenerateVideoThumbnail(file, thumbnailPath); err != nil {
    //     log.Printf("Error generating video thumbnail: %v", err)
    //     return "", "", "", fmt.Errorf("error generating video thumbnail: %v", err)
    // }

    // Step 3: Upload the thumbnail to S3
    // if err := m.UploadThumbnailToStorage(thumbnailPath, bucketName, folderName); err != nil {
    //     log.Printf("Error uploading video thumbnail to S3: %v", err)
    //     return "", "", "", fmt.Errorf("error uploading video thumbnail to S3: %v", err)
    // }

    // Step 4: Clean up temporary files
    // if err := os.Remove(thumbnailPath); err != nil {
    //     log.Printf("Error cleaning up thumbnail file: %v", err)
    // }

    log.Printf("Successfully processed video file: %s", mediaFile.Filename)

    return feedURL, thumbnailURL, fullSizeURL, nil
}


func (m *mediaService) SaveToStorage(file multipart.File, fileName, bucketName, folderName string) error {
	// Upload file to S3 using the repository method
	fileURL, err := m.mediaRepo.UploadMediaToS3(file, &multipart.FileHeader{Filename: fileName}, bucketName, folderName)
	if err != nil {
		return fmt.Errorf("error uploading file to S3: %v", err)
	}

	// Log the file URL
	log.Printf("File successfully uploaded to S3: %s", fileURL)

	// You can return the file URL or save it to the database if needed
	return nil
}

func (m *mediaService) GenerateVideoThumbnail(file multipart.File, outputPath string) error {
    // Use FFmpeg command to extract a thumbnail at a specified time (e.g., 1 second)
    cmd := exec.Command("ffmpeg", "-i", "/dev/stdin", "-ss", "00:00:01", "-vframes", "1", outputPath)
    cmd.Stdin = file
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to generate thumbnail: %w", err)
    }
    log.Printf("Generated thumbnail at %s", outputPath)
    return nil
}

func (m *mediaService) uploadThumbnailToStorage(thumbnailFilePath string) (string, error) {
	// Open the thumbnail file
	file, err := os.Open(thumbnailFilePath)
	if err != nil {
		return "", fmt.Errorf("unable to open thumbnail file: %v", err)
	}
	defer file.Close()

	// Retrieve file info for creating FileHeader
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to get file info: %v", err)
	}

	// Create a FileHeader using the file's information
	fileHeader := &multipart.FileHeader{
		Filename: fileInfo.Name(),
		Size:     fileInfo.Size(),
	}

	// Get bucket and folder names from environment variables
	bucketName := os.Getenv("BUCKET_NAME")
	folderName := os.Getenv("FOLDER_NAME")

	// Use UploadMediaToS3 to upload the thumbnail
	thumbnailURL, err := m.mediaRepo.UploadMediaToS3(file, fileHeader, bucketName, folderName)
	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail to S3: %v", err)
	}

	return thumbnailURL, nil
}

func (s *mediaService) ProcessImageFile(mediaFile *multipart.FileHeader, userID uint, reportIDStr string) (string, string, string, error) {
    // Open the image file
    file, err := mediaFile.Open()
    if err != nil {
        log.Printf("Error opening image file: %v", err)
        return "", "", "", fmt.Errorf("error opening image file: %v", err)
    }
    defer file.Close()

    // Generate a unique identifier (using UUID and timestamp)
    uniqueID := fmt.Sprintf("%s_%d", uuid.New().String(), time.Now().UnixNano())

    // Sanitize and define file storage paths
    sanitizedFilename := strings.ReplaceAll(mediaFile.Filename, " ", "_")
    imageFileName := fmt.Sprintf("%d_%s_%s", userID, uniqueID, sanitizedFilename)

    feedURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s",
        os.Getenv("AWS_BUCKET"),
        os.Getenv("AWS_REGION"),
        "media2", // Folder name in S3
        imageFileName,
    )
    fullSizeURL := feedURL // Full-size image URL would be the same for now

    // Generate a unique filename for the thumbnail
    thumbnailFileName := fmt.Sprintf("%d_%s_thumbnail.jpg", userID, uniqueID)
    thumbnailURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s",
        os.Getenv("AWS_BUCKET"),
        os.Getenv("AWS_REGION"),
        "media2", // Folder name in S3
        thumbnailFileName,
    )

    // Define S3 bucket and folder name
    bucketName := os.Getenv("AWS_BUCKET") // Use the AWS_BUCKET environment variable
    folderName := "media2"               // Folder name where the file will be stored in S3

    // Step 1: Save the original image file to S3 storage
    if err := s.SaveToStorage(file, imageFileName, bucketName, folderName); err != nil {
        log.Printf("Error saving image to storage: %v", err)
        return "", "", "", fmt.Errorf("error saving image to storage: %v", err)
    }

    // Step 2: Generate a thumbnail for the image
    // Ensure the directory for the thumbnail exists
    thumbnailDir := "/tmp" // Temporary processing directory
    if err := os.MkdirAll(thumbnailDir, os.ModePerm); err != nil {
        log.Printf("Error creating thumbnail directory: %v", err)
        return "", "", "", fmt.Errorf("error creating thumbnail directory: %v", err)
    }

    thumbnailPath := fmt.Sprintf("%s/%s", thumbnailDir, thumbnailFileName) // Full path for the thumbnail
    if err := s.GenerateImageThumbnail(mediaFile, thumbnailPath); err != nil {
        log.Printf("Error generating image thumbnail: %v", err)
        return "", "", "", fmt.Errorf("error generating image thumbnail: %v", err)
    }

    // Step 3: Upload the thumbnail to S3 storage
    thumbnailURL, err = s.UploadThumbnailToStorage(thumbnailPath, bucketName, folderName)
    if err != nil {
        log.Printf("Error uploading thumbnail: %v", err)
        return "", "", "", fmt.Errorf("error uploading thumbnail: %v", err)
    }

    // Clean up the temporary thumbnail file
    if err := os.Remove(thumbnailPath); err != nil {
        log.Printf("Error deleting temporary thumbnail file: %v", err)
    }

    log.Printf("Processed image file successfully: %s", mediaFile.Filename)

    return feedURL, thumbnailURL, fullSizeURL, nil
}



func (s *mediaService) GenerateImageThumbnail(mediaFile *multipart.FileHeader, thumbnailPath string) error {
    // Open the file from the multipart.FileHeader
    file, err := mediaFile.Open()
    if err != nil {
        log.Printf("Error opening media file: %v", err)
        return fmt.Errorf("error opening media file: %v", err)
    }
    defer file.Close()

    // Decode the image
    img, _, err := image.Decode(file)
    if err != nil {
        log.Printf("Error decoding image: %v", err)
        return fmt.Errorf("error decoding image: %v", err)
    }

    // Resize or process the image as needed (using an example thumbnail size)
    thumbnail := resize.Resize(200, 0, img, resize.Lanczos3) // Requires github.com/nfnt/resize

    // Create the thumbnail file
    outFile, err := os.Create(thumbnailPath)
    if err != nil {
        log.Printf("Error creating thumbnail file: %v", err)
        return fmt.Errorf("error creating thumbnail file: %v", err)
    }
    defer outFile.Close()

    // Encode the thumbnail as a JPEG
    if err := jpeg.Encode(outFile, thumbnail, nil); err != nil {
        log.Printf("Error encoding thumbnail to JPEG: %v", err)
        return fmt.Errorf("error encoding thumbnail to JPEG: %v", err)
    }

    log.Printf("Thumbnail generated successfully: %s", thumbnailPath)
    return nil
}

func (s *mediaService) UploadFileToS3(mediaFile *multipart.FileHeader, userID uint, fileType string) (string, error) {
	// Open the media file
	file, err := mediaFile.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Generate a unique file key for the S3 bucket
	fileExtension := filepath.Ext(mediaFile.Filename)
	fileKey := fmt.Sprintf("media/%d_%s_%s%s", userID, fileType, uuid.New().String(), fileExtension)

	// Prepare the S3 bucket name from environment variables
	bucketName := os.Getenv("AWS_BUCKET")
	if bucketName == "" {
		return "", fmt.Errorf("S3 bucket name is not configured")
	}

	// Load AWS config
	cfg, err := fig.LoadDefaultConfig(context.TODO(),
		fig.WithRegion(os.Getenv("AWS_REGION")),
		fig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		),
	)
	if err != nil {
		return "", fmt.Errorf("unable to load AWS config: %v", err)
	}

	// Create S3 client
	svc := s3.NewFromConfig(cfg)

	// Stream file directly to S3
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(fileKey),
		Body:        file,                       // Stream the file content
		ACL:         "public-read",              // Make the file publicly accessible
		ContentType: aws.String(mediaFile.Header.Get("Content-Type")), // Use the file's content type
	}

	// Upload the file
	_, err = svc.PutObject(context.TODO(), putObjectInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	// Construct the file URL
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, os.Getenv("AWS_REGION"), fileKey)

	return fileURL, nil
}
