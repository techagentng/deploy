package main

import (
	"context"
	"log"
	_ "net/url"
	"os"

	"firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/db"
	"github.com/techagentng/citizenx/mailingservices"
	"github.com/techagentng/citizenx/server"
	"github.com/techagentng/citizenx/services"
	"google.golang.org/api/option"
)

var firebaseApp *firebase.App
var messagingClient *messaging.Client

func InitFirebase() {
	// Load Firebase credentials JSON file
	opt := option.WithCredentialsFile("./google-services.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing Firebase app: %v", err)
	}
	firebaseApp = app
	log.Println("Firebase initialized")

	// Initialize Messaging client
	client, err := firebaseApp.Messaging(context.Background())
	if err != nil {
		log.Fatalf("error getting Messaging client: %v", err)
	}
	messagingClient = client
	log.Println("Firebase Messaging client initialized")
}


func main() {
	os.Setenv("GOOGLE_CLOUD_PROJECT", "achat-f2008")
	InitFirebase()
	// os.Setenv("GOOGLE_CLOUD_PROJECT", "achat-f2008")
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
	postRepo := db.NewPostRepo(gormDB)

	authService := services.NewAuthService(authRepo, conf)
	mediaService := services.NewMediaService(mediaRepo, rewardRepo, incidentReportRepo, conf)
	incidentReportService := services.NewIncidentReportService(incidentReportRepo, rewardRepo, mediaRepo, conf)
	rewardService := services.NewRewardService(rewardRepo, incidentReportRepo, conf)
	likeService := services.NewLikeService(likeRepo, conf)
	postService := services.NewPostService(postRepo, conf)

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
	}

	// r := gin.Default()
	// r.Use(cors.Default())
	// r.ForwardedByClientIP = true
	// r.SetTrustedProxies([]string{"127.0.0.1"})

	// r.Run(":8080")
	s.Start()
}
