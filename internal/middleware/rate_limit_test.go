package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contextFor := func(remoteAddr, userAgent, userID string) *gin.Context {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		ctx.Request.RemoteAddr = remoteAddr
		ctx.Request.Header.Set("User-Agent", userAgent)
		if userID != "" {
			ctx.Set("userID", userID)
		}
		return ctx
	}

	first := rateLimitIdentity(contextFor("192.0.2.1:1234", "Example   Agent", ""), IdentityIPUserAgent)
	normalized := rateLimitIdentity(contextFor("192.0.2.1:9999", "Example Agent", ""), IdentityIPUserAgent)
	if first != normalized {
		t.Fatal("equivalent IP/User-Agent identities must produce the same key")
	}
	differentAgent := rateLimitIdentity(contextFor("192.0.2.1:1234", "Other Agent", ""), IdentityIPUserAgent)
	if first == differentAgent {
		t.Fatal("different User-Agents must produce different public identities")
	}
	ipOnlyA := rateLimitIdentity(contextFor("192.0.2.1:1234", "Agent A", ""), IdentityIP)
	ipOnlyB := rateLimitIdentity(contextFor("192.0.2.1:9999", "Agent B", ""), IdentityIP)
	if ipOnlyA != ipOnlyB {
		t.Fatal("IP-only identity must not vary with port or User-Agent")
	}

	userA := rateLimitIdentity(contextFor("192.0.2.1:1234", "Agent A", "user-1"), IdentityUser)
	userB := rateLimitIdentity(contextFor("198.51.100.2:9999", "Agent B", "user-1"), IdentityUser)
	if userA != userB {
		t.Fatal("authenticated identity must be stable across IP and User-Agent changes")
	}
}

func TestValidateRateLimitPolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  RateLimitPolicy
		wantErr bool
	}{
		{"sliding", SlidingPolicy("public.create", 5, 72*time.Hour, IdentityIPUserAgent, false), false},
		{"bucket", BucketPolicy("auth.login", 10, 1.0/60.0, IdentityIPUserAgent, false), false},
		{"empty name", BucketPolicy("", 10, 1, IdentityIPUserAgent, false), true},
		{"unsafe name", BucketPolicy("auth:login", 10, 1, IdentityIPUserAgent, false), true},
		{"zero limit", BucketPolicy("auth.login", 0, 1, IdentityIPUserAgent, false), true},
		{"zero window", SlidingPolicy("public.create", 5, 0, IdentityIPUserAgent, false), true},
		{"zero refill", BucketPolicy("auth.login", 10, 0, IdentityIPUserAgent, false), true},
		{"invalid identity", BucketPolicy("auth.login", 10, 1, RateLimitIdentity("invalid"), false), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotErr := validateRateLimitPolicy(tt.policy) != nil; gotErr != tt.wantErr {
				t.Fatalf("validateRateLimitPolicy() error = %v, want error %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestRateLimiterRedisFailureModes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisClient := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
		MaxRetries:   -1,
	})
	t.Cleanup(func() { _ = redisClient.Close() })
	limiter := NewRateLimiter(redisClient, nil)

	t.Run("fail closed", func(t *testing.T) {
		router := gin.New()
		router.GET("/", limiter.Limit(BucketPolicy("test.closed", 1, 1, IdentityIPUserAgent, false)), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("fail open", func(t *testing.T) {
		router := gin.New()
		router.GET("/", limiter.Limit(BucketPolicy("test.open", 1, 1, IdentityIPUserAgent, true)), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
	})
}

func TestSetRateLimitHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	reset := time.Unix(1_700_000_000, 0)
	setRateLimitHeaders(ctx, BucketPolicy("test", 10, 1, IdentityIPUserAgent, false), rateLimitDecision{
		Remaining: 0, ResetAt: reset, RetryAfter: 1500 * time.Millisecond,
	})
	if got := recorder.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("Retry-After = %q, want %q", got, "2")
	}
	if got := recorder.Header().Get("X-RateLimit-Reset"); got != "1700000000" {
		t.Fatalf("X-RateLimit-Reset = %q", got)
	}
}
