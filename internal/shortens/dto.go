package shortens

import "time"

// Shorten Request

type CreateShortenPublicRequest struct {
	OriginalURL string `json:"original_url" validata:"required,url"`
	ShortCode   string `json:"-"`
}

type CreateShortenAuthorizedRequest struct {
	OriginalURL string     `json:"original_url" validate:"required,url"`
	ShortCode   string     `json:"-"`
	Owner       string     `json:"-"`
	ExpiresAt   *time.Time `json:"expires_at"   validate:"omitempty"`
}

type UpdateShortenAuthorizedRequest struct {
	ID          string     `json:"-"`
	OriginalURL *string    `json:"original_url" validate:"omitempty,url"`
	ShortCode   *string    `json:"-"`
	Owner       string     `json:"-"`
	IsActive    *bool      `json:"-" form:"active"`
	ExpiresAt   *time.Time `json:"expires_at"   validate:"omitempty"`
}

// Shorten Response

type ShortenResponse struct {
	ID          string     `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	Owner       *string    `json:"owner,omitempty"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}