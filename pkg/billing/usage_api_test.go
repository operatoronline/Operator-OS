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

	_ "modernc.org/sqlite"
)

func newUsageTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func withUserCtx(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), contextKeyUserID("user_id"), userID)
	return r.WithContext(ctx)
}

func seedUsageEvents(t *testing.T, store UsageStore, userID string) {
	t.Helper()
	now := time.Now().UTC()
	events := []*UsageEvent{
		{ID: "ev1", UserID: userID, Model: "gpt-4", Provider: "openai", InputTokens: 100, OutputTokens: 50, EstimatedCost: 0.05, CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "ev2", UserID: userID, Model: "gpt-4", Provider: "openai", InputTokens: 200, OutputTokens: 100, EstimatedCost: 0.10, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "ev3", UserID: userID, Model: "claude-3", Provider: "anthropic", InputTokens: 300, OutputTokens: 150, EstimatedCost: 0.08, CreatedAt: now.Add(-1 * time.Hour)},
	}
	for _, ev := range events {
		require.NoError(t, store.Record(ev))
	}
}

func TestUsageAPI_GetSummary(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "summary")
	assert.Contains(t, resp, "period_start")
	assert.Contains(t, resp, "period_end")

	summary := resp["summary"].(map[string]any)
	assert.Greater(t, summary["total_tokens"].(float64), float64(0))
	assert.Greater(t, summary["total_requests"].(float64), float64(0))
}

func TestUsageAPI_GetSummary_WithTimeRange(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	now := time.Now().UTC()
	since := now.Add(-2 * time.Hour).Format(time.RFC3339)
	until := now.Format(time.RFC3339)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/billing/usage?since=%s&until=%s", since, until), nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	summary := resp["summary"].(map[string]any)
	// Only ev3 should be in range (1 hour ago).
	assert.Equal(t, float64(1), summary["total_requests"].(float64))
}

func TestUsageAPI_GetSummary_Unauthorized(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUsageAPI_GetSummary_NoStore(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUsageAPI_GetDailyUsage(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/daily", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "daily")
	assert.Contains(t, resp, "count")

	daily := resp["daily"].([]any)
	assert.Greater(t, len(daily), 0)
}

func TestUsageAPI_GetDailyUsage_WithDaysParam(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/daily?days=7", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	daily := resp["daily"].([]any)
	assert.Greater(t, len(daily), 0)
}

func TestUsageAPI_GetDailyUsage_Empty(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/daily", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	daily := resp["daily"].([]any)
	assert.Len(t, daily, 0)
}

func TestUsageAPI_GetDailyUsage_Unauthorized(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/daily", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUsageAPI_GetModelUsage(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/models", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	models := resp["models"].([]any)
	assert.Len(t, models, 2) // gpt-4 + claude-3
}

func TestUsageAPI_GetModelUsage_Empty(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/models", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	models := resp["models"].([]any)
	assert.Len(t, models, 0)
}

func TestUsageAPI_GetModelUsage_Unauthorized(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/models", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUsageAPI_ListEvents(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/events", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	events := resp["events"].([]any)
	assert.Len(t, events, 3)
}

func TestUsageAPI_ListEvents_WithModelFilter(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/events?model=gpt-4", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	events := resp["events"].([]any)
	assert.Len(t, events, 2)
}

func TestUsageAPI_ListEvents_Pagination(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/events?limit=2&offset=0", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	events := resp["events"].([]any)
	assert.Len(t, events, 2)
}

func TestUsageAPI_ListEvents_WithTimeRange(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	now := time.Now().UTC()
	since := now.Add(-2 * time.Hour).Format(time.RFC3339)
	until := now.Format(time.RFC3339)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/billing/usage/events?since=%s&until=%s", since, until), nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	events := resp["events"].([]any)
	assert.Len(t, events, 1)
}

func TestUsageAPI_ListEvents_Unauthorized(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUsageAPI_ListEvents_NoStore(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/events", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUsageAPI_GetLimits_FreePlan(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	catalogue := NewCatalogue(DefaultPlans())
	api := NewUsageAPI(usageStore, nil, catalogue)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/limits", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(PlanFree), resp["plan_id"])
	assert.Contains(t, resp, "tokens")
	assert.Contains(t, resp, "messages")

	tokens := resp["tokens"].(map[string]any)
	assert.Greater(t, tokens["used"].(float64), float64(0))
	assert.Greater(t, tokens["limit"].(float64), float64(0))

	messages := resp["messages"].(map[string]any)
	assert.Greater(t, messages["used"].(float64), float64(0))
}

func TestUsageAPI_GetLimits_WithSubscription(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	subStore, err := NewSQLiteSubscriptionStore(db)
	require.NoError(t, err)

	catalogue := NewCatalogue(DefaultPlans())
	api := NewUsageAPI(usageStore, subStore, catalogue)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create a pro subscription.
	now := time.Now().UTC()
	sub := &Subscription{
		ID:                 "sub1",
		UserID:             "user1",
		PlanID:             PlanPro,
		Status:             SubStatusActive,
		CurrentPeriodStart: now.AddDate(0, 0, -15),
		CurrentPeriodEnd:   now.AddDate(0, 0, 15),
	}
	require.NoError(t, subStore.Create(sub))

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/limits", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, string(PlanPro), resp["plan_id"])
}

func TestUsageAPI_GetLimits_Unauthorized(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/limits", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUsageAPI_GetLimits_NoStore(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/limits", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUsageAPI_GetSummary_WithSubscriptionPeriod(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	subStore, err := NewSQLiteSubscriptionStore(db)
	require.NoError(t, err)

	catalogue := NewCatalogue(DefaultPlans())
	api := NewUsageAPI(usageStore, subStore, catalogue)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	now := time.Now().UTC()
	sub := &Subscription{
		ID:                 "sub1",
		UserID:             "user1",
		PlanID:             PlanStarter,
		Status:             SubStatusActive,
		CurrentPeriodStart: now.AddDate(0, 0, -10),
		CurrentPeriodEnd:   now.AddDate(0, 0, 20),
	}
	require.NoError(t, subStore.Create(sub))

	seedUsageEvents(t, usageStore, "user1")

	req := httptest.NewRequest("GET", "/api/v1/billing/usage", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "summary")
}

func TestUsageAPI_MultiUserIsolation(t *testing.T) {
	db := newUsageTestDB(t)
	usageStore, err := NewSQLiteUsageStore(db)
	require.NoError(t, err)

	api := NewUsageAPI(usageStore, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	seedUsageEvents(t, usageStore, "user1")

	// user2 should see no data.
	req := httptest.NewRequest("GET", "/api/v1/billing/usage", nil)
	req = withUserCtx(req, "user2")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	summary := resp["summary"].(map[string]any)
	assert.Equal(t, float64(0), summary["total_requests"].(float64))
}

func TestBeginningOfMonth(t *testing.T) {
	ts := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	bom := beginningOfMonth(ts)
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), bom)
}

func TestUsageAPI_RegisterRoutes(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	// Should not panic.
	api.RegisterRoutes(mux)
}

func TestUsageAPI_GetModelUsage_NoStore(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/models", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUsageAPI_GetDailyUsage_NoStore(t *testing.T) {
	api := NewUsageAPI(nil, nil, NewCatalogue(DefaultPlans()))
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/usage/daily", nil)
	req = withUserCtx(req, "user1")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
