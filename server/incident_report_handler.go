package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// "github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/techagentng/citizenx/errors"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
	jwtPackage "github.com/techagentng/citizenx/services/jwt"
	"gorm.io/gorm"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	DefaultPageSize = 20
	DefaultPage     = 1
	MaxChunkSize    = 5 << 20 // 5 MB
)

// Define mediaResult struct
type mediaResult struct {
	FeedURL      string
	ThumbnailURL string
	FullSizeURL  string
	FileType     string
	Error        error
}

// saveChunk processes and saves the media file, associating it with the given reportID and userID.
func (s *Server) saveChunk(fileHeader *multipart.FileHeader, reportID, userID string, results chan<- mediaResult) {
	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to open file %s: %v", fileHeader.Filename, err)}
		return
	}
	defer file.Close()

	// Define a temporary directory using the userID and reportID for better organization
	tempDir := filepath.Join("./path/to/temp", userID, reportID) // Replace with actual temp directory path
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to create temp directory %s: %v", tempDir, err)}
		return
	}

	log.Printf("Processing file: %s, size: %d bytes", fileHeader.Filename, fileHeader.Size)

	// Create a unique temporary file path to store the uploaded content
	tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to create temp file %s: %v", tempFilePath, err)}
		return
	}
	defer func() {
		tempFile.Close()
		// Clean up the temp file after use
		if err := os.Remove(tempFilePath); err != nil {
			log.Printf("Failed to remove temp file %s: %v", tempFilePath, err)
		}
	}()

	// Copy the uploaded file content to the temporary file
	if _, err = io.Copy(tempFile, file); err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to copy file content to temp file %s: %v", tempFilePath, err)}
		return
	}

	// Replace the following block with actual logic to process the media file and generate real URLs

	// Generate URLs using reportID and userID for storage organization
	feedURL := fmt.Sprintf("/media/feed/%s/%s/%s", userID, reportID, fileHeader.Filename)
	thumbnailURL := fmt.Sprintf("/media/thumbnail/%s/%s/%s", userID, reportID, fileHeader.Filename)
	fullSizeURL := fmt.Sprintf("/media/fullsize/%s/%s/%s", userID, reportID, fileHeader.Filename)

	fileType := fileHeader.Header.Get("Content-Type")

	// Send the result with URLs and file type back through the channel
	results <- mediaResult{
		FeedURL:      feedURL,
		ThumbnailURL: thumbnailURL,
		FullSizeURL:  fullSizeURL,
		FileType:     fileType,
		Error:        nil,
	}
}

func GetUserFromContext(c *gin.Context) (*models.User, error) {
	if userI, exists := c.Get("user"); exists {
		if user, ok := userI.(*models.User); ok {
			return user, nil
		}
		return nil, errors.New("User is not logged in", http.StatusUnauthorized)
	}
	return nil, errors.New("user is not logged in", http.StatusUnauthorized)
}

// Function to check the supported file type
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

type MediaCount struct {
	Images           int
	Videos           int
	Audios           int
	UserID           uint
	IncidentReportID string
}

// Function to create media counts from mediaTypeCounts map
func CreateMediaCount(mediaTypeCounts map[string]int) (imageCount, videoCount, audioCount int) {
	for fileType, count := range mediaTypeCounts {
		switch fileType {
		case ".png", ".jpeg", ".jpg":
			imageCount += count
		case ".mp4":
			videoCount += count
		case ".mp3", ".wav", ".ogg":
			audioCount += count
		}
	}
	return imageCount, videoCount, audioCount
}

// Function to calculate the total number of media points
func calculateMediaPoints(mediaTypeCounts map[string]int) int {
	totalPoints := 0
	for _, count := range mediaTypeCounts {
		totalPoints += count
	}
	return totalPoints
}

type GeocodingResponse struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		Types []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

// Helper function to check if a slice contains a given element
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateIDx() uuid.UUID {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	return id

}

