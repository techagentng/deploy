package server

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/models"
	"html/template"
	jwtPackage "github.com/techagentng/citizenx/services/jwt"
)

func (s *Server) handleCreatePost() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle file upload
		file, fileHeader, err := c.Request.FormFile("postImage")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid file"})
			return
		}

		// Validate file type and size (same as for profile images)
		if err := validateFile(fileHeader); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

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

		var userID uint
		switch userIDValue := accessClaims["id"].(type) {
		case float64:
			userID = uint(userIDValue)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID format"})
			return
		}

		// Validate form fields
		title := c.PostForm("title")
		postCategory := c.PostForm("post_category")
		postDescription := c.PostForm("post_description")

		if title == "" || postCategory == "" || postDescription == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title, category, and description are required"})
			return
		}

		// Create S3 client
		s3Client, err := createS3Client()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create S3 client"})
			return
		}

		userIDString := strconv.FormatUint(uint64(userID), 10)

		// Generate unique filename
		filename := userIDString + "_" + fileHeader.Filename

		// Upload file to S3
		filepath, err := uploadFileToS3(s3Client, file, os.Getenv("AWS_BUCKET"), filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to S3"})
			return
		}

		// Create a new post
		post := models.Post{
			UserID:          userID,
			Title:           title,
			PostCategory:    postCategory,
			Image:           filepath,
			PostDescription: postDescription,
		}

		// Save the post to the database
		if err := s.PostRepository.CreatePost(&post); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Post created successfully",
			"post":    post,
		})
	}
}

func (s *Server) handleGetPostsByUserID() gin.HandlerFunc {
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

		var userID uint
		switch userIDValue := accessClaims["id"].(type) {
		case float64:
			userID = uint(userIDValue)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userID format"})
			return
		}

		// Fetch all posts by the user from the database
		posts, err := s.PostRepository.GetPostsByUserID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve posts"})
			return
		}

		// Check if the user has no posts
		if len(posts) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No posts found for this user"})
			return
		}

		// Respond with the list of posts
		c.JSON(http.StatusOK, gin.H{
			"message": "Posts retrieved successfully",
			"posts":   posts,
		})
	}
}

// server.go or where you define your handlers
func (s *Server) HandleGetAllPosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call the repository method to retrieve all posts
		posts, err := s.PostRepository.GetAllPosts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve posts"})
			return
		}

		// Return the posts as a JSON response
		c.JSON(http.StatusOK, gin.H{"posts": posts})
	}
}

func (s *Server) GetPostByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		postID := c.Param("id")

		post, err := s.PostRepository.GetPostByID(postID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
			return
		}

		c.JSON(http.StatusOK, post)
	}
}

func (s *Server) GetAppPostByID() gin.HandlerFunc{
	return func(c *gin.Context) {
    postID := c.Param("id")

    report, err := s.PostRepository.GetPublicReportByID(postID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "id":          report.ID,
        "title":       "CitizenX Nigeria",
        "description": report.Description,
        "report_type": report.ReportType.Category,
        "created_at":  report.CreatedAt,
    })
}
}

func (s *Server) GetPostPreviewByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Get post from DB using repository
		post, err := s.PostRepository.GetPublicReportByID(id)
		if err != nil {
			c.String(http.StatusNotFound, "Post not found")
			return
		}

		// Load template
		tmpl, err := template.ParseFiles("templates/preview_post.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Template error")
			return
		}

		// Fill template data
		data := map[string]interface{}{
			"ID":          post.ID,
			"Title":       "CitizenX Nigeria",
			"ReportType":  post.ReportType.Category,
			"Description": post.Description,
			"ImageURL":    post.FeedURLs,
		}

		// Render template
		c.Writer.Header().Set("Content-Type", "text/html")
		tmpl.Execute(c.Writer, data)
	}
}
