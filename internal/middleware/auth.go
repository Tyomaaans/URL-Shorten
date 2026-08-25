package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"url-shorten/internal/tokens"
	"url-shorten/pkg"
)

type (
	contextKey string
)

const (
	IsAdminKey contextKey = "isAdmin"
)


type AuthMiddleware struct {
	jwtService tokens.JWTService
	adminSecret string
}

func NewAuthMiddleware(jwtService tokens.JWTService, adminSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		adminSecret: adminSecret,
	}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, ok := extractBearerToken(c)
		if !ok {
			abortWithError(c, http.StatusUnauthorized, "missing or invalid authorization header", "AUTH_TOKEN_MISSING")
			return
		}

		claims, err := m.jwtService.ValidateAccessToken(c.Request.Context(), accessToken)
		if err != nil {
			switch {
			case errors.Is(err, pkg.ErrTokenExpired):
				abortWithError(c, http.StatusUnauthorized, "access token expired", "AUTH_TOKEN_EXPIRED")
			case errors.Is(err, pkg.ErrTokenRevoked):
				abortWithError(c, http.StatusUnauthorized, "token has been revoked", "AUTH_TOKEN_REVOKED")
			default:
				abortWithError(c, http.StatusUnauthorized, "invalid token", "AUTH_TOKEN_INVALID")
			}
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)

		c.Next()
	}
}

func (m *AuthMiddleware) AdminSecretMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminToken := c.GetHeader("X-Admin-Secret")

		if adminToken != m.adminSecret {
			abortWithError(c, http.StatusForbidden, "access denied", "AUTHZ_ROLE_FORBIDDEN")
			return
		}

		ctx := context.WithValue(c.Request.Context(), IsAdminKey, true)
		c.Request = c.Request.WithContext(ctx)
		
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
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

