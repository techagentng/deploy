package main

import (
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/mailingservices"
	"github.com/techagentng/citizenx/server"
	"github.com/techagentng/citizenx/services"
	"log"
	_ "net/url"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Mailgun client
	mailgunClient := &mailingservices.Mailgun{}
	mailgunClient.Init()

	gormDB := db.GetDB(conf)
	// Seed roles
	if err := db.SeedRoles(gormDB.DB); err != nil {
		log.Fatalf("error seeding roles: %v", err)
	}
	authRepo := db.NewAuthRepo(gormDB)
	mediaRepo := db.NewMediaRepo(gormDB)
	incidentReportRepo := db.NewIncidentReportRepo(gormDB)
	rewardRepo := db.NewRewardRepo(gormDB)
	likeRepo := db.NewLikeRepo(gormDB)

	authService := services.NewAuthService(authRepo, conf)
	mediaService := services.NewMediaService(mediaRepo, rewardRepo, incidentReportRepo, conf)
	incidentReportService := services.NewIncidentReportService(incidentReportRepo, rewardRepo, mediaRepo, conf)
	rewardService := services.NewRewardService(rewardRepo, incidentReportRepo, conf)
	likeService := services.NewLikeService(likeRepo, conf)

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
		LikeService:              likeService,
		DB:                       db.GormDB{},
	}

	// r := gin.Default()
	// r.Use(cors.Default())
	// r.ForwardedByClientIP = true
	// r.SetTrustedProxies([]string{"127.0.0.1"})

	// r.Run(":8080")
	s.Start()
}
