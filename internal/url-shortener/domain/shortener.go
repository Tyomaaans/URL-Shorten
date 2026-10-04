package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Shorten Entity

type ShortenEntity struct {
	ID          string
	OriginalURL string
	ShortCode   string
	Owner       *string
	IsActive    *bool
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}

type UpdateShortenEntity struct {
	ID          string
	Owner       string
	OriginalURL *string
	IsActive    *bool
	ExpiresAt   NullableTime
}

type NullableTime struct {
	Value *time.Time
	Set   bool
}

func (n *NullableTime) UnmarshalJSON(data []byte) error {
	n.Set = true

	if string(data) == "null" {
		n.Value = nil
		return nil
	}

	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	n.Value = &t
	return nil
}

// Shorten Cache

type ShortenCacheEntity struct {
	ID          string
	OriginalURL string
	Owner       *string
	IsActive    *bool
	ExpiresAt   *time.Time
}

// Shorten Repository Interface

type ShortenRepository interface {
	CreateShorten(ctx context.Context, shorten *ShortenEntity) (*ShortenEntity, error)
	UpdateShorten(ctx context.Context, update *UpdateShortenEntity) (*ShortenEntity, error)
	GetShortens(ctx context.Context, page, limit int) ([]ShortenEntity, int64, error)
	GetShortenByID(ctx context.Context, id string) (*ShortenEntity, error)
	GetShortenByOwner(ctx context.Context, owner string, page, limit int) ([]ShortenEntity, int64, error)
	GetShortenByShortCode(ctx context.Context, shortCode string) (*ShortenEntity, error)
	DeactivateByShortCode(ctx context.Context, shortCode string, isActive *bool) error
	DeleteShorten(ctx context.Context, id string) error
}
