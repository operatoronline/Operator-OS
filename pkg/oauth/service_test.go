package oauth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func testService(t *testing.T) (*Service, *ProviderRegistry, *sql.DB) {
	t.Helper()
	db := testDB(t)
	store, err := NewSQLiteStateStore(db)
	require.NoError(t, err)

	reg := NewProviderRegistry()
	svc, err := NewService(ServiceConfig{
		Registry:   reg,
		StateStore: store,
	})
	require.NoError(t, err)
	return svc, reg, db
}

func registerTestProvider(t *testing.T, reg *ProviderRegistry, tokenURL string) {
	t.Helper()
	require.NoError(t, reg.Register(&Provider{
		ID:          "test-provider",
		Name:        "Test",
		AuthURL:     "https://provider.example.com/authorize",
		TokenURL:    tokenURL,
		ClientID:    "test-client-id",
		RedirectURL: "https://app.example.com/api/v1/oauth/callback",
		Scopes:      []string{"openid", "email"},
		UsePKCE:     true,
	}))
}

func TestNewService_NilRegistry(t *testing.T) {
	_, err := NewService(ServiceConfig{StateStore: &SQLiteStateStore{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry is required")
}

func TestNewService_NilStore(t *testing.T) {
	_, err := NewService(ServiceConfig{Registry: NewProviderRegistry()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state store is required")
}

func TestNewService_OK(t *testing.T) {
	svc, _, _ := testService(t)
	require.NotNil(t, svc)
}

func TestStartFlow_EmptyUserID(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.StartFlow("", "google", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestStartFlow_EmptyProvider(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.StartFlow("user-1", "", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider ID is required")
}

func TestStartFlow_ProviderNotFound(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.StartFlow("user-1", "nonexistent", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStartFlow_Success(t *testing.T) {
	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, "https://example.com/token")

	result, err := svc.StartFlow("user-1", "test-provider", nil, "/dashboard")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.AuthURL, "https://provider.example.com/authorize")
	assert.Contains(t, result.AuthURL, "client_id=test-client-id")
	assert.Contains(t, result.AuthURL, "code_challenge=")
	assert.Contains(t, result.AuthURL, "code_challenge_method=S256")
	assert.Contains(t, result.AuthURL, "state="+result.State)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Len(t, result.State, 64)
}

func TestStartFlow_WithExtraScopes(t *testing.T) {
	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, "https://example.com/token")

	result, err := svc.StartFlow("user-1", "test-provider", []string{"drive.readonly"}, "")
	require.NoError(t, err)

	// Should contain both default and extra scopes.
	assert.Contains(t, result.AuthURL, "scope=")
}

func TestStartFlow_WithExtraAuthParams(t *testing.T) {
	svc, reg, _ := testService(t)
	require.NoError(t, reg.Register(&Provider{
		ID:          "google-extra",
		Name:        "Google",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		ClientID:    "client-id",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid"},
		UsePKCE:     true,
		ExtraAuthParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	}))

	result, err := svc.StartFlow("user-1", "google-extra", nil, "")
	require.NoError(t, err)
	assert.Contains(t, result.AuthURL, "access_type=offline")
	assert.Contains(t, result.AuthURL, "prompt=consent")
}

func TestStartFlow_NoPKCE(t *testing.T) {
	svc, reg, _ := testService(t)
	require.NoError(t, reg.Register(&Provider{
		ID:          "no-pkce",
		Name:        "No PKCE",
		AuthURL:     "https://example.com/auth",
		TokenURL:    "https://example.com/token",
		ClientID:    "client-id",
		RedirectURL: "https://app.example.com/callback",
		UsePKCE:     false,
	}))

	result, err := svc.StartFlow("user-1", "no-pkce", nil, "")
	require.NoError(t, err)
	assert.NotContains(t, result.AuthURL, "code_challenge")
}

func TestHandleCallback_EmptyState(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.HandleCallback("", "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state token is required")
}

func TestHandleCallback_EmptyCode(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.HandleCallback("state", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authorization code is required")
}

func TestHandleCallback_InvalidState(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.HandleCallback("nonexistent", "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or unknown")
}

func TestHandleCallback_ExpiredState(t *testing.T) {
	svc, reg, db := testService(t)
	registerTestProvider(t, reg, "https://example.com/token")

	// Manually insert an expired state.
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO oauth_states (id, user_id, provider_id, state, code_verifier, redirect_uri, scopes, created_at, expires_at, used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"exp-id", "user-1", "test-provider", "expired-state", "verifier", "", "",
		now.Add(-20*time.Minute).Format(time.RFC3339Nano),
		now.Add(-10*time.Minute).Format(time.RFC3339Nano),
		false,
	)
	require.NoError(t, err)

	_, err = svc.HandleCallback("expired-state", "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestHandleCallback_UsedState(t *testing.T) {
	svc, reg, db := testService(t)
	registerTestProvider(t, reg, "https://example.com/token")

	// Insert a used state.
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO oauth_states (id, user_id, provider_id, state, code_verifier, redirect_uri, scopes, created_at, expires_at, used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"used-id", "user-1", "test-provider", "used-state", "verifier", "", "",
		now.Format(time.RFC3339Nano),
		now.Add(10*time.Minute).Format(time.RFC3339Nano),
		true,
	)
	require.NoError(t, err)

	_, err = svc.HandleCallback("used-state", "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

func TestHandleCallback_Success(t *testing.T) {
	// Set up a mock token endpoint.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.Equal(t, "test-code", r.FormValue("code"))
		assert.NotEmpty(t, r.FormValue("code_verifier"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token-123",
			"refresh_token": "refresh-token-456",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "openid email",
		})
	}))
	defer tokenServer.Close()

	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, tokenServer.URL+"/token")

	// Start a flow.
	result, err := svc.StartFlow("user-1", "test-provider", nil, "/dashboard")
	require.NoError(t, err)

	// Handle callback.
	tokenResp, err := svc.HandleCallback(result.State, "test-code")
	require.NoError(t, err)
	require.NotNil(t, tokenResp)

	assert.Equal(t, "access-token-123", tokenResp.AccessToken)
	assert.Equal(t, "refresh-token-456", tokenResp.RefreshToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Equal(t, 3600, tokenResp.ExpiresIn)
	assert.Equal(t, "test-provider", tokenResp.ProviderID)
	assert.Equal(t, "user-1", tokenResp.UserID)
}

func TestHandleCallback_ReplayPrevention(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token",
			"token_type":   "Bearer",
		})
	}))
	defer tokenServer.Close()

	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, tokenServer.URL+"/token")

	result, err := svc.StartFlow("user-1", "test-provider", nil, "")
	require.NoError(t, err)

	// First callback succeeds.
	_, err = svc.HandleCallback(result.State, "code")
	require.NoError(t, err)

	// Second callback fails (replay prevention).
	_, err = svc.HandleCallback(result.State, "code")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

func TestRefreshToken_EmptyProvider(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.RefreshToken("", "refresh-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider ID is required")
}

func TestRefreshToken_EmptyToken(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.RefreshToken("google", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token is required")
}

func TestRefreshToken_ProviderNotFound(t *testing.T) {
	svc, _, _ := testService(t)
	_, err := svc.RefreshToken("nonexistent", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRefreshToken_Success(t *testing.T) {
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "old-refresh-token", r.FormValue("refresh_token"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer refreshServer.Close()

	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, refreshServer.URL+"/token")

	tokenResp, err := svc.RefreshToken("test-provider", "old-refresh-token")
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", tokenResp.AccessToken)
	assert.Equal(t, "new-refresh-token", tokenResp.RefreshToken)
	assert.Equal(t, "test-provider", tokenResp.ProviderID)
}

func TestRefreshToken_PreservesRefreshToken(t *testing.T) {
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Response without refresh_token (some providers don't rotate).
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer refreshServer.Close()

	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, refreshServer.URL+"/token")

	tokenResp, err := svc.RefreshToken("test-provider", "original-refresh")
	require.NoError(t, err)
	assert.Equal(t, "original-refresh", tokenResp.RefreshToken)
}

func TestRefreshToken_WithClientSecret(t *testing.T) {
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "test-secret", r.FormValue("client_secret"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token",
			"token_type":   "Bearer",
		})
	}))
	defer refreshServer.Close()

	svc, reg, _ := testService(t)
	require.NoError(t, reg.Register(&Provider{
		ID:           "confidential",
		Name:         "Confidential",
		AuthURL:      "https://example.com/auth",
		TokenURL:     refreshServer.URL + "/token",
		ClientID:     "client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/callback",
	}))

	_, err := svc.RefreshToken("confidential", "refresh-token")
	require.NoError(t, err)
}

func TestRefreshToken_ProviderError(t *testing.T) {
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_grant",
		})
	}))
	defer errServer.Close()

	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, errServer.URL+"/token")

	_, err := svc.RefreshToken("test-provider", "bad-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token refresh failed")
}

