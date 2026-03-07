package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Threshold Tests ---

func TestDefaultOverageThresholds(t *testing.T) {
	th := DefaultOverageThresholds()
	assert.Equal(t, 0.80, th.Warning)
	assert.Equal(t, 0.90, th.SoftCap)
	assert.Equal(t, 1.00, th.HardCap)
	assert.Equal(t, 1.20, th.BlockAt)
	assert.NoError(t, th.Validate())
}

func TestOverageThresholds_Validate(t *testing.T) {
	tests := []struct {
		name      string
		th        OverageThresholds
		wantError bool
	}{
		{"valid defaults", DefaultOverageThresholds(), false},
		{"valid custom", OverageThresholds{0.50, 0.75, 1.00, 1.50}, false},
		{"warning too low", OverageThresholds{0, 0.90, 1.00, 1.20}, true},
		{"warning too high", OverageThresholds{1.0, 1.10, 1.20, 1.30}, true},
		{"soft_cap <= warning", OverageThresholds{0.80, 0.80, 1.00, 1.20}, true},
		{"hard_cap <= soft_cap", OverageThresholds{0.80, 0.90, 0.90, 1.20}, true},
		{"block_at <= hard_cap", OverageThresholds{0.80, 0.90, 1.00, 1.00}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.th.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Enforcer Creation Tests ---

func TestNewOverageEnforcer_NilUsageStore(t *testing.T) {
	_, err := NewOverageEnforcer(OverageEnforcerConfig{
		Catalogue: NewCatalogue(nil),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "usage store is required")
}

func TestNewOverageEnforcer_NilCatalogue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	_, err = NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: store,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "catalogue is required")
}

func TestNewOverageEnforcer_InvalidThresholds(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	bad := OverageThresholds{0.90, 0.80, 1.00, 1.20}
	_, err = NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: store,
		Catalogue:  NewCatalogue(nil),
		Thresholds: &bad,
	})
	assert.Error(t, err)
}

func TestNewOverageEnforcer_Success(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	enforcer, err := NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: store,
		Catalogue:  NewCatalogue(nil),
	})
	require.NoError(t, err)
	assert.NotNil(t, enforcer)
}

func TestNewOverageEnforcer_CustomConfig(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	th := OverageThresholds{0.70, 0.85, 1.00, 1.30}
	tc := ThrottleConfig{FallbackModel: "llama-3", DelayMs: 200, ReducedRatePercent: 30}
	enforcer, err := NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: store,
		Catalogue:  NewCatalogue(nil),
		Thresholds: &th,
		Throttle:   &tc,
	})
	require.NoError(t, err)
	assert.NotNil(t, enforcer)
}

// --- Evaluate Tests ---

func TestEvaluate_None(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("tokens", 400, 1000)
	assert.Equal(t, OverageLevelNone, status.Level)
	assert.Equal(t, ActionNone, status.Action)
	assert.Equal(t, int64(400), status.Usage)
	assert.Equal(t, int64(1000), status.Limit)
	assert.InDelta(t, 0.40, status.Percentage, 0.01)
	assert.Empty(t, status.Message)
}

func TestEvaluate_Warning(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("tokens", 800, 1000)
	assert.Equal(t, OverageLevelWarning, status.Level)
	assert.Equal(t, ActionWarn, status.Action)
	assert.Contains(t, status.Message, "80%")
	assert.Contains(t, status.Message, "upgrading")
}

func TestEvaluate_SoftCap(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("tokens", 950, 1000)
	assert.Equal(t, OverageLevelSoftCap, status.Level)
	assert.Equal(t, ActionDowngradeModel, status.Action)
	assert.Equal(t, "gpt-4o-mini", status.FallbackModel)
	assert.Contains(t, status.Message, "approaching hard cap")
}

func TestEvaluate_HardCap(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("messages", 1000, 1000)
	assert.Equal(t, OverageLevelHardCap, status.Level)
	assert.Equal(t, ActionThrottle, status.Action)
	assert.Equal(t, "gpt-4o-mini", status.FallbackModel)
	assert.Contains(t, status.Message, "throttled")
}

func TestEvaluate_HardCap_Over(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("tokens", 1100, 1000) // 110%
	assert.Equal(t, OverageLevelHardCap, status.Level)
	assert.Equal(t, ActionThrottle, status.Action)
}

func TestEvaluate_Blocked(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("tokens", 1200, 1000) // 120%
	assert.Equal(t, OverageLevelBlocked, status.Level)
	assert.Equal(t, ActionBlock, status.Action)
	assert.Contains(t, status.Message, "blocked")
}

func TestEvaluate_Blocked_WayOver(t *testing.T) {
	enforcer := mustEnforcer(t)
	status := enforcer.evaluate("tokens", 2000, 1000) // 200%
	assert.Equal(t, OverageLevelBlocked, status.Level)
	assert.Equal(t, ActionBlock, status.Action)
}

