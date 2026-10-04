package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	AppPort              string
	AppURL               string
	DatabaseURL          string
	JWTSecretKey         string
	JWTExpiry            time.Duration
	DefaultRefreshExpiry time.Duration
	ShortRefreshExpiry   time.Duration
	RedisAddr            string
	RedisPassword        string
	AdminPassword        string
	TrustedProxies       []string
	SecureCookies        bool
}

func NewConfig() AppConfig {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found")
	}

	appPort       := requiredEnv("APP_PORT")
	appURL        := requiredEnv("APP_URL")
	dsn           := requiredEnv("DATABASE_URL")
	jwtSecret     := requiredEnv("JWT_SECRET_KEY")
	redisAddr     := requiredEnv("REDIS_ADDR")
	redisPassword := requiredEnv("REDIS_PASSWORD")
	adminPassword := requiredEnv("ADMIN_PASSWORD")

	jwtExpiry            := parseDuration("JWT_EXPIRY")
	defaultRefreshExpiry := parseDuration("DEFAULT_REFRESH_EXPIRY")
	shortRefreshExpiry   := parseDuration("SHORT_REFRESH_EXPIRY")

	var trustedProxies []string
	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			trustedProxies = append(trustedProxies, proxy)
		}
	}

	return AppConfig{
		AppPort:              appPort,
		AppURL:               appURL,
		DatabaseURL:          dsn,
		JWTSecretKey:         jwtSecret,
		JWTExpiry:            jwtExpiry,
		DefaultRefreshExpiry: defaultRefreshExpiry,
		ShortRefreshExpiry:   shortRefreshExpiry,
		RedisAddr:            redisAddr,
		RedisPassword:        redisPassword,
		AdminPassword:        adminPassword,
		TrustedProxies:       trustedProxies,
		SecureCookies:        strings.EqualFold(os.Getenv("APP_ENV"), "production"),
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s environment variable is required", name)
	}
	return value
}

func parseDuration(name string) time.Duration {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil {
		log.Fatalf("invalid %s format", name)
	}
	return value
}
