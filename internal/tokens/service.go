package tokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"url-shorten/internal/domains"
	"url-shorten/internal/sessions"
	"url-shorten/pkg"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Interface

type JWTService interface {
	GenerateTokenPair(ctx context.Context, userID, userAgent, ip string, rememberMe bool) (*domains.TokenPairEntity, error)
	ValidateAccessToken(ctx context.Context, tokenStr string) (*domains.ClaimsEntity, error)
	RefreshTokens(ctx context.Context, rawRefreshToken string) (*domains.TokenPairEntity, error)
	RevokeTokens(ctx context.Context, accessToken, rawRefreshToken string) error
}

// Implementation

const (
	refreshTokenPrefix = "refresh:"
	lockPrefix         = "lock:refresh:"
	refreshTokenLength = 32
)

type jwtClaims struct {
	UserID    string `json:"sub"`
	SessionID string `json:"sid"`
	gojwt.RegisteredClaims
}

type refreshTokenPayload struct {
	UserID    string
	SessionID string
	CreatedAt time.Time
	ExpiresAt time.Time
	TTL       time.Duration
}

type jwtService struct {
	secretKey               string
	accessTokenExpiry       time.Duration
	defaultRefreshExpiry    time.Duration
	shortRefreshTokenExpiry time.Duration
	redisClient             *redis.Client
	sessionSvc              sessions.SessionService
}

func NewJWTService(
	secretKey string,
	accessTokenExpiry time.Duration,
	defaultRefreshExpiry time.Duration,
	shortRefreshTokenExpiry time.Duration,
	redisClient *redis.Client,
	sessionSvc sessions.SessionService,
) JWTService {
	return &jwtService{
		secretKey:               secretKey,
		accessTokenExpiry:       accessTokenExpiry,
		defaultRefreshExpiry:    defaultRefreshExpiry,
		shortRefreshTokenExpiry: shortRefreshTokenExpiry,
		redisClient:             redisClient,
		sessionSvc:              sessionSvc,
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *jwtService) generateAccessToken(userID, sessionID string) (string, error) {
	now := time.Now()
	claims := &jwtClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(now.Add(s.accessTokenExpiry)),
			IssuedAt:  gojwt.NewNumericDate(now),
		},
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}
	return signed, nil
}

func (s *jwtService) parseAccessToken(tokenStr string) (*jwtClaims, error) {
	claims := &jwtClaims{}
	_, err := gojwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *gojwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
				return nil, pkg.ErrTokenInvalid
			}
			return []byte(s.secretKey), nil
		},
		gojwt.WithoutClaimsValidation(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", pkg.ErrTokenInvalid, err)
	}
	return claims, nil
}

