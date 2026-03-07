package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/standardws/operator/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func testAPI(t *testing.T) (*API, *ProviderRegistry) {
	t.Helper()
	svc, reg, _ := testService(t)
	api, err := NewAPI(svc)
	require.NoError(t, err)
	return api, reg
}

func withUser(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), users.ContextKeyUserID(), userID)
	return r.WithContext(ctx)
}

func TestNewAPI_NilService(t *testing.T) {
	_, err := NewAPI(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service is required")
}

func TestAPI_ListProviders_Empty(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/oauth/providers", nil)
	w := httptest.NewRecorder()
	api.handleListProviders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	providers := resp["providers"].([]any)
	assert.Empty(t, providers)
}

func TestAPI_ListProviders_WithProviders(t *testing.T) {
	api, reg := testAPI(t)
	registerTestProvider(t, reg, "https://example.com/token")

	req := httptest.NewRequest("GET", "/api/v1/oauth/providers", nil)
	w := httptest.NewRecorder()
	api.handleListProviders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	providers := resp["providers"].([]any)
	assert.Len(t, providers, 1)

	p := providers[0].(map[string]any)
	assert.Equal(t, "test-provider", p["id"])
	assert.Equal(t, "Test", p["name"])
	// Should NOT expose client_id, client_secret, auth_url, token_url.
	assert.Nil(t, p["client_id"])
	assert.Nil(t, p["client_secret"])
}

func TestAPI_Authorize_Unauthorized(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"provider":"google"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/authorize", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.handleAuthorize(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI_Authorize_InvalidJSON(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/oauth/authorize", bytes.NewBufferString("not json"))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleAuthorize(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Authorize_MissingProvider(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"scopes":["email"]}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/authorize", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleAuthorize(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "missing_provider", resp["error"])
}

func TestAPI_Authorize_ProviderNotFound(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"provider":"nonexistent"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/authorize", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleAuthorize(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Authorize_Success(t *testing.T) {
	api, reg := testAPI(t)
	registerTestProvider(t, reg, "https://example.com/token")

	body := `{"provider":"test-provider","redirect_after":"/dashboard"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/authorize", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleAuthorize(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp["auth_url"], "https://provider.example.com/authorize")
	assert.NotEmpty(t, resp["state"])
	assert.Equal(t, "test-provider", resp["provider"])
}

func TestAPI_Callback_MissingState(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/oauth/callback?code=abc", nil)
	w := httptest.NewRecorder()
	api.handleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "missing_state", resp["error"])
}

func TestAPI_Callback_MissingCode(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/oauth/callback?state=abc", nil)
	w := httptest.NewRecorder()
	api.handleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "missing_code", resp["error"])
}

func TestAPI_Callback_ProviderError(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/oauth/callback?error=access_denied&error_description=User+denied", nil)
	w := httptest.NewRecorder()
	api.handleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "provider_error", resp["error"])
	assert.Equal(t, "User denied", resp["message"])
}

func TestAPI_Callback_InvalidState(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/oauth/callback?state=bad&code=abc", nil)
	w := httptest.NewRecorder()
	api.handleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Callback_Success(t *testing.T) {
	// Mock token server.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-123",
			"refresh_token": "refresh-456",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "openid email",
		})
	}))
	defer tokenServer.Close()

	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)
	reg := NewProviderRegistry()
	svc, _ := NewService(ServiceConfig{
		Registry:   reg,
		StateStore: store,
	})
	api, _ := NewAPI(svc)

	registerTestProvider(t, reg, tokenServer.URL+"/token")

	// Start flow to get a valid state.
	result, err := svc.StartFlow("user-1", "test-provider", nil, "")
	require.NoError(t, err)

	// Simulate callback.
	req := httptest.NewRequest("GET", "/api/v1/oauth/callback?state="+result.State+"&code=auth-code", nil)
	w := httptest.NewRecorder()
	api.handleCallback(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "access-123", resp["access_token"])
	assert.Equal(t, "refresh-456", resp["refresh_token"])
	assert.Equal(t, "test-provider", resp["provider"])
	assert.Equal(t, "user-1", resp["user_id"])
}

func TestAPI_Refresh_Unauthorized(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"provider":"google","refresh_token":"token"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/refresh", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPI_Refresh_InvalidJSON(t *testing.T) {
	api, _ := testAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/oauth/refresh", bytes.NewBufferString("bad"))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Refresh_MissingProvider(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"refresh_token":"token"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/refresh", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "missing_provider", resp["error"])
}

func TestAPI_Refresh_MissingToken(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"provider":"google"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/refresh", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Refresh_ProviderNotFound(t *testing.T) {
	api, _ := testAPI(t)

	body := `{"provider":"nonexistent","refresh_token":"token"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/refresh", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Refresh_Success(t *testing.T) {
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer refreshServer.Close()

	db := testDB(t)
	store, _ := NewSQLiteStateStore(db)
	reg := NewProviderRegistry()
	svc, _ := NewService(ServiceConfig{
		Registry:   reg,
		StateStore: store,
	})
	api, _ := NewAPI(svc)
	registerTestProvider(t, reg, refreshServer.URL+"/token")

	body := `{"provider":"test-provider","refresh_token":"old-token"}`
	req := httptest.NewRequest("POST", "/api/v1/oauth/refresh", bytes.NewBufferString(body))
	req = withUser(req, "user-1")
	w := httptest.NewRecorder()
	api.handleRefresh(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "new-access", resp["access_token"])
	assert.Equal(t, "test-provider", resp["provider"])
}

func TestAPI_RegisterRoutes(t *testing.T) {
	api, _ := testAPI(t)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Verify routes are registered by making requests.
	req := httptest.NewRequest("GET", "/api/v1/oauth/providers", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
