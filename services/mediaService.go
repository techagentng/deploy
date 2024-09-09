package services

import (
	"bytes"
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
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/models"
)

type MediaService interface {
	ProcessMedia(c *gin.Context, formMedia []*multipart.FileHeader, userID uint, reportID string) ([]string, []string, []string, []string, error)
	SaveMedia(media models.Media, reportID string, userID uint, imageCount int, videoCount int, audioCount int, totalPoints int) error
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

func (m *mediaService) ProcessMedia(c *gin.Context, formMedia []*multipart.FileHeader, userID uint, reportID string) ([]string, []string, []string, []string, error) {
    var feedURLs, thumbnailURLs, fullsizeURLs, fileTypes []string
    bucketName := os.Getenv("AWS_BUCKET") // Replace with your actual bucket name

    for _, fileHeader := range formMedia {
        file, err := fileHeader.Open()
        if err != nil {
            return nil, nil, nil, nil, fmt.Errorf("failed to open file: %v", err)
        }

        // Read the file content
        fileBytes, err := ioutil.ReadAll(file)
        if err != nil {
            return nil, nil, nil, nil, fmt.Errorf("failed to read file: %v", err)
        }

		// Reset file pointer to the beginning after reading it
		file.Seek(0, io.SeekStart)

        fileType := getFileType(fileBytes)
        var feedURL, thumbnailURL, fullsizeURL string

        // Define the folder name based on the file type
        folderName := ""
        switch fileType {
        case "image":
            folderName = "images"
            feedURL, thumbnailURL, fullsizeURL, err = processAndStoreImage(fileBytes)
            if err != nil {
                return nil, nil, nil, nil, fmt.Errorf("failed to process and store image: %v", err)
            }
        case "video":
            folderName = "videos"
            feedURL, thumbnailURL, fullsizeURL, err = processAndStoreVideo(fileBytes)
            if err != nil {
                return nil, nil, nil, nil, fmt.Errorf("failed to process and store video: %v", err)
            }
        case "audio":
            folderName = "audio"
            feedURL, thumbnailURL, err = processAndStoreAudio(fileBytes)
            if err != nil {
                return nil, nil, nil, nil, fmt.Errorf("failed to process and store audio: %v", err)
            }
        default:
            return nil, nil, nil, nil, fmt.Errorf("unsupported file type")
        }

        // Upload the processed media to S3
        feedURL, err = m.mediaRepo.UploadMediaToS3(file, fileHeader, bucketName, folderName)
        if err != nil {
            return nil, nil, nil, nil, fmt.Errorf("failed to upload media to S3: %v", err)
        }

        // Append the URLs and file type to their respective slices
        feedURLs = append(feedURLs, feedURL)
        thumbnailURLs = append(thumbnailURLs, thumbnailURL)
        fullsizeURLs = append(fullsizeURLs, fullsizeURL)
        fileTypes = append(fileTypes, fileType)
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

    fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxT", rewardPoints)

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
