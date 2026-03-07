package integrations

import (
	"encoding/json"
	"net/http"
	"strings"
)

// API provides REST endpoints for the integration registry and user integrations.
type API struct {
	registry *IntegrationRegistry
	store    UserIntegrationStore
}

// NewAPI creates an integration API handler.
func NewAPI(registry *IntegrationRegistry, store UserIntegrationStore) *API {
	return &API{registry: registry, store: store}
}

// RegisterRoutes registers the integration API routes on the given mux.
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/integrations", a.handleIntegrations)
	mux.HandleFunc("/api/v1/integrations/categories", a.handleCategories)
	mux.HandleFunc("/api/v1/integrations/", a.handleIntegrationByID)
	mux.HandleFunc("/api/v1/user/integrations", a.handleUserIntegrations)
	mux.HandleFunc("/api/v1/user/integrations/", a.handleUserIntegrationByID)
}

// handleIntegrations lists available integrations.
// GET /api/v1/integrations?category=email&status=active
func (a *API) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResp("method_not_allowed", "GET only"))
		return
	}
	category := r.URL.Query().Get("category")
	var list []*Integration
	if category != "" {
		list = a.registry.ListByCategory(category)
	} else {
		list = a.registry.ListActive()
	}

	// Sanitize: remove OAuth secrets from response
	sanitized := make([]integrationSummary, len(list))
	for i, integ := range list {
		sanitized[i] = summarizeIntegration(integ)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"integrations": sanitized,
		"count":        len(sanitized),
	})
}

// handleCategories returns available categories.
// GET /api/v1/integrations/categories
func (a *API) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResp("method_not_allowed", "GET only"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"categories": a.registry.Categories(),
	})
}

// handleIntegrationByID gets a single integration.
// GET /api/v1/integrations/{id}
func (a *API) handleIntegrationByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResp("method_not_allowed", "GET only"))
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/integrations/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("missing_id", "Integration ID is required"))
		return
	}
	integ := a.registry.Get(id)
	if integ == nil {
		writeJSON(w, http.StatusNotFound, errorResp("not_found", "Integration not found"))
		return
	}
	writeJSON(w, http.StatusOK, summarizeIntegration(integ))
}

// handleUserIntegrations lists or creates user integrations.
// GET /api/v1/user/integrations?status=active
// POST /api/v1/user/integrations
func (a *API) handleUserIntegrations(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromRequest(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, errorResp("unauthorized", "Authentication required"))
		return
	}
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResp("not_configured", "User integration store not configured"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		list, err := a.store.ListByUser(userID, status)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResp("internal", err.Error()))
			return
		}
		if list == nil {
			list = []*UserIntegration{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"integrations": list,
			"count":        len(list),
		})

	case http.MethodPost:
		var req struct {
			IntegrationID string            `json:"integration_id"`
			Config        map[string]string `json:"config,omitempty"`
			Scopes        []string          `json:"scopes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp("invalid_json", "Invalid request body"))
			return
		}
		if req.IntegrationID == "" {
			writeJSON(w, http.StatusBadRequest, errorResp("missing_integration_id", "integration_id is required"))
			return
		}
		// Verify integration exists in registry
		if a.registry.Get(req.IntegrationID) == nil {
			writeJSON(w, http.StatusNotFound, errorResp("integration_not_found", "Integration not found in registry"))
			return
		}
		ui := &UserIntegration{
			UserID:        userID,
			IntegrationID: req.IntegrationID,
			Config:        req.Config,
			Scopes:        req.Scopes,
			Status:        UserIntegrationPending,
		}
		if err := a.store.Create(ui); err != nil {
			if strings.Contains(err.Error(), "already connected") {
				writeJSON(w, http.StatusConflict, errorResp("already_connected", err.Error()))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResp("internal", err.Error()))
			return
		}
		writeJSON(w, http.StatusCreated, ui)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResp("method_not_allowed", "GET or POST only"))
	}
}

// handleUserIntegrationByID handles single user integration operations.
// GET /api/v1/user/integrations/{id}
// DELETE /api/v1/user/integrations/{id}
func (a *API) handleUserIntegrationByID(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromRequest(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, errorResp("unauthorized", "Authentication required"))
		return
	}
	if a.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResp("not_configured", "User integration store not configured"))
		return
	}

	integrationID := strings.TrimPrefix(r.URL.Path, "/api/v1/user/integrations/")
	if integrationID == "" {
		writeJSON(w, http.StatusBadRequest, errorResp("missing_id", "Integration ID is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		ui, err := a.store.Get(userID, integrationID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSON(w, http.StatusNotFound, errorResp("not_found", "User integration not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResp("internal", err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, ui)

	case http.MethodDelete:
		err := a.store.Delete(userID, integrationID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSON(w, http.StatusNotFound, errorResp("not_found", "User integration not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResp("internal", err.Error()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResp("method_not_allowed", "GET or DELETE only"))
	}
}

// --- helpers ---

type integrationSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon,omitempty"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	AuthType    string   `json:"auth_type"`
	ToolCount   int      `json:"tool_count"`
	ToolNames   []string `json:"tool_names"`
	RequiredPlan string  `json:"required_plan,omitempty"`
	Status      string   `json:"status,omitempty"`
	Version     string   `json:"version,omitempty"`
	HasOAuth    bool     `json:"has_oauth"`
}

func summarizeIntegration(i *Integration) integrationSummary {
	return integrationSummary{
		ID:           i.ID,
		Name:         i.Name,
		Icon:         i.Icon,
		Category:     i.Category,
		Description:  i.Description,
		AuthType:     i.AuthType,
		ToolCount:    len(i.Tools),
		ToolNames:    i.ToolNames(),
		RequiredPlan: i.RequiredPlan,
		Status:       i.Status,
		Version:      i.Version,
		HasOAuth:     i.OAuth != nil,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errorResp(code, message string) map[string]any {
	return map[string]any{"error": code, "message": message}
}

// userIDFromRequest extracts user_id from the request context.
// Mirrors the auth middleware pattern.
func userIDFromRequest(r *http.Request) string {
	return userIDFromContext(r.Context())
}
