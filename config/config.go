package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug                        bool   `envconfig:"debug"`
	Port                         int    `envconfig:"port"`
	PostgresHost                 string `envconfig:"postgres_host"`
	PostgresUser                 string `envconfig:"postgres_user"`
	PostgresDB                   string `envconfig:"postgres_db"`
	MailgunApiKey                string `envconfig:"mg_public_api_key"`
	MgEmailFrom                  string `envconfig:"email_from"`
	BaseUrl                      string `envconfig:"base_url"`
	Env                          string `envconfig:"env"`
	PostgresPort                 int    `envconfig:"postgres_port"`
	PostgresPassword             string `envconfig:"postgres_password"`
	JWTSecret                    string `envconfig:"jwt_secret"`
	MgDomain                     string `envconfig:"mg_domain"`
	Host                         string `envconfig:"host"`
	GoogleClientID               string `envconfig:"google_client_id"`
	GoogleClientSecret           string `envconfig:"google_client_secret"`
	GoogleRedirectURL            string `envconfig:"google_redirect_url"`
	GoogleApplicationCredentials string `envconfig:"google_application_credentials"`
	FacebookAppId                string `envconfig:"facebook_app_id"`
	FacebookAppSecret            string `envconfig:"facebook_app_secret"`
	FacebookRedirectURL          string `envconfig:"facebook_redirect_url"`
	GoogleMapsApiKey             string `envconfig:"google_maps_api_key"`
	AccessControlAllowOrigin     string `envconfig:"accessc_control_allow_origin"`
}

func Load() (*Config, error) {
	env := os.Getenv("GIN_MODE")
	if env != "release" {
		if err := godotenv.Load("./.env"); err != nil {
			log.Printf("couldn't load env vars: %v", err)
		}
	}

	c := &Config{}
	err := envconfig.Process("citizenx", c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
