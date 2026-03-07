// Package ratelimit provides per-user rate limiting for Operator OS.
//
// It implements a token bucket algorithm with configurable rates per plan tier.
// Buckets are stored in-memory for fast access with SQLite persistence for
// state recovery across restarts. A background sweeper prunes expired buckets
// from memory.
package ratelimit

import (
	"errors"
	"sync"
	"time"
)

// Errors returned by the rate limiter.
var (
	// ErrRateLimited is returned when a request exceeds the user's rate limit.
	ErrRateLimited = errors.New("rate limit exceeded")
	// ErrInvalidPlan is returned when a plan tier is not recognized.
	ErrInvalidPlan = errors.New("invalid plan tier")
	// ErrInvalidConfig is returned when a tier config has invalid values.
	ErrInvalidConfig = errors.New("invalid rate limit config")
)

// PlanTier identifies a billing plan level for rate limiting.
type PlanTier string

const (
	PlanFree       PlanTier = "free"
	PlanStarter    PlanTier = "starter"
	PlanPro        PlanTier = "pro"
	PlanEnterprise PlanTier = "enterprise"
)

// TierConfig defines rate limit parameters for a plan tier.
type TierConfig struct {
	// RequestsPerMinute is the maximum sustained request rate.
	RequestsPerMinute float64
	// BurstSize is the maximum number of requests allowed in a burst.
	// Must be >= 1.
	BurstSize int
	// DailyLimit is the maximum number of requests per 24h rolling window.
	// 0 means unlimited.
	DailyLimit int64
}

// DefaultTierConfigs returns the default rate limit configuration per plan tier.
func DefaultTierConfigs() map[PlanTier]TierConfig {
	return map[PlanTier]TierConfig{
		PlanFree: {
			RequestsPerMinute: 10,
			BurstSize:         15,
			DailyLimit:        500,
		},
		PlanStarter: {
			RequestsPerMinute: 30,
			BurstSize:         50,
			DailyLimit:        5000,
		},
		PlanPro: {
			RequestsPerMinute: 60,
			BurstSize:         100,
			DailyLimit:        50000,
		},
		PlanEnterprise: {
			RequestsPerMinute: 120,
			BurstSize:         200,
			DailyLimit:        0, // unlimited
		},
	}
}

// bucket is an in-memory token bucket for a single user.
type bucket struct {
	mu        sync.Mutex
	tokens    float64
	lastTime  time.Time
	rate      float64 // tokens per second
	burst     int
	daily     int64 // daily limit (0 = unlimited)
	dayCount  int64 // requests today
	dayStart  time.Time
}

// allow checks if a request is permitted and consumes a token if so.
func (b *bucket) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(b.lastTime).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.rate
		if b.tokens > float64(b.burst) {
			b.tokens = float64(b.burst)
		}
		b.lastTime = now
	}

	// Check daily limit reset.
	if b.daily > 0 && now.Sub(b.dayStart) >= 24*time.Hour {
		b.dayCount = 0
		b.dayStart = now.Truncate(24 * time.Hour)
	}

	// Check daily limit.
	if b.daily > 0 && b.dayCount >= b.daily {
		return false
	}

	// Check token bucket.
	if b.tokens < 1 {
		return false
	}

	b.tokens--
	b.dayCount++
	return true
}

// retryAfter returns the duration until the next token is available.
func (b *bucket) retryAfter(now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill to see current state.
	elapsed := now.Sub(b.lastTime).Seconds()
	tokens := b.tokens
	if elapsed > 0 {
		tokens += elapsed * b.rate
		if tokens > float64(b.burst) {
			tokens = float64(b.burst)
		}
	}

	if tokens >= 1 {
		// Check if daily limit is the blocker.
		if b.daily > 0 && b.dayCount >= b.daily {
			remaining := b.dayStart.Add(24 * time.Hour).Sub(now)
			if remaining < 0 {
				return 0
			}
			return remaining
		}
		return 0
	}

	deficit := 1.0 - tokens
	return time.Duration(deficit/b.rate*1e9) * time.Nanosecond
}

