package agents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SQLiteUserAgentStore implements UserAgentStore backed by SQLite.
type SQLiteUserAgentStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteUserAgentStore opens (or creates) a SQLite database at dbPath and
// initialises the user_agents schema.
func NewSQLiteUserAgentStore(dbPath string) (*SQLiteUserAgentStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)

	if err := initUserAgentsSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init user_agents schema: %w", err)
	}

	return &SQLiteUserAgentStore{db: db}, nil
}

// NewSQLiteUserAgentStoreFromDB wraps an existing *sql.DB connection.
// The caller is responsible for schema initialization.
func NewSQLiteUserAgentStoreFromDB(db *sql.DB) *SQLiteUserAgentStore {
	return &SQLiteUserAgentStore{db: db}
}

func initUserAgentsSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS user_agents (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    system_prompt   TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL DEFAULT '',
    model_fallbacks TEXT NOT NULL DEFAULT '[]',
    tools           TEXT NOT NULL DEFAULT '[]',
    skills          TEXT NOT NULL DEFAULT '[]',
    max_tokens      INTEGER NOT NULL DEFAULT 0,
    temperature     REAL DEFAULT NULL,
    max_iterations  INTEGER NOT NULL DEFAULT 0,
    is_default      INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'active',
    allowed_integrations TEXT NOT NULL DEFAULT '[]',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_user_agents_user_id ON user_agents(user_id);
CREATE INDEX IF NOT EXISTS idx_user_agents_user_default ON user_agents(user_id, is_default);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_agents_user_name ON user_agents(user_id, name);
`
	_, err := db.Exec(schema)
	return err
}

// Create inserts a new agent. Generates a UUID if agent.ID is empty.
func (s *SQLiteUserAgentStore) Create(agent *UserAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}

	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now

	if agent.Status == "" {
		agent.Status = AgentStatusActive
	}

	isDefault := 0
	if agent.IsDefault {
		isDefault = 1
	}

	fallbacksJSON := marshalStringSlice(agent.ModelFallbacks)
	toolsJSON := marshalStringSlice(agent.Tools)
	skillsJSON := marshalStringSlice(agent.Skills)
	integrationsJSON := marshalIntegrationScopes(agent.AllowedIntegrations)

	var tempVal sql.NullFloat64
	if agent.Temperature != nil {
		tempVal = sql.NullFloat64{Float64: *agent.Temperature, Valid: true}
	}

	_, err := s.db.Exec(
		`INSERT INTO user_agents (id, user_id, name, description, system_prompt, model,
		 model_fallbacks, tools, skills, max_tokens, temperature, max_iterations,
		 is_default, status, allowed_integrations, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID,
		agent.UserID,
		agent.Name,
		agent.Description,
		agent.SystemPrompt,
		agent.Model,
		fallbacksJSON,
		toolsJSON,
		skillsJSON,
		agent.MaxTokens,
		tempVal,
		agent.MaxIterations,
		isDefault,
		agent.Status,
		integrationsJSON,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameExists
		}
		return fmt.Errorf("insert agent: %w", err)
	}

	return nil
}

// GetByID returns the agent with the given ID.
func (s *SQLiteUserAgentStore) GetByID(id string) (*UserAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getBy("id", id)
}

