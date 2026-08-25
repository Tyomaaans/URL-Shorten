package postgres

import (
	"time"
)

type UserStorage struct {
	ID         string    `gorm:"primaryKey"`
	Name       string    `gorm:"varchar(255);not null"`
	Email      string    `gorm:"varchar(255);uniqueIndex;not null"`
	Password   string    `gorm:"varchar(255);not null"`
	RememberMe bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`

	Shortens []ShortenStorage `gorm:"foreignKey:Owner"`
}