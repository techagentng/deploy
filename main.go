package main

import (
	// "context"
	"log"

	// "firebase.google.com/go"
	// "google.golang.org/api/option"

	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/mailingservices"
	"github.com/techagentng/citizenx/server"
	"github.com/techagentng/citizenx/services"
	 "github.com/go-redis/redis/v8"
)

func main() {
	// Load configuration
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

    redisClient := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Adjust to your Redis server address
        Password: "",               // No password by default
        DB:       0,                // Default DB
    })

	// Initialize Mailgun client
	mailgunClient := &mailingservices.Mailgun{}
	mailgunClient.Init()

	// Initialize database
	gormDB := db.GetDB(conf)

	// Seed roles
	if err := db.SeedRoles(gormDB.DB); err != nil {
		log.Fatalf("error seeding roles: %v", err)
	}

	// Repositories
	authRepo := db.NewAuthRepo(gormDB)
	mediaRepo := db.NewMediaRepo(gormDB)
	incidentReportRepo := db.NewIncidentReportRepo(gormDB)
	rewardRepo := db.NewRewardRepo(gormDB)
	likeRepo := db.NewLikeRepo(gormDB)
	postRepo := db.NewPostRepo(gormDB)

	// Services
	authService := services.NewAuthService(authRepo, conf)
	mediaService := services.NewMediaService(mediaRepo, rewardRepo, incidentReportRepo, conf)
	incidentReportService := services.NewIncidentReportService(incidentReportRepo, rewardRepo, mediaRepo, conf)
	rewardService := services.NewRewardService(rewardRepo, incidentReportRepo, conf)
	likeService := services.NewLikeService(likeRepo, conf)
	postService := services.NewPostService(postRepo, conf)

	// Server setup
	s := &server.Server{
		Mail:                     mailgunClient,
		Config:                   conf,
		AuthRepository:           authRepo,
		AuthService:              authService,
		MediaRepository:          mediaRepo,
		MediaService:             mediaService,
		IncidentReportService:    incidentReportService,
		IncidentReportRepository: incidentReportRepo,
		RewardService:            rewardService,
		RewardRepository:         rewardRepo,
		LikeService:              likeService,
		PostService:              postService,
		PostRepository:           postRepo,
		DB:                       db.GormDB{},
		RedisClient:              redisClient,
	}

	// Start server
	s.Start()
}
