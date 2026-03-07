package billing

import (
	"encoding/json"
	"net/http"
)

// API provides HTTP handlers for billing plan endpoints.
type API struct {
	catalogue *Catalogue
	store     SubscriptionStore
}

// NewAPI creates a billing API with the given catalogue.
// store may be nil if subscription management is not yet enabled.
func NewAPI(catalogue *Catalogue, store SubscriptionStore) *API {
	return &API{catalogue: catalogue, store: store}
}

// RegisterRoutes registers billing routes on the given mux.
// All routes are under /api/v1/billing/.
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/billing/plans", a.handleListPlans)
	mux.HandleFunc("GET /api/v1/billing/plans/{id}", a.handleGetPlan)
}

// handleListPlans returns all active plans.
func (a *API) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plans := a.catalogue.ListActive()
	writeJSON(w, http.StatusOK, map[string]any{
		"plans": plans,
		"count": len(plans),
	})
}

// handleGetPlan returns a single plan by ID.
func (a *API) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	id := PlanID(r.PathValue("id"))
	plan := a.catalogue.Get(id)
	if plan == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "plan not found",
			"code":  "not_found",
		})
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
