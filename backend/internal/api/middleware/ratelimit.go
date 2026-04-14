package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// visitor tracks the rate limiter and last-seen time for a single client.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting for HTTP handlers.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      rate.Limit // requests per second
	burst    int
}

// NewRateLimiter creates a rate limiter that allows rps requests per second
// with the given burst size per client IP.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go rl.cleanup()
	return rl
}

// getVisitor returns the rate limiter for the given key, creating one if needed.
func (rl *RateLimiter) getVisitor(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists {
		limiter := rate.NewLimiter(rl.rps, rl.burst)
		rl.visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup removes stale visitors every minute.
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for key, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that rate-limits by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		limiter := rl.getVisitor(ip)

		if !limiter.Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"type":  "error",
				"error": map[string]string{"type": "rate_limit_error", "message": "too many requests, please retry later"},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP returns the client IP string from the request for use as a rate-limit key.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (leftmost) IP, which is the original client
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
