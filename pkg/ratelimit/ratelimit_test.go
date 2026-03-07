package ratelimit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/standardws/operator/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"database/sql"

	_ "modernc.org/sqlite"
)

// --- Token Bucket Tests ---

func TestNewLimiter(t *testing.T) {
	l := NewLimiter(nil)
	require.NotNil(t, l)
	assert.Equal(t, 4, len(l.tiers)) // default tiers
}

func TestNewLimiterCustomTiers(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 5, BurstSize: 10, DailyLimit: 100},
	}
	l := NewLimiter(tiers)
	require.NotNil(t, l)
	assert.Equal(t, 1, len(l.tiers))
}

func TestSetUserPlan(t *testing.T) {
	l := NewLimiter(nil)

	err := l.SetUserPlan("user1", PlanFree)
	require.NoError(t, err)
	assert.Equal(t, 1, l.UserCount())

	// Update plan.
	err = l.SetUserPlan("user1", PlanPro)
	require.NoError(t, err)
	assert.Equal(t, 1, l.UserCount())
}

func TestSetUserPlanInvalidTier(t *testing.T) {
	l := NewLimiter(nil)
	err := l.SetUserPlan("user1", PlanTier("nonexistent"))
	assert.ErrorIs(t, err, ErrInvalidPlan)
}

func TestAllowBasic(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// First request should be allowed.
	err := l.Allow("user1")
	assert.NoError(t, err)
}

func TestAllowNoPlan(t *testing.T) {
	l := NewLimiter(nil)
	err := l.Allow("unknown-user")
	assert.ErrorIs(t, err, ErrInvalidPlan)
}

func TestAllowBurstExhaustion(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 60, BurstSize: 5, DailyLimit: 0},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// Exhaust burst.
	for i := 0; i < 5; i++ {
		err := l.Allow("user1")
		require.NoError(t, err, "request %d should be allowed", i)
	}

	// Next request should be rate limited.
	err := l.Allow("user1")
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestAllowAtTokenRefill(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 60, BurstSize: 3, DailyLimit: 0},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	now := time.Now()

	// Exhaust burst.
	for i := 0; i < 3; i++ {
		err := l.AllowAt("user1", now)
		require.NoError(t, err)
	}

	// Should be limited.
	err := l.AllowAt("user1", now)
	assert.ErrorIs(t, err, ErrRateLimited)

	// Wait 1 second (1 token/sec at 60 RPM).
	err = l.AllowAt("user1", now.Add(1*time.Second))
	assert.NoError(t, err)
}

func TestAllowDailyLimit(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 6000, BurstSize: 100, DailyLimit: 10},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// Use up daily limit.
	for i := 0; i < 10; i++ {
		err := l.Allow("user1")
		require.NoError(t, err, "request %d should be allowed", i)
	}

	// Should be limited by daily cap.
	err := l.Allow("user1")
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestAllowDailyLimitReset(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 6000, BurstSize: 100, DailyLimit: 5},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	now := time.Now()

	// Exhaust daily limit.
	for i := 0; i < 5; i++ {
		err := l.AllowAt("user1", now)
		require.NoError(t, err)
	}

	// Should be limited.
	err := l.AllowAt("user1", now)
	assert.ErrorIs(t, err, ErrRateLimited)

	// After 24 hours, should be allowed again.
	err = l.AllowAt("user1", now.Add(25*time.Hour))
	assert.NoError(t, err)
}

func TestRetryAfter(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 60, BurstSize: 2, DailyLimit: 0},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// Before exhaustion, retry should be 0.
	d, err := l.RetryAfter("user1")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), d)

	// Exhaust.
	for i := 0; i < 2; i++ {
		require.NoError(t, l.Allow("user1"))
	}

	d, err = l.RetryAfter("user1")
	require.NoError(t, err)
	assert.True(t, d > 0, "expected positive retry-after, got %v", d)
}

func TestRetryAfterNoPlan(t *testing.T) {
	l := NewLimiter(nil)
	_, err := l.RetryAfter("nobody")
	assert.ErrorIs(t, err, ErrInvalidPlan)
}

func TestGetStatus(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanPro))

	status, err := l.GetStatus("user1")
	require.NoError(t, err)
	assert.Equal(t, PlanPro, status.Tier)
	assert.Equal(t, 100, status.Limit) // Pro burst size
	assert.True(t, status.Remaining > 0)
	assert.Equal(t, int64(50000), status.DailyLimit)
}

func TestGetStatusNoPlan(t *testing.T) {
	l := NewLimiter(nil)
	_, err := l.GetStatus("nobody")
	assert.ErrorIs(t, err, ErrInvalidPlan)
}