func fetchGeocodingData(lat, lng float64, c *gin.Context, reportID string) (*models.LGA, *models.State, *models.ReportType, string, string, error) {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s", lat, lng, apiKey)
	response, err := http.Get(url)
	if err != nil {
		return nil, nil, nil, "", "", fmt.Errorf("error fetching geocoding data: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, nil, nil, "", "", fmt.Errorf("unexpected status code: %v", response.StatusCode)
	}

	var geocodingResponse GeocodingResponse
	if err := json.NewDecoder(response.Body).Decode(&geocodingResponse); err != nil {
		return nil, nil, nil, "", "", fmt.Errorf("error decoding JSON response: %v", err)
	}

	var locality, state string
	for _, result := range geocodingResponse.Results {
		for _, component := range result.AddressComponents {
			if contains(component.Types, "locality") {
				locality = component.LongName
			} else if contains(component.Types, "administrative_area_level_1") {
				state = component.LongName
			}
		}
	}

	log.Printf("Fetched LGA: %s, State: %s", locality, state) // Log fetched values

	lga := &models.LGA{
		ID:   generateIDx(),
		Name: locality,
	}

	stateStruct := &models.State{
		ID:   generateIDx(),
		State: state,
	}

	// Check if user exists in the context
	userI, exists := c.Get("user")
	if !exists {
		log.Println("User not found in context")
		return nil, nil, nil, "", "", fmt.Errorf("user not found in context")
	}
	userId := userI.(*models.User).ID

	// Create reportType struct using PostForm values
	reportType := &models.ReportType{
		ID:                   generateIDx(),
		UserID:               userId,
		IncidentReportID:     parseUUID(reportID),
		StateName:            c.PostForm("state_name"),
		LGAName:              c.PostForm("lga_name"),
		Category:             c.PostForm("category"),
		IncidentReportRating: c.PostForm("rating"),
		DateOfIncidence:      time.Now(),
	}

	return lga, stateStruct, reportType, locality, state, nil
}

// Helper function to parse reportID from string to UUID
func parseUUID(reportID string) uuid.UUID {
	id, err := uuid.Parse(reportID)
	if err != nil {
		log.Fatalf("Invalid UUID format: %v", err)
	}
	return id
}

// Utility function to split URLs in a slice of strings if needed
func splitUrlSlice(urls []string) []string {
	var result []string
	for _, url := range urls {
		result = append(result, strings.Split(url, ",")...)
	}
	return result
}

func generateIDls() uuid.UUID {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	return id
}

var vulgarWords = []string {
    "Arsehole",
    "Asshat",
    "Asshole",
    "Bastard (slang)",
    "Big black cock",
    "Bitch (slang)",
    "Bloody",
    "Blowjob",
    "Bollocks",
    "Bugger",
    "Bullshit",
    "Chicken shit",
    "Clusterfuck",
    "Cock (slang)",
    "Cocksucker",
    "Coonass",
    "Cornhole (slang)",
    "Cox–Zucker machine",
    "Cracker (term)",
    "Cunt",
    "Damn",
    "Dick (slang)",
    "Enshittification",
    "Faggot",
    "Feck",
    "Fuck",
    "Fuck her right in the pussy",
    "Fuck Joe Biden",
    "Fuck, marry, kill",
    "Fuckery",
    "Grab 'em by the pussy",
    "Healslut",
    "Jesus fucking christ",
    "Kike",
    "Motherfucker",
    "Nigga",
    "Nigger",
    "Paki (slur)",
    "Poof",
    "Poofter",
    "Prick (slang)",
    "Pussy",
    "Ratfucking",
    "Retard (pejorative)",
    "Russian warship, go fuck yourself",
    "Shit",
    "Shit happens",
    "Shithouse",
    "Shitposting",
    "Shitter",
    "Shut the fuck up",
    "Shut the hell up",
    "Slut",
    "Son of a bitch",
    "Spic",
    "Taking the piss",
    "Twat",
    "Unclefucker",
    "Wanker",
    "Whore",
}


func containsVulgarWords(description string, words []string) bool {
	lowerDescription := strings.ToLower(description)
	for _, word := range words {
		if strings.Contains(lowerDescription, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

func (s *Server) handleIncidentReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve user from the context
		userI, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		user, ok := userI.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type"})
			return
		}

		// Generate new UUID for the report ID
		reportID := uuid.New()

		// Parse latitude and longitude from the form
		lat, lng, err := parseCoordinates(c)
		if err != nil {
			response.JSON(c, "Invalid latitude or longitude", http.StatusBadRequest, nil, err)
			return
		}

		// Retrieve description from the form
		description := c.PostForm("description")

		// Check for vulgar words in the description
		if containsVulgarWords(description, vulgarWords) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Description contains inappropriate language"})
			return
		}

		// Other data retrieval and processing...
		fullName := c.GetString("fullName")
		username := c.GetString("username")
		profileImage := c.GetString("profile_image")

		incidentReport := &models.IncidentReport{
			ID:              reportID,
			UserID:          user.ID,
			UserFullname:    fullName,
			UserUsername:    username,
			DateOfIncidence: c.PostForm("date_of_incidence"),
			Description:     description,
			StateName:       c.PostForm("state_name"),
			LGAName:         c.PostForm("lga_name"),
			Latitude:        lat,
			Longitude:       lng,
			Telephone:       c.PostForm("telephone"),
			Email:           c.PostForm("email"),
			Address:         c.PostForm("address"),
			Rating:          c.PostForm("rating"),
			Category:        c.PostForm("category"),
			ThumbnailURLs:   profileImage,
			TimeofIncidence: time.Now(),
		}

		// Save incident report and respond...
		savedIncidentReport, err := s.IncidentReportService.SaveReport(user.ID, lat, lng, incidentReport, reportID.String(), 0)
		if err != nil {
			log.Printf("Error saving incident report: %v\n", err)
			response.JSON(c, "Unable to save incident report", http.StatusInternalServerError, nil, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":             "Incident Report Submitted Successfully",
			"reportID":            reportID.String(),
			"savedIncidentReport": savedIncidentReport,
		})
	}
}

// Helper function to parse coordinates from the request form
func parseCoordinates(c *gin.Context) (float64, float64, error) {
	lat, lng := 0.0, 0.0
	var err error

	if latStr := strings.TrimSpace(c.PostForm("latitude")); latStr != "" {
		lat, err = strconv.ParseFloat(latStr, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid latitude: %v", err)
		}
	}

	if lngStr := strings.TrimSpace(c.PostForm("longitude")); lngStr != "" {
		lng, err = strconv.ParseFloat(lngStr, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid longitude: %v", err)
		}
	}

	return lat, lng, nil
}

