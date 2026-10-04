package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type (
	RateLimitAlgorithm string
	RateLimitIdentity  string
)

const (
	SlidingWindow RateLimitAlgorithm = "sliding-window"
	TokenBucket   RateLimitAlgorithm = "token-bucket"
)

const (
	IdentityIP          RateLimitIdentity = "ip"
	IdentityIPUserAgent RateLimitIdentity = "ip-user-agent"
	IdentityUser        RateLimitIdentity = "user"
)

type RateLimitPolicy struct {
	Name       string
	Algorithm  RateLimitAlgorithm
	Identity   RateLimitIdentity
	Limit      int
	Window     time.Duration
	RefillRate float64
	FailOpen   bool
}

type rateLimitDecision struct {
	Allowed    bool
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

type RateLimiter struct {
	redis  *redis.Client
	logger *slog.Logger
}

func NewRateLimiter(redisClient *redis.Client, logger *slog.Logger) *RateLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	return &RateLimiter{redis: redisClient, logger: logger}
}

func SlidingPolicy(name string, limit int, window time.Duration, identity RateLimitIdentity, failOpen bool) RateLimitPolicy {
	return RateLimitPolicy{Name: name, Algorithm: SlidingWindow, Identity: identity, Limit: limit, Window: window, FailOpen: failOpen}
}

func BucketPolicy(name string, capacity int, refillPerSecond float64, identity RateLimitIdentity, failOpen bool) RateLimitPolicy {
	return RateLimitPolicy{Name: name, Algorithm: TokenBucket, Identity: identity, Limit: capacity, RefillRate: refillPerSecond, FailOpen: failOpen}
}

func (m *RateLimiter) Limit(policy RateLimitPolicy) gin.HandlerFunc {
	if err := validateRateLimitPolicy(policy); err != nil {
		panic(err)
	}
	return func(c *gin.Context) {
		identity := rateLimitIdentity(c, policy.Identity)
		key := "ratelimit:v1:" + policy.Name + ":" + identity

		decision, err := m.evaluate(c.Request.Context(), key, policy)
		if err != nil {
			m.logger.Error("rate limiter evaluation failed", slog.String("policy", policy.Name), slog.Any("error", err))
			c.Header("X-RateLimit-Status", "degraded")
			if policy.FailOpen {
				c.Header("X-RateLimit-Policy", policy.Name)
				c.Next()
				return
			}
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "rate limiter temporarily unavailable",
				"error":   "rate limiter temporarily unavailable",
				"code":    "RATE_LIMIT_UNAVAILABLE",
			})
			return
		}

		setRateLimitHeaders(c, policy, decision)
		if !decision.Allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success":       false,
				"message":       "request limit reached; please try again later",
				"error":         "request limit reached; please try again later",
				"code":          "RATE_LIMIT_EXCEEDED",
				"requests_left": decision.Remaining,
				"reset_at":      decision.ResetAt.UTC().Format(time.RFC3339),
			})
			return
		}
		c.Next()
	}
}

func validateRateLimitPolicy(policy RateLimitPolicy) error {
	if policy.Name == "" || strings.ContainsAny(policy.Name, " \t\r\n:") {
		return fmt.Errorf("rate limiter: invalid policy name %q", policy.Name)
	}
	if policy.Limit <= 0 {
		return fmt.Errorf("rate limiter: policy %q limit must be positive", policy.Name)
	}
	if policy.Identity != IdentityIP && policy.Identity != IdentityIPUserAgent && policy.Identity != IdentityUser {
		return fmt.Errorf("rate limiter: policy %q has unsupported identity", policy.Name)
	}
	switch policy.Algorithm {
	case SlidingWindow:
		if policy.Window <= 0 {
			return fmt.Errorf("rate limiter: policy %q window must be positive", policy.Name)
		}
	case TokenBucket:
		if policy.RefillRate <= 0 {
			return fmt.Errorf("rate limiter: policy %q refill rate must be positive", policy.Name)
		}
	default:
		return fmt.Errorf("rate limiter: policy %q has unsupported algorithm", policy.Name)
	}
	return nil
}

func (m *RateLimiter) evaluate(ctx context.Context, key string, policy RateLimitPolicy) (rateLimitDecision, error) {
	if policy.Algorithm == SlidingWindow {
		return m.slidingWindow(ctx, key, policy)
	}
	return m.tokenBucket(ctx, key, policy)
}

func (m *RateLimiter) slidingWindow(ctx context.Context, key string, policy RateLimitPolicy) (rateLimitDecision, error) {
	result, err := slidingWindowScript.Run(ctx, m.redis, []string{key},
		policy.Window.Milliseconds(), policy.Limit, uuid.NewString()).Slice()
	if err != nil {
		return rateLimitDecision{}, err
	}
	if len(result) != 4 {
		return rateLimitDecision{}, fmt.Errorf("rate limiter: unexpected sliding-window result")
	}
	allowed, err := redisInt64(result[0])
	if err != nil {
		return rateLimitDecision{}, err
	}
	remaining, err := redisInt64(result[1])
	if err != nil {
		return rateLimitDecision{}, err
	}
	resetMS, err := redisInt64(result[2])
	if err != nil {
		return rateLimitDecision{}, err
	}
	retryMS, err := redisInt64(result[3])
	if err != nil {
		return rateLimitDecision{}, err
	}
	return rateLimitDecision{
		Allowed: allowed == 1, Remaining: int(remaining),
		ResetAt: time.UnixMilli(resetMS), RetryAfter: time.Duration(retryMS) * time.Millisecond,
	}, nil
}

