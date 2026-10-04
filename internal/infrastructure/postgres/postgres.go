package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"url-shorten/internal/auth-session/user"
	"url-shorten/internal/url-shortener/shortener"
)

func NewPostgresDB(dsn, adminPassword string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to connect database %w", err)
	}

	if err := db.AutoMigrate(&user.UserStorage{}, &shortener.ShortenStorage{}); err != nil {
		return nil, fmt.Errorf("postgres: failed to migrate %w", err)
	}

	if err := user.SeedAdmin(db, adminPassword); err != nil {
		return nil, fmt.Errorf("postgres: failed to seed database %w", err)
	}

	return db, nil
}
