package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect database: %w", err)
    }

    if err := db.AutoMigrate(
    	&UserStorage{},
		&ShortenStorage{},
    ); err != nil {
        return nil, fmt.Errorf("failed to migrate: %w", err)
    }

    return db, nil
}