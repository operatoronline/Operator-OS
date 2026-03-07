// Package agents provides per-user agent configuration management for
// Operator OS. It defines a UserAgentStore interface for pluggable backends
// and includes an SQLite implementation, along with HTTP API handlers
// for authenticated CRUD operations.
package agents

import (
	"errors"
	"time"
)

// UserAgent represents a user-defined agent configuration.
type UserAgent struct {
	ID             string   `json:"id"`
	UserID         string   `json:"user_id"`
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
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Agent status constants.
const (
	AgentStatusActive   = "active"
	AgentStatusArchived = "archived"
)

// Errors.
var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrNameExists    = errors.New("agent name already exists for this user")
	ErrNameRequired  = errors.New("agent name is required")
	ErrNameTooLong   = errors.New("agent name must be 100 characters or fewer")
	ErrPromptTooLong = errors.New("system prompt must be 50000 characters or fewer")
	ErrInvalidStatus = errors.New("invalid agent status")
	ErrMaxAgents     = errors.New("maximum number of agents reached")
)

// MaxAgentsPerUser is the default limit on agents a single user can create.
const MaxAgentsPerUser = 50

// UserAgentStore abstracts per-user agent configuration persistence.
// Implementations must be safe for concurrent use.
type UserAgentStore interface {
	// Create inserts a new agent. Returns ErrNameExists if the user already
	// has an agent with the same name.
	Create(agent *UserAgent) error
	// GetByID returns the agent with the given ID, or ErrAgentNotFound.
	GetByID(id string) (*UserAgent, error)
	// Update saves changes to an existing agent. Returns ErrAgentNotFound
	// if the agent does not exist.
	Update(agent *UserAgent) error
	// Delete removes an agent by ID. Returns ErrAgentNotFound if missing.
	Delete(id string) error
	// ListByUser returns all agents for a user, ordered by created_at ascending.
	ListByUser(userID string) ([]*UserAgent, error)
	// CountByUser returns the number of agents a user has.
	CountByUser(userID string) (int64, error)
	// GetDefault returns the user's default agent, or ErrAgentNotFound.
	GetDefault(userID string) (*UserAgent, error)
	// SetDefault marks one agent as default and clears the flag on others.
	SetDefault(userID, agentID string) error
	// Close releases any resources held by the store.
	Close() error
}
