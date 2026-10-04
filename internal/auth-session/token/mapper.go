package token

import (
	"url-shorten/internal/auth-session/domain"
)

// Entity to Response

func ToTokenPairResponse(e domain.TokenPairEntity) *TokenPairResponse {
	return &TokenPairResponse{
		UserID:       e.UserID,
		AccessToken:  e.AccessToken,
		RefreshToken: e.RefreshToken,
		RememberMe:   e.RememberMe,
	}
}
