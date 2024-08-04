package server

import (
	"fmt"

	// rateLimit "github.com/JGLTechnologies/gin-rate-limit"
	// "net/http"
	"os"
	// "path/filepath"
	// "runtime"
	"time"

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
	// r.Static("/static", "./build/static")

	// staticFiles := "server/templates/static"
	// htmlFiles := "server/templates/*.html"
	// if s.Config.Env == "test" {
	// 	_, b, _, _ := runtime.Caller(0)
	// 	basepath := filepath.Dir(b)
	// 	staticFiles = basepath + "/templates/static"
	// 	htmlFiles = basepath + "/templates/*.html"
	// }
	// r.StaticFS("static", http.Dir(staticFiles))
	// r.LoadHTMLGlob(htmlFiles)

	// LoggerWithFormatter middleware will write the logs to gin.DefaultWriter
	// By default gin.DefaultWriter = os.Stdout
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
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

	// allowedOrigins := []string{"http://localhost:3001"}
	// if os.Getenv("GIN_MODE") == "release" {
	// 	allowedOrigins = []string{"https://citizenx-dashboard-sbqx.onrender.com"} 
	// }
	// Use CORS middleware with appropriate configuration
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.MaxMultipartMemory = 32 << 20
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
	apirouter.GET("/google/login", s.HandleGoogleLogin())
	apirouter.GET("auth/google/callback", s.HandleGoogleCallback())
	apirouter.GET("/incident_reports/state/:state", s.handleGetAllReportsByState())
	apirouter.GET("/incident_reports/lga/:lga", s.handleGetAllReportsByLGA())
	apirouter.GET("/incident_reports/report_type/:report_type", s.handleGetAllReportsByReportType())
	// apirouter.GET("/verifyEmail/:token", s.HandleVerifyEmail())
	apirouter.POST("/password/forgot", s.HandleForgotPassword())
	apirouter.POST("/password/reset/:token", s.ResetPassword())
	apirouter.POST("/report-type/states", s.HandleGetVariadicBarChart())

	authorized := apirouter.Group("/")
	authorized.Use(s.Authorize())
	// Upload endpoint
	authorized.GET("/logout", s.handleLogout())
	authorized.GET("/users/online", s.handleGetOnlineUsers())
	authorized.POST("/user/report/", s.handleIncidentReport())
	authorized.GET("/categories", s.handleGetAllCategories())
	authorized.GET("/states", s.handleGetAllStates())
	authorized.PUT("/me/updateUserProfile", s.handleEditUserProfile())
	authorized.GET("/me", s.handleShowProfile())
	authorized.PUT("/user/:reportID/like", s.handleLikeReport())
	authorized.GET("/user/:reportID/bookmark", s.SaveBookmarkReport())
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
	apirouter.GET("/lgas", s.handleGetLGAs())
	apirouter.GET("/lgas/lat/lng", s.IncidentMarkersHandler())
	apirouter.DELETE("/incident-report/:id", s.DeleteIncidentReportHandler())
	apirouter.GET("/incident-report/state/count", s.HandleGetStateReportCounts())
	apirouter.PUT("/upload", s.handleUpdateUserImageUrl())
	apirouter.GET("/report/rating", s.handleGetRatingPercentages())
	apirouter.GET("/report/lga/count", s.handleGetAllReportsByState())
	apirouter.GET("/state/report/count", s.handleListAllStatesWithReportCounts())
	apirouter.GET("/report/total/count", s.handleGetTotalReportCount())
	// authorized.DELETE("/delete/:folder/:fileName", s.handleDeleteDocument())
	// authorized.POST("/user/medications", s.handleCreateMedication())
	// authorized.GET("/user/medications/:id", s.handleGetMedDetail())
	// apirouter.GET("/documents", s.handleGetAllDocuments())
	// authorized.PUT("/user/medications/:medicationID", s.handleUpdateMedication())
	// authorized.GET("/user/medications/next", s.handleGetNextMedication())
	// authorized.GET("/user/medications/search", s.handleFindMedication())

	// authorized.PUT("/user/medication-history/:id", s.handleUpdateMedicationHistory())
	// authorized.GET("/user/medication-history", s.handleGetAllMedicationHistoryByUser())
	// authorized.POST("/notifications/add-token", s.authorizeNotificationsForDevice())

}
