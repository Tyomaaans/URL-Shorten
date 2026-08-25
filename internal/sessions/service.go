package sessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"url-shorten/internal/domains"
	"url-shorten/pkg"

	"github.com/redis/go-redis/v9"
)

// Interface

type SessionService interface {
	CreateSession(ctx context.Context, session *domains.SessionEntity, ttl time.Duration) error
	IsSessionActive(ctx context.Context, sessionID string) (bool, error)
	GetActiveSessions(ctx context.Context, userID string) ([]domains.SessionEntity, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	RevokeAllOtherSessions(ctx context.Context, userID string, exceptSessionID string) error
	RevokeAllSessions(ctx context.Context, userID string) error
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

func (s *sessionService) CreateSession(ctx context.Context, session *domains.SessionEntity, ttl time.Duration) error {
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
	hashedSessionID := hashID(sessionID)
	sessionKey := sessionPrefix + hashedSessionID

	exists, err := s.redisClient.Exists(ctx, sessionKey).Result()
	if err != nil {
		return false, pkg.HandleRedisError(err)
	}
	return exists > 0, nil
}

func (s *sessionService) GetActiveSessions(ctx context.Context, userID string) ([]domains.SessionEntity, error) {
	userSetKey := userSessionsPrefix + userID

	hashedIDs, err := s.redisClient.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return nil, pkg.HandleRedisError(err)
	}

	var activeSessions []domains.SessionEntity
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

		var sess domains.SessionEntity
		if err := json.Unmarshal(data, &sess); err == nil {
			activeSessions = append(activeSessions, sess)
		}
	}

	if len(expiredHashedIDs) > 0 {
		s.redisClient.SRem(ctx, userSetKey, expiredHashedIDs...)
	}

	return activeSessions, nil
}

func (s *sessionService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	hashedSessionID := hashID(sessionID)
    sessionKey      := sessionPrefix + hashedSessionID
    userSetKey      := userSessionsPrefix + userID

	data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
    if err != nil {
        domainErr := pkg.HandleRedisError(err)
        if errors.Is(domainErr, pkg.ErrRedisKeyNotFound) {
            return nil // Session sudah tidak ada / expired
        }
        return domainErr
    }

    var sess domains.SessionEntity
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
            var sess domains.SessionEntity
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

func (s *sessionService) RevokeAllSessions(ctx context.Context, userID string) error {
    userSetKey := userSessionsPrefix + userID

    hashedIDs, err := s.redisClient.SMembers(ctx, userSetKey).Result()
    if err != nil {
        return pkg.HandleRedisError(err)
    }

    pipe := s.redisClient.Pipeline()

    for _, hID := range hashedIDs {
        sessionKey := sessionPrefix + hID

        data, err := s.redisClient.Get(ctx, sessionKey).Bytes()
        if err == nil {
            var sess domains.SessionEntity
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