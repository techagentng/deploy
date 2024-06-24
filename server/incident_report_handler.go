package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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

// saveChunk handles each file chunk and passes the file header to the ProcessMedia call.
func (s *Server) saveChunk(fileHeader *multipart.FileHeader, results chan<- mediaResult) {
	file, err := fileHeader.Open()
	if err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to open file: %v", err)}
		return
	}
	defer file.Close()

	// Ensure the temp directory exists
	tempDir := "./path/to/temp" // Change this to the actual path of your temp directory
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		// Create the temp directory if it doesn't exist
		if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
			results <- mediaResult{Error: fmt.Errorf("failed to create temp directory: %v", err)}
			return
		}
	}

	log.Printf("Processing file: %s, size: %d", fileHeader.Filename, fileHeader.Size)

	// Save the file to a temporary location
	tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
	out, err := os.Create(tempFilePath)
	if err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to create temp file: %v", err)}
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		results <- mediaResult{Error: fmt.Errorf("failed to copy file to temp location: %v", err)}
		return
	}

	// Simulate processing and generating URLs
	feedURL := "./media/feed/" + fileHeader.Filename
	thumbnailURL := "./media/thumbnail/" + fileHeader.Filename
	fullSizeURL := "./media/fullsize/" + fileHeader.Filename

	fileType := fileHeader.Header.Get("Content-Type")

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
func generateIDx() string {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	return id.String()
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

	// Decode the JSON response
	var geocodingResponse GeocodingResponse
	if err := json.NewDecoder(response.Body).Decode(&geocodingResponse); err != nil {
		return nil, nil, nil, "", "", fmt.Errorf("error decoding JSON response: %v", err)
	}

	// Extract the locality (LGA) and administrative area (State) from the response
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

	// Print or use the locality and state information
	fmt.Println("LGA:", locality)
	fmt.Println("State:", state)

	lga := &models.LGA{
		ID:       generateIDx(),
		Name:     locality,
		ReportTypeID: reportID,
	}

	stateStruct := &models.State{
		ID:       generateIDx(),
		Name:     state,
		ReportID: reportID,
	}

	userI, exists := c.Get("user")
	if !exists {
		log.Println("User not found in context")
		return nil, nil, nil, "", "", fmt.Errorf("user not found in context")
	}
	userId := userI.(*models.User).ID

	reportType := &models.ReportType{
		ID:        generateIDx(),
		UserID:    userId,
		ReportID:  reportID,
		StateName: state,
		LGAName:   locality,
		Category:      c.PostForm("category"),
	}

	return lga, stateStruct, reportType, locality, state, nil
}

