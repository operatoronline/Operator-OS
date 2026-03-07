package billing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAPI() (*API, *http.ServeMux) {
	cat := NewCatalogue(nil)
	api := NewAPI(cat, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	return api, mux
}

func TestAPIListPlans(t *testing.T) {
	_, mux := setupAPI()

	req := httptest.NewRequest("GET", "/api/v1/billing/plans", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Plans []Plan `json:"plans"`
		Count int    `json:"count"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 4, resp.Count)
	assert.Len(t, resp.Plans, 4)

	// Verify sort order.
	assert.Equal(t, PlanFree, resp.Plans[0].ID)
	assert.Equal(t, PlanStarter, resp.Plans[1].ID)
	assert.Equal(t, PlanPro, resp.Plans[2].ID)
	assert.Equal(t, PlanEnterprise, resp.Plans[3].ID)
}

func TestAPIListPlansFiltersInactive(t *testing.T) {
	plans := DefaultPlans()
	plans[PlanStarter].Active = false
	cat := NewCatalogue(plans)
	api := NewAPI(cat, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/billing/plans", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp struct {
		Plans []Plan `json:"plans"`
		Count int    `json:"count"`
	}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, 3, resp.Count)
}

func TestAPIGetPlan(t *testing.T) {
	_, mux := setupAPI()

	req := httptest.NewRequest("GET", "/api/v1/billing/plans/pro", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var plan Plan
	err := json.Unmarshal(rec.Body.Bytes(), &plan)
	require.NoError(t, err)
	assert.Equal(t, PlanPro, plan.ID)
	assert.Equal(t, "Pro", plan.Name)
	assert.Equal(t, int64(2900), plan.PriceMonthly)
}

func TestAPIGetPlanNotFound(t *testing.T) {
	_, mux := setupAPI()

	req := httptest.NewRequest("GET", "/api/v1/billing/plans/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "not_found", resp["code"])
}

func TestAPIGetPlanFree(t *testing.T) {
	_, mux := setupAPI()

	req := httptest.NewRequest("GET", "/api/v1/billing/plans/free", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var plan Plan
	json.Unmarshal(rec.Body.Bytes(), &plan)
	assert.Equal(t, PlanFree, plan.ID)
	assert.Equal(t, int64(0), plan.PriceMonthly)
	assert.Equal(t, 1, plan.Limits.MaxAgents)
}

func TestAPIGetPlanEnterprise(t *testing.T) {
	_, mux := setupAPI()

	req := httptest.NewRequest("GET", "/api/v1/billing/plans/enterprise", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var plan Plan
	json.Unmarshal(rec.Body.Bytes(), &plan)
	assert.Equal(t, PlanEnterprise, plan.ID)
	assert.Equal(t, 0, plan.Limits.MaxAgents) // unlimited
}

func TestAPIContentType(t *testing.T) {
	_, mux := setupAPI()

	req := httptest.NewRequest("GET", "/api/v1/billing/plans", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
