package session

import (
	"encoding/base64"
	"url-shorten/internal/auth-session/domain"

	"github.com/google/uuid"
)

// Entity to Response

func ToSessionResponse(e domain.SessionEntity) (*SessionResponse, error) {
	sid, err := uuidToBase64(e.SessionID)
	if err != nil {
		return nil, err
	}

	sub, err := uuidToBase64(e.UserID)
	if err != nil {
		return nil, err
	}

	return &SessionResponse{
		SessionID:    sid,
		UserID:       sub,
		RefreshToken: e.RefreshToken,
		UserAgent:    e.UserAgent,
		IPAddress:    e.IPAddress,
		IsActive:     e.IsActive,
		LastActiveAt: e.LastActiveAt,
		IsCurrent:    e.IsCurrent,
		ExpiresAt:    e.ExpiresAt,
		IssuedAt:     e.IssuedAt,
	}, nil
}

func ToSessionListResponse(list []domain.SessionEntity) ([]SessionResponse, error) {
	result := make([]SessionResponse, len(list))

	for i := range list {
		res, err := ToSessionResponse(list[i])
		if err != nil {
			return nil, err
		}
		result[i] = *res
	}

	return result, nil
}

// Internal Helper

func uuidToBase64(uuidStr string) (string, error) {
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(parsed[:]), nil
}
