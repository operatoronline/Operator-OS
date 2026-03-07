package integrations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAPI(t *testing.T) (*API, *http.ServeMux) {
	t.Helper()
	db := openTestDB(t)
	t.Cleanup(func() { db.Close() })

	store, err := NewSQLiteUserIntegrationStore(db)
	require.NoError(t, err)

	reg := NewIntegrationRegistry()
	g := validIntegration("google")
	g.Category = "email"
	s := validIntegration("shopify")
	s.Category = "ecommerce"
	require.NoError(t, reg.Register(g))
	require.NoError(t, reg.Register(s))

	api := NewAPI(reg, store)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	return api, mux
}

func authRequest(method, path string, body any) *http.Request {
	var buf *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		buf = bytes.NewBuffer(data)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req = req.WithContext(WithUserID(req.Context(), "test-user"))
	return req
}

func unauthRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

// --- List integrations ---

func TestAPI_ListIntegrations(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/integrations", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])
	integrations := resp["integrations"].([]any)
	// Verify secrets are not exposed
	for _, i := range integrations {
		integ := i.(map[string]any)
		_, hasOAuth := integ["oauth"]
		assert.False(t, hasOAuth, "OAuth config should not be in summary")
		assert.NotEmpty(t, integ["tool_names"])
	}
}

func TestAPI_ListIntegrations_ByCategory(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/integrations?category=email", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["count"])
}

func TestAPI_ListIntegrations_MethodNotAllowed(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/integrations", nil))
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- Categories ---

func TestAPI_Categories(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/integrations/categories", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	cats := resp["categories"].([]any)
	assert.Len(t, cats, 2)
}

// --- Get integration by ID ---

func TestAPI_GetIntegration(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/integrations/google", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "google", resp["id"])
}

func TestAPI_GetIntegration_NotFound(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/integrations/nonexistent", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- User integrations: List ---

func TestAPI_UserIntegrations_List_Empty(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("GET", "/api/v1/user/integrations", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["count"])
}

func TestAPI_UserIntegrations_List_Unauthorized(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, unauthRequest("GET", "/api/v1/user/integrations"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- User integrations: Create ---

func TestAPI_UserIntegrations_Create_Success(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	body := map[string]any{
		"integration_id": "google",
		"config":         map[string]string{"domain": "gmail.com"},
		"scopes":         []string{"email"},
	}
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", body))
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp UserIntegration
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "google", resp.IntegrationID)
	assert.Equal(t, UserIntegrationPending, resp.Status)
}

func TestAPI_UserIntegrations_Create_MissingIntegrationID(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", map[string]any{}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_UserIntegrations_Create_IntegrationNotFound(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	body := map[string]any{"integration_id": "nonexistent"}
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", body))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_UserIntegrations_Create_Duplicate(t *testing.T) {
	_, mux := setupAPI(t)
	body := map[string]any{"integration_id": "google"}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", body))
	assert.Equal(t, http.StatusCreated, w.Code)

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", body))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAPI_UserIntegrations_Create_InvalidJSON(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	req := authRequest("POST", "/api/v1/user/integrations", nil)
	req.Body = http.NoBody
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_UserIntegrations_Create_Unauthorized(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, unauthRequest("POST", "/api/v1/user/integrations"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- User integrations: Get by ID ---

func TestAPI_UserIntegration_Get_Success(t *testing.T) {
	_, mux := setupAPI(t)
	// Create first
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", map[string]any{"integration_id": "google"}))
	require.Equal(t, http.StatusCreated, w.Code)

	// Get
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("GET", "/api/v1/user/integrations/google", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var resp UserIntegration
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "google", resp.IntegrationID)
}

func TestAPI_UserIntegration_Get_NotFound(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("GET", "/api/v1/user/integrations/nonexistent", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- User integrations: Delete ---

func TestAPI_UserIntegration_Delete_Success(t *testing.T) {
	_, mux := setupAPI(t)
	// Create first
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("POST", "/api/v1/user/integrations", map[string]any{"integration_id": "google"}))
	require.Equal(t, http.StatusCreated, w.Code)

	// Delete
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("DELETE", "/api/v1/user/integrations/google", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify gone
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("GET", "/api/v1/user/integrations/google", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_UserIntegration_Delete_NotFound(t *testing.T) {
	_, mux := setupAPI(t)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("DELETE", "/api/v1/user/integrations/nonexistent", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- No store configured ---

func TestAPI_UserIntegrations_NoStore(t *testing.T) {
	reg := NewIntegrationRegistry()
	api := NewAPI(reg, nil)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authRequest("GET", "/api/v1/user/integrations", nil))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- RegisterRoutes ---

func TestAPI_RegisterRoutes(t *testing.T) {
	reg := NewIntegrationRegistry()
	api := NewAPI(reg, nil)
	mux := http.NewServeMux()
	// Should not panic
	api.RegisterRoutes(mux)
}