func (s *jwtService) generateRefreshToken(ctx context.Context, userID, sessionID string, expiryDuration time.Duration) (string, error) {
	b := make([]byte, refreshTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	rawToken := hex.EncodeToString(b)

	now := time.Now()
	payload := refreshTokenPayload{
		UserID:    userID,
		SessionID: sessionID,
		CreatedAt: now,
		ExpiresAt: now.Add(expiryDuration),
		TTL:       expiryDuration,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", pkg.ErrRedisMarshal
	}

	hashedKey := refreshTokenPrefix + hashToken(rawToken)

	success, err := s.redisClient.SetNX(ctx, hashedKey, data, expiryDuration).Result()
	if err := pkg.HandleRedisSetNX(success, err, false); err != nil {
		return "", err
	}

	return rawToken, nil
}

func (s *jwtService) GenerateTokenPair(ctx context.Context, userID, userAgent, ip string, rememberMe bool) (*domains.TokenPairEntity, error) {
    refreshExpiry := s.defaultRefreshExpiry
    if !rememberMe {
        refreshExpiry = s.shortRefreshTokenExpiry
    }

    sessionID := uuid.New().String()
    now       := time.Now()

    // 1. Generate Raw Refresh Token & Hash
    b := make([]byte, refreshTokenLength)
    if _, err := rand.Read(b); err != nil {
        return nil, fmt.Errorf("failed to generate random token: %w", err)
    }
    rawRefreshToken    := hex.EncodeToString(b)
    hashedRefreshToken := hashToken(rawRefreshToken)

    sess := &domains.SessionEntity{
        SessionID:    sessionID,
        UserID:       userID,
        UserAgent:    userAgent,
        IPAddress:    ip,
        RefreshToken: hashedRefreshToken,
        IssuedAt:     now,
        ExpiresAt:    now.Add(refreshExpiry),
    }

    if err := s.sessionSvc.CreateSession(ctx, sess, refreshExpiry); err != nil {
        return nil, err
    }

    // 3. Simpan Refresh Token Payload ke Redis
    payload := refreshTokenPayload{
        UserID:    userID,
        SessionID: sessionID,
        CreatedAt: now,
        ExpiresAt: now.Add(refreshExpiry),
        TTL:       refreshExpiry,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return nil, pkg.ErrRedisMarshal
    }

    hashedKey    := refreshTokenPrefix + hashedRefreshToken
    success, err := s.redisClient.SetNX(ctx, hashedKey, data, refreshExpiry).Result()
    if err := pkg.HandleRedisSetNX(success, err, false); err != nil {
        return nil, err
    }

    accessToken, err := s.generateAccessToken(userID, sessionID)
    if err != nil {
        return nil, err
    }

    return &domains.TokenPairEntity{
        UserID:       userID,
        AccessToken:  accessToken,
        RefreshToken: rawRefreshToken,
    }, nil
}

func (s *jwtService) ValidateAccessToken(ctx context.Context, tokenStr string) (*domains.ClaimsEntity, error) {
	claims, err := s.parseAccessToken(tokenStr)
	if err != nil {
		return nil, pkg.ErrTokenInvalid
	}

	active, err := s.sessionSvc.IsSessionActive(ctx, claims.SessionID)
	if err != nil {
		return nil, pkg.ErrTokenBlocked
	}
	if !active {
		return nil, pkg.ErrTokenRevoked
	}

	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, pkg.ErrTokenExpired
	}

	return &domains.ClaimsEntity{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		ExpiresAt: claims.ExpiresAt.Time,
		IssuedAt:  claims.IssuedAt.Time,
	}, nil
}

func (s *jwtService) RefreshTokens(ctx context.Context, rawRefreshToken string) (*domains.TokenPairEntity, error) {
	hashedToken := hashToken(rawRefreshToken)
	lockKey     := lockPrefix + hashedToken

	locked, err := s.redisClient.SetNX(ctx, lockKey, "locked", 5*time.Second).Result()
	if err := pkg.HandleRedisSetNX(locked, err, true); err != nil {
		if errors.Is(err, pkg.ErrRedisLockNotAcquired) {
			return nil, pkg.ErrRefreshTokenConcurrent
		}
		return nil, err
	}
	defer s.redisClient.Del(ctx, lockKey)

	hashedKey := refreshTokenPrefix + hashedToken
	data, err := s.redisClient.Get(ctx, hashedKey).Bytes()
	if err != nil {
		domainErr := pkg.HandleRedisError(err)
		if errors.Is(domainErr, pkg.ErrRedisKeyNotFound) {
			return nil, pkg.ErrRefreshTokenInvalid
		}
		return nil, domainErr
	}

	var payload refreshTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, pkg.ErrRedisUnmarshal
	}

	active, err := s.sessionSvc.IsSessionActive(ctx, payload.SessionID)
	if err != nil || !active {
		return nil, pkg.ErrTokenRevoked
	}

	if err := s.redisClient.Del(ctx, hashedKey).Err(); err != nil {
		return nil, pkg.HandleRedisError(err)
	}

	newAccessToken, err := s.generateAccessToken(payload.UserID, payload.SessionID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.generateRefreshToken(ctx, payload.UserID, payload.SessionID, payload.TTL)
	if err != nil {
		return nil, err
	}

	return &domains.TokenPairEntity{
		UserID:       payload.UserID,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *jwtService) RevokeTokens(ctx context.Context, accessToken, rawRefreshToken string) error {
	claims, err := s.parseAccessToken(accessToken)
	if err == nil {
		_ = s.sessionSvc.RevokeSession(ctx, claims.UserID, claims.SessionID)
	}

	hashedKey := refreshTokenPrefix + hashToken(rawRefreshToken)
	if err := s.redisClient.Del(ctx, hashedKey).Err(); err != nil {
		return pkg.HandleRedisError(err)
	}

	return nil
}