func TestEvaluate_ExactBoundaries(t *testing.T) {
	enforcer := mustEnforcer(t)

	// Exactly at warning (80%)
	s := enforcer.evaluate("tokens", 800, 1000)
	assert.Equal(t, OverageLevelWarning, s.Level)

	// Just below warning
	s = enforcer.evaluate("tokens", 799, 1000)
	assert.Equal(t, OverageLevelNone, s.Level)

	// Exactly at soft cap (90%)
	s = enforcer.evaluate("tokens", 900, 1000)
	assert.Equal(t, OverageLevelSoftCap, s.Level)

	// Exactly at hard cap (100%)
	s = enforcer.evaluate("tokens", 1000, 1000)
	assert.Equal(t, OverageLevelHardCap, s.Level)

	// Exactly at block (120%)
	s = enforcer.evaluate("tokens", 1200, 1000)
	assert.Equal(t, OverageLevelBlocked, s.Level)
}

// --- CheckTokens / CheckMessages Tests ---

func TestCheckTokens_Unlimited(t *testing.T) {
	enforcer := mustEnforcer(t)
	plan := &Plan{Limits: PlanLimits{MaxTokensPerMonth: 0}} // unlimited
	status, err := enforcer.CheckTokens("user1", time.Now().UTC(), plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, status.Level)
	assert.Equal(t, ActionNone, status.Action)
}

func TestCheckTokens_NilPlan(t *testing.T) {
	enforcer := mustEnforcer(t)
	_, err := enforcer.CheckTokens("user1", time.Now().UTC(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan is nil")
}

func TestCheckTokens_UnderLimit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	enforcer := mustEnforcerWithDB(t, db)

	store, _ := NewSQLiteUsageStore(db)
	now := time.Now().UTC()
	periodStart := beginningOfMonth(now)

	// Record 100 tokens — well under the free plan limit of 500,000.
	require.NoError(t, store.Record(&UsageEvent{
		ID: "evt1", UserID: "user1", Model: "gpt-4o-mini",
		InputTokens: 50, OutputTokens: 50, CreatedAt: now,
	}))

	plan := NewCatalogue(nil).Get(PlanFree)
	status, err := enforcer.CheckTokens("user1", periodStart, plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, status.Level)
	assert.Equal(t, int64(100), status.Usage)
}

func TestCheckTokens_OverLimit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	periodStart := beginningOfMonth(now)

	// Record usage at 100% of a 1000 token limit.
	require.NoError(t, store.Record(&UsageEvent{
		ID: "evt1", UserID: "user1", Model: "gpt-4o-mini",
		TotalTokens: 1000, CreatedAt: now,
	}))

	enforcer := mustEnforcerWithDB(t, db)
	plan := &Plan{Limits: PlanLimits{MaxTokensPerMonth: 1000}}
	status, err := enforcer.CheckTokens("user1", periodStart, plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelHardCap, status.Level)
	assert.Equal(t, ActionThrottle, status.Action)
}

func TestCheckMessages_Unlimited(t *testing.T) {
	enforcer := mustEnforcer(t)
	plan := &Plan{Limits: PlanLimits{MaxMessagesPerMonth: 0}}
	status, err := enforcer.CheckMessages("user1", time.Now().UTC(), plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, status.Level)
}

func TestCheckMessages_NilPlan(t *testing.T) {
	enforcer := mustEnforcer(t)
	_, err := enforcer.CheckMessages("user1", time.Now().UTC(), nil)
	assert.Error(t, err)
}

func TestCheckMessages_Warning(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	periodStart := beginningOfMonth(now)

	// Record 85 messages out of 100 limit (85% — in warning zone).
	for i := 0; i < 85; i++ {
		require.NoError(t, store.Record(&UsageEvent{
			ID: fmt.Sprintf("evt%d", i), UserID: "user1", Model: "gpt-4o-mini",
			TotalTokens: 10, CreatedAt: now,
		}))
	}

	enforcer := mustEnforcerWithDB(t, db)
	plan := &Plan{Limits: PlanLimits{MaxMessagesPerMonth: 100}}
	status, err := enforcer.CheckMessages("user1", periodStart, plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelWarning, status.Level)
	assert.Equal(t, ActionWarn, status.Action)
	assert.Equal(t, int64(85), status.Usage)
}

// --- CheckAll Tests ---