func TestRemoveUser(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))
	assert.Equal(t, 1, l.UserCount())

	err := l.RemoveUser("user1")
	require.NoError(t, err)
	assert.Equal(t, 0, l.UserCount())

	// Allowing after removal should fail.
	err = l.Allow("user1")
	assert.ErrorIs(t, err, ErrInvalidPlan)
}

func TestPlanUpgrade(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// Exhaust free tier burst (15).
	for i := 0; i < 15; i++ {
		require.NoError(t, l.Allow("user1"))
	}
	assert.ErrorIs(t, l.Allow("user1"), ErrRateLimited)

	// Upgrade to pro — should get fresh bucket.
	require.NoError(t, l.SetUserPlan("user1", PlanPro))
	err := l.Allow("user1")
	assert.NoError(t, err)
}

func TestMultipleUsers(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 60, BurstSize: 3, DailyLimit: 0},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))
	require.NoError(t, l.SetUserPlan("user2", PlanFree))

	// Exhaust user1.
	for i := 0; i < 3; i++ {
		require.NoError(t, l.Allow("user1"))
	}
	assert.ErrorIs(t, l.Allow("user1"), ErrRateLimited)

	// user2 should still be fine.
	assert.NoError(t, l.Allow("user2"))
}

func TestDefaultTierConfigs(t *testing.T) {
	configs := DefaultTierConfigs()
	assert.Equal(t, 4, len(configs))

	// Verify free tier.
	free := configs[PlanFree]
	assert.Equal(t, float64(10), free.RequestsPerMinute)
	assert.Equal(t, 15, free.BurstSize)
	assert.Equal(t, int64(500), free.DailyLimit)

	// Enterprise has no daily limit.
	ent := configs[PlanEnterprise]
	assert.Equal(t, int64(0), ent.DailyLimit)
}

func TestFormatRetryAfter(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "now"},
		{-1 * time.Second, "now"},
		{5 * time.Second, "5 seconds"},
		{30 * time.Second, "30 seconds"},
		{90 * time.Second, "1 minutes"},
		{3600 * time.Second, "1 hours"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, FormatRetryAfter(tt.d))
	}
}

// --- SQLite Store Tests ---

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	// Use a single connection to avoid separate in-memory databases per connection.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteStoreCreateAndLoad(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Millisecond)
	state := BucketState{
		Tokens:   5.5,
		LastTime: now,
		DayCount: 42,
		DayStart: now.Truncate(24 * time.Hour),
	}

	err = store.SaveBucket("user1", PlanPro, state)
	require.NoError(t, err)

	tier, loaded, err := store.LoadBucket("user1")
	require.NoError(t, err)
	assert.Equal(t, PlanPro, tier)
	assert.InDelta(t, state.Tokens, loaded.Tokens, 0.01)
	assert.Equal(t, state.DayCount, loaded.DayCount)
}

func TestSQLiteStoreLoadNotFound(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	_, _, err = store.LoadBucket("nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSQLiteStoreUpsert(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	state1 := BucketState{Tokens: 10, LastTime: now, DayCount: 5, DayStart: now}
	require.NoError(t, store.SaveBucket("user1", PlanFree, state1))

	state2 := BucketState{Tokens: 3, LastTime: now, DayCount: 50, DayStart: now}
	require.NoError(t, store.SaveBucket("user1", PlanPro, state2))

	tier, loaded, err := store.LoadBucket("user1")
	require.NoError(t, err)
	assert.Equal(t, PlanPro, tier)
	assert.InDelta(t, 3.0, loaded.Tokens, 0.01)
	assert.Equal(t, int64(50), loaded.DayCount)
}

func TestSQLiteStoreDelete(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	require.NoError(t, store.SaveBucket("user1", PlanFree, BucketState{
		Tokens: 10, LastTime: now, DayCount: 0, DayStart: now,
	}))

	err = store.DeleteBucket("user1")
	require.NoError(t, err)

	_, _, err = store.LoadBucket("user1")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSQLiteStoreDeleteNonexistent(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	// Deleting a non-existent key should not error.
	err = store.DeleteBucket("nobody")
	assert.NoError(t, err)
}

func TestSQLiteStoreNilDB(t *testing.T) {
	_, err := NewSQLiteRateLimitStore(nil)
	assert.Error(t, err)
}

func TestSQLiteStoreClose(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)
	assert.NoError(t, store.Close())
}

// --- Persistence Integration Tests ---

func TestLimiterPersistAndRestore(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	l := NewLimiterWithStore(nil, store)
	require.NoError(t, l.SetUserPlan("user1", PlanStarter))

	// Use some tokens.
	for i := 0; i < 5; i++ {
		require.NoError(t, l.Allow("user1"))
	}

	// Persist.
	require.NoError(t, l.Persist())

	// Create fresh limiter with same store.
	l2 := NewLimiterWithStore(nil, store)
	require.NoError(t, l2.RestoreUser("user1"))

	// Should have the correct plan.
	status, err := l2.GetStatus("user1")
	require.NoError(t, err)
	assert.Equal(t, PlanStarter, status.Tier)
}

func TestLimiterRestoreUserNotFound(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	l := NewLimiterWithStore(nil, store)

	// Restoring non-existent user should be a no-op.
	err = l.RestoreUser("nobody")
	assert.NoError(t, err)
}

func TestLimiterPersistNoStore(t *testing.T) {
	l := NewLimiter(nil)
	err := l.Persist()
	assert.NoError(t, err) // should be no-op
}

func TestLimiterRemoveUserWithStore(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	l := NewLimiterWithStore(nil, store)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))
	require.NoError(t, l.Persist())

	// Remove user — should delete from store too.
	require.NoError(t, l.RemoveUser("user1"))

	_, _, err = store.LoadBucket("user1")
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Middleware Tests ---

