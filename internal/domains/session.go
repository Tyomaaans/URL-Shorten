package domains

import (
	"time"
)

type SessionEntity struct {
	SessionID    string
	UserID       string
	RefreshToken string
	UserAgent    string
	IPAddress    string
	ExpiresAt    time.Time
	IssuedAt     time.Time
}