package token

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

	"url-shorten/internal/auth-session/domain"
	"url-shorten/internal/auth-session/session"
	"url-shorten/pkg"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Interface

type JWTService interface {
	GenerateTokenPair(ctx context.Context, userID, userAgent, ip, role string, rememberMe bool) (*domain.TokenPairEntity, string, error)
	ValidateAccessToken(ctx context.Context, tokenStr string) (*domain.ClaimsEntity, error)
	RefreshTokens(ctx context.Context, rawRefreshToken string) (*domain.TokenPairEntity, error)
	RevokeTokens(ctx context.Context, accessToken, rawRefreshToken string) error
	RevokeAllUserTokens(ctx context.Context, userID string) (int, error)
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
	Role      string `json:"role"`
	gojwt.RegisteredClaims
}

type refreshTokenPayload struct {
	UserID     string
	SessionID  string
	Role       string
	RememberMe bool
	CreatedAt  time.Time
	ExpiresAt  time.Time
	TTL        time.Duration
}

type jwtService struct {
	secretKey               string
	accessTokenExpiry       time.Duration
	defaultRefreshExpiry    time.Duration
	shortRefreshTokenExpiry time.Duration
	redisClient             *redis.Client
	sessionSvc              session.SessionService
}

func NewJWTService(
	secretKey               string,
	accessTokenExpiry       time.Duration,
	defaultRefreshExpiry    time.Duration,
	shortRefreshTokenExpiry time.Duration,
	redisClient             *redis.Client,
	sessionSvc              session.SessionService,
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

// Main Token Service

func (s *jwtService) GenerateTokenPair(ctx context.Context, userID, userAgent, ip, role string, rememberMe bool) (*domain.TokenPairEntity, string, error) {
	refreshExpiry := s.defaultRefreshExpiry
	if !rememberMe {
		refreshExpiry = s.shortRefreshTokenExpiry
	}

	sessionID := uuid.New().String()
	now := time.Now()

	b := make([]byte, refreshTokenLength)
	if _, err := rand.Read(b); err != nil {
		return nil, "", fmt.Errorf("failed to generate random token: %w", err)
	}
	rawRefreshToken := hex.EncodeToString(b)
	hashedRefreshToken := hashToken(rawRefreshToken)

	sess := &domain.SessionEntity{
		SessionID:    sessionID,
		UserID:       userID,
		UserAgent:    userAgent,
		IPAddress:    ip,
		IsActive:     true,
		LastActiveAt: time.Now(),
		RefreshToken: hashedRefreshToken,
		IsCurrent:    true,
		IssuedAt:     now,
		ExpiresAt:    now.Add(refreshExpiry),
	}

	if err := s.sessionSvc.CreateSession(ctx, sess, refreshExpiry); err != nil {
		return nil, "", err
	}

	payload := refreshTokenPayload{
		UserID:     userID,
		SessionID:  sessionID,
		Role:       role,
		RememberMe: rememberMe,
		CreatedAt:  now,
		ExpiresAt:  now.Add(refreshExpiry),
		TTL:        refreshExpiry,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", pkg.ErrRedisMarshal
	}

	hashedKey := refreshTokenPrefix + hashedRefreshToken
	success, err := s.redisClient.SetNX(ctx, hashedKey, data, refreshExpiry).Result()
	if err := pkg.HandleRedisSetNX(success, err, false); err != nil {
		_ = s.sessionSvc.RevokeSession(ctx, userID, sessionID)
		return nil, "", err
	}

	accessToken, err := s.generateAccessToken(userID, sessionID, role)
	if err != nil {
		return nil, "", err
	}

	return &domain.TokenPairEntity{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		RememberMe:   rememberMe,
	}, sessionID, nil
}

func (s *jwtService) ValidateAccessToken(ctx context.Context, tokenStr string) (*domain.ClaimsEntity, error) {
	claims, err := s.parseAccessToken(tokenStr)
	if err != nil {
		return nil, pkg.ErrTokenInvalid
	}

	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, pkg.ErrTokenExpired
	}

	active, err := s.sessionSvc.IsSessionActive(ctx, claims.SessionID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, pkg.ErrTokenRevoked
	}

	return &domain.ClaimsEntity{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		Role:      domain.Role(claims.Role),
		ExpiresAt: claims.ExpiresAt.Time,
		IssuedAt:  claims.IssuedAt.Time,
	}, nil
}

