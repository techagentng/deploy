package server

import (
	"context"
	"fmt"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/mailingservices"
	"github.com/techagentng/citizenx/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/go-redis/redis/v8"
)

type Server struct {
	Config                   *config.Config
	AuthRepository           db.AuthRepository
	AuthService              services.AuthService
	Mail                     mailingservices.Mailer
	MediaRepository          db.MediaRepository
	MediaService             services.MediaService
	IncidentReportService    services.IncidentReportService
	IncidentReportRepository db.IncidentReportRepository
	RewardService            services.RewardService
	RewardRepository         db.RewardRepository
	LikeService              services.LikeService
	PostService              services.PostService
	PostRepository           db.PostRepository
	DB                       db.GormDB
	SessionSecret            string
	RedisClient              *redis.Client
}

// Server serves requests to DB with rout
func (s *Server) Start() {
	r := s.setupRouter()
	// TODO: user config.PORT here
	PORT := fmt.Sprintf(":%s", os.Getenv("PORT"))
	if PORT == ":" {
		PORT = ":8080"
	}
	srv := &http.Server{
		Addr:    PORT,
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("Server started on %s\n", PORT)
	gracefulShutdown(srv)
}

func gracefulShutdown(srv *http.Server) {
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}
