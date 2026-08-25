package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	APPport              string
	DSN                  string
	JWTSecretKey         string
	JWTExpiry            time.Duration
    DefaultRefreshExpiry time.Duration
	ShortRefreshExpiry   time.Duration
	REDISaddr            string
	REDISpassword        string
    APPurl               string
	AdminSecretKey       string
}

func NewConfig() AppConfig {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found!")
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
        log.Fatal("APP_PORT environment variable is required!")
    }

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environtment variable is required!")
	}

	accessExpiryStr := os.Getenv("JWT_EXPIRY")
	accessExpiry, err := time.ParseDuration(accessExpiryStr)
	if err != nil {
		log.Fatal("invalid JWT_EXPIRY format!")
	}

    defaultExpiryStr := os.Getenv("DEFAULT_REFRESH_EXPIRY")
	defaultExpiry, err := time.ParseDuration(defaultExpiryStr)
	if err != nil {
		log.Fatal("invalid DEFAULT_REFRESH_EXPIRY format!")
	}

	shortExpiryStr := os.Getenv("SHORT_REFRESH_EXPIRY")
	shortExpiry, err := time.ParseDuration(shortExpiryStr)
	if err != nil {
		log.Fatal("invalid DEFAULT_REFRESH_EXPIRY format!")
	} 
	
    secretKey := os.Getenv("JWT_SECRET_KEY")
    if secretKey == "" {
        log.Fatal("JWT_SECRET_KEY environment variable is required!")
    }

	redisAddr := os.Getenv("REDIS_ADDR")
    if redisAddr == "" {
        log.Fatal("REDIS_ADDR environment variable is required!")
    }

	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword == "" {
        log.Fatal("REDIS_PASSWORD environment variable is required!")
    }

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		log.Fatal("APP_URL environment variable is required!")
	}

	adminSecret := os.Getenv("ADMIN_SECRET_KEY")
	if appURL == "" {
		log.Fatal("ADMIN_SECRET_KEY environment variable is required!")
	}

	return AppConfig{
		APPport:              appPort,
		DSN:                  dsn,
		JWTSecretKey:         secretKey,
        JWTExpiry:            accessExpiry,
        DefaultRefreshExpiry: defaultExpiry,
		ShortRefreshExpiry:   shortExpiry,
		REDISaddr:            redisAddr,
		REDISpassword:        redisPassword,
		APPurl:               appURL,
		AdminSecretKey:       adminSecret,
	}
}