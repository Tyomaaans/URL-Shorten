package domains

import (
	"time"
)

type ClaimsEntity struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type TokenPairEntity struct {
	UserID       string
	AccessToken  string
	RefreshToken string
}