package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redis   *redis.Client
	limit   int
	window  time.Duration
	prefix  string
}

func NewRateLimiter(redisClient *redis.Client, limit int, window time.Duration, prefix string) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		limit:  limit,
		window: window,
		prefix: prefix,
	}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		key := fmt.Sprintf("%s:%s", rl.prefix, ip)

		ctx := r.Context()
		allowed, remaining, resetTime, err := rl.isAllowed(ctx, key)
		if err != nil {
			// On Redis error, allow the request but log it
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", resetTime-time.Now().Unix()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Rate limit exceeded. Please try again later."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) isAllowed(ctx context.Context, key string) (allowed bool, remaining int, resetTime int64, err error) {
	now := time.Now()
	windowStart := now.Truncate(rl.window)
	windowEnd := windowStart.Add(rl.window)

	// Use a sliding window counter
	pipe := rl.redis.Pipeline()

	// Increment the counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiry on first request
	pipe.Expire(ctx, key, rl.window)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return true, rl.limit, windowEnd.Unix(), err
	}

	count := int(incrCmd.Val())
	remaining = rl.limit - count
	if remaining < 0 {
		remaining = 0
	}

	return count <= rl.limit, remaining, windowEnd.Unix(), nil
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for reverse proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ip, _, _ := net.SplitHostPort(xff)
		if ip == "" {
			ip = xff
		}
		return ip
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Stricter rate limiter for auth endpoints
func NewAuthRateLimiter(redisClient *redis.Client) *RateLimiter {
	return NewRateLimiter(redisClient, 5, time.Minute, "ratelimit:auth")
}

// General API rate limiter
func NewAPIRateLimiter(redisClient *redis.Client) *RateLimiter {
	return NewRateLimiter(redisClient, 100, time.Minute, "ratelimit:api")
}