func (s *jwtService) RefreshTokens(ctx context.Context, rawRefreshToken string) (*domain.TokenPairEntity, error) {
	hashedToken := hashToken(rawRefreshToken)
	lockKey := lockPrefix + hashedToken

	lockVal := uuid.New().String()
	locked, err := s.redisClient.SetNX(ctx, lockKey, lockVal, 5*time.Second).Result()
	if err := pkg.HandleRedisSetNX(locked, err, true); err != nil {
		if errors.Is(err, pkg.ErrRedisLockNotAcquired) {
			return nil, pkg.ErrRefreshTokenConcurrent
		}
		return nil, err
	}
	defer s.releaseLock(lockKey, lockVal)

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
	if !payload.RememberMe && payload.TTL == s.defaultRefreshExpiry {
		payload.RememberMe = true
	}

	active, err := s.sessionSvc.IsSessionActive(ctx, payload.SessionID)
	if err != nil || !active {
		return nil, pkg.ErrTokenRevoked
	}

	if err := s.redisClient.Del(ctx, hashedKey).Err(); err != nil {
		return nil, pkg.HandleRedisError(err)
	}

	newAccessToken, err := s.generateAccessToken(payload.UserID, payload.SessionID, payload.Role)
	if err != nil {
		return nil, err
	}

	newRawRefresh, newHashedRefresh, err := s.generateRefreshToken(ctx, payload.UserID, payload.SessionID, payload.Role, payload.RememberMe, payload.TTL)
	if err != nil {
		return nil, err
	}

	if err := s.sessionSvc.ExtendSession(ctx, payload.SessionID, newHashedRefresh, payload.TTL); err != nil {
		_ = s.redisClient.Del(ctx, refreshTokenPrefix+newHashedRefresh).Err()
		return nil, err
	}

	return &domain.TokenPairEntity{
		UserID:       payload.UserID,
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefresh,
		RememberMe:   payload.RememberMe,
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

func (s *jwtService) RevokeAllUserTokens(ctx context.Context, userID string) (int, error) {
	if userID == "" {
		return 0, pkg.ErrTokenInvalid
	}

	purged, err := s.sessionSvc.PurgeUserSessions(ctx, userID)
	if err != nil {
		return purged, err
	}

	if err := s.sweepOrphanRefreshTokens(ctx, userID); err != nil {
		return purged, err
	}

	return purged, nil
}

// Internal Helper

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *jwtService) generateAccessToken(userID, sessionID, role string) (string, error) {
	now := time.Now()
	claims := &jwtClaims{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
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

func (s *jwtService) generateRefreshToken(ctx context.Context, userID, sessionID, role string, rememberMe bool, expiryDuration time.Duration) (raw string, hashed string, err error) {
	b := make([]byte, refreshTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}
	raw = hex.EncodeToString(b)
	hashed = hashToken(raw)

	now := time.Now()
	payload := refreshTokenPayload{
		UserID:     userID,
		SessionID:  sessionID,
		Role:       role,
		RememberMe: rememberMe,
		CreatedAt:  now,
		ExpiresAt:  now.Add(expiryDuration),
		TTL:        expiryDuration,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", "", pkg.ErrRedisMarshal
	}

	success, err := s.redisClient.SetNX(ctx, refreshTokenPrefix+hashed, data, expiryDuration).Result()
	if err := pkg.HandleRedisSetNX(success, err, false); err != nil {
		return "", "", err
	}
	return raw, hashed, nil
}

var releaseLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

func (s *jwtService) releaseLock(lockKey, lockVal string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = releaseLockScript.Run(ctx, s.redisClient, []string{lockKey}, lockVal).Err()
}

func (s *jwtService) sweepOrphanRefreshTokens(ctx context.Context, userID string) error {
	var cursor uint64
	for {
		keys, next, err := s.redisClient.Scan(ctx, cursor, refreshTokenPrefix+"*", 200).Result()
		if err != nil {
			return pkg.HandleRedisError(err)
		}

		if len(keys) > 0 {
			getPipe := s.redisClient.Pipeline()
			cmds := make([]*redis.StringCmd, len(keys))
			for i, k := range keys {
				cmds[i] = getPipe.Get(ctx, k)
			}
			_, _ = getPipe.Exec(ctx)

			var toDelete []string
			for i, k := range keys {
				raw, err := cmds[i].Bytes()
				if err != nil {
					continue
				}
				var p refreshTokenPayload
				if json.Unmarshal(raw, &p) == nil && p.UserID == userID {
					toDelete = append(toDelete, k)
				}
			}
			if len(toDelete) > 0 {
				if err := s.redisClient.Del(ctx, toDelete...).Err(); err != nil {
					return pkg.HandleRedisError(err)
				}
			}
		}

		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
