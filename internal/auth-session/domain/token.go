package domain

import (
	"time"
)

type ClaimsEntity struct {
	UserID    string
	SessionID string
	Role      Role
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type TokenPairEntity struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	RememberMe   bool
}
