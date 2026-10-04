package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"url-shorten/internal/auth-session/domain"
	"url-shorten/pkg"

	"github.com/redis/go-redis/v9"
)

// Interface

type SessionService interface {
	CreateSession(ctx context.Context, session *domain.SessionEntity, ttl time.Duration) error
	IsSessionActive(ctx context.Context, sessionID string) (bool, error)
	GetActiveSessions(ctx context.Context, userID, sessionID string) ([]domain.SessionEntity, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	RevokeAllOtherSessions(ctx context.Context, userID string, exceptSessionID string) error
	ExtendSession(ctx context.Context, sessionID, newHashedRefreshToken string, ttl time.Duration) error
	PurgeUserSessions(ctx context.Context, userID string) (int, error)
	Heartbeat(ctx context.Context, sessionID string, state string) error
}

// Implementation

const (
	sessionPrefix      = "session:"
	userSessionsPrefix = "user_sessions:"
)

type sessionService struct {
	redisClient *redis.Client
}

func NewSessionService(redisClient *redis.Client) SessionService {
	return &sessionService{
		redisClient: redisClient,
	}
}

func hashID(id string) string {
	h := sha256.Sum256([]byte(id))
	return hex.EncodeToString(h[:])
}

func (s *sessionService) CreateSession(ctx context.Context, session *domain.SessionEntity, ttl time.Duration) error {
	hashedSessionID := hashID(session.SessionID)
	sessionKey := sessionPrefix + hashedSessionID
	userSetKey := userSessionsPrefix + session.UserID

	data, err := json.Marshal(session)
	if err != nil {
		return pkg.ErrRedisMarshal
	}

	created, err := s.redisClient.SetNX(ctx, sessionKey, data, ttl).Result()
	if err := pkg.HandleRedisSetNX(created, err, false); err != nil {
		if errors.Is(err, pkg.ErrRedisCollision) {
			return pkg.ErrSessionExists
		}
		return err
	}

	pipe := s.redisClient.Pipeline()
	pipe.SAdd(ctx, userSetKey, hashedSessionID)
	pipe.Expire(ctx, userSetKey, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		_ = s.redisClient.Del(ctx, sessionKey)
		return pkg.HandleRedisError(err)
	}

	return nil
}

func (s *sessionService) IsSessionActive(ctx context.Context, sessionID string) (bool, error) {
	sessionKey := sessionPrefix + hashID(sessionID)

	exists, err := s.redisClient.Exists(ctx, sessionKey).Result()
	if err != nil {
		return false, pkg.HandleRedisError(err)
	}
	return exists > 0, nil
}

func (s *sessionService) GetActiveSessions(ctx context.Context, userID, sessionID string) ([]domain.SessionEntity, error) {
	userSetKey := userSessionsPrefix + userID

	hashedIDs, err := s.redisClient.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return nil, pkg.HandleRedisError(err)
	}

	if len(hashedIDs) == 0 {
		return nil, pkg.ErrSessionNotFound
	}

	var activeSessions []domain.SessionEntity
	var expiredHashedIDs []interface{}

	for _, hID := range hashedIDs {
		sessionKey := sessionPrefix + hID
		data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
		if err != nil {
			domainErr := pkg.HandleRedisError(err)
			if errors.Is(domainErr, pkg.ErrRedisKeyNotFound) {
				expiredHashedIDs = append(expiredHashedIDs, hID)
			}
			continue
		}

		var sess domain.SessionEntity
		if err := json.Unmarshal(data, &sess); err == nil {
			if time.Since(sess.LastActiveAt) > (120 * time.Second) {
				sess.IsActive = false
			} else {
				sess.IsActive = true
			}

			if sessionID != "" {
				if sessionID == sess.SessionID {
					sess.IsCurrent = true
				} else {
					sess.IsCurrent = false
				}
			}

			activeSessions = append(activeSessions, sess)
		}
	}

	if len(expiredHashedIDs) > 0 {
		s.redisClient.SRem(ctx, userSetKey, expiredHashedIDs...)
	}

	if len(activeSessions) == 0 {
		return nil, pkg.ErrSessionNotFound
	}

	return activeSessions, nil
}

func (s *sessionService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	hashedSessionID := hashID(sessionID)
	sessionKey := sessionPrefix + hashedSessionID
	userSetKey := userSessionsPrefix + userID

	data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
	if err != nil {
		domainErr := pkg.HandleRedisError(err)
		if errors.Is(domainErr, pkg.ErrRedisKeyNotFound) {
			return nil
		}
		return domainErr
	}

	var sess domain.SessionEntity
	if err := json.Unmarshal(data, &sess); err != nil {
		return pkg.ErrRedisUnmarshal
	}

	pipe := s.redisClient.Pipeline()
	pipe.Del(ctx, sessionKey)
	pipe.SRem(ctx, userSetKey, hashedSessionID)

	if sess.RefreshToken != "" {
		refreshTokenKey := "refresh:" + sess.RefreshToken
		pipe.Del(ctx, refreshTokenKey)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return pkg.HandleRedisError(err)
	}

	return nil
}

