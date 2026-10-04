package shortener

import (
	"time"
	"url-shorten/internal/url-shortener/domain"
)

// Shorten Request

type CreateShortenPublicRequest struct {
	OriginalURL string `json:"original_url" validate:"required,http_url"`
	ShortCode   string `json:"-"`
}

type CreateShortenAuthorizedRequest struct {
	OriginalURL string     `json:"original_url" validate:"required,http_url"`
	ShortCode   string     `json:"-"`
	Owner       string     `json:"-"`
	ExpiresAt   *time.Time `json:"expires_at"   validate:"omitempty"`
}

type UpdateShortenAuthorizedRequest struct {
	ID          string              `json:"-"`
	OriginalURL *string             `json:"original_url" validate:"omitempty,http_url"`
	Owner       string              `json:"-"`
	IsActive    *bool               `json:"-"`
	ExpiresAt   domain.NullableTime `json:"expires_at"   validate:"omitempty"`
}

// Shorten Response

type ShortenResponse struct {
	ID          string     `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	Owner       *string    `json:"owner,omitempty"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	PageSize    int   `json:"page_size"`
	TotalData   int64 `json:"total_data"`
}

type PaginatedShortenResponse struct {
	Data []ShortenResponse `json:"users"`
	Meta PaginationMeta    `json:"meta"`
}
