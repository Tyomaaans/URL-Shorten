package postgres

import (
	"time"
)

type ShortenStorage struct {
	ID          string         `gorm:"primaryKey"`
	OriginalURL string         `gorm:"type:varchar(255);not null"`
	ShortCode   string         `gorm:"type:varchar(255);not null"`
	Owner       *string        `gorm:"type:uuid"`
	IsActive    *bool          `gorm:"not null;default:true"`
	ExpiresAt   *time.Time     `gorm:"index"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`

	User *UserStorage `gorm:"foreignKey:Owner;constraint:OnDelete:CASCADE"`
}