func (s *Server) handleUploadMedia() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Retrieve report ID
        reportID := c.PostForm("report_id")
        if reportID == "" {
            response.JSON(c, "Missing report type ID", http.StatusBadRequest, nil, nil)
            return
        }

        // Process media files and collect URLs and types
        feedURLs, fullsizeURLs, _, _, fileTypes, err := s.processAndSaveMedia(c)
        if err != nil {
            log.Printf("Error processing media: %v", err)
            response.JSON(c, "Unable to process media files", http.StatusInternalServerError, nil, err)
            return
        }

        // Calculate points (10 points per media file)
        mediaCount := len(feedURLs) + len(fileTypes) // Count all media items
        points := mediaCount * 10

        // Retrieve user ID from context
        userI, exists := c.Get("user")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
            return
        }

        user, ok := userI.(*models.User)
        if !ok {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type"})
            return
        }

        // Save the reward
        reward := &models.Reward{
            IncidentReportID: reportID,
            UserID:           user.ID,
            RewardType:       "media_upload",
            Point:            points,
            Balance:          points, // Start with the same value for new rewards
        }

        if err := s.RewardRepository.SaveReward(reward); err != nil {
            log.Printf("Error saving reward: %v\n", err)
            response.JSON(c, "Unable to save reward", http.StatusInternalServerError, nil, err)
            return
        }

        // Respond with successful media upload and reward details
        response.JSON(c, "Media added to report successfully", http.StatusOK, gin.H{
            "reportID":     reportID,
            "feedURLs":     feedURLs,
            "fullsizeURLs": fullsizeURLs,
            "fileTypes":    fileTypes,
            "reward":       gin.H{"points": points, "balance": reward.Balance},
        }, nil)
    }
}



func (s *Server) processAndSaveMedia(c *gin.Context) ([]string, []string, []string, []string, []string, error) {
	// Parse multipart form with a size limit
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil { // 100 MB limit
		log.Printf("Error parsing multipart form: %v", err)
		return nil, nil, nil, nil, nil, fmt.Errorf("error parsing multipart form: %v", err)
	}

	// Retrieve files
	formMedia := c.Request.MultipartForm.File["mediaFiles"]
	if formMedia == nil {
		log.Println("No media files found in the request")
		return nil, nil, nil, nil, nil, fmt.Errorf("no media files found in the request")
	}

	// Initialize URL and file type slices
	var feedURLs, thumbnailURLs, fullsizeURLs, fileTypes []string
	var videoURL, audioURL string

	userID, exists := c.Get("userID")
	if !exists {
		log.Println("Error: userID not found in context")
		return nil, nil, nil, nil, nil, fmt.Errorf("unauthorized: userID not found in context")
	}

	userIDUint := userID.(uint)

	// Fetch the last report ID of the current user
	reportIDStr, err := s.IncidentReportRepository.GetLastReportIDByUserID(userIDUint)
	if err != nil {
		log.Printf("Error fetching last report ID: %v\n", err)
		return nil, nil, nil, nil, nil, fmt.Errorf("error fetching last report ID: %v", err)
	}

	// Iterate through uploaded media files
	for _, mediaFile := range formMedia {
		log.Printf("Processing file: %s, Size: %d, Content-Type: %s", mediaFile.Filename, mediaFile.Size, mediaFile.Header.Get("Content-Type"))

		fileType := detectFileType(mediaFile.Filename)
		var processedFeedURL, processedThumbnailURL, processedFullsizeURL string

		switch fileType {
		case "image":
			processedFeedURL, processedThumbnailURL, processedFullsizeURL, err = s.MediaService.ProcessImageFile(mediaFile, userIDUint, reportIDStr)
			if err == nil {
				feedURLs = append(feedURLs, processedFeedURL)
				thumbnailURLs = append(thumbnailURLs, processedThumbnailURL)
				fullsizeURLs = append(fullsizeURLs, processedFullsizeURL)
				fileTypes = append(fileTypes, "image")
			}
		case "video":
			if mediaFile.Size > 100*1024*1024 {
				return nil, nil, nil, nil, nil, fmt.Errorf("video file %s exceeds the 100 MB size limit", mediaFile.Filename)
			}
			videoURL, err = s.MediaService.UploadFileToS3(mediaFile, userIDUint, "video")
			if err == nil {
				feedURLs = append(feedURLs, videoURL)
				fileTypes = append(fileTypes, "video")
			}
		case "audio":
			if mediaFile.Size > 50*1024*1024 {
				return nil, nil, nil, nil, nil, fmt.Errorf("audio file %s exceeds the 50 MB size limit", mediaFile.Filename)
			}
			audioURL, err = s.MediaService.UploadFileToS3(mediaFile, userIDUint, "audio")
			if err == nil {
				feedURLs = append(feedURLs, audioURL)
				fileTypes = append(fileTypes, "audio")
			}
		default:
			log.Printf("Unsupported file type for %s", mediaFile.Filename)
			return nil, nil, nil, nil, nil, fmt.Errorf("unsupported file type: %s", fileType)
		}

		if err != nil {
			log.Printf("Error processing media file %s: %v\n", mediaFile.Filename, err)
			return nil, nil, nil, nil, nil, err
		}
	}

	// Update incident report with the processed URLs
	incidentReport, err := s.IncidentReportRepository.GetIncidentReportByID(reportIDStr)
	if err != nil {
		log.Printf("Error retrieving report: %v\n", err)
		return nil, nil, nil, nil, nil, fmt.Errorf("error retrieving report: %v", err)
	}

	incidentReport.FeedURLs = strings.Join(feedURLs, ",")
	incidentReport.ThumbnailURLs = strings.Join(thumbnailURLs, ",")
	incidentReport.FullSizeURLs = strings.Join(fullsizeURLs, ",")
	incidentReport.VideoURL = videoURL
	incidentReport.AudioURL = audioURL

	// Save the updated report
	if err := s.IncidentReportRepository.UpdateIncidentReport(incidentReport); err != nil {
		log.Printf("Error updating incident report: %v\n", err)
		return nil, nil, nil, nil, nil, fmt.Errorf("error updating incident report: %v", err)
	}

	// Return the final URLs
	log.Println("Media processed and saved successfully")
	return feedURLs, fullsizeURLs, fileTypes, nil, nil, err
}




