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

func SeedStates(db *gorm.DB) error {
	states := []models.State{
		{
			ID:            uuid.New(),
			State:         "Lagos",
			Governor:      "Babajide Sanwo-Olu",
			LGAC:          "Ikeja",
			GovernorImage: "https://drive.google.com/file/d/15bz1tOUIZ9JjRiJ8ivWpgoZvA1fMF_tC/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/19EDqjGe8Mk8oyZ10VN5nU1h5LsxBVHtB/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Kano",
			Governor:      "Abba Kabir Yusuf",
			LGAC:          "Kano Municipal",
			GovernorImage: "https://drive.google.com/file/d/1JPrQT5de8GwWV1U8OVI73Ugo449ekE5S/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1vn8TYw_KQIFx5J8joDZ9NhzUMEjSKn0f/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Rivers",
			Governor:      "Siminalayi Fubara",
			LGAC:          "Port Harcourt",
			GovernorImage: "https://drive.google.com/file/d/1I8IZLVCfu5ZX7rdQakeDN27XNA03Bkjb/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/12Jakf6t7AwrBp4q2HZoHGT_xZh_PmhJE/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Kaduna",
			Governor:      "Uba Sani",
			LGAC:          "Kaduna North",
			GovernorImage: "https://drive.google.com/file/d/1QIoQA7nhzn8eaB2hff_sIP6_ItyHMS0o/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1CJ98lHl9tywa6Se6ugBFiQeNlyP07FA-/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Ogun",
			Governor:      "Dapo Abiodun",
			LGAC:          "Abeokuta South",
			GovernorImage: "https://drive.google.com/file/d/183IjH5P6vhJEsOk6_hnQnMUVYf-CWcBW/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1Rs4HDH2ALL4isMtjywZRO2tKDqUk5ISS/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Anambra",
			Governor:      "Chukwuma Soludo",
			LGAC:          "Awka",
			GovernorImage: "https://drive.google.com/file/d/1-R7SbgT34m0KIqwKf-C-zaPK-4bmOXN8/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1pS6WFSqhxwBNpvrJ44W9ijIVe6xnU1eo/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Enugu",
			Governor:      "Peter Mbah",
			LGAC:          "Enugu",
			GovernorImage: "https://drive.google.com/file/d/1GSg8XEMyC79Lb4XMNJEaM-qZxn5LZGxm/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1G19Cy1gsrKKbQ_V7UQnl8HqPgjmBljjP/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Oyo",
			Governor:      "Seyi Makinde",
			LGAC:          "Ibadan North",
			GovernorImage: "https://drive.google.com/file/d/1bmlygAeuKSkq9aKBoQDfm3e1VSpv9QyL/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1QoPlQW88KIzRyoMVBIi4SV4WQgsjAHPv/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Edo",
			Governor:      "Godwin Obaseki",
			LGAC:          "Benin City",
			GovernorImage: "https://example.com/obaseki.jpg",
			LgacImage:     "https://example.com/benin.jpg",
		},
		{
			ID:            uuid.New(),
			State:         "Delta",
			Governor:      "Sheriff Oborevwori",
			LGAC:          "Asaba",
			GovernorImage: "https://drive.google.com/file/d/1i8dkwiaQRwjOGmd138PaJp_VNkFHVznj/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/19q1d_AtQOjI1-4_jLESTWcyr_Rhj7GIP/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Benue",
			Governor:      "Hyacinth Alia",
			LGAC:          "Makurdi",
			GovernorImage: "https://drive.google.com/file/d/13mx7-uJU2-vZF00ALTzBvRJj5ECioJPa/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1nZQgOnSHIsT9kpqGc2_kdlO6xG5OtCWC/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Borno",
			Governor:      "Babagana Zulum",
			LGAC:          "Maiduguri",
			GovernorImage: "https://drive.google.com/file/d/1VqN9pjaPqIrW8RQjDkRO-7ABG9OL4yzK/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1AYEX0q7B9VcAdDIIrV69LwCQQkO1n2vM/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Plateau",
			Governor:      "Caleb Mutfwang",
			LGAC:          "Jos",
			GovernorImage: "https://drive.google.com/file/d/1Xn819RjpzyBT8DmfRUz0POp3NDKy5LPn/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1CK4hjqUdnmuI4dr6tDBQotrjYv_JOgTl/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Cross River",
			Governor:      "Bassey Otu",
			LGAC:          "Calabar",
			GovernorImage: "https://drive.google.com/file/d/11DUOgsa6CUgf6SWUPSvkcshVpODw3emh/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1gnZ0D5QV07YHvTDxHNlRSXm9wVgM5Oya/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Akwa Ibom",
			Governor:      "Umo Eno",
			LGAC:          "Uyo",
			GovernorImage: "https://drive.google.com/file/d/1-pnZkCYuPg_sSMWd3SHd5XncnVlDKYxY/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1fCWlNOKLU_uNLpMV8HjZPOVHMRhZ07j-/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Nasarawa",
			Governor:      "Abdullahi Sule",
			LGAC:          "Lafia",
			GovernorImage: "https://drive.google.com/file/d/1cZRCspOB1z9SyyTMd-R9NQxS-__i4DUg/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1CML8fyhrT8PPzRN21rBZFUJEPqMMVQ3m/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Taraba",
			Governor:      "Agbu Kefas",
			LGAC:          "Jalingo",
			GovernorImage: "https://drive.google.com/file/d/1vn-_gq-LgoHGH6BsO4UA-iHoEZ73D6qx/view?usp=sharing",
			LgacImage:     "https://example.com/jalingo.jpg",
		},
		{
			ID:            uuid.New(),
			State:         "Zamfara",
			Governor:      "Dauda Lawal",
			LGAC:          "Gusau",
			GovernorImage: "https://drive.google.com/file/d/1VqKBdJURgEaeb0A1TOQSRT-fxFbBq49p/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1bjjLA2zDjpU_ubsiPBdp0c48QNRefFL-/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Yobe",
			Governor:      "Mai Mala Buni",
			LGAC:          "Damaturu",
			GovernorImage: "https://drive.google.com/file/d/1L-BwRqPyfjeod6Up5nUV8Kb86GN4X44p/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1TdULjESf2ZmHw_GJNzAfS6vs5mGO0STA/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Kogi",
			Governor:      "Ahmed Usman Ododo",
			LGAC:          "Lokoja",
			GovernorImage: "https://drive.google.com/file/d/1RboKD-zpu3IUDqFPcNEEl8q4pv-3QIGM/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1WdDmch2S4s9_Hus6ako6y9nQU4WrRi4g/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Sokoto",
			Governor:      "Ahmad Aliyu",
			LGAC:          "Sokoto North",
			GovernorImage: "https://drive.google.com/file/d/1WpzqTbONg8AN2FpKHtggHNANq77sZqTN/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1qqYgCPKQg6uU1-aPPbDp-dR9njlYNLwl/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Kebbi",
			Governor:      "Nasir Idris",
			LGAC:          "Birnin Kebbi",
			GovernorImage: "https://drive.google.com/file/d/1gSUWtk1q2ZpksApjbKOVi1rHNQtJYOB3/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1mRmi7uqwxnbjyJA4rqCxGRK2C4GoAiQA/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Bauchi",
			Governor:      "Bala Mohammed",
			LGAC:          "Bauchi",
			GovernorImage: "https://drive.google.com/file/d/1KdQTri2UqqKRqXkOhpYLa2_njrP9FDNH/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1cgrMu1NX0eLDe6oEZt4w1f3v1ZTbyc47/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Jigawa",
			Governor:      "Umar Namadi",
			LGAC:          "Dutse",
			GovernorImage: "https://drive.google.com/file/d/1OjcK258GFw3-SQ95V3wOo27PLUYUH-_m/view?usp=sharing",
			LgacImage:     "https://example.com/dutse.jpg",
		},
		{
			ID:            uuid.New(),
			State:         "Katsina",
			Governor:      "Dikko Radda",
			LGAC:          "Katsina",
			GovernorImage: "https://drive.google.com/file/d/1OsAwAHJnbpVdiBcgsZm2F3VGu30t5lbm/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1jt5-nzIahFEX0nTmuZYInlQeTdCLRGNR/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Adamawa",
			Governor:      "Ahmadu Fintiri",
			LGAC:          "Yola",
			GovernorImage: "https://drive.google.com/file/d/1VqkgX_84hJR8TnREs2zYRACrHKZ4l5mk/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1X3ugnSHm-YNu43H2KVJKN9S7YypBJh87/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Niger",
			Governor:      "Mohammed Bago",
			LGAC:          "Minna",
			GovernorImage: "https://drive.google.com/file/d/1LxE79u85HCT9Wv8BZ-u38OVo4w-gyNTV/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1mYy1uH7Xk1Rdw7wYhHaVMo--nfcRyaCX/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Kwara",
			Governor:      "AbdulRahman AbdulRazaq",
			LGAC:          "Ilorin",
			GovernorImage: "https://drive.google.com/file/d/1X5HD18ij5q_EjOjJGX-s2tOXKoRw2p54/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1hgsZxUXBkeyhpDoHJfsE5cb12UvICkQa/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Osun",
			Governor:      "Ademola Adeleke",
			LGAC:          "Osogbo",
			GovernorImage: "https://drive.google.com/file/d/1WH6096Ypu1664Tv6OXIElhnPaW8W6R4t/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1_v_XE-IIuQ8fZX3tYWxy517IvXaUdd-o/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Ekiti",
			Governor:      "Biodun Oyebanji",
			LGAC:          "Ado-Ekiti",
			GovernorImage: "https://drive.google.com/file/d/1vZWfq-cQJTExJCzgLmQut72KCfHqNFfX/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1FFwdO7iBkAhHerV2rlmwmmZQU4TcFtIv/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Bayelsa",
			Governor:      "Douye Diri",
			LGAC:          "Yenagoa",
			GovernorImage: "https://drive.google.com/file/d/18g7LNMIP7Eoqw9fro1zMs_bKFCcTyUHj/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1naYZOXZMm1XakfD9ckXskYLMilfjws39/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Ebonyi",
			Governor:      "Francis Nwifuru",
			LGAC:          "Abakaliki",
			GovernorImage: "https://drive.google.com/file/d/1GI123JAXB0lm0mYmz398lwJ7EknOAbAw/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1CP-XtNNy9ZTcvRv9mv-nALj7-umaBCDU/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Imo",
			Governor:      "Hope Uzodinma",
			LGAC:          "Owerri",
			GovernorImage: "https://drive.google.com/file/d/14QbuX8Mxz7L4iVNW6ey3Q8piiqXciWmT/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1NX2KmsCM1sqwbE77paCQu4xf8GpnZW6T/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Abia",
			Governor:      "Alex Otti",
			LGAC:          "Umuahia",
			GovernorImage: "https://drive.google.com/file/d/1XOX3lmQnbBMU4qIP6bDXBe_BvMQUim52/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1TO91cw1SskIwD2m_yCGSFVNnSWuj_s2d/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Gombe",
			Governor:      "Inuwa Yahaya",
			LGAC:          "Gombe",
			GovernorImage: "https://drive.google.com/file/d/10HRkiN-DNxPb59Ey6CHmokDddCbgWOJC/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1mjKrGVs0N8VoigAZ8ARHI9EQiVHuGRVC/view?usp=sharing",
		},
		{
			ID:            uuid.New(),
			State:         "Ondo",
			Governor:      "Lucky Aiyedatiwa",
			LGAC:          "Akure",
			GovernorImage: "https://drive.google.com/file/d/1WH6096Ypu1664Tv6OXIElhnPaW8W6R4t/view?usp=sharing",
			LgacImage:     "https://drive.google.com/file/d/1lpGyI_PtCA3GmJxEcrPqq1ezO9Bvjjs8/view?usp=sharing",
		},
	}

	for _, state := range states {
		if err := db.FirstOrCreate(&state, models.State{State: state.State}).Error; err != nil {
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
	if err := SeedRoles(db); err != nil {
		return fmt.Errorf("seeding roles error: %v", err)
	}

	// Seed states
	if err := SeedStates(db); err != nil {
		return fmt.Errorf("seeding states error: %v", err)
	}

	// Add any additional migrations or seeds here if needed

	return nil
}