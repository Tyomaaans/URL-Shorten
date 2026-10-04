package domain

import (
	"time"
)

type SessionEntity struct {
	SessionID    string
	UserID       string
	Role         Role
	RefreshToken string
	UserAgent    string
	IPAddress    string
	IsActive     bool
	LastActiveAt time.Time
	IsCurrent    bool
	ExpiresAt    time.Time
	IssuedAt     time.Time
}