// Helper function to detect file type based on extension (expand as needed)
func detectFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		return "image"
	}
	if ext == ".mp4" || ext == ".mov" {
		return "video"
	}
	if ext == ".mp3" || ext == ".wav" {
		return "audio"
	}
	return "unknown"
}

func (s *Server) handleGetAllReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		pageStr := c.Query("page")
		if pageStr == "" {
			pageStr = "1"
		}
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
			return
		}

		reports, err := s.IncidentReportService.GetAllReports(page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"incident_reports": reports})
	}
}

func (s *Server) handleGetAllReportsByState() gin.HandlerFunc {
	return func(c *gin.Context) {
		state := c.Param("state")
		if state == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "State parameter is required"})
			return
		}

		page, err := getPageFromQuery(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
			return
		}

		reports, err := s.IncidentReportService.GetAllReportsByState(state, page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"incident_reports": reports})
	}
}

func (s *Server) handleGetAllReportsByLGA() gin.HandlerFunc {
	return func(c *gin.Context) {
		lga := c.Param("lga")
		if lga == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "LGA parameter is required"})
			return
		}

		page, err := getPageFromQuery(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
			return
		}

		reports, err := s.IncidentReportService.GetAllReportsByLGA(lga, page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"incident_reports": reports})
	}
}

func (s *Server) handleGetAllReportsByReportType() gin.HandlerFunc {
	return func(c *gin.Context) {
		report_type := c.Param("report_type")
		if report_type == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report type parameter is required"})
			return
		}

		page, err := getPageFromQuery(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
			return
		}

		reports, err := s.IncidentReportService.GetAllReportsByReportType(report_type, page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"incident_reports": reports})
	}
}

func getPageFromQuery(c *gin.Context) (int, error) {
	pageStr := c.Query("page")
	if pageStr == "" {
		return DefaultPage, nil
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, err
	}

	return page, nil
}

func generateID() (string, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return newUUID.String(), nil
}

func (s *Server) handleApproveReportPoints() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the report by ID
		reportID := c.Param("reportID")
		userID := c.Param("userID")
		userID64, err := strconv.ParseUint(userID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID"})
		}

		if reportID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
			return
		}

		report, err := s.IncidentReportRepository.GetReportByID(reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Check if the report is already approved
		if report.ReportStatus == "approved" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report already approved"})
			return
		}

		// Reward points to the user for the approved report
		if err := s.RewardService.ApproveReportPoints(reportID, uint(userID64)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Report approved and points rewarded successfully"})
	}
}

func (s *Server) handleRejectReportPoints() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the report by ID
		reportID := c.Param("reportID")
		userID := c.Param("userID")
		userID64, err := strconv.ParseUint(userID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID"})
		}

		if reportID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
			return
		}

		report, err := s.IncidentReportRepository.GetReportByID(reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Check if the report is already approved
		if report.ReportStatus == "rejected" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report already approved"})
			return
		}

		// Reward points to the user for the approved report
		if err := s.RewardService.RejectReportPoints(reportID, uint(userID64)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Report rejected successfully"})
	}
}

func (s *Server) handleAcceptReportPoints() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the report by ID
		reportID := c.Param("reportID")
		userID := c.Param("userID")
		userID64, err := strconv.ParseUint(userID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID"})
		}

		if reportID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
			return
		}

		report, err := s.IncidentReportRepository.GetReportByID(reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Check if the report is already approved
		if report.ReportStatus == "accepted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Report already approved"})
			return
		}

		// Reward points to the user for the approved report
		if err := s.RewardService.AcceptReportPoints(reportID, uint(userID64)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Report accepted successfully"})
	}
}

func (s *Server) handleGetReportPercentageByState() gin.HandlerFunc {
	return func(c *gin.Context) {
		percentages, err := s.IncidentReportService.GetReportPercentageByState()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": percentages})
	}
}