func TestCheckAll_ReturnsMostSevere(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	periodStart := beginningOfMonth(now)

	// Record 95 messages (soft cap at 90%) and 100 tokens (under warning at 10%).
	for i := 0; i < 95; i++ {
		require.NoError(t, store.Record(&UsageEvent{
			ID: fmt.Sprintf("evt%d", i), UserID: "user1", Model: "gpt-4o-mini",
			TotalTokens: 1, CreatedAt: now,
		}))
	}

	enforcer := mustEnforcerWithDB(t, db)
	plan := &Plan{Limits: PlanLimits{
		MaxMessagesPerMonth: 100,
		MaxTokensPerMonth:   1000,
	}}

	status, err := enforcer.CheckAll("user1", periodStart, plan)
	require.NoError(t, err)
	// Messages at 95% (soft cap) is more severe than tokens at ~10% (none).
	assert.Equal(t, OverageLevelSoftCap, status.Level)
	assert.Equal(t, "messages", status.Resource)
}

func TestCheckAll_BothUnlimited(t *testing.T) {
	enforcer := mustEnforcer(t)
	plan := &Plan{Limits: PlanLimits{
		MaxMessagesPerMonth: 0,
		MaxTokensPerMonth:   0,
	}}

	status, err := enforcer.CheckAll("user1", time.Now().UTC(), plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, status.Level)
}

// --- CheckUser Tests ---

func TestCheckUser_FreePlan(t *testing.T) {
	enforcer := mustEnforcer(t)
	status, err := enforcer.CheckUser("user1")
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, status.Level) // no usage
}

func TestCheckUser_WithSubscription(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)
	subStore, err := NewSQLiteSubscriptionStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	periodStart := now.Add(-24 * time.Hour)
	periodEnd := now.Add(24 * time.Hour)

	// Create a starter subscription.
	require.NoError(t, subStore.Create(&Subscription{
		ID: "sub1", UserID: "user1", PlanID: PlanStarter,
		Status: SubStatusActive, BillingInterval: IntervalMonthly,
		CurrentPeriodStart: periodStart, CurrentPeriodEnd: periodEnd,
	}))

	enforcer, err := NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: usageStore,
		SubStore:   subStore,
		Catalogue:  NewCatalogue(nil),
	})
	require.NoError(t, err)

	status, err := enforcer.CheckUser("user1")
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, status.Level)
}

// --- GetFullStatus Tests ---

func TestGetFullStatus(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)
	now := time.Now().UTC()

	// Record some usage.
	for i := 0; i < 10; i++ {
		require.NoError(t, store.Record(&UsageEvent{
			ID: fmt.Sprintf("evt%d", i), UserID: "user1", Model: "gpt-4o-mini",
			TotalTokens: 100, CreatedAt: now,
		}))
	}

	enforcer := mustEnforcerWithDB(t, db)
	statuses, err := enforcer.GetFullStatus("user1")
	require.NoError(t, err)
	assert.Len(t, statuses, 2)
	assert.Equal(t, "tokens", statuses[0].Resource)
	assert.Equal(t, "messages", statuses[1].Resource)
}

// --- levelSeverity Tests ---

func TestLevelSeverity(t *testing.T) {
	assert.Equal(t, 0, levelSeverity(OverageLevelNone))
	assert.Equal(t, 1, levelSeverity(OverageLevelWarning))
	assert.Equal(t, 2, levelSeverity(OverageLevelSoftCap))
	assert.Equal(t, 3, levelSeverity(OverageLevelHardCap))
	assert.Equal(t, 4, levelSeverity(OverageLevelBlocked))
	assert.Equal(t, 0, levelSeverity("unknown"))
}

// --- Middleware Tests ---

