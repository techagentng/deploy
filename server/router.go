package server

import (
	"fmt"

	// rateLimit "github.com/JGLTechnologies/gin-rate-limit"
	// "net/http"
	"os"
	// "path/filepath"
	// "runtime"
	"time"
	// "github.com/gin-contrib/cors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) setupRouter() *gin.Engine {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "test" {
		r := gin.New()
		s.defineRoutes(r)
		return r
	}

	r := gin.New()

	// Logger middleware with custom format
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"https://www.citizenx.ng","http://localhost:3001","https://citizenx-dashboard-sbqx.onrender.com"}, // Replace with your frontend's origin
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Authorization", "Content-Type", "X-Client-State"},
		ExposeHeaders: []string{"Content-Length", "X-Client-State"},
		AllowCredentials: true,
		MaxAge: 12 * time.Hour,
	}))
	
	// Increase memory limit for multipart forms
	r.MaxMultipartMemory = 32 << 20
	// Define application routes
	s.defineRoutes(r)

	return r
}

func (s *Server) defineRoutes(router *gin.Engine) {
	// store := rateLimit.InMemoryStore(&rateLimit.InMemoryOptions{})
	// limitRate := limitRateForPasswordReset(store)

	apirouter := router.Group("/api/v1")
	apirouter.POST("/auth/signup", s.handleSignup())
	apirouter.POST("/auth/login", s.handleLogin())
	apirouter.POST("/no-cred/login", restrictAccessToProtectedRoutes(), s.handleNonCredentialLogin())
	apirouter.GET("/fb/auth", s.handleFBLogin())
	apirouter.GET("fb/callback", s.handleFBCallback())
	apirouter.GET("/incident_reports", s.handleGetAllReport()) 
	apirouter.POST("/google/user/login", s.handleGoogleUserLogin())
	apirouter.POST("/facebook/user/login", s.handleFacebookUserLogin())
	apirouter.GET("/google/login", s.HandleGoogleLogin())
	apirouter.GET("/auth/google/callback", s.HandleGoogleCallback()) //
	apirouter.GET("/incident_reports/state/:state", s.handleGetAllReportsByState())
	apirouter.GET("/incident_reports/lga/:lga", s.handleGetAllReportsByLGA())
	apirouter.GET("/incident_reports/report_type/:report_type", s.handleGetAllReportsByReportType())
	apirouter.POST("/password/forgot", s.HandleForgotPassword())
	apirouter.POST("/password/reset/:token", s.ResetPasswordHandler()) //
	apirouter.POST("/report-type/states", s.HandleGetVariadicBarChart())
	apirouter.GET("/all/publications", s.HandleGetAllPosts())
	apirouter.GET("/publication/:id", s.GetPostByID())
	apirouter.PUT("/incident-report/block-request/:post_id", s.UpdateBlockRequestHandler())
	

	authorized := apirouter.Group("/")
	authorized.Use(s.Authorize())
	// Upload endpoint
	authorized.GET("/logout", s.handleLogout())
	authorized.GET("/users/online", s.handleGetOnlineUsers())
	authorized.POST("/user/report/", s.handleIncidentReport())  //
	authorized.POST("/user/report/media", s.handleUploadMedia())
	authorized.GET("/categories", s.handleGetAllCategories())
	authorized.GET("/states", s.handleGetAllStates())
	authorized.PUT("/me/updateUserProfile", s.handleEditUserProfile())
	authorized.GET("/me", s.handleShowProfile())
	authorized.GET("/user/bookmark/:reportID", s.HandleBookmarkReport())
	authorized.GET("/user/bookmarked/report", s.HandleGetBookmarkedReports()) //
	authorized.GET("/approve/:reportID/:userID/report", s.handleApproveReportPoints())
	authorized.GET("/reject/:reportID/:userID/report", s.handleRejectReportPoints())
	authorized.GET("/accept/:reportID/:userID/report", s.handleAcceptReportPoints())
	authorized.GET("/report-percentage-by-state", s.handleGetReportPercentageByState())
	authorized.GET("/today/report", s.handleGetTodayReportCount())
	authorized.GET("/all/user", s.handleGetTotalUserCount())
	authorized.GET("/users/lga/:lga/count", s.GetRegisteredUsersCountByLGA())
	authorized.GET("/reports/state/:state", s.handleGetAllReportsByStateByTime())
	authorized.GET("/user/is_online", s.handleGetUserActivity())
	authorized.GET("/users/all", s.handleGetAllUsers())
	authorized.GET("/count/all/rewards", s.handleSumAllRewardsBalance())
	authorized.GET("/users/lga/:lga/report-type/:reportType", s.handleGetReportsByTypeAndLGA())
	authorized.GET("/rewards/list", s.handleGetAllRewardsList())
	authorized.GET("/report/type/count", s.handleGetReportTypeCounts())
	authorized.GET("/lgas", s.handleGetLGAs())
	authorized.GET("/lgas/lat/lng", s.IncidentMarkersHandler())
	authorized.DELETE("/incident-report/:id", s.DeleteIncidentReportHandler())
	authorized.GET("/incident-report/state/count", s.HandleGetStateReportCounts())
	authorized.PUT("/upload", s.handleUpdateUserImageUrl())
	authorized.GET("/report/rating", s.handleGetRatingPercentages())
	authorized.GET("/report/lga/count", s.handleGetAllReportsByState())
	authorized.GET("/state/report/count", s.handleListAllStatesWithReportCounts())
	authorized.GET("/report/total/count", s.handleGetTotalReportCount())
	authorized.GET("/report/category/sub", s.handleGetNamesByCategory())
	authorized.GET("/report/sub_reports", s.HandleGetSubReportsByCategory())
	authorized.PUT("/report/upvote/:reportID", s.HandleUpvoteReport())
	authorized.PUT("/report/downvote/:reportID", s.HandleDownvoteReport())
	authorized.GET("/user/reports", s.HandleGetAllReportsByUser())  //
	authorized.GET("/report/votecounts/:reportID", s.HandleGetVoteCounts())
	authorized.GET("/report/counts/lga/:lga", s.GetReportTypeCountsByLGA())
	authorized.GET("/report/counts/state/:state", s.GetReportCountsByStateAndLGA())
	authorized.DELETE("/delete/user", s.handleDeleteUser())
	authorized.GET("/top/report/categories", s.handleGetTopCategories())
	authorized.GET("/report/type/id", s.GetReportsByCategory())
	authorized.GET("/get/user/balance", s.handleGetUserRewardBalance())
	authorized.GET("reports/filters", s.handleGetReportsByFilters())
	authorized.POST("posts/create", s.handleCreatePost())
	authorized.GET("/all/posts/:userID", s.handleGetPostsByUserID())
	authorized.PUT("/users/report/:userID", s.ReportUserHandler())
	authorized.PUT("/users/block/:userID", s.BlockUserHandler())
	authorized.PUT("/users/:user_id/role", s.handleChangeUserRole())
	apirouter.GET("/auth/google/state", s.GenerateGoogleState())
	authorized.POST("/reports/follow/:report_id", s.HandleFollowReport())
	authorized.GET("/reports/followers/:report_id", s.HandleGetFollowersByReport())
	apirouter.GET("/incident_reports/lga/:lga/count", s.handleGetReportCountByLGA())
	apirouter.GET("/incident_reports/state/:state/count", s.handleGetReportCountByState())
	apirouter.GET("/incident_reports/count", s.handleGetOverallReportCount())
	apirouter.GET("/state/governor", s.handleGetGovernorDetails())
	apirouter.POST("/create/governor", s.CreateState())
	apirouter.GET("/reports/count", s.handleGetReportCountByLGA())
	apirouter.GET("/api/users/total", s.GetTotalUserCount)
	apirouter.GET("/lgas/:state", s.FetchLGAsByState())
	apirouter.GET("/map/state/count", s.handleGetReportTypeCountsState())
}
