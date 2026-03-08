package gdpr

import (
	"encoding/json"
	"net/http"
)

// userIDFromContext extracts the user ID from the request context.
// It checks for the key used by the auth middleware.
func userIDFromContext(r *http.Request) string {
	if v := r.Context().Value(contextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// contextKeyType is a private type to avoid context key collisions.
type contextKeyType string

const contextKeyUserID contextKeyType = "user_id"

// API provides HTTP handlers for GDPR compliance endpoints.
type API struct {
	service *Service
}

// NewAPI creates a new GDPR API handler.
func NewAPI(service *Service) *API {
	return &API{service: service}
}

// RegisterRoutes registers GDPR API endpoints on the given mux.
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/gdpr/export", a.handleExport)
	mux.HandleFunc("/api/v1/gdpr/erase", a.handleErase)
	mux.HandleFunc("/api/v1/gdpr/requests", a.handleRequests)
	mux.HandleFunc("/api/v1/gdpr/requests/", a.handleRequestByID)
	mux.HandleFunc("/api/v1/gdpr/retention", a.handleRetention)
}

// handleExport initiates a data export for the authenticated user.
func (a *API) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GDPR service not configured"})
		return
	}
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	export, err := a.service.ExportUserData(r.Context(), userID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, export)
}

// handleErase initiates data erasure for the authenticated user.
func (a *API) handleErase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GDPR service not configured"})
		return
	}
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Parse optional confirmation
	var body struct {
		Confirm bool `json:"confirm"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if !body.Confirm {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "confirmation required",
			"message": "Set confirm=true to proceed with data erasure. This action is irreversible.",
		})
		return
	}

	report, err := a.service.EraseUserData(r.Context(), userID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// handleRequests lists the user's DSRs.
func (a *API) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GDPR service not configured"})
		return
	}
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	requests, err := a.service.ListUserRequests(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if requests == nil {
		requests = []*DataSubjectRequest{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"requests": requests,
		"count":    len(requests),
	})
}

// handleRequestByID returns or cancels a specific DSR.
func (a *API) handleRequestByID(w http.ResponseWriter, r *http.Request) {
	if a.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GDPR service not configured"})
		return
	}
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Extract ID from path: /api/v1/gdpr/requests/{id}
	id := r.URL.Path[len("/api/v1/gdpr/requests/"):]
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request ID required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		req, err := a.service.GetRequest(id)
		if err != nil {
			if err == ErrRequestNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// Ensure user can only see their own requests
		if req.UserID != userID {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found"})
			return
		}
		writeJSON(w, http.StatusOK, req)

	case http.MethodDelete:
		// Cancel a pending request
		req, err := a.service.GetRequest(id)
		if err != nil {
			if err == ErrRequestNotFound {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if req.UserID != userID {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found"})
			return
		}
		if err := a.service.CancelRequest(id); err != nil {
			if err == ErrAlreadyProcessed {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "request already processed"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "canceled"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleRetention returns the current retention policy.
func (a *API) handleRetention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if a.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GDPR service not configured"})
		return
	}
	userID := userIDFromContext(r)
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	type retentionResponse struct {
		AuditLogDays    int `json:"audit_log_days"`
		UsageDataDays   int `json:"usage_data_days"`
		SessionDays     int `json:"session_days"`
		DeletedUserDays int `json:"deleted_user_days"`
	}

	resp := retentionResponse{
		AuditLogDays:    int(a.service.retention.AuditLogRetention.Hours() / 24),
		UsageDataDays:   int(a.service.retention.UsageDataRetention.Hours() / 24),
		SessionDays:     int(a.service.retention.SessionRetention.Hours() / 24),
		DeletedUserDays: int(a.service.retention.DeletedUserRetention.Hours() / 24),
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