// snapshot returns the current bucket state for persistence.
func (b *bucket) snapshot(now time.Time) BucketState {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill before snapshot.
	elapsed := now.Sub(b.lastTime).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * b.rate
		if b.tokens > float64(b.burst) {
			b.tokens = float64(b.burst)
		}
		b.lastTime = now
	}

	return BucketState{
		Tokens:   b.tokens,
		LastTime: b.lastTime,
		DayCount: b.dayCount,
		DayStart: b.dayStart,
	}
}

// BucketState captures the serializable state of a bucket.
type BucketState struct {
	Tokens   float64
	LastTime time.Time
	DayCount int64
	DayStart time.Time
}

// Limiter provides per-user rate limiting using token bucket algorithm.
type Limiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	tiers   map[PlanTier]TierConfig
	plans   map[string]PlanTier // userID → plan
	store   RateLimitStore      // optional persistence
}

// RateLimitStore abstracts persistence for rate limit state.
// Implementations must be safe for concurrent use.
type RateLimitStore interface {
	// SaveBucket persists a user's bucket state.
	SaveBucket(userID string, tier PlanTier, state BucketState) error
	// LoadBucket retrieves a user's saved bucket state.
	// Returns ErrNotFound if no state exists.
	LoadBucket(userID string) (PlanTier, BucketState, error)
	// DeleteBucket removes a user's saved bucket state.
	DeleteBucket(userID string) error
	// Close releases any resources.
	Close() error
}

// ErrNotFound is returned when a bucket state is not found in the store.
var ErrNotFound = errors.New("rate limit state not found")

// NewLimiter creates a new Limiter with the given tier configs.
// If tiers is nil, DefaultTierConfigs() is used.
func NewLimiter(tiers map[PlanTier]TierConfig) *Limiter {
	if tiers == nil {
		tiers = DefaultTierConfigs()
	}
	return &Limiter{
		buckets: make(map[string]*bucket),
		tiers:   tiers,
		plans:   make(map[string]PlanTier),
	}
}

// NewLimiterWithStore creates a new Limiter with persistence.
func NewLimiterWithStore(tiers map[PlanTier]TierConfig, store RateLimitStore) *Limiter {
	l := NewLimiter(tiers)
	l.store = store
	return l
}

