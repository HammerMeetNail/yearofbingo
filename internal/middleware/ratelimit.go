package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HammerMeetNail/yearofbingo/internal/logging"
)

type RateLimiter struct {
	redis  *redis.Client
	limit  int64
	window time.Duration
	prefix string
	keyFn  func(r *http.Request) string
}

func NewRateLimiter(redis *redis.Client, limit int64, window time.Duration, prefix string, keyFn func(r *http.Request) string) *RateLimiter {
	return &RateLimiter{
		redis:  redis,
		limit:  limit,
		window: window,
		prefix: prefix,
		keyFn:  keyFn,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rl.redis == nil {
			next.ServeHTTP(w, r)
			return
		}

		keySuffix := rl.keyFn(r)
		if keySuffix == "" {
			// Fallback to IP if key function returns empty string
			keySuffix = GetClientIP(r)
		}

		key := fmt.Sprintf("%s%s", rl.prefix, keySuffix)
		ctx := r.Context()

		count, err := rl.redis.Incr(ctx, key).Result()
		if err != nil {
			logging.Error("Rate limit Redis error", map[string]interface{}{"error": err.Error()})
			next.ServeHTTP(w, r) // Fail open
			return
		}

		if count == 1 {
			rl.redis.Expire(ctx, key, rl.window)
		}

		if count > rl.limit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Rate limit exceeded"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetClientIP extracts the client IP from the request, respecting X-Forwarded-For
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by Cloudflare/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs; the first one is the client
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