func (s *sessionService) RevokeAllOtherSessions(ctx context.Context, userID string, exceptSessionID string) error {
	userSetKey := userSessionsPrefix + userID

	hashedIDs, err := s.redisClient.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return pkg.HandleRedisError(err)
	}

	hashedExceptID := ""
	if exceptSessionID != "" {
		hashedExceptID = hashID(exceptSessionID)
	}

	pipe := s.redisClient.Pipeline()

	for _, hID := range hashedIDs {
		if hID == hashedExceptID {
			continue
		}

		sessionKey := sessionPrefix + hID

		data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
		if err == nil {
			var sess domain.SessionEntity
			if err := json.Unmarshal(data, &sess); err == nil && sess.RefreshToken != "" {
				pipe.Del(ctx, "refresh:"+sess.RefreshToken)
			}
		}

		pipe.Del(ctx, sessionKey)
		pipe.SRem(ctx, userSetKey, hID)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return pkg.HandleRedisError(err)
	}

	return nil
}

func (s *sessionService) ExtendSession(ctx context.Context, sessionID, newHashedRefreshToken string, ttl time.Duration) error {
	hashedSessionID := hashID(sessionID)
	sessionKey := sessionPrefix + hashedSessionID

	data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
	if err != nil {
		domainErr := pkg.HandleRedisError(err)
		if errors.Is(domainErr, pkg.ErrRedisKeyNotFound) {
			return pkg.ErrSessionNotFound
		}
		return domainErr
	}

	var sess domain.SessionEntity
	if err := json.Unmarshal(data, &sess); err != nil {
		return pkg.ErrRedisUnmarshal
	}

	now := time.Now()
	sess.RefreshToken = newHashedRefreshToken
	sess.LastActiveAt = now
	sess.IsActive = true
	sess.ExpiresAt = now.Add(ttl)

	updated, err := json.Marshal(sess)
	if err != nil {
		return pkg.ErrRedisMarshal
	}

	userSetKey := userSessionsPrefix + sess.UserID

	pipe := s.redisClient.Pipeline()
	pipe.Set(ctx, sessionKey, updated, ttl)
	pipe.Expire(ctx, userSetKey, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return pkg.HandleRedisError(err)
	}
	return nil
}

func (s *sessionService) PurgeUserSessions(ctx context.Context, userID string) (int, error) {
	userSetKey := userSessionsPrefix + userID

	hashedIDs, err := s.redisClient.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return 0, pkg.HandleRedisError(err)
	}
	if len(hashedIDs) == 0 {
		_ = s.redisClient.Del(ctx, userSetKey).Err()
		return 0, nil
	}

	getPipe := s.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(hashedIDs))
	for i, hID := range hashedIDs {
		cmds[i] = getPipe.Get(ctx, sessionPrefix+hID)
	}
	_, _ = getPipe.Exec(ctx)

	delPipe := s.redisClient.Pipeline()
	for i, hID := range hashedIDs {
		delPipe.Del(ctx, sessionPrefix+hID)

		raw, err := cmds[i].Bytes()
		if err != nil {
			continue
		}
		var sess domain.SessionEntity
		if json.Unmarshal(raw, &sess) == nil && sess.RefreshToken != "" {
			delPipe.Del(ctx, "refresh:"+sess.RefreshToken)
		}
	}
	delPipe.Del(ctx, userSetKey)

	if _, err := delPipe.Exec(ctx); err != nil {
		return 0, pkg.HandleRedisError(err)
	}
	return len(hashedIDs), nil
}

func (s *sessionService) Heartbeat(ctx context.Context, sessionID string, state string) error {
	hashedSessionID := hashID(sessionID)
	sessionKey := sessionPrefix + hashedSessionID

	data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
	if err != nil {
		domainErr := pkg.HandleRedisError(err)
		if errors.Is(domainErr, pkg.ErrRedisKeyNotFound) {
			return pkg.ErrSessionNotFound
		}
		return domainErr
	}

	var sess domain.SessionEntity
	if err := json.Unmarshal(data, &sess); err != nil {
		return pkg.ErrRedisUnmarshal
	}

	isActive := (state == "active")
	sess.IsActive = isActive
	if isActive {
		sess.LastActiveAt = time.Now()
	}

	updatedData, err := json.Marshal(sess)
	if err != nil {
		return pkg.ErrRedisMarshal
	}

	pipe := s.redisClient.Pipeline()
	pipe.Set(ctx, sessionKey, updatedData, redis.KeepTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return pkg.HandleRedisError(err)
	}

	return nil
}