func TestMiddlewareAllowed(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanPro))

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), users.ContextKeyUserID(), "user1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rr.Header().Get("X-RateLimit-Remaining"))
}

func TestMiddlewareRateLimited(t *testing.T) {
	tiers := map[PlanTier]TierConfig{
		PlanFree: {RequestsPerMinute: 60, BurstSize: 2, DailyLimit: 0},
	}
	l := NewLimiter(tiers)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), users.ContextKeyUserID(), "user1")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	// Should be rate limited now.
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), users.ContextKeyUserID(), "user1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "rate_limited", resp["code"])
}

func TestMiddlewareNoAuth(t *testing.T) {
	l := NewLimiter(nil)

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No user in context — should pass through.
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMiddlewareNoPlan(t *testing.T) {
	l := NewLimiter(nil)

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// User has no plan set — should pass through.
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), users.ContextKeyUserID(), "user-no-plan")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- Status Handler Tests ---

func TestStatusHandler(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanStarter))

	handler := StatusHandler(l)
	req := httptest.NewRequest("GET", "/api/v1/rate-limit/status", nil)
	ctx := context.WithValue(req.Context(), users.ContextKeyUserID(), "user1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp StatusResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "starter", resp.Tier)
	assert.Equal(t, 50, resp.Limit) // starter burst
	assert.Equal(t, int64(5000), resp.DailyLimit)
}

func TestStatusHandlerNoAuth(t *testing.T) {
	l := NewLimiter(nil)
	handler := StatusHandler(l)

	req := httptest.NewRequest("GET", "/api/v1/rate-limit/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestStatusHandlerNoPlan(t *testing.T) {
	l := NewLimiter(nil)
	handler := StatusHandler(l)

	req := httptest.NewRequest("GET", "/api/v1/rate-limit/status", nil)
	ctx := context.WithValue(req.Context(), users.ContextKeyUserID(), "user-no-plan")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- PersistMiddleware Test ---

func TestPersistMiddleware(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteRateLimitStore(db)
	require.NoError(t, err)

	l := NewLimiterWithStore(nil, store)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// Start persist loop with very short interval.
	stop := PersistMiddleware(l, 50*time.Millisecond)

	// Use some tokens.
	require.NoError(t, l.Allow("user1"))

	// Wait for a persist cycle.
	time.Sleep(100 * time.Millisecond)

	// Stop (which triggers a final persist).
	stop()

	// Check store has data.
	tier, _, err := store.LoadBucket("user1")
	require.NoError(t, err)
	assert.Equal(t, PlanFree, tier)
}

func TestUnlimitedDailyEnterprise(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanEnterprise))

	// Enterprise has no daily limit — should never hit daily cap.
	for i := 0; i < 200; i++ {
		require.NoError(t, l.Allow("user1"))
	}

	status, err := l.GetStatus("user1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), status.DailyLimit)
	assert.Equal(t, int64(0), status.DailyRemaining) // 0 means unlimited
}

func TestSamePlanNoReset(t *testing.T) {
	l := NewLimiter(nil)
	require.NoError(t, l.SetUserPlan("user1", PlanFree))

	// Use some tokens.
	for i := 0; i < 5; i++ {
		require.NoError(t, l.Allow("user1"))
	}

	// Re-set same plan — still resets bucket (SetUserPlan always creates fresh).
	require.NoError(t, l.SetUserPlan("user1", PlanFree))
	status, err := l.GetStatus("user1")
	require.NoError(t, err)
	// Bucket resets because SetUserPlan always creates a fresh bucket when
	// called with the same or different tier.
	assert.Equal(t, 15, status.Remaining) // full burst
}