func (s *Server) handleGetTodayReportCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		todayReport, err := s.IncidentReportRepository.GetReportsPostedTodayCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": todayReport})
	}
}

// func (s *Server) handleWebSocket() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// Upgrade HTTP connection to WebSocket connection
// 		conn, err := websocket.Upgrade(c.Writer, c.Request, nil, 1024, 1024)
// 		if err != nil {
// 			// Handle upgrade error
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}
// 		defer conn.Close()

// 		// Infinite loop to listen for changes and broadcast updates
// 		for {
// 			// Fetch latest report count
// 			// count, err := s.IncidentReportService.GetReportsPostedTodayCount()
// 			if err != nil {
// 				// Handle error
// 				continue
// 			}

// 			// Send data to client
// 			err = conn.WriteJSON(map[string]int64{"count": count})
// 			if err != nil {
// 				// Handle write error
// 				break
// 			}

//				// Sleep for some time (e.g., 5 seconds) before checking again
//				time.Sleep(5 * time.Second)
//			}
//		}
//	}
func (s *Server) handleGetTotalUserCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := s.IncidentReportService.GetTotalUserCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"total_users": count})
	}
}

func (s *Server) GetRegisteredUsersCountByLGA() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the LGA from the URL parameter
		lga := c.Param("lga")
		var user models.User
		user.LGAName = lga
		// Call the service method to get the count of registered users by LGA
		count, err := s.IncidentReportService.GetRegisteredUsersCountByLGA(lga)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return the count as JSON response
		c.JSON(http.StatusOK, gin.H{"registered_users_count": count})
	}
}

func (s *Server) handleGetAllReportsByStateByTime() gin.HandlerFunc {
	return func(c *gin.Context) {
		state := c.Param("state")
		startTimeStr := c.Query("start_time")
		endTimeStr := c.Query("end_time")
		pageStr := c.DefaultQuery("page", "1")

		// Parse the start and end time
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format"})
			return
		}

		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format"})
			return
		}

		// Parse the page number
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
			return
		}

		// Fetch the reports from the repository
		reports, err := s.IncidentReportRepository.GetAllReportsByStateByTime(state, startTime, endTime, page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, reports)
	}
}

// Handler to check user activity and update status to online
func (s *Server) handleGetUserActivity() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user from the context set by the Authorize middleware
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No user found"})
			return
		}

		// Type assert user to models.User
		u, ok := user.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User data is corrupted"})
			return
		}

		// Update user's online status in the database
		u.Online = true
		if err := s.AuthRepository.UpdateUserStatus(u); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User status updated to online", "user": u})
	}
}

func (s *Server) handleGetReportsByTypeAndLGA() gin.HandlerFunc {
	return func(c *gin.Context) {
		reportType := c.Query("reportType")
		lga := c.Query("lga")
		if reportType == "" || lga == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reportType and lga are required"})
			return
		}

		reports, err := s.IncidentReportService.GetReportsByTypeAndLGA(reportType, lga)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"reports": reports})
	}
}

func (s *Server) handleGetReportTypeCounts() gin.HandlerFunc {
	return func(c *gin.Context) {
		state := strings.TrimSpace(c.Query("state"))
		lga := strings.TrimSpace(c.Query("lga"))
		startDate := strings.TrimSpace(c.Query("start_date"))
		endDate := strings.TrimSpace(c.Query("end_date"))

		if state == "" || lga == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "State and LGA are required"})
			return
		}

		reportTypes, reportCounts, totalUsers, totalReports, topStates, err := s.IncidentReportService.GetReportTypeCounts(state, lga, &startDate, &endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert the slice of StateReportCount to a map
		topStatesMap := make(map[string]int)
		for _, stateReport := range topStates {
			topStatesMap[stateReport.StateName] = stateReport.ReportCount
		}

		c.JSON(http.StatusOK, gin.H{
			"report_types":  reportTypes,
			"report_counts": reportCounts,
			"total_users":   totalUsers,
			"total_reports": totalReports,
			"top_states":    topStatesMap,
		})
	}
}

// Handler function to get LGAs in a state
func (s *Server) handleGetLGAs() gin.HandlerFunc {
	return func(c *gin.Context) {
		stateName := c.Query("state")
		if stateName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "State name is required"})
			return
		}

		apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
		if apiKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Google API key is not set"})
			return
		}

		northeast, southwest, err := getStateBounds(stateName, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		lgas, err := getLGAsInState(northeast, southwest, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"lgas": lgas})
	}
}

func getLGAsInState(northeast, southwest [2]float64, apiKey string) ([]string, error) {
	lat := (northeast[0] + southwest[0]) / 2
	lng := (northeast[1] + southwest[1]) / 2
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/textsearch/json?query=local+government+area&location=%f,%f&radius=50000&key=%s", lat, lng, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var placesResponse PlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&placesResponse); err != nil {
		return nil, err
	}

	if placesResponse.Status != "OK" {
		return nil, fmt.Errorf("places API error: %s", placesResponse.Status)
	}

	var lgas []string
	for _, result := range placesResponse.Results {
		lgas = append(lgas, result.Name)
	}

	return lgas, nil
}