func (s *SQLiteUserAgentStore) getBy(column, value string) (*UserAgent, error) {
	query := fmt.Sprintf(
		`SELECT id, user_id, name, description, system_prompt, model,
		 model_fallbacks, tools, skills, max_tokens, temperature, max_iterations,
		 is_default, status, allowed_integrations, created_at, updated_at
		 FROM user_agents WHERE %s = ?`, column,
	)

	var a UserAgent
	var isDefault int
	var fallbacksJSON, toolsJSON, skillsJSON string
	var integrationsJSON sql.NullString
	var tempVal sql.NullFloat64
	var createdStr, updatedStr string

	err := s.db.QueryRow(query, value).Scan(
		&a.ID, &a.UserID, &a.Name, &a.Description, &a.SystemPrompt, &a.Model,
		&fallbacksJSON, &toolsJSON, &skillsJSON,
		&a.MaxTokens, &tempVal, &a.MaxIterations,
		&isDefault, &a.Status, &integrationsJSON, &createdStr, &updatedStr,
	)
	if err == sql.ErrNoRows {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query agent by %s: %w", column, err)
	}

	a.IsDefault = isDefault == 1
	a.ModelFallbacks = unmarshalStringSlice(fallbacksJSON)
	a.Tools = unmarshalStringSlice(toolsJSON)
	a.Skills = unmarshalStringSlice(skillsJSON)
	if integrationsJSON.Valid {
		a.AllowedIntegrations = unmarshalIntegrationScopes(integrationsJSON.String)
	}
	if tempVal.Valid {
		a.Temperature = &tempVal.Float64
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	a.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)

	return &a, nil
}

// Update saves changes to an existing agent.
func (s *SQLiteUserAgentStore) Update(agent *UserAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent.UpdatedAt = time.Now()

	isDefault := 0
	if agent.IsDefault {
		isDefault = 1
	}

	fallbacksJSON := marshalStringSlice(agent.ModelFallbacks)
	toolsJSON := marshalStringSlice(agent.Tools)
	skillsJSON := marshalStringSlice(agent.Skills)
	integrationsJSON := marshalIntegrationScopes(agent.AllowedIntegrations)

	var tempVal sql.NullFloat64
	if agent.Temperature != nil {
		tempVal = sql.NullFloat64{Float64: *agent.Temperature, Valid: true}
	}

	res, err := s.db.Exec(
		`UPDATE user_agents SET name = ?, description = ?, system_prompt = ?, model = ?,
		 model_fallbacks = ?, tools = ?, skills = ?, max_tokens = ?, temperature = ?,
		 max_iterations = ?, is_default = ?, status = ?, allowed_integrations = ?, updated_at = ?
		 WHERE id = ?`,
		agent.Name,
		agent.Description,
		agent.SystemPrompt,
		agent.Model,
		fallbacksJSON,
		toolsJSON,
		skillsJSON,
		agent.MaxTokens,
		tempVal,
		agent.MaxIterations,
		isDefault,
		agent.Status,
		integrationsJSON,
		agent.UpdatedAt.Format(time.RFC3339Nano),
		agent.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrNameExists
		}
		return fmt.Errorf("update agent: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAgentNotFound
	}
	return nil
}

// Delete removes an agent by ID.
func (s *SQLiteUserAgentStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`DELETE FROM user_agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAgentNotFound
	}
	return nil
}

// ListByUser returns all agents for a user, ordered by created_at ascending.
func (s *SQLiteUserAgentStore) ListByUser(userID string) ([]*UserAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, user_id, name, description, system_prompt, model,
		 model_fallbacks, tools, skills, max_tokens, temperature, max_iterations,
		 is_default, status, allowed_integrations, created_at, updated_at
		 FROM user_agents WHERE user_id = ? ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*UserAgent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents: %w", err)
	}

	if agents == nil {
		agents = []*UserAgent{}
	}
	return agents, nil
}

// CountByUser returns the number of agents a user has.
func (s *SQLiteUserAgentStore) CountByUser(userID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM user_agents WHERE user_id = ?`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count agents: %w", err)
	}
	return count, nil
}

