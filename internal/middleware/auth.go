package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"url-shorten/internal/auth-session/token"
	"url-shorten/pkg"
)

type AuthMiddleware struct {
	tokenSvc token.JWTService
}

func NewAuthMiddleware(tokenSvc token.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		tokenSvc: tokenSvc,
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, ok := extractBearerToken(c)
		if !ok {
			abortWithError(c, http.StatusUnauthorized, "missing or invalid authorization header", "AUTH_TOKEN_MISSING")
			return
		}

		claims, err := m.tokenSvc.ValidateAccessToken(c.Request.Context(), accessToken)
		if err != nil {
			switch {
			case errors.Is(err, pkg.ErrTokenExpired):
				abortWithError(c, http.StatusUnauthorized, "access token expired", "AUTH_TOKEN_EXPIRED")
			case errors.Is(err, pkg.ErrTokenRevoked):
				abortWithError(c, http.StatusUnauthorized, "token has been revoked", "AUTH_TOKEN_REVOKED")
			case errors.Is(err, pkg.ErrTokenBlocked):
				abortWithError(c, http.StatusUnauthorized, "token is blocked", "AUTH_TOKEN_BLOCKED")
			default:
				abortWithError(c, http.StatusUnauthorized, "invalid token", "AUTH_TOKEN_INVALID")
			}
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Set("role", string(claims.Role))

		c.Next()
	}
}

func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			abortWithError(c, http.StatusForbidden, "role not found in token", "AUTHZ_ROLE_MISSING")
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			abortWithError(c, http.StatusForbidden, "invalid role format", "AUTHZ_ROLE_INVALID")
			return
		}

		for _, r := range roles {
			if roleStr == r {
				c.Next()
				return
			}
		}

		abortWithError(c, http.StatusForbidden, "access denied", "AUTHZ_ROLE_FORBIDDEN")
	}
}

func extractBearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return "", false
	}

	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" {
		return "", false
	}

	return token, true
}

func abortWithError(c *gin.Context, status int, message, code string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}
