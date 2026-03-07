package agents

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempStore(t *testing.T) *SQLiteUserAgentStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteUserAgentStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestCreateAndGetByID(t *testing.T) {
	store := tempStore(t)

	temp := 0.8
	agent := &UserAgent{
		UserID:       "user-1",
		Name:         "My Agent",
		Description:  "A test agent",
		SystemPrompt: "You are helpful.",
		Model:        "gpt-4",
		ModelFallbacks: []string{"gpt-3.5-turbo"},
		Tools:        []string{"read_file", "write_file"},
		Skills:       []string{"coding"},
		MaxTokens:    4096,
		Temperature:  &temp,
		MaxIterations: 10,
	}

	err := store.Create(agent)
	require.NoError(t, err)
	assert.NotEmpty(t, agent.ID)
	assert.Equal(t, AgentStatusActive, agent.Status)
	assert.False(t, agent.CreatedAt.IsZero())

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Equal(t, agent.ID, got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "My Agent", got.Name)
	assert.Equal(t, "A test agent", got.Description)
	assert.Equal(t, "You are helpful.", got.SystemPrompt)
	assert.Equal(t, "gpt-4", got.Model)
	assert.Equal(t, []string{"gpt-3.5-turbo"}, got.ModelFallbacks)
	assert.Equal(t, []string{"read_file", "write_file"}, got.Tools)
	assert.Equal(t, []string{"coding"}, got.Skills)
	assert.Equal(t, 4096, got.MaxTokens)
	require.NotNil(t, got.Temperature)
	assert.InDelta(t, 0.8, *got.Temperature, 0.001)
	assert.Equal(t, 10, got.MaxIterations)
	assert.Equal(t, AgentStatusActive, got.Status)
}

func TestCreateDuplicateName(t *testing.T) {
	store := tempStore(t)

	a1 := &UserAgent{UserID: "user-1", Name: "Agent"}
	require.NoError(t, store.Create(a1))

	a2 := &UserAgent{UserID: "user-1", Name: "Agent"}
	err := store.Create(a2)
	assert.ErrorIs(t, err, ErrNameExists)
}

func TestCreateSameNameDifferentUsers(t *testing.T) {
	store := tempStore(t)

	a1 := &UserAgent{UserID: "user-1", Name: "Agent"}
	require.NoError(t, store.Create(a1))

	a2 := &UserAgent{UserID: "user-2", Name: "Agent"}
	require.NoError(t, store.Create(a2))
}

func TestGetByIDNotFound(t *testing.T) {
	store := tempStore(t)

	_, err := store.GetByID("nonexistent")
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestUpdate(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{UserID: "user-1", Name: "Original"}
	require.NoError(t, store.Create(agent))

	agent.Name = "Updated"
	agent.Description = "New desc"
	agent.SystemPrompt = "New prompt"
	agent.Model = "claude-3"
	agent.Tools = []string{"exec"}
	newTemp := 0.5
	agent.Temperature = &newTemp

	err := store.Update(agent)
	require.NoError(t, err)

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, "New desc", got.Description)
	assert.Equal(t, "New prompt", got.SystemPrompt)
	assert.Equal(t, "claude-3", got.Model)
	assert.Equal(t, []string{"exec"}, got.Tools)
	require.NotNil(t, got.Temperature)
	assert.InDelta(t, 0.5, *got.Temperature, 0.001)
}