func TestParseTokenResponse_OAuthError(t *testing.T) {
	body := `{"error":"invalid_grant","error_description":"Token has expired"}`
	_, err := parseTokenResponse([]byte(body))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Token has expired")
}

func TestParseTokenResponse_NoAccessToken(t *testing.T) {
	body := `{"token_type":"Bearer"}`
	_, err := parseTokenResponse([]byte(body))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no access token")
}

func TestParseTokenResponse_InvalidJSON(t *testing.T) {
	_, err := parseTokenResponse([]byte("not json"))
	require.Error(t, err)
}

func TestParseTokenResponse_ExpiresAt(t *testing.T) {
	body := `{"access_token":"tok","token_type":"Bearer","expires_in":3600}`
	resp, err := parseTokenResponse([]byte(body))
	require.NoError(t, err)
	assert.False(t, resp.ExpiresAt.IsZero())
	assert.True(t, resp.ExpiresAt.After(time.Now()))
}

func TestMultiUserIsolation(t *testing.T) {
	svc, reg, _ := testService(t)
	registerTestProvider(t, reg, "https://example.com/token")

	// Two users start flows.
	r1, err := svc.StartFlow("user-1", "test-provider", nil, "")
	require.NoError(t, err)

	r2, err := svc.StartFlow("user-2", "test-provider", nil, "")
	require.NoError(t, err)

	// States are different.
	assert.NotEqual(t, r1.State, r2.State)
}
