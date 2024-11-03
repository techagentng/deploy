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
		Name: state,
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

		// Retrieve full name and profile image from context
		fullNameInterface, exists := c.Get("fullName")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Full name not found"})
			return
		}

		fullName, ok := fullNameInterface.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid type for full name"})
			return
		}
		userNameInterface, exists := c.Get("username")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Full name not found"})
			return
		}

		username, ok := userNameInterface.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid type for full name"})
			return
		}
		profileImageInterface, exists := c.Get("profile_image")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Profile image not found"})
			return
		}

		profileImage, ok := profileImageInterface.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid type for profile image"})
			return
		}

		// Create and populate the IncidentReport model
		incidentReport := &models.IncidentReport{
			ID:              reportID,
			UserFullname:    fullName,
			UserUsername:    username,
			DateOfIncidence: c.PostForm("date_of_incidence"),
			Description:     c.PostForm("description"),
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
		}

		// Create and populate the ReportType model
		reportType := &models.ReportType{
			ID:                   uuid.New(),
			UserID:               user.ID,
			IncidentReportID:     reportID,
			Category:             incidentReport.Category,
			StateName:            incidentReport.StateName,
			LGAName:              incidentReport.LGAName,
			IncidentReportRating: incidentReport.Rating,
			DateOfIncidence:      time.Now(),
		}

		// Save ReportType
		if _, err := s.IncidentReportRepository.SaveReportType(reportType); err != nil {
			log.Printf("Error saving report type: %v\n", err)
			response.JSON(c, "Unable to save report type", http.StatusInternalServerError, nil, err)
			return
		}

		// Create and populate the SubReport model
		subReport := &models.SubReport{
			ID:            uuid.New(),
			ReportTypeID:  reportType.ID,
			SubReportType: c.PostForm("sub_report_type"),
		}

		// Save SubReport
		savedSubReport, err := s.IncidentReportRepository.SaveSubReport(subReport)
		if err != nil {
			log.Printf("Error saving sub-report: %v\n", err)
			response.JSON(c, "Unable to save sub-report", http.StatusInternalServerError, nil, err)
			return
		}

		// Save the incident report to the database
		savedIncidentReport, err := s.IncidentReportService.SaveReport(user.ID, lat, lng, incidentReport, reportID.String(), 0)
		if err != nil {
			log.Printf("Error saving incident report: %v\n", err)
			response.JSON(c, "Unable to save incident report", http.StatusInternalServerError, nil, err)
			return
		}

		// Return reportID, reportTypeID, and subReportID in the response
		response.JSON(c, "Incident Report Submitted Successfully", http.StatusCreated, gin.H{
			"reportID":            reportID.String(),
			"reportTypeID":        reportType.ID.String(),
			"subReportID":         savedSubReport.ID.String(),
			"savedIncidentReport": savedIncidentReport,
		}, nil)
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
		// Extract and validate reportTypeID from the form-data
		reportID := c.PostForm("report_id")
		if reportID == "" {
			response.JSON(c, "Missing report type ID", http.StatusBadRequest, nil, nil)
			return
		}

		// Process and save media files, while updating the report with media links and reward points
		feedURLs, thumbnailURLs, fullsizeURLs, fileTypes, err := s.processAndSaveMedia(c)
		if err != nil {
			log.Printf("Error processing media: %v", err)
			response.JSON(c, "Unable to process media files", http.StatusInternalServerError, nil, err)
			return
		}

		// Successful media upload response
		response.JSON(c, "Media added to report successfully", http.StatusOK, gin.H{
			"reportID":      reportID,
			"feedURLs":      feedURLs,
			"thumbnailURLs": thumbnailURLs,
			"fullsizeURLs":  fullsizeURLs,
			"fileTypes":     fileTypes,
		}, nil)
	}
}