func TestUpdateNotFound(t *testing.T) {
	store := tempStore(t)

	err := store.Update(&UserAgent{ID: "nonexistent", Name: "X"})
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestUpdateDuplicateName(t *testing.T) {
	store := tempStore(t)

	a1 := &UserAgent{UserID: "user-1", Name: "Alpha"}
	a2 := &UserAgent{UserID: "user-1", Name: "Beta"}
	require.NoError(t, store.Create(a1))
	require.NoError(t, store.Create(a2))

	a2.Name = "Alpha"
	err := store.Update(a2)
	assert.ErrorIs(t, err, ErrNameExists)
}

func TestDelete(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{UserID: "user-1", Name: "ToDelete"}
	require.NoError(t, store.Create(agent))

	err := store.Delete(agent.ID)
	require.NoError(t, err)

	_, err = store.GetByID(agent.ID)
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestDeleteNotFound(t *testing.T) {
	store := tempStore(t)

	err := store.Delete("nonexistent")
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestListByUser(t *testing.T) {
	store := tempStore(t)

	// Create agents for two users.
	require.NoError(t, store.Create(&UserAgent{UserID: "user-1", Name: "A1"}))
	require.NoError(t, store.Create(&UserAgent{UserID: "user-1", Name: "A2"}))
	require.NoError(t, store.Create(&UserAgent{UserID: "user-2", Name: "B1"}))

	list, err := store.ListByUser("user-1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "A1", list[0].Name)
	assert.Equal(t, "A2", list[1].Name)

	list2, err := store.ListByUser("user-2")
	require.NoError(t, err)
	assert.Len(t, list2, 1)

	empty, err := store.ListByUser("user-3")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestCountByUser(t *testing.T) {
	store := tempStore(t)

	require.NoError(t, store.Create(&UserAgent{UserID: "user-1", Name: "A1"}))
	require.NoError(t, store.Create(&UserAgent{UserID: "user-1", Name: "A2"}))

	count, err := store.CountByUser("user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = store.CountByUser("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestGetDefault(t *testing.T) {
	store := tempStore(t)

	a1 := &UserAgent{UserID: "user-1", Name: "Agent1", IsDefault: true}
	require.NoError(t, store.Create(a1))

	got, err := store.GetDefault("user-1")
	require.NoError(t, err)
	assert.Equal(t, "Agent1", got.Name)
	assert.True(t, got.IsDefault)
}

func TestGetDefaultNotFound(t *testing.T) {
	store := tempStore(t)

	_, err := store.GetDefault("user-1")
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestSetDefault(t *testing.T) {
	store := tempStore(t)

	a1 := &UserAgent{UserID: "user-1", Name: "Agent1", IsDefault: true}
	a2 := &UserAgent{UserID: "user-1", Name: "Agent2"}
	require.NoError(t, store.Create(a1))
	require.NoError(t, store.Create(a2))

	// Switch default to a2.
	err := store.SetDefault("user-1", a2.ID)
	require.NoError(t, err)

	// Verify a2 is now default.
	got, err := store.GetDefault("user-1")
	require.NoError(t, err)
	assert.Equal(t, a2.ID, got.ID)
	assert.True(t, got.IsDefault)

	// Verify a1 is no longer default.
	got1, err := store.GetByID(a1.ID)
	require.NoError(t, err)
	assert.False(t, got1.IsDefault)
}

func TestSetDefaultNotFound(t *testing.T) {
	store := tempStore(t)

	err := store.SetDefault("user-1", "nonexistent")
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestSetDefaultWrongUser(t *testing.T) {
	store := tempStore(t)

	a1 := &UserAgent{UserID: "user-1", Name: "Agent1"}
	require.NoError(t, store.Create(a1))

	err := store.SetDefault("user-2", a1.ID)
	assert.ErrorIs(t, err, ErrAgentNotFound)
}

func TestNilTemperature(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{UserID: "user-1", Name: "NoTemp"}
	require.NoError(t, store.Create(agent))

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Nil(t, got.Temperature)
}

func TestEmptySlices(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{UserID: "user-1", Name: "Minimal"}
	require.NoError(t, store.Create(agent))

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Nil(t, got.ModelFallbacks)
	assert.Nil(t, got.Tools)
	assert.Nil(t, got.Skills)
}

func TestCustomID(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{ID: "custom-id", UserID: "user-1", Name: "Custom"}
	require.NoError(t, store.Create(agent))
	assert.Equal(t, "custom-id", agent.ID)

	got, err := store.GetByID("custom-id")
	require.NoError(t, err)
	assert.Equal(t, "custom-id", got.ID)
}

func TestFromDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "shared.db")

	// Create with full store (handles schema).
	store1, err := NewSQLiteUserAgentStore(dbPath)
	require.NoError(t, err)

	agent := &UserAgent{UserID: "user-1", Name: "Shared"}
	require.NoError(t, store1.Create(agent))
	store1.Close()

	// Re-open with full store to verify persistence.
	store2, err := NewSQLiteUserAgentStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	got, err := store2.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Equal(t, "Shared", got.Name)
}

func TestMarshalUnmarshalSlices(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{"nil", nil, "[]"},
		{"empty", []string{}, "[]"},
		{"one", []string{"a"}, `["a"]`},
		{"multiple", []string{"a", "b", "c"}, `["a","b","c"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalStringSlice(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnmarshalStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", nil},
		{"empty array", "[]", nil},
		{"one", `["a"]`, []string{"a"}},
		{"multiple", `["a","b"]`, []string{"a", "b"}},
		{"invalid", "not-json", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unmarshalStringSlice(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAllowedIntegrations_CreateAndGet(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{
		UserID: "user-1",
		Name:   "Scoped Agent",
		AllowedIntegrations: []AgentIntegrationScope{
			{
				IntegrationID: "google-gmail",
				AllowedTools:  []string{"gmail_list_messages", "gmail_get_message"},
				AllowedScopes: []string{"gmail.readonly"},
			},
			{
				IntegrationID: "shopify-products",
			},
		},
	}
	require.NoError(t, store.Create(agent))

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	require.Len(t, got.AllowedIntegrations, 2)
	assert.Equal(t, "google-gmail", got.AllowedIntegrations[0].IntegrationID)
	assert.Equal(t, []string{"gmail_list_messages", "gmail_get_message"}, got.AllowedIntegrations[0].AllowedTools)
	assert.Equal(t, []string{"gmail.readonly"}, got.AllowedIntegrations[0].AllowedScopes)
	assert.Equal(t, "shopify-products", got.AllowedIntegrations[1].IntegrationID)
	assert.Nil(t, got.AllowedIntegrations[1].AllowedTools)
	assert.Nil(t, got.AllowedIntegrations[1].AllowedScopes)
}

func TestAllowedIntegrations_EmptySlice(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{
		UserID: "user-1",
		Name:   "Open Agent",
		// No AllowedIntegrations = open access
	}
	require.NoError(t, store.Create(agent))

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Nil(t, got.AllowedIntegrations)
}

func TestAllowedIntegrations_Update(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{
		UserID: "user-1",
		Name:   "Agent To Scope",
	}
	require.NoError(t, store.Create(agent))

	// Update to add scopes.
	agent.AllowedIntegrations = []AgentIntegrationScope{
		{IntegrationID: "google-drive", AllowedTools: []string{"drive_list_files"}},
	}
	require.NoError(t, store.Update(agent))

	got, err := store.GetByID(agent.ID)
	require.NoError(t, err)
	require.Len(t, got.AllowedIntegrations, 1)
	assert.Equal(t, "google-drive", got.AllowedIntegrations[0].IntegrationID)
	assert.Equal(t, []string{"drive_list_files"}, got.AllowedIntegrations[0].AllowedTools)

	// Update to clear scopes (back to open access).
	agent.AllowedIntegrations = nil
	require.NoError(t, store.Update(agent))

	got, err = store.GetByID(agent.ID)
	require.NoError(t, err)
	assert.Nil(t, got.AllowedIntegrations)
}

func TestAllowedIntegrations_ListByUser(t *testing.T) {
	store := tempStore(t)

	agent1 := &UserAgent{
		UserID: "user-1",
		Name:   "Agent 1",
		AllowedIntegrations: []AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		},
	}
	agent2 := &UserAgent{
		UserID: "user-1",
		Name:   "Agent 2",
		// Open access
	}
	require.NoError(t, store.Create(agent1))
	require.NoError(t, store.Create(agent2))

	agents, err := store.ListByUser("user-1")
	require.NoError(t, err)
	require.Len(t, agents, 2)
	assert.Len(t, agents[0].AllowedIntegrations, 1)
	assert.Nil(t, agents[1].AllowedIntegrations)
}

func TestAllowedIntegrations_GetDefault(t *testing.T) {
	store := tempStore(t)

	agent := &UserAgent{
		UserID:    "user-1",
		Name:      "Default Scoped",
		IsDefault: true,
		AllowedIntegrations: []AgentIntegrationScope{
			{IntegrationID: "shopify-orders", AllowedTools: []string{"shopify_list_orders"}},
		},
	}
	require.NoError(t, store.Create(agent))

	got, err := store.GetDefault("user-1")
	require.NoError(t, err)
	require.Len(t, got.AllowedIntegrations, 1)
	assert.Equal(t, "shopify-orders", got.AllowedIntegrations[0].IntegrationID)
}

func TestAllowedIntegrations_Persistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")
	store1, err := NewSQLiteUserAgentStore(dbPath)
	require.NoError(t, err)

	agent := &UserAgent{
		UserID: "user-1",
		Name:   "Persist Test",
		AllowedIntegrations: []AgentIntegrationScope{
			{IntegrationID: "google-calendar", AllowedScopes: []string{"calendar.readonly"}},
		},
	}
	require.NoError(t, store1.Create(agent))
	store1.Close()

	store2, err := NewSQLiteUserAgentStore(dbPath)
	require.NoError(t, err)
	defer store2.Close()

	got, err := store2.GetByID(agent.ID)
	require.NoError(t, err)
	require.Len(t, got.AllowedIntegrations, 1)
	assert.Equal(t, "google-calendar", got.AllowedIntegrations[0].IntegrationID)
	assert.Equal(t, []string{"calendar.readonly"}, got.AllowedIntegrations[0].AllowedScopes)
}

func TestMarshalUnmarshalIntegrationScopes(t *testing.T) {
	tests := []struct {
		name  string
		input []AgentIntegrationScope
		want  string
	}{
		{"nil", nil, "[]"},
		{"empty", []AgentIntegrationScope{}, "[]"},
		{"one", []AgentIntegrationScope{
			{IntegrationID: "test"},
		}, `[{"integration_id":"test"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalIntegrationScopes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnmarshalIntegrationScopes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []AgentIntegrationScope
	}{
		{"empty string", "", nil},
		{"empty array", "[]", nil},
		{"null", "null", nil},
		{"invalid", "not-json", nil},
		{"valid", `[{"integration_id":"test","allowed_tools":["a"]}]`,
			[]AgentIntegrationScope{{IntegrationID: "test", AllowedTools: []string{"a"}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unmarshalIntegrationScopes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