// GetDefault returns the user's default agent.
func (s *SQLiteUserAgentStore) GetDefault(userID string) (*UserAgent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var a UserAgent
	var isDefault int
	var fallbacksJSON, toolsJSON, skillsJSON string
	var tempVal sql.NullFloat64
	var createdStr, updatedStr string

	var integrationsJSON sql.NullString
	err := s.db.QueryRow(
		`SELECT id, user_id, name, description, system_prompt, model,
		 model_fallbacks, tools, skills, max_tokens, temperature, max_iterations,
		 is_default, status, allowed_integrations, created_at, updated_at
		 FROM user_agents WHERE user_id = ? AND is_default = 1`, userID,
	).Scan(
		&a.ID, &a.UserID, &a.Name, &a.Description, &a.SystemPrompt, &a.Model,
		&fallbacksJSON, &toolsJSON, &skillsJSON,
		&a.MaxTokens, &tempVal, &a.MaxIterations,
		&isDefault, &a.Status, &integrationsJSON, &createdStr, &updatedStr,
	)
	if err == sql.ErrNoRows {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query default agent: %w", err)
	}

	a.IsDefault = true
	a.ModelFallbacks = unmarshalStringSlice(fallbacksJSON)
	a.Tools = unmarshalStringSlice(toolsJSON)
	a.Skills = unmarshalStringSlice(skillsJSON)
	if integrationsJSON.Valid {
		a.AllowedIntegrations = unmarshalIntegrationScopes(integrationsJSON.String)
	}
	if tempVal.Valid {
		a.Temperature = &tempVal.Float64
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	a.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)

	return &a, nil
}

// SetDefault marks one agent as default and clears the flag on all others
// for the same user. Both operations run in a transaction.
func (s *SQLiteUserAgentStore) SetDefault(userID, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Verify agent exists and belongs to user.
	var ownerID string
	err = tx.QueryRow(`SELECT user_id FROM user_agents WHERE id = ?`, agentID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return ErrAgentNotFound
	}
	if err != nil {
		return fmt.Errorf("check agent owner: %w", err)
	}
	if ownerID != userID {
		return ErrAgentNotFound
	}

	// Clear all defaults for this user.
	if _, err := tx.Exec(`UPDATE user_agents SET is_default = 0 WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("clear defaults: %w", err)
	}

	// Set the new default.
	if _, err := tx.Exec(`UPDATE user_agents SET is_default = 1 WHERE id = ?`, agentID); err != nil {
		return fmt.Errorf("set default: %w", err)
	}

	return tx.Commit()
}

// Close closes the underlying database connection.
func (s *SQLiteUserAgentStore) Close() error {
	return s.db.Close()
}

// scanAgent scans a single row into a UserAgent.
func scanAgent(rows *sql.Rows) (*UserAgent, error) {
	var a UserAgent
	var isDefault int
	var fallbacksJSON, toolsJSON, skillsJSON string
	var integrationsJSON sql.NullString
	var tempVal sql.NullFloat64
	var createdStr, updatedStr string

	if err := rows.Scan(
		&a.ID, &a.UserID, &a.Name, &a.Description, &a.SystemPrompt, &a.Model,
		&fallbacksJSON, &toolsJSON, &skillsJSON,
		&a.MaxTokens, &tempVal, &a.MaxIterations,
		&isDefault, &a.Status, &integrationsJSON, &createdStr, &updatedStr,
	); err != nil {
		return nil, fmt.Errorf("scan agent: %w", err)
	}

	a.IsDefault = isDefault == 1
	a.ModelFallbacks = unmarshalStringSlice(fallbacksJSON)
	a.Tools = unmarshalStringSlice(toolsJSON)
	a.Skills = unmarshalStringSlice(skillsJSON)
	if integrationsJSON.Valid {
		a.AllowedIntegrations = unmarshalIntegrationScopes(integrationsJSON.String)
	}
	if tempVal.Valid {
		a.Temperature = &tempVal.Float64
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	a.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)

	return &a, nil
}

// marshalIntegrationScopes encodes integration scopes to JSON. Returns "[]" for nil/empty.
func marshalIntegrationScopes(scopes []AgentIntegrationScope) string {
	if len(scopes) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(scopes)
	return string(b)
}

// unmarshalIntegrationScopes decodes integration scopes from JSON. Returns nil for empty/invalid.
func unmarshalIntegrationScopes(s string) []AgentIntegrationScope {
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	var result []AgentIntegrationScope
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// marshalStringSlice encodes a string slice to JSON. Returns "[]" for nil/empty.
func marshalStringSlice(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(s)
	return string(b)
}

// unmarshalStringSlice decodes a JSON string slice. Returns nil for empty/invalid.
func unmarshalStringSlice(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// isUniqueViolation checks if a SQLite error is a UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
