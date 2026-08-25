package pkg

import (
	"errors"
)

var (
	// Session
	ErrSessionExists   = errors.New("session already exists")
	ErrSessionNotFound = errors.New("session not found")

	// Token
	ErrTokenExpired           = errors.New("access token expired")
	ErrTokenRevoked           = errors.New("token has been revoked")
	ErrTokenInvalid           = errors.New("invalid token")
	ErrTokenBlocked           = errors.New("failed to check token status")
	ErrRefreshTokenInvalid    = errors.New("refresh token expired or already used")
	ErrRefreshTokenConcurrent = errors.New("refresh token rotation is currently processing")

	// Credentials
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrForbidden          = errors.New("access denied")

	// Input
	ErrInvalidInput = errors.New("")
)