func getStateBounds(stateName, apiKey string) (northeast, southwest [2]float64, err error) {
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s", stateName, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return [2]float64{}, [2]float64{}, err
	}
	defer resp.Body.Close()

	var geocodingResponse GeocodingResponses
	if err := json.NewDecoder(resp.Body).Decode(&geocodingResponse); err != nil {
		return [2]float64{}, [2]float64{}, err
	}

	if geocodingResponse.Status != "OK" {
		return [2]float64{}, [2]float64{}, fmt.Errorf("geocoding API error: %s", geocodingResponse.Status)
	}

	northeast[0] = geocodingResponse.Results[0].Geometry.Bounds.Northeast.Lat
	northeast[1] = geocodingResponse.Results[0].Geometry.Bounds.Northeast.Lng
	southwest[0] = geocodingResponse.Results[0].Geometry.Bounds.Southwest.Lat
	southwest[1] = geocodingResponse.Results[0].Geometry.Bounds.Southwest.Lng

	return northeast, southwest, nil
}

type GeocodingResponses struct {
	Results []struct {
		Geometry struct {
			Bounds struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"bounds"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}

// Define the structure of the Google Places API response
type PlacesResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
	Status string `json:"status"`
}

func (s *Server) IncidentMarkersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		markers, err := s.IncidentReportRepository.GetIncidentMarkers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, markers)
	}
}

func (s *Server) DeleteIncidentReportHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		err := s.IncidentReportRepository.DeleteByID(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Incident report not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete incident report"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Incident report deleted successfully"})
	}
}

func (s *Server) HandleGetStateReportCounts() gin.HandlerFunc {
	return func(c *gin.Context) {
		reportCounts, err := s.IncidentReportRepository.GetStateReportCounts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": reportCounts})
	}
}

func (s *Server) HandleGetVariadicBarChart() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse request body into ReportCriteria struct
		var criteria models.ReportCriteria
		if err := c.BindJSON(&criteria); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Call function to get report counts
		stateReportCounts, err := s.IncidentReportRepository.GetVariadicStateReportCounts(
			criteria.ReportTypes, // Include report types
			criteria.States,
			criteria.StartDate,
			criteria.EndDate,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Respond with report counts (assuming suitable format for bar chart)
		c.JSON(http.StatusOK, stateReportCounts)
	}
}

func (s *Server) handleGetAllCategories() gin.HandlerFunc {
	return func(c *gin.Context) {
		categories, err := s.IncidentReportRepository.GetAllCategories()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"categories": categories})
	}
}

func (s *Server) handleGetAllStates() gin.HandlerFunc {
	return func(c *gin.Context) {
		states, err := s.IncidentReportRepository.GetAllStates()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"states": states})
	}
}

func (s *Server) handleGetRatingPercentages() gin.HandlerFunc {
	return func(c *gin.Context) {
		reportType := c.Query("reportType")
		state := c.Query("state")

		if reportType == "" || state == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reportType and state query parameters are required"})
			return
		}

		percentages, err := s.IncidentReportRepository.GetRatingPercentages(reportType, state)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "No data found for the specified report type and state"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rating percentages"})
			}
			return
		}

		c.JSON(http.StatusOK, percentages)
	}
}

func (s *Server) handleGetAllReportsByStateAndLGA() gin.HandlerFunc {
	return func(c *gin.Context) {
		reportCounts, err := s.IncidentReportRepository.GetReportCountsByStateAndLGA()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"report_counts": reportCounts})
	}
}

func (s *Server) handleListAllStatesWithReportCounts() gin.HandlerFunc {
	return func(c *gin.Context) {
		statesWithReports, err := s.IncidentReportService.ListAllStatesWithReportCounts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"top_states": statesWithReports,
		})
	}
}

// Define the handler function
func (s *Server) handleGetTotalReportCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call the service to get the total report count
		totalCount, err := s.IncidentReportService.GetTotalReportCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return the total count as JSON
		c.JSON(http.StatusOK, gin.H{
			"total_report_count": totalCount,
		})
	}
}

func (s *Server) handleGetNamesByCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve parameters from query
		stateName := c.Query("state_name")
		lga := c.Query("lga")
		reportTypeCategory := c.Query("category")

		// Call the service method
		names, err := s.IncidentReportService.GetNamesByCategory(stateName, lga, reportTypeCategory)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return the names as JSON response
		c.JSON(http.StatusOK, gin.H{"names": names})
	}
}

func (s *Server) HandleGetSubReportsByCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the category from query parameters
		category := c.Query("category")
		if category == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
			return
		}

		// Fetch sub-reports from the repository
		subReports, err := s.IncidentReportRepository.GetSubReportsByCategory(category)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Respond with the sub-reports data
		c.JSON(http.StatusOK, gin.H{"data": subReports})
	}
}

func (s *Server) HandleGetAllReportsByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from the context
		userIDCtx, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
			return
		}

		// Assert the type of userID as uint
		userID, ok := userIDCtx.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			return
		}

		// Fetch page and limit query parameters for pagination
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))  // Default to page 1
		// limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20")) 

		// Fetch reports for the user
		reports, err := s.IncidentReportRepository.GetAllIncidentReportsByUser(userID, page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports: " + err.Error()})
			return
		}

		// Return the reports as a JSON response
		c.JSON(http.StatusOK, gin.H{"reports": reports})
	}
}