func (s *Server) processAndSaveMedia(c *gin.Context) ([]string, []string, []string, []string, error) {
	// Retrieve media files from the multipart form
	formMedia := c.Request.MultipartForm.File["mediaFiles"]
	if formMedia == nil {
		return nil, nil, nil, nil, fmt.Errorf("no media files found in the request")
	}

	// Initialize URL and file type slices
	var feedURLs, thumbnailURLs, fullsizeURLs, fileTypes []string
	var imageCount, videoCount, audioCount int

	userID, exists := c.Get("userID")
	if !exists {
		return nil, nil, nil, nil, fmt.Errorf("unauthorized: userID not found in context")
	}

	userIDUint := userID.(uint)

	// Fetch the last report ID of the current user
	reportIDStr, err := s.IncidentReportRepository.GetLastReportIDByUserID(userIDUint)
	if err != nil {
		log.Printf("Error fetching last report ID: %v\n", err)
		return nil, nil, nil, nil, fmt.Errorf("error fetching last report ID: %v", err)
	}

	processedFeedURLs, processedThumbnailURLs, processedFullsizeURLs, processedFileTypes, err := s.MediaService.ProcessMedia(c, formMedia, userIDUint, reportIDStr)
	if err != nil {
		log.Printf("Error processing media: %v\n", err)
		return nil, nil, nil, nil, fmt.Errorf("error processing media: %v", err)
	}

	// Append the processed URLs and types to the respective slices
	feedURLs = append(feedURLs, processedFeedURLs...)
	thumbnailURLs = append(thumbnailURLs, processedThumbnailURLs...)
	fullsizeURLs = append(fullsizeURLs, processedFullsizeURLs...)
	fileTypes = append(fileTypes, processedFileTypes...)

	// Retrieve the incident report by reportID using the repository
	incidentReport, err := s.IncidentReportRepository.GetIncidentReportByID(reportIDStr)
	if err != nil {
		log.Printf("Error retrieving report: %v\n", err)
		return nil, nil, nil, nil, fmt.Errorf("error retrieving report: %v", err)
	}

	// Check if the incident report has an associated ReportType
	if incidentReport.ReportTypeID != uuid.Nil {
		log.Printf("Incident report has an associated ReportType: %v", incidentReport.ReportTypeID)
		// We are not updating or creating a new ReportType since it already exists
	} else {
		// If for some reason, the report doesn't have an associated ReportType, handle that here
		log.Printf("Incident report is missing a ReportType, which is unexpected")
		return nil, nil, nil, nil, fmt.Errorf("incident report is missing a ReportType")
	}

	// Update the fields in the incident report with the processed media URLs
	incidentReport.FeedURLs = strings.Join(feedURLs, ",")
	incidentReport.ThumbnailURLs = strings.Join(thumbnailURLs, ",")
	incidentReport.FullSizeURLs = strings.Join(fullsizeURLs, ",")

	// Use the repository function to update the incident report
	if err := s.IncidentReportRepository.UpdateIncidentReport(incidentReport); err != nil {
		log.Printf("Error updating incident report: %v\n", err)
		return nil, nil, nil, nil, fmt.Errorf("error updating incident report: %v", err)
	}

	// Update counters based on the processed file types
	for _, fileType := range processedFileTypes {
		switch fileType {
		case "image":
			imageCount++
		case "video":
			videoCount++
		case "audio":
			audioCount++
		}
	}

	// Save each processed media to the database
	for i := 0; i < len(processedFeedURLs); i++ {
		mediaModel := models.Media{
			UserID:       userIDUint,
			FeedURL:      processedFeedURLs[i],
			ThumbnailURL: processedThumbnailURLs[i],
			FullSizeURL:  processedFullsizeURLs[i],
			FileType:     processedFileTypes[i],
		}

		// Calculate total points (example logic, adjust as needed)
		totalPoints := (imageCount * 5) + (videoCount * 10) + (audioCount * 8)

		// Save the processed media with the correct parameters
		if err := s.MediaService.SaveMedia(mediaModel, reportIDStr, userIDUint, imageCount, videoCount, audioCount, totalPoints); err != nil {
			log.Printf("Error saving media: %v\n", err)
			return nil, nil, nil, nil, fmt.Errorf("error saving media: %v", err)
		}
	}

	return feedURLs, thumbnailURLs, fullsizeURLs, fileTypes, nil
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

		// Fetch reports for the user
		reports, err := s.IncidentReportRepository.GetAllIncidentReportsByUser(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			"message":    "BlockRequest updated successfully",
			"report_id":  reportID,
			"user_message": payload.Message,
		})
	}
}