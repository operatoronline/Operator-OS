package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/standardws/operator/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuthMiddleware injects a user ID into the request context for testing.
func fakeAuthMiddleware(userID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), users.ContextKeyUserID(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setupAPI(t *testing.T) (*API, *http.ServeMux) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteUserAgentStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	api := NewAPI(store)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, fakeAuthMiddleware("user-1"))
	return api, mux
}

func TestAPICreateAgent(t *testing.T) {
	_, mux := setupAPI(t)

	body := `{"name":"Test Agent","description":"A test","system_prompt":"Be helpful","model":"gpt-4","tools":["read_file"]}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "Test Agent", resp.Name)
	assert.Equal(t, "A test", resp.Description)
	assert.Equal(t, "Be helpful", resp.SystemPrompt)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, []string{"read_file"}, resp.Tools)
	assert.Equal(t, AgentStatusActive, resp.Status)
}

func TestAPICreateAgentMissingName(t *testing.T) {
	_, mux := setupAPI(t)

	body := `{"description":"No name"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name_required")
}

func TestAPICreateAgentNameTooLong(t *testing.T) {
	_, mux := setupAPI(t)

	longName := strings.Repeat("a", 101)
	body := `{"name":"` + longName + `"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name_too_long")
}

func TestAPICreateAgentDuplicate(t *testing.T) {
	_, mux := setupAPI(t)

	body := `{"name":"Agent"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Duplicate.
	req = httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "name_exists")
}

func TestAPICreateAgentInvalidJSON(t *testing.T) {
	_, mux := setupAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_json")
}

func TestAPIGetAgent(t *testing.T) {
	_, mux := setupAPI(t)

	// Create first.
	body := `{"name":"GetMe"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))

	// Get it.
	req = httptest.NewRequest("GET", "/api/v1/agents/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var got AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "GetMe", got.Name)
}

func TestAPIGetAgentNotFound(t *testing.T) {
	_, mux := setupAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/agents/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIListAgents(t *testing.T) {
	_, mux := setupAPI(t)

	// Create two.
	for _, name := range []string{"Agent1", "Agent2"} {
		body := `{"name":"` + name + `"}`
		req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AgentListResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Count)
	assert.Len(t, resp.Agents, 2)
}

func TestAPIListAgentsEmpty(t *testing.T) {
	_, mux := setupAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AgentListResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Count)
	assert.Empty(t, resp.Agents)
}

func TestAPIUpdateAgent(t *testing.T) {
	_, mux := setupAPI(t)

	// Create.
	body := `{"name":"Original","model":"gpt-4"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))

	// Update partial fields.
	updateBody := map[string]any{
		"name":  "Updated",
		"model": "claude-3",
		"tools": []string{"exec", "read_file"},
	}
	b, _ := json.Marshal(updateBody)
	req = httptest.NewRequest("PUT", "/api/v1/agents/"+created.ID, bytes.NewReader(b))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var updated AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&updated))
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "claude-3", updated.Model)
	assert.Equal(t, []string{"exec", "read_file"}, updated.Tools)
}

func TestAPIUpdateAgentNotFound(t *testing.T) {
	_, mux := setupAPI(t)

	body := `{"name":"New"}`
	req := httptest.NewRequest("PUT", "/api/v1/agents/nonexistent", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIUpdateAgentInvalidStatus(t *testing.T) {
	_, mux := setupAPI(t)

	body := `{"name":"Agent"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))

	updateBody := `{"status":"invalid"}`
	req = httptest.NewRequest("PUT", "/api/v1/agents/"+created.ID, strings.NewReader(updateBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_status")
}

func TestAPIDeleteAgent(t *testing.T) {
	_, mux := setupAPI(t)

	// Create.
	body := `{"name":"ToDelete"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))

	// Delete.
	req = httptest.NewRequest("DELETE", "/api/v1/agents/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify gone.
	req = httptest.NewRequest("GET", "/api/v1/agents/"+created.ID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIDeleteAgentNotFound(t *testing.T) {
	_, mux := setupAPI(t)

	req := httptest.NewRequest("DELETE", "/api/v1/agents/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPISetDefault(t *testing.T) {
	_, mux := setupAPI(t)

	// Create two agents.
	body1 := `{"name":"Agent1","is_default":true}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body1))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var a1 AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&a1))

	body2 := `{"name":"Agent2"}`
	req = httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body2))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var a2 AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&a2))

	// Set a2 as default.
	req = httptest.NewRequest("POST", "/api/v1/agents/"+a2.ID+"/default", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.IsDefault)
	assert.Equal(t, a2.ID, resp.ID)
}

func TestAPISetDefaultNotFound(t *testing.T) {
	_, mux := setupAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/agents/nonexistent/default", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPICreateAgentWithTemperature(t *testing.T) {
	_, mux := setupAPI(t)

	body := `{"name":"Temp Agent","temperature":0.3}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp AgentResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Temperature)
	assert.InDelta(t, 0.3, *resp.Temperature, 0.001)
}

func TestAPIPromptTooLong(t *testing.T) {
	_, mux := setupAPI(t)

	longPrompt := strings.Repeat("x", 50001)
	body := `{"name":"Long","system_prompt":"` + longPrompt + `"}`
	req := httptest.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "prompt_too_long")
}
