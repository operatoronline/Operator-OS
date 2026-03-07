package ratelimit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/standardws/operator/pkg/users"
)

// Middleware returns an HTTP middleware that enforces per-user rate limits.
// It extracts the user ID from the request context (set by AuthMiddleware)
// and checks the rate limiter. Rate limit headers are added to all responses.
//
// When rate limited, responds with 429 Too Many Requests and includes
// Retry-After header.
func Middleware(limiter *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := users.UserIDFromContext(r.Context())
			if userID == "" {
				// No authenticated user — skip rate limiting.
				next.ServeHTTP(w, r)
				return
			}

			err := limiter.Allow(userID)
			if err != nil {
				if err == ErrRateLimited {
					retryAfter, _ := limiter.RetryAfter(userID)
					retrySeconds := int(retryAfter.Seconds()) + 1
					if retrySeconds < 1 {
						retrySeconds = 1
					}

					// Set rate limit headers.
					setRateLimitHeaders(w, limiter, userID)
					w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":       "Rate limit exceeded",
						"code":        "rate_limited",
						"retry_after": retrySeconds,
					})
					return
				}
				// Unknown plan — let through but don't enforce.
				// This handles users who haven't been assigned a plan yet.
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers for successful requests.
			setRateLimitHeaders(w, limiter, userID)

			next.ServeHTTP(w, r)
		})
	}
}

// setRateLimitHeaders adds standard rate limit headers to the response.
func setRateLimitHeaders(w http.ResponseWriter, limiter *Limiter, userID string) {
	status, err := limiter.GetStatus(userID)
	if err != nil {
		return
	}

	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(status.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(status.Remaining))
	if status.DailyLimit > 0 {
		w.Header().Set("X-RateLimit-Daily-Limit", strconv.FormatInt(status.DailyLimit, 10))
		w.Header().Set("X-RateLimit-Daily-Remaining", strconv.FormatInt(status.DailyRemaining, 10))
	}
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(status.ResetAt.Unix(), 10))
}

// StatusHandler returns an HTTP handler that responds with the user's
// current rate limit status as JSON.
func StatusHandler(limiter *Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := users.UserIDFromContext(r.Context())
		if userID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Authentication required",
				"code":  "unauthorized",
			})
			return
		}

		status, err := limiter.GetStatus(userID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "No rate limit configured for user",
				"code":  "no_plan",
			})
			return
		}

		resp := StatusResponse{
			Tier:           string(status.Tier),
			Remaining:      status.Remaining,
			Limit:          status.Limit,
			DailyRemaining: status.DailyRemaining,
			DailyLimit:     status.DailyLimit,
			RetryAfterMs:   status.RetryAfter.Milliseconds(),
			ResetAt:        status.ResetAt.UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// StatusResponse is the JSON response for the rate limit status endpoint.
type StatusResponse struct {
	Tier           string `json:"tier"`
	Remaining      int    `json:"remaining"`
	Limit          int    `json:"limit"`
	DailyRemaining int64  `json:"daily_remaining"`
	DailyLimit     int64  `json:"daily_limit"`
	RetryAfterMs   int64  `json:"retry_after_ms"`
	ResetAt        string `json:"reset_at"`
}

// RegisterRoutes registers rate limit status endpoints on the given ServeMux.
// Routes are expected to be behind AuthMiddleware.
func RegisterRoutes(mux *http.ServeMux, limiter *Limiter) {
	mux.HandleFunc("GET /api/v1/rate-limit/status", StatusHandler(limiter))
}

// PersistMiddleware wraps the limiter to periodically save state.
// It returns a function that should be deferred to save on shutdown.
func PersistMiddleware(limiter *Limiter, interval time.Duration) (stop func()) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	var once sync.Once

	go func() {
		for {
			select {
			case <-ticker.C:
				_ = limiter.Persist()
			case <-done:
				ticker.Stop()
				_ = limiter.Persist() // final save
				return
			}
		}
	}()

	return func() {
		once.Do(func() { close(done) })
	}
}

// FormatRetryAfter returns a human-readable retry-after string.
func FormatRetryAfter(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%d seconds", s)
	}
	m := s / 60
	if m < 60 {
		return fmt.Sprintf("%d minutes", m)
	}
	h := m / 60
	return fmt.Sprintf("%d hours", h)
}
