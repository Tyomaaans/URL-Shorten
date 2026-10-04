package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"url-shorten/internal/auth-session/domain"
)

type UserStorage struct {
	ID         string      `gorm:"primaryKey"`
	Name       string      `gorm:"type:varchar(255);not null"`
	Email      string      `gorm:"type:varchar(255);uniqueIndex;not null"`
	Password   string      `gorm:"type:varchar(255);not null"`
	Role       domain.Role `gorm:"type:varchar(100);not null"`
	RememberMe bool        `gorm:"not null;default:false"`
	CreatedAt  time.Time   `gorn:"not null"`
	UpdatedAt  time.Time   `gorm:"not null"`
}

func SeedAdmin(db *gorm.DB, plainPassword string) error {
	var count int64
	db.Model(&UserStorage{}).Where("role = ?", domain.Admin).Count(&count)
	if count > 0 {
		return nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seedAdmin: failed to hash password %w", err)
	}

	now := time.Now()
	admin := UserStorage{
		ID:         uuid.NewString(),
		Name:       "Admin Test",
		Email:      "admin@test.com",
		Password:   string(hashed),
		Role:       domain.Admin,
		RememberMe: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("seedAdmin: failed to insert admin %w", err)
	}

	return nil
}