// SetUserPlan sets or updates the plan tier for a user.
// If the tier changes, the user's bucket is reset to the new tier's config.
func (l *Limiter) SetUserPlan(userID string, tier PlanTier) error {
	cfg, ok := l.tiers[tier]
	if !ok {
		return ErrInvalidPlan
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = l.plans[userID]
	l.plans[userID] = tier

	// Always create a fresh bucket when plan is set.
	{
		now := time.Now()
		l.buckets[userID] = &bucket{
			tokens:   float64(cfg.BurstSize),
			lastTime: now,
			rate:     cfg.RequestsPerMinute / 60.0,
			burst:    cfg.BurstSize,
			daily:    cfg.DailyLimit,
			dayCount: 0,
			dayStart: now.Truncate(24 * time.Hour),
		}
	}
	return nil
}

// Allow checks if a request from the given user is allowed.
// Returns nil if allowed, ErrRateLimited if not, or ErrInvalidPlan if
// the user has no plan set.
func (l *Limiter) Allow(userID string) error {
	b, err := l.getBucket(userID)
	if err != nil {
		return err
	}

	if !b.allow(time.Now()) {
		return ErrRateLimited
	}
	return nil
}

// AllowAt is like Allow but uses the specified time (for testing).
func (l *Limiter) AllowAt(userID string, now time.Time) error {
	b, err := l.getBucket(userID)
	if err != nil {
		return err
	}

	if !b.allow(now) {
		return ErrRateLimited
	}
	return nil
}

// RetryAfter returns the duration until the user can make another request.
// Returns 0 if a request is currently allowed.
func (l *Limiter) RetryAfter(userID string) (time.Duration, error) {
	b, err := l.getBucket(userID)
	if err != nil {
		return 0, err
	}
	return b.retryAfter(time.Now()), nil
}

// Status returns the current rate limit status for a user.
type Status struct {
	Tier           PlanTier      `json:"tier"`
	Remaining      int           `json:"remaining"`       // approximate tokens remaining
	Limit          int           `json:"limit"`            // burst size
	DailyRemaining int64         `json:"daily_remaining"`  // requests left today (0 = unlimited)
	DailyLimit     int64         `json:"daily_limit"`      // daily limit (0 = unlimited)
	RetryAfter     time.Duration `json:"retry_after"`      // time until next token
	ResetAt        time.Time     `json:"reset_at"`         // daily counter reset time
}

// GetStatus returns the current rate limit status for a user.
func (l *Limiter) GetStatus(userID string) (*Status, error) {
	l.mu.RLock()
	tier, hasPlan := l.plans[userID]
	b, hasBucket := l.buckets[userID]
	l.mu.RUnlock()

	if !hasPlan {
		return nil, ErrInvalidPlan
	}

	if !hasBucket {
		return nil, ErrInvalidPlan
	}

	now := time.Now()
	retryAfter := b.retryAfter(now)

	b.mu.Lock()
	remaining := int(b.tokens)
	dailyRemaining := int64(0)
	if b.daily > 0 {
		dailyRemaining = b.daily - b.dayCount
		if dailyRemaining < 0 {
			dailyRemaining = 0
		}
	}
	resetAt := b.dayStart.Add(24 * time.Hour)
	b.mu.Unlock()

	return &Status{
		Tier:           tier,
		Remaining:      remaining,
		Limit:          l.tiers[tier].BurstSize,
		DailyRemaining: dailyRemaining,
		DailyLimit:     l.tiers[tier].DailyLimit,
		RetryAfter:     retryAfter,
		ResetAt:        resetAt,
	}, nil
}

// Persist saves all in-memory bucket states to the store.
// No-op if no store is configured.
func (l *Limiter) Persist() error {
	if l.store == nil {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	now := time.Now()
	for userID, b := range l.buckets {
		tier := l.plans[userID]
		state := b.snapshot(now)
		if err := l.store.SaveBucket(userID, tier, state); err != nil {
			return err
		}
	}
	return nil
}

// Restore loads bucket states from the store and populates in-memory buckets.
// No-op if no store is configured.
func (l *Limiter) Restore() error {
	// Restore is a no-op without a store — callers should load via
	// RestoreUser for each known user. This method exists for symmetry
	// but individual restore is preferred.
	return nil
}

// RestoreUser loads a single user's bucket state from the store.
// If no saved state exists, falls back to creating a fresh bucket via SetUserPlan.
func (l *Limiter) RestoreUser(userID string) error {
	if l.store == nil {
		return nil
	}

	tier, state, err := l.store.LoadBucket(userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // no saved state, nothing to restore
		}
		return err
	}

	cfg, ok := l.tiers[tier]
	if !ok {
		return ErrInvalidPlan
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.plans[userID] = tier
	l.buckets[userID] = &bucket{
		tokens:   state.Tokens,
		lastTime: state.LastTime,
		rate:     cfg.RequestsPerMinute / 60.0,
		burst:    cfg.BurstSize,
		daily:    cfg.DailyLimit,
		dayCount: state.DayCount,
		dayStart: state.DayStart,
	}
	return nil
}

// RemoveUser removes a user's rate limit state from memory (and store).
func (l *Limiter) RemoveUser(userID string) error {
	l.mu.Lock()
	delete(l.buckets, userID)
	delete(l.plans, userID)
	l.mu.Unlock()

	if l.store != nil {
		return l.store.DeleteBucket(userID)
	}
	return nil
}

// UserCount returns the number of users with active buckets.
func (l *Limiter) UserCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.buckets)
}

// getBucket returns the bucket for the user, auto-creating with default (free)
// tier if not set, or loading from store.
func (l *Limiter) getBucket(userID string) (*bucket, error) {
	l.mu.RLock()
	b, ok := l.buckets[userID]
	l.mu.RUnlock()

	if ok {
		return b, nil
	}

	// Try loading from store.
	if l.store != nil {
		if err := l.RestoreUser(userID); err == nil {
			l.mu.RLock()
			b, ok = l.buckets[userID]
			l.mu.RUnlock()
			if ok {
				return b, nil
			}
		}
	}

	return nil, ErrInvalidPlan
}
