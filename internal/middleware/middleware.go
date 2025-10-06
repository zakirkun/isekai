package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/zakirkun/isekai/pkg/logger"
	"github.com/zakirkun/isekai/pkg/response"
)

// Logger middleware logs incoming requests
func Logger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			log.Infof("%s %s - %d (%v) - %s",
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
				r.RemoteAddr,
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORS middleware adds CORS headers
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Recovery middleware recovers from panics
func Recovery(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("Panic recovered: %v", err)
					response.InternalServerError(w, "Internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter represents a simple rate limiter
type RateLimiter struct {
	mu          sync.Mutex
	requests    map[string][]time.Time
	limit       int
	window      time.Duration
	cleanupTick *time.Ticker
	log         *logger.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int, log *logger.Logger) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		limit:       requestsPerSecond,
		window:      time.Second,
		cleanupTick: time.NewTicker(time.Minute),
		log:         log,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old entries from the rate limiter
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTick.C {
		rl.mu.Lock()
		now := time.Now()
		for key, times := range rl.requests {
			// Remove entries older than the window
			cutoff := now.Add(-rl.window)
			valid := make([]time.Time, 0)
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get requests for this client
	times, exists := rl.requests[clientIP]
	if !exists {
		rl.requests[clientIP] = []time.Time{now}
		return true
	}

	// Filter out old requests
	valid := make([]time.Time, 0)
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	// Check if limit exceeded
	if len(valid) >= rl.limit {
		rl.requests[clientIP] = valid
		return false
	}

	// Add new request
	valid = append(valid, now)
	rl.requests[clientIP] = valid
	return true
}

// Stop stops the rate limiter cleanup
func (rl *RateLimiter) Stop() {
	rl.cleanupTick.Stop()
}

// RateLimit middleware limits requests per client
func RateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr

			if !rl.Allow(clientIP) {
				response.Error(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Timeout middleware adds a timeout to requests
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r)
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				response.Error(w, http.StatusGatewayTimeout, "Request timeout")
				return
			}
		})
	}
}