func (m *RateLimiter) tokenBucket(ctx context.Context, key string, policy RateLimitPolicy) (rateLimitDecision, error) {
	recovery := time.Duration(math.Ceil(float64(policy.Limit)/policy.RefillRate*2)) * time.Second
	if recovery < time.Minute {
		recovery = time.Minute
	}
	result, err := tokenBucketScript.Run(ctx, m.redis, []string{key},
		policy.Limit, strconv.FormatFloat(policy.RefillRate, 'f', 9, 64), recovery.Milliseconds()).Slice()
	if err != nil {
		return rateLimitDecision{}, err
	}
	if len(result) != 4 {
		return rateLimitDecision{}, fmt.Errorf("rate limiter: unexpected token-bucket result")
	}
	allowed, err := redisInt64(result[0])
	if err != nil {
		return rateLimitDecision{}, err
	}
	remaining, err := redisInt64(result[1])
	if err != nil {
		return rateLimitDecision{}, err
	}
	resetMS, err := redisInt64(result[2])
	if err != nil {
		return rateLimitDecision{}, err
	}
	retryMS, err := redisInt64(result[3])
	if err != nil {
		return rateLimitDecision{}, err
	}
	return rateLimitDecision{
		Allowed: allowed == 1, Remaining: int(remaining),
		ResetAt: time.UnixMilli(resetMS), RetryAfter: time.Duration(retryMS) * time.Millisecond,
	}, nil
}

func rateLimitIdentity(c *gin.Context, mode RateLimitIdentity) string {
	var raw string
	if mode == IdentityUser {
		if userID := strings.TrimSpace(c.GetString("userID")); userID != "" {
			raw = "user\x00" + userID
		}
	}
	if raw == "" {
		ip := strings.TrimSpace(c.ClientIP())
		if parsed := net.ParseIP(ip); parsed != nil {
			ip = parsed.String()
		}
		if mode == IdentityIPUserAgent {
			ua := strings.Join(strings.Fields(c.GetHeader("User-Agent")), " ")
			raw = "client\x00" + ip + "\x00" + ua
		} else {
			raw = "ip\x00" + ip
		}
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func setRateLimitHeaders(c *gin.Context, policy RateLimitPolicy, decision rateLimitDecision) {
	c.Header("X-RateLimit-Policy", policy.Name)
	c.Header("X-RateLimit-Limit", strconv.Itoa(policy.Limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(max(decision.Remaining, 0)))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(decision.ResetAt.Unix(), 10))
	if !decision.Allowed {
		retrySeconds := int64(math.Ceil(decision.RetryAfter.Seconds()))
		if retrySeconds < 1 {
			retrySeconds = 1
		}
		c.Header("Retry-After", strconv.FormatInt(retrySeconds, 10))
	}
}

func redisInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("rate limiter: unexpected Redis value %T", value)
	}
}

var slidingWindowScript = redis.NewScript(`
local now_parts = redis.call('TIME')
local now = (tonumber(now_parts[1]) * 1000) + math.floor(tonumber(now_parts[2]) / 1000)
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - window)
local count = redis.call('ZCARD', KEYS[1])
local allowed = 0
if count < limit then
  redis.call('ZADD', KEYS[1], now, ARGV[3])
  count = count + 1
  allowed = 1
end
redis.call('PEXPIRE', KEYS[1], window)
local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
local reset_at = now + window
if #oldest == 2 then reset_at = tonumber(oldest[2]) + window end
local retry = 0
if allowed == 0 then retry = math.max(reset_at - now, 1) end
return {allowed, math.max(limit - count, 0), reset_at, retry}
`)

var tokenBucketScript = redis.NewScript(`
local now_parts = redis.call('TIME')
local now = (tonumber(now_parts[1]) * 1000) + math.floor(tonumber(now_parts[2]) / 1000)
local capacity = tonumber(ARGV[1])
local refill_per_ms = tonumber(ARGV[2]) / 1000
local state = redis.call('HMGET', KEYS[1], 'tokens', 'updated_at')
local tokens = tonumber(state[1]) or capacity
local updated_at = tonumber(state[2]) or now
tokens = math.min(capacity, tokens + math.max(now - updated_at, 0) * refill_per_ms)
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'updated_at', now)
redis.call('PEXPIRE', KEYS[1], tonumber(ARGV[3]))
local retry = 0
if allowed == 0 then retry = math.ceil((1 - tokens) / refill_per_ms) end
local reset_at = now + math.ceil((capacity - tokens) / refill_per_ms)
return {allowed, math.floor(tokens), reset_at, retry}
`)
