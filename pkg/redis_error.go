package pkg

import (
	"errors"
	"fmt"
	"net"

	"github.com/redis/go-redis/v9"
)

// Domain errors
var (
	ErrRedisKeyNotFound      = errors.New("key not found")
	ErrRedisKeyExpired       = errors.New("key has expired")
	ErrRedisCollision        = errors.New("key collision detected, please retry")
	ErrRedisConnectionFailed = errors.New("redis connection failed")
	ErrRedisTimeout          = errors.New("redis operation timed out")
	ErrRedisLockNotAcquired  = errors.New("resource is locked, please retry")
	ErrRedisMarshal          = errors.New("failed to serialize data")
	ErrRedisUnmarshal        = errors.New("failed to deserialize data")
	ErrRedisInternal         = errors.New("internal redis error")
)

// HandleRedisError map raw Redis error to domain error to prevent it from leaking out.
func HandleRedisError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, redis.Nil) {
		return ErrRedisKeyNotFound
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %s", ErrRedisConnectionFailed, netErr.Op)
	}

	if isTimeoutError(err) {
		return ErrRedisTimeout
	}

	return fmt.Errorf("%w: %s", ErrRedisInternal, err.Error())
}

// HandleRedisSetNX handle response SetNX (lock/deduplication).
func HandleRedisSetNX(success bool, err error, isLock bool) error {
	if err != nil {
		return HandleRedisError(err)
	}
	if !success {
		if isLock {
			return ErrRedisLockNotAcquired
		}
		return ErrRedisCollision
	}
	return nil
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "timeout") || contains(msg, "deadline exceeded") || contains(msg, "context canceled")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}