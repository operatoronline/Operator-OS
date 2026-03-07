package agents

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/standardws/operator/pkg/users"
)

// API provides HTTP handlers for per-user agent configuration.
type API struct {
	store UserAgentStore
}

// NewAPI creates a new agents API with the given store.
func NewAPI(store UserAgentStore) *API {
	return &API{store: store}
}

// RegisterRoutes registers agent CRUD endpoints on the given ServeMux.
// All routes require AuthMiddleware to be applied upstream.
func (a *API) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler {
		return authMiddleware(fn)
	}

	mux.Handle("GET /api/v1/agents", wrap(a.handleList))
	mux.Handle("POST /api/v1/agents", wrap(a.handleCreate))
	mux.Handle("GET /api/v1/agents/{id}", wrap(a.handleGet))
	mux.Handle("PUT /api/v1/agents/{id}", wrap(a.handleUpdate))
	mux.Handle("DELETE /api/v1/agents/{id}", wrap(a.handleDelete))
	mux.Handle("POST /api/v1/agents/{id}/default", wrap(a.handleSetDefault))
}

// CreateAgentRequest is the JSON body for creating an agent.
type CreateAgentRequest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	SystemPrompt   string   `json:"system_prompt,omitempty"`
	Model          string   `json:"model,omitempty"`
	ModelFallbacks []string `json:"model_fallbacks,omitempty"`
	Tools          []string `json:"tools,omitempty"`
	Skills         []string `json:"skills,omitempty"`
	MaxTokens      int      `json:"max_tokens,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxIterations  int      `json:"max_iterations,omitempty"`
	IsDefault      bool     `json:"is_default,omitempty"`
}

// UpdateAgentRequest is the JSON body for updating an agent.
type UpdateAgentRequest struct {
	Name           *string  `json:"name,omitempty"`
	Description    *string  `json:"description,omitempty"`
	SystemPrompt   *string  `json:"system_prompt,omitempty"`
	Model          *string  `json:"model,omitempty"`
	ModelFallbacks []string `json:"model_fallbacks,omitempty"`
	Tools          []string `json:"tools,omitempty"`
	Skills         []string `json:"skills,omitempty"`
	MaxTokens      *int     `json:"max_tokens,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxIterations  *int     `json:"max_iterations,omitempty"`
	IsDefault      *bool    `json:"is_default,omitempty"`
	Status         *string  `json:"status,omitempty"`
}

// AgentResponse is the JSON response for agent operations.
type AgentResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	SystemPrompt   string   `json:"system_prompt,omitempty"`
	Model          string   `json:"model,omitempty"`
	ModelFallbacks []string `json:"model_fallbacks,omitempty"`
	Tools          []string `json:"tools,omitempty"`
	Skills         []string `json:"skills,omitempty"`
	MaxTokens      int      `json:"max_tokens,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxIterations  int      `json:"max_iterations,omitempty"`
	IsDefault      bool     `json:"is_default"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// AgentListResponse wraps a list of agents.
type AgentListResponse struct {
	Agents []*AgentResponse `json:"agents"`
	Count  int              `json:"count"`
}

// ErrorResponse is a standard error JSON response.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func (a *API) handleList(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	agents, err := a.store.ListByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "Failed to list agents")
		return
	}

	resp := AgentListResponse{
		Agents: make([]*AgentResponse, len(agents)),
		Count:  len(agents),
	}
	for i, ag := range agents {
		resp.Agents[i] = agentToResponse(ag)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) handleCreate(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Validate.
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name_required", ErrNameRequired.Error())
		return
	}
	if len(req.Name) > 100 {
		writeError(w, http.StatusBadRequest, "name_too_long", ErrNameTooLong.Error())
		return
	}
	if len(req.SystemPrompt) > 50000 {
		writeError(w, http.StatusBadRequest, "prompt_too_long", ErrPromptTooLong.Error())
		return
	}

	// Check agent count limit.
	count, err := a.store.CountByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "Failed to check agent count")
		return
	}
	if count >= int64(MaxAgentsPerUser) {
		writeError(w, http.StatusConflict, "max_agents", ErrMaxAgents.Error())
		return
	}

	agent := &UserAgent{
		UserID:         userID,
		Name:           req.Name,
		Description:    strings.TrimSpace(req.Description),
		SystemPrompt:   req.SystemPrompt,
		Model:          strings.TrimSpace(req.Model),
		ModelFallbacks: req.ModelFallbacks,
		Tools:          req.Tools,
		Skills:         req.Skills,
		MaxTokens:      req.MaxTokens,
		Temperature:    req.Temperature,
		MaxIterations:  req.MaxIterations,
		IsDefault:      req.IsDefault,
	}

	if err := a.store.Create(agent); err != nil {
		if errors.Is(err, ErrNameExists) {
			writeError(w, http.StatusConflict, "name_exists", "An agent with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "Failed to create agent")
		return
	}

	writeJSON(w, http.StatusCreated, agentToResponse(agent))
}

func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	agentID := r.PathValue("id")
	agent, err := a.store.GetByID(agentID)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Agent not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "Failed to retrieve agent")
		return
	}

	// Ensure the agent belongs to the authenticated user.
	if agent.UserID != userID {
		writeError(w, http.StatusNotFound, "not_found", "Agent not found")
		return
	}

	writeJSON(w, http.StatusOK, agentToResponse(agent))
}