func TestOverageMiddleware_NoAuth(t *testing.T) {
	enforcer := mustEnforcer(t)
	handler := OverageMiddleware(enforcer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	// No overage headers when not authenticated.
	assert.Empty(t, rec.Header().Get("X-Overage-Level"))
}

func TestOverageMiddleware_UnderLimit(t *testing.T) {
	enforcer := mustEnforcer(t)
	handler := OverageMiddleware(enforcer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = setTestUserID(req, "user1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, string(OverageLevelNone), rec.Header().Get("X-Overage-Level"))
	assert.Equal(t, string(ActionNone), rec.Header().Get("X-Overage-Action"))
}

func TestOverageMiddleware_Blocked(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	// Record usage at 125% of the free plan's token limit (500,000).
	require.NoError(t, store.Record(&UsageEvent{
		ID: "evt1", UserID: "user1", Model: "gpt-4o-mini",
		TotalTokens: 625_000, CreatedAt: now,
	}))

	enforcer := mustEnforcerWithDB(t, db)

	called := false
	handler := OverageMiddleware(enforcer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = setTestUserID(req, "user1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.False(t, called, "handler should not be called when blocked")
	assert.Equal(t, string(OverageLevelBlocked), rec.Header().Get("X-Overage-Level"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "overage_blocked", body["code"])
}

func TestOverageMiddleware_Throttled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	// Record usage at 105% of free plan's token limit.
	require.NoError(t, store.Record(&UsageEvent{
		ID: "evt1", UserID: "user1", Model: "gpt-4o-mini",
		TotalTokens: 525_000, CreatedAt: now,
	}))

	// Use 0 delay so test doesn't sleep.
	th := DefaultOverageThresholds()
	tc := ThrottleConfig{FallbackModel: "gpt-4o-mini", DelayMs: 0, ReducedRatePercent: 50}
	enforcer, err := NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: store,
		Catalogue:  NewCatalogue(nil),
		Thresholds: &th,
		Throttle:   &tc,
	})
	require.NoError(t, err)

	called := false
	handler := OverageMiddleware(enforcer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = setTestUserID(req, "user1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should be called when throttled")
	assert.Equal(t, string(OverageLevelHardCap), rec.Header().Get("X-Overage-Level"))
	assert.Equal(t, string(ActionThrottle), rec.Header().Get("X-Overage-Action"))
	assert.Equal(t, "gpt-4o-mini", rec.Header().Get("X-Overage-Fallback-Model"))
}

// --- Overage API Tests ---

func TestOverageAPI_Unauthorized(t *testing.T) {
	enforcer := mustEnforcer(t)
	api := NewOverageAPI(enforcer)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/overage", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOverageAPI_Success(t *testing.T) {
	enforcer := mustEnforcer(t)
	api := NewOverageAPI(enforcer)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/overage", nil)
	req = setTestUserID(req, "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, string(OverageLevelNone), body["overall_level"])
	assert.Equal(t, string(ActionNone), body["overall_action"])

	resources, ok := body["resources"].([]any)
	require.True(t, ok)
	assert.Len(t, resources, 2)
}

func TestOverageAPI_WithUsage(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	// Record 450 messages — 90% of free plan's 500 message limit.
	for i := 0; i < 450; i++ {
		require.NoError(t, store.Record(&UsageEvent{
			ID: fmt.Sprintf("evt%d", i), UserID: "user1", Model: "gpt-4o-mini",
			TotalTokens: 10, CreatedAt: now,
		}))
	}

	enforcer := mustEnforcerWithDB(t, db)
	api := NewOverageAPI(enforcer)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/overage", nil)
	req = setTestUserID(req, "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	// Messages at 90% should be soft cap.
	assert.Equal(t, string(OverageLevelSoftCap), body["overall_level"])
}

// --- DefaultThrottleConfig Tests ---

func TestDefaultThrottleConfig(t *testing.T) {
	tc := DefaultThrottleConfig()
	assert.Equal(t, "gpt-4o-mini", tc.FallbackModel)
	assert.Equal(t, 500, tc.DelayMs)
	assert.Equal(t, 50, tc.ReducedRatePercent)
}

// --- Multi-user isolation ---

func TestOverageEnforcer_MultiUserIsolation(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	now := time.Now().UTC()
	periodStart := beginningOfMonth(now)

	// User1 at 95% of limit, user2 at 10%.
	for i := 0; i < 95; i++ {
		require.NoError(t, store.Record(&UsageEvent{
			ID: fmt.Sprintf("u1evt%d", i), UserID: "user1", Model: "gpt-4o-mini",
			TotalTokens: 10, CreatedAt: now,
		}))
	}
	for i := 0; i < 10; i++ {
		require.NoError(t, store.Record(&UsageEvent{
			ID: fmt.Sprintf("u2evt%d", i), UserID: "user2", Model: "gpt-4o-mini",
			TotalTokens: 10, CreatedAt: now,
		}))
	}

	enforcer := mustEnforcerWithDB(t, db)
	plan := &Plan{Limits: PlanLimits{MaxMessagesPerMonth: 100}}

	s1, err := enforcer.CheckMessages("user1", periodStart, plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelSoftCap, s1.Level)

	s2, err := enforcer.CheckMessages("user2", periodStart, plan)
	require.NoError(t, err)
	assert.Equal(t, OverageLevelNone, s2.Level)
}

// --- RegisterRoutes ---

func TestOverageAPI_RegisterRoutes(t *testing.T) {
	enforcer := mustEnforcer(t)
	api := NewOverageAPI(enforcer)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/overage", nil)
	req = setTestUserID(req, "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- Helpers ---

func mustEnforcer(t *testing.T) *OverageEnforcer {
	t.Helper()
	db := openTestDB(t)
	t.Cleanup(func() { db.Close() })
	return mustEnforcerWithDB(t, db)
}

func mustEnforcerWithDB(t *testing.T, db *sql.DB) *OverageEnforcer {
	t.Helper()
	store, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	enforcer, err := NewOverageEnforcer(OverageEnforcerConfig{
		UsageStore: store,
		Catalogue:  NewCatalogue(nil),
	})
	require.NoError(t, err)
	return enforcer
}

func setTestUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), contextKeyUserID("user_id"), userID)
	return r.WithContext(ctx)
}