// Handle the upload of media file
func (s *Server) handleIncidentReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		userI, exists := c.Get("user")
		if !exists {
			log.Println("User not found in context")
			response.JSON(c, "", http.StatusUnauthorized, nil, errors.ErrNotFound)
			return
		}
		userId := userI.(*models.User).ID

		const MaxFileSize = 32 << 20 // 32 MB

		// Read the request body into a buffer
		buf := new(bytes.Buffer)
		_, err := io.CopyN(buf, c.Request.Body, MaxFileSize)
		if err != nil && err != io.EOF {
			log.Printf("Error reading request body: %v\n", err)
			response.JSON(c, "unable to read media", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}

		c.Request.Body = io.NopCloser(buf)

		log.Println("About to parse multipart form")
		if err := c.Request.ParseMultipartForm(MaxFileSize); err != nil {
			log.Printf("Error parsing multipart form: %v\n", err)
			log.Printf("Request headers: %v\n", c.Request.Header)
			log.Printf("Request content length: %v\n", c.Request.ContentLength)
			response.JSON(c, "unable to parse media", http.StatusInternalServerError, nil, errors.ErrInternalServerError)
			return
		}
		log.Println("Parsed multipart form successfully")

		formMedia := c.Request.MultipartForm.File["mediaFiles"]
		log.Printf("Number of files received: %d\n", len(formMedia))

		reportID, err := generateID()
		if err != nil {
			log.Printf("Error generating ID: %v\n", err)
			return
		}

		results := make(chan mediaResult)
		mediaTypeCounts := make(map[string]int)

		var wg sync.WaitGroup
		mu := &sync.Mutex{}
		for _, fileHeader := range formMedia {
			wg.Add(1)
			go func(fileHeader *multipart.FileHeader) {
				defer wg.Done()
				s.saveChunk(fileHeader, results)

				supported, fileType := CheckSupportedFile(fileHeader.Filename)
				if !supported {
					log.Printf("Unsupported file type: %s\n", fileHeader.Filename)
					return
				}

				mu.Lock()
				mediaTypeCounts[fileType]++
				log.Printf("File type: %s, Count: %d\n", fileType, mediaTypeCounts[fileType])
				mu.Unlock()
			}(fileHeader)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		var feedURLs, thumbnailURLs, fullsizeURLs, fileTypes []string

		for result := range results {
			if result.Error != nil {
				log.Printf("Error processing media: %v\n", result.Error)
				response.JSON(c, "Unable to process media", http.StatusInternalServerError, nil, result.Error)
				return
			}
			feedURLs = append(feedURLs, result.FeedURL)
			thumbnailURLs = append(thumbnailURLs, result.ThumbnailURL)
			fullsizeURLs = append(fullsizeURLs, result.FullSizeURL)
			fileTypes = append(fileTypes, result.FileType)
		}

		if len(feedURLs) > 0 {
			processedFeedURLs, processedThumbnailURLs, processedFullsizeURLs, processedFileTypes, err := s.MediaService.ProcessMedia(c, formMedia, userId, reportID)
			if err != nil {
				log.Printf("Error processing media: %v\n", err)
				response.JSON(c, "Unable to process media", http.StatusInternalServerError, nil, err)
				return
			}
			feedURLs = append(feedURLs, processedFeedURLs...)
			thumbnailURLs = append(thumbnailURLs, processedThumbnailURLs...)
			fullsizeURLs = append(fullsizeURLs, processedFullsizeURLs...)
			fileTypes = append(fileTypes, processedFileTypes...)
		}
		wg.Wait()

		for fileType, count := range mediaTypeCounts {
			log.Printf("File type: %s, Count: %d\n", fileType, count)
		}

		imageCount, videoCount, audioCount := CreateMediaCount(mediaTypeCounts)
		totalPoints := calculateMediaPoints(mediaTypeCounts)
		log.Println("Image count:", imageCount)
		log.Println("Total points:", totalPoints)

		var feedURL, thumbnailURL, fullsizeURL, fileType string
		if len(feedURLs) > 0 {
			feedURL = strings.Join(feedURLs, ",")
			thumbnailURL = strings.Join(thumbnailURLs, ",")
			fullsizeURL = strings.Join(fullsizeURLs, ",")
			fileType = fileTypes[0]
		} else {
			feedURL, thumbnailURL, fullsizeURL, fileType = "", "", "", "unknown"
		}

		media := models.Media{
			UserID:       userId,
			FeedURL:      feedURL,
			ThumbnailURL: thumbnailURL,
			FullSizeURL:  fullsizeURL,
			FileType:     fileType,
		}

		if err := s.MediaService.SaveMedia(media, reportID, userId, imageCount, videoCount, audioCount, totalPoints); err != nil {
			log.Printf("Error saving media: %v\n", err)
			response.JSON(c, "Unable to save media", http.StatusInternalServerError, nil, err)
			return
		}

		lat, err := strconv.ParseFloat(strings.TrimSpace(c.PostForm("latitude")), 64)
		if err != nil {
			response.JSON(c, "Invalid latitude", http.StatusBadRequest, nil, err)
			return
		}

		lng, err := strconv.ParseFloat(strings.TrimSpace(c.PostForm("longitude")), 64)
		if err != nil {
			response.JSON(c, "Invalid longitude", http.StatusBadRequest, nil, err)
			return
		}

		// Parse date_of_incidence into time.Time
		dateStr := c.PostForm("date_of_incidence")
		dateOfIncidence, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date_of_incidence format"})
			return
		}

		lga, stateStruct, category, lgastring, statestring, err := fetchGeocodingData(lat, lng, c, reportID)
		if err != nil {
			log.Printf("Error fetching geocoding data: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		incidentReport := &models.IncidentReport{
			ID:                 reportID,
			CreatedAt:          0,
			DateOfIncidence:    dateOfIncidence,
			Description:        c.PostForm("description"),
			FeedURLs:           feedURL,
			FullSizeURLs:       fullsizeURL,
			ProductName:        c.PostForm("product_name"),
			StateName:          c.PostForm("state_name"),
			LGAName:            c.PostForm("lga_name"),
			Latitude:           lat,
			Longitude:          lng,
			UserIsAnonymous:    false,
			Address:            c.PostForm("address"),
			UserUsername:       c.PostForm("user_name"),
			View:               0,
			IsVerified:         false,
			UserID:             userId,
			AdminID:            0,
			Landmark:           c.PostForm("landmark"),
			LikeCount:          0,
			BookmarkedReports:  []*models.User{},
			IsResponse:         false,
			TimeofIncidence:    time.Now(),
			ReportStatus:       "Pending",
			RewardPoint:        0,
			ActionTypeName:     "",
			ReportTypeName:     c.PostForm("report_type"),
			IsState:            false,
			Rating:             c.PostForm("rating"),
			HospitalName:       c.PostForm("hospital_name"),
			Department:         c.PostForm("department"),
			DepartmentHeadName: c.PostForm("department_head_name"),
			AccidentCause:      c.PostForm("accident_cause"),
			SchoolName:         c.PostForm("school_name"),
			VicePrincipal:      c.PostForm("vice_principal"),
			OutageLength:       c.PostForm("outage_length"),
			NoWater:            true,
		}

		sub := &models.SubReport{
			ID:   reportID,
			Name: c.PostForm("sub_report_type"),
		}

		// Check if stateName and lgaName are empty, replace with statestring and lgastring if so
		if c.PostForm("state_name") == "" && c.PostForm("lga_name") == "" {
			incidentReport.StateName = statestring
			incidentReport.LGAName = lgastring
		}

		// bad naming
		err = s.IncidentReportRepository.SaveStateLgaTime(lga, stateStruct, category, sub)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Unable to save report",
				"error":   err.Error(),
			})
			return
		}

		savedIncidentReport, err := s.IncidentReportService.SaveReport(userId, lat, lng, incidentReport, reportID, totalPoints)
		if err != nil {
			log.Printf("Error saving incident report: %v\n", err)
			response.JSON(c, "Unable to save incident report", http.StatusInternalServerError, nil, err)
			return
		}

		response.JSON(c, "Incident Report Submitted Successfully", http.StatusCreated, savedIncidentReport, nil)
	}
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
	const idLength = 14
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	id := make([]byte, idLength)
	for i := range id {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		id[i] = charset[num.Int64()]
	}
	return string(id), nil
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
		log.Printf("State: %s, LGA: %s, Start Date: %s, End Date: %s\n", state, lga, startDate, endDate)

		if state == "" || lga == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "State and LGA are required"})
			return
		}

		reportTypes, reportCounts, err := s.IncidentReportService.GetReportTypeCounts(state, lga, &startDate, &endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"report_types":  reportTypes,
			"report_counts": reportCounts,
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
		stateReportCounts, err := s.IncidentReportRepository.GetVariadicStateReportCounts(criteria.ReportType, criteria.States...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Respond with report counts (assuming suitable format for bar chart)
		c.JSON(http.StatusOK, stateReportCounts)
	}
}
