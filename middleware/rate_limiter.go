package middleware

import (
	"net/http"
	"sync"
	"time"

	"questionarie-service/utils"
)

type rateLimitEntry struct {
	count    int
	windowStart time.Time
}

// RateLimiter is a simple in-memory per-IP rate limiter.
type RateLimiter struct {
	mu          sync.Mutex
	entries     map[string]*rateLimitEntry
	maxRequests int
	window      time.Duration
}

// NewRateLimiter creates a new rate limiter with the given max requests per window.
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries:     make(map[string]*rateLimitEntry),
		maxRequests: maxRequests,
		window:      window,
	}

	// Start background cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes expired entries every 2 minutes.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.entries {
			if now.Sub(entry.windowStart) > rl.window {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Handler returns an HTTP middleware that enforces rate limiting per IP.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		// Use X-Forwarded-For if present (behind reverse proxy)
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		rl.mu.Lock()
		now := time.Now()
		entry, exists := rl.entries[ip]

		if !exists || now.Sub(entry.windowStart) > rl.window {
			// New window
			rl.entries[ip] = &rateLimitEntry{
				count:       1,
				windowStart: now,
			}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if entry.count >= rl.maxRequests {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", "60")
			utils.RespondWithError(w, http.StatusTooManyRequests, "rate limit exceeded, try again later")
			return
		}

		entry.count++
		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
