package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/techagentng/citizenx/models"
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
        
        // Add proper headers
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
        c.Writer.Header().Set("Cache-Control", "public, max-age=3600")
        c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
        
        
        post, err := s.PostRepository.GetPublicReportByID(id)
        if err != nil {
            c.String(http.StatusNotFound, "Post not found")
            return
        }

        // Format image URL properly
        var imageURL string
        if post.FeedURLs != "" {
            // If FeedURLs is a JSON array, parse it
            var urls []string
            if err := json.Unmarshal([]byte(post.FeedURLs), &urls); err == nil && len(urls) > 0 {
                imageURL = urls[0]
            } else {
                // If it's a single URL
                imageURL = post.FeedURLs
            }
            
            // Ensure URL is absolute
            if !strings.HasPrefix(imageURL, "http") {
                imageURL = "https://citizenx.ng" + imageURL
            }
        }

        // Add proper meta tags for social sharing
        data := map[string]interface{}{
            "ID":          post.ID,
            "Title":       "CitizenX Nigeria",
            "Description": post.Description,
            "ImageURL":    imageURL,
            "URL":         fmt.Sprintf("https://www.citizenx.ng/preview/post/%s", post.ID),
            "Type":        "article",
            "SiteName":    "CitizenX",
        }

        tmpl, err := template.ParseFiles("templates/preview_post.html")
        if err != nil {
            c.String(http.StatusInternalServerError, "Template error")
            return
        }

        c.Writer.Header().Set("Content-Type", "text/html")
        tmpl.Execute(c.Writer, data)
    }
}