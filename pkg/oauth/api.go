package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/standardws/operator/pkg/users"
)

// API provides HTTP handlers for OAuth 2.0 flows.
type API struct {
	service *Service
}

// NewAPI creates a new OAuth API handler.
func NewAPI(service *Service) (*API, error) {
	if service == nil {
		return nil, fmt.Errorf("service is required")
	}
	return &API{service: service}, nil
}

// RegisterRoutes registers OAuth API routes on the given mux.
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/oauth/providers", a.handleListProviders)
	mux.HandleFunc("POST /api/v1/oauth/authorize", a.handleAuthorize)
	mux.HandleFunc("GET /api/v1/oauth/callback", a.handleCallback)
	mux.HandleFunc("POST /api/v1/oauth/refresh", a.handleRefresh)
}

// handleListProviders returns available OAuth providers.
func (a *API) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := a.service.GetRegistry().List()

	type providerInfo struct {
		ID     string   `json:"id"`
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}

	result := make([]providerInfo, 0, len(providers))
	for _, p := range providers {
		result = append(result, providerInfo{
			ID:     p.ID,
			Name:   p.Name,
			Scopes: p.Scopes,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"providers": result})
}

// handleAuthorize initiates an OAuth flow for the authenticated user.
func (a *API) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "unauthorized",
		})
		return
	}

	var req struct {
		Provider      string   `json:"provider"`
		Scopes        []string `json:"scopes"`
		RedirectAfter string   `json:"redirect_after"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid_json",
		})
		return
	}

	if req.Provider == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "missing_provider",
		})
		return
	}

	result, err := a.service.StartFlow(userID, req.Provider, req.Scopes, req.RedirectAfter)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":   "provider_not_found",
				"message": fmt.Sprintf("Provider %q is not configured", req.Provider),
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "flow_start_failed",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleCallback processes the OAuth provider callback.
func (a *API) handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errParam := r.URL.Query().Get("error")

	if errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		if errDesc == "" {
			errDesc = errParam
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "provider_error",
			"message": errDesc,
		})
		return
	}

	if state == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "missing_state",
		})
		return
	}

	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "missing_code",
		})
		return
	}

	tokenResp, err := a.service.HandleCallback(state, code)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "callback_failed"
		if strings.Contains(err.Error(), "invalid or unknown") ||
			strings.Contains(err.Error(), "expired") ||
			strings.Contains(err.Error(), "already used") {
			status = http.StatusBadRequest
			errorCode = "invalid_state"
		}
		writeJSON(w, status, map[string]any{
			"error":   errorCode,
			"message": err.Error(),
		})
		return
	}

	// Return tokens (in production, you'd store these in the vault
	// and redirect the user, but the API layer handles that).
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"token_type":    tokenResp.TokenType,
		"expires_in":    tokenResp.ExpiresIn,
		"scope":         tokenResp.Scope,
		"provider":      tokenResp.ProviderID,
		"user_id":       tokenResp.UserID,
	})
}

// handleRefresh exchanges a refresh token for a new access token.
func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error": "unauthorized",
		})
		return
	}

	var req struct {
		Provider     string `json:"provider"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid_json",
		})
		return
	}

	if req.Provider == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "missing_provider",
		})
		return
	}
	if req.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "missing_refresh_token",
		})
		return
	}

	tokenResp, err := a.service.RefreshToken(req.Provider, req.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error":   "provider_not_found",
				"message": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":   "refresh_failed",
			"message": err.Error(),
		})
		return
	}

	tokenResp.UserID = userID

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"token_type":    tokenResp.TokenType,
		"expires_in":    tokenResp.ExpiresIn,
		"scope":         tokenResp.Scope,
		"provider":      tokenResp.ProviderID,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
