package sessions

import (
	"time"
)

type SessionResponse struct {
	SessionID    string    `json:"sid"`
	UserID       string    `json:"sub"`
	RefreshToken string    `json:"rt"`
	UserAgent    string    `json:"ua"`
	IPAddress    string    `json:"ip"`
	ExpiresAt    time.Time `json:"exp"`
	IssuedAt     time.Time `json:"iat"`
}