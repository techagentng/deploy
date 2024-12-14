package db

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/techagentng/citizenx/config"
	"github.com/techagentng/citizenx/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormDB struct {
	DB *gorm.DB
}

func GetDB(c *config.Config) *GormDB {
	gormDB := &GormDB{}
	gormDB.Init(c)
	return gormDB
}

func (g *GormDB) Init(c *config.Config) {
	g.DB = getPostgresDB(c)

	if err := migrate(g.DB); err != nil {
		log.Fatalf("unable to run migrations: %v", err)
	}
}

func getPostgresDB(c *config.Config) *gorm.DB {
	log.Printf("Connecting to postgres: %+v", c)
	postgresDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d TimeZone=Africa/Lagos",
		c.PostgresHost, c.PostgresUser, c.PostgresPassword, c.PostgresDB, c.PostgresPort)

	// Create GORM DB instance
	gormConfig := &gorm.Config{}
	if c.Env != "prod" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN: postgresDSN,
	}), gormConfig)
	if err != nil {
		log.Fatal(err)
	}

	return gormDB
}

func SeedRoles(db *gorm.DB) error {
	roles := []models.Role{
		{ID: uuid.New(), Name: "Admin"},
		{ID: uuid.New(), Name: "User"},
	}

	for _, role := range roles {
		if err := db.FirstOrCreate(&role, models.Role{Name: role.Name}).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrate(db *gorm.DB) error {
	// AutoMigrate all the models
	err := db.AutoMigrate(
		&models.User{},
		&models.Blacklist{},
		&models.IncidentReport{},
		&models.Media{},
		&models.Reward{},
		&models.Like{},
		&models.Notification{},
		&models.Comment{},
		&models.ReportType{},
		&models.IncidentReportUser{},
		&models.LGA{},
		&models.State{},
		&models.Bookmark{},
		&models.StateReportPercentage{},
		&models.MediaCount{},
		&models.LoginRequestMacAddress{},
		&models.UserImage{},
		&models.ReportCount{},
		&models.SubReport{},
		&models.Votes{},
		&models.UserPoints{},
		&models.Role{},
		&models.Post{},
		&models.ReportPostRequest{},
		&models.ReportUserRequest{},
		&models.Follow{},
		&models.OAuthState{},
		&models.Conversation{},
		&models.Message{},
	)
	
	if err != nil {
		return fmt.Errorf("migrations error: %v", err)
	}

	// Seed roles
	// if err := seedRoles(db); err != nil {
	//     return fmt.Errorf("seeding error: %v", err)
	// }

	// Add any additional migrations or seeds here if needed

	return nil
}