func (s *Server) HandleGetVoteCounts() gin.HandlerFunc {
	return func(c *gin.Context) {
		reportID := c.Param("reportID")
		upvotes, downvotes, err := s.LikeService.GetVoteCounts(reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"upvotes":   upvotes,
			"downvotes": downvotes,
		})
	}
}

func (s *Server) HandleBookmarkReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get userID from context
		userIDCtx, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user not authenticated",
			})
			return
		}

		userID, ok := userIDCtx.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "invalid user ID format",
			})
			return
		}

		// Get and parse reportID
		reportIDStr := c.Param("reportID")
		reportID, err := uuid.Parse(reportIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid report ID format",
			})
			return
		}

		// Call the bookmark service
		err = s.IncidentReportService.BookmarkReport(userID, reportID)
		if err != nil {
			status := http.StatusInternalServerError

			// Handle specific error cases
			switch err.Error() {
			case "report not found":
				status = http.StatusNotFound
			case "report already bookmarked":
				status = http.StatusConflict
			}

			c.JSON(status, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Success response
		c.JSON(http.StatusOK, gin.H{
			"message": "Report bookmarked successfully",
		})
	}
}

func (s *Server) HandleGetBookmarkedReports() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from context
		userIDCtx, ok := c.Get("userID")
		if !ok {
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID not found in context", http.StatusInternalServerError))
			return
		}

		// Assert the type of userID as uint
		userID, ok := userIDCtx.(uint)
		if !ok {
			response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("userID is not of type uint", http.StatusInternalServerError))
			return
		}

		// Fetch bookmarked reports
		bookmarkedReports, err := s.IncidentReportService.GetBookmarkedReports(userID)
		if err != nil {
			response.JSON(c, "", http.StatusInternalServerError, nil, err)
			return
		}

		// Return the bookmarked reports
		c.JSON(http.StatusOK, gin.H{
			"bookmarked_reports": bookmarkedReports,
		})
	}
}

func (s *Server) GetReportTypeCountsByLGA() gin.HandlerFunc {
	return func(c *gin.Context) {
		lga := c.Param("lga")

		// Get the result as a map from the service
		result, err := s.IncidentReportService.GetReportTypeCountsByLGA(lga)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Send the result map as JSON
		c.JSON(http.StatusOK, result)
	}
}

func (s *Server) GetReportCountsByStateAndLGA() gin.HandlerFunc {
	return func(c *gin.Context) {
		state := c.Param("state")

		lgas, reportCounts, err := s.IncidentReportRepository.GetReportCountsByState(state)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"lgas":          lgas,
			"report_counts": reportCounts,
		})
	}
}

func (s *Server) handleGetTopCategories() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call the repository function to get top categories and their counts
		categories, counts, err := s.IncidentReportRepository.GetTopCategories()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Respond with the categories and their counts
		c.JSON(http.StatusOK, gin.H{
			"categories":    categories,
			"report_counts": counts,
		})
	}
}

func (s *Server) GetReportsByCategory() gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")

		if category == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
			return
		}

		reports, err := s.IncidentReportRepository.GetReportsByCategory(category)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if len(reports) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No reports found for the specified category"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"reports": reports,
		})
	}
}

func (s *Server) handleGetReportsByFilters() gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")
		state := c.Query("state")
		lga := c.Query("lga")

		// Call the repository function with all filters
		reports, filters, err := s.IncidentReportRepository.GetFilteredIncidentReports(category, state, lga)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if len(reports) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No reports found for the specified filters"})
			return
		}

		// Return the reports and the applied filters
		c.JSON(http.StatusOK, gin.H{
			"reports": reports,
			"filters": filters,
		})
	}
}

// UpdateBlockRequestHandler handles the request to update BlockRequest
func (s *Server) UpdateBlockRequestHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the report ID from the request URL parameters
		reportIDParam := c.Param("post_id")
		reportID, err := uuid.Parse(reportIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
			return
		}

		// Bind the JSON payload to the struct
		var payload models.ReportPostRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload", "details": err.Error()})
			return
		}

		// Call the repository function to update the BlockRequest field
		err = s.IncidentReportRepository.UpdateBlockRequest(c.Request.Context(), reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return a success response with the provided message and report ID
		c.JSON(http.StatusOK, gin.H{
			"message":      "BlockRequest updated successfully",
			"report_id":    reportID,
			"user_message": payload.Message,
		})
	}
}

func (s *Server) BlockUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from URL parameters and parse it as uint
		userIDStr := c.Param("userID")
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
			return
		}

		// Convert userID to uint
		uintUserID := uint(userID)

		// Call the BlockUser function
		if err := s.IncidentReportRepository.BlockUser(c.Request.Context(), uintUserID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to block user"})
			return
		}

		// Respond with success message
		c.JSON(http.StatusOK, gin.H{"message": "user blocked successfully"})
	}
}

