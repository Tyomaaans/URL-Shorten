package domains

import (
	"context"
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
}

type UpdateShortenEntity struct {
	ID          string
	Owner       string
	OriginalURL *string
	ShortCode   *string
	IsActive    *bool
	ExpiresAt   *time.Time
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
	GetShortens(ctx context.Context) ([]ShortenEntity, error)
	GetShortenByID(ctx context.Context, id string) (*ShortenEntity, error)
	GetShortenByOwner(ctx context.Context, owner string) ([]ShortenEntity, error)
	GetShortenByShortCode(ctx context.Context, shortCode string) (*ShortenEntity, error)
	GetActiveWithExpiry(ctx context.Context) ([]ShortenEntity, error)
	DeleteShorten(ctx context.Context, id string) error
}