func (a *API) handleUpdate(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	agentID := r.PathValue("id")
	agent, err := a.store.GetByID(agentID)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Agent not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "Failed to retrieve agent")
		return
	}

	if agent.UserID != userID {
		writeError(w, http.StatusNotFound, "not_found", "Agent not found")
		return
	}

	var req UpdateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}

	// Apply partial updates.
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			writeError(w, http.StatusBadRequest, "name_required", ErrNameRequired.Error())
			return
		}
		if len(name) > 100 {
			writeError(w, http.StatusBadRequest, "name_too_long", ErrNameTooLong.Error())
			return
		}
		agent.Name = name
	}
	if req.Description != nil {
		agent.Description = strings.TrimSpace(*req.Description)
	}
	if req.SystemPrompt != nil {
		if len(*req.SystemPrompt) > 50000 {
			writeError(w, http.StatusBadRequest, "prompt_too_long", ErrPromptTooLong.Error())
			return
		}
		agent.SystemPrompt = *req.SystemPrompt
	}
	if req.Model != nil {
		agent.Model = strings.TrimSpace(*req.Model)
	}
	if req.ModelFallbacks != nil {
		agent.ModelFallbacks = req.ModelFallbacks
	}
	if req.Tools != nil {
		agent.Tools = req.Tools
	}
	if req.Skills != nil {
		agent.Skills = req.Skills
	}
	if req.MaxTokens != nil {
		agent.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		agent.Temperature = req.Temperature
	}
	if req.MaxIterations != nil {
		agent.MaxIterations = *req.MaxIterations
	}
	if req.IsDefault != nil {
		agent.IsDefault = *req.IsDefault
	}
	if req.Status != nil {
		status := *req.Status
		if status != AgentStatusActive && status != AgentStatusArchived {
			writeError(w, http.StatusBadRequest, "invalid_status", ErrInvalidStatus.Error())
			return
		}
		agent.Status = status
	}

	if err := a.store.Update(agent); err != nil {
		if errors.Is(err, ErrNameExists) {
			writeError(w, http.StatusConflict, "name_exists", "An agent with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "Failed to update agent")
		return
	}

	writeJSON(w, http.StatusOK, agentToResponse(agent))
}

func (a *API) handleDelete(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	agentID := r.PathValue("id")
	agent, err := a.store.GetByID(agentID)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Agent not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "Failed to retrieve agent")
		return
	}

	if agent.UserID != userID {
		writeError(w, http.StatusNotFound, "not_found", "Agent not found")
		return
	}

	if err := a.store.Delete(agentID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "Failed to delete agent")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleSetDefault(w http.ResponseWriter, r *http.Request) {
	userID := users.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	agentID := r.PathValue("id")
	if err := a.store.SetDefault(userID, agentID); err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Agent not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "Failed to set default agent")
		return
	}

	// Return the updated agent.
	agent, err := a.store.GetByID(agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "Failed to retrieve agent")
		return
	}

	writeJSON(w, http.StatusOK, agentToResponse(agent))
}

func agentToResponse(a *UserAgent) *AgentResponse {
	return &AgentResponse{
		ID:             a.ID,
		Name:           a.Name,
		Description:    a.Description,
		SystemPrompt:   a.SystemPrompt,
		Model:          a.Model,
		ModelFallbacks: a.ModelFallbacks,
		Tools:          a.Tools,
		Skills:         a.Skills,
		MaxTokens:      a.MaxTokens,
		Temperature:    a.Temperature,
		MaxIterations:  a.MaxIterations,
		IsDefault:      a.IsDefault,
		Status:         a.Status,
		CreatedAt:      a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}