func (s *Server) ReportUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the access token from the authorization header
		accessToken := getTokenFromHeader(c)
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Validate and decode the access token to get the userID
		secret := s.Config.JWTSecret
		accessClaims, err := jwtPackage.ValidateAndGetClaims(accessToken, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Parse userID from the access claims as a uint
		var userID uint
		if id, ok := accessClaims["id"].(float64); ok {
			userID = uint(id)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID format"})
			return
		}

		// Set the is_queried field of the user to true
		if err := s.IncidentReportRepository.ReportUser(c.Request.Context(), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to report user"})
			return
		}

		// Respond with success message
		c.JSON(http.StatusOK, gin.H{"message": "User reported successfully"})
	}
}

func (s *Server) handleChangeUserRole() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Parse the user ID from the URL parameter
        userIDParam := c.Param("user_id")
        if userIDParam == "" {
            response.JSON(c, "User ID is required", http.StatusBadRequest, nil, nil)
            return
        }

        // Convert user ID to uint
        userID, err := strconv.ParseUint(userIDParam, 10, 32)
        if err != nil {
            response.JSON(c, "Invalid user ID", http.StatusBadRequest, nil, err)
            return
        }

        // Parse the new role name from the POST form data
        newRoleName := c.PostForm("role_name")
        if newRoleName == "" {
            response.JSON(c, "Role Name is required", http.StatusBadRequest, nil, nil)
            return
        }

        // Fetch the role by name
        role, err := s.AuthService.GetRoleByName(newRoleName)
        if err != nil {
            response.JSON(c, "Role not found", http.StatusBadRequest, nil, err)
            return
        }

        // Fetch the user by ID
        user, err := s.AuthRepository.GetUserByID(uint(userID)) // Convert to uint
        if err != nil {
            response.JSON(c, "User not found", http.StatusNotFound, nil, err)
            return
        }

        // Update the user's RoleID
        user.RoleID = role.ID
        err = s.AuthRepository.UpdateUserStatus(user)
        if err != nil {
            response.JSON(c, "Failed to update user role", http.StatusInternalServerError, nil, err)
            return
        }

        response.JSON(c, "User role updated successfully", http.StatusOK, user, nil)
    }
}

func (s *Server) HandleFollowReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse multipart form data with a max size of 20 MB
		if err := c.Request.ParseMultipartForm(20 << 20); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 20 MB"})
			return
		}

// Extract and parse the report_id from the URL
reportIDParam := c.Param("report_id")
reportID, err := uuid.Parse(reportIDParam)
if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID format"})
    return
}

		// Extract followText from the form data
		followText := c.PostForm("followText")

		// Extract followMedia (image/video) from the form data
		var mediaURL string
		file, handler, err := c.Request.FormFile("followMedia")
		if err == nil {
			defer file.Close()

			// Validate the file type (only image or video)
			if !isValidMediaType(handler.Header.Get("Content-Type")) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media type. Only image or video files are allowed"})
				return
			}

			// Create S3 client
			s3Client, err := createS3Client()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create S3 client"})
				return
			}

			// Generate a unique filename for the media
			mediaFilename := fmt.Sprintf("%s_%s", reportID.String(), handler.Filename)

			// Upload the file to S3
			mediaURL, err = uploadFileToS3(s3Client, file, os.Getenv("AWS_BUCKET"), mediaFilename)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload media"})
				return
			}
		}

		// Create a Follow instance with userID from context and reportID from URL
		userID := c.MustGet("userID").(uint) // UserID is set in the context by the authorization middleware
		follow := models.Follow{
			UserID:   userID,
			ReportID: reportID,
			FollowText: followText,
			FollowMedia: mediaURL,
		}

		// Call the repository to create a follow record
		if err := s.IncidentReportRepository.CreateFollow(follow); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow report"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully followed the report"})
	}
}

// Helper function to validate media type (image or video)
func isValidMediaType(contentType string) bool {
	// Allow only image and video types
	validTypes := []string{"image/jpeg", "image/png", "image/gif", "video/mp4", "video/avi", "video/mkv"}
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

func (s *Server) HandleGetFollowersByReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get reportID from the URL parameter and convert it to uuid.UUID
		reportIDStr := c.Param("report_id")
		reportID, err := uuid.Parse(reportIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
			return
		}

		// Fetch followers from the repository
		followers, err := s.IncidentReportRepository.GetFollowersByReport(reportID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch followers"})
			return
		}

		// Return the followers as a JSON response
		c.JSON(http.StatusOK, gin.H{"followers": followers})
	}
}

func (s *Server) handleGetReportCountByLGA() gin.HandlerFunc {
    return func(c *gin.Context) {
        lga := c.Param("lga")
        if lga == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "LGA parameter is required"})
            return
        }

        count, err := s.IncidentReportService.GetReportCountByLGA(lga)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, gin.H{"total_reports": count})
    }
}

func (s *Server) handleGetReportCountByState() gin.HandlerFunc {
    return func(c *gin.Context) {
        state := c.Param("state")
        if state == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "State parameter is required"})
            return
        }

        count, err := s.IncidentReportService.GetReportCountByState(state)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, gin.H{"total_reports": count})
    }
}
