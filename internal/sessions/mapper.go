package sessions

import (
	"url-shorten/internal/domains"
)

// Entity to Response

func ToSessionResponse(e domains.SessionEntity) *SessionResponse {
	return &SessionResponse{
		SessionID:    e.SessionID,
		UserID:       e.UserID,
		RefreshToken: e.RefreshToken,
		UserAgent:    e.UserAgent, 
		IPAddress:    e.IPAddress,
		ExpiresAt:    e.ExpiresAt,
		IssuedAt:     e.IssuedAt,
	}
}

func ToSessionListResponse(list []domains.SessionEntity) []SessionResponse {
	result := make([]SessionResponse, len(list))

	for i := range list {
		result[i] = *ToSessionResponse(list[i])
	}

	return result
}

