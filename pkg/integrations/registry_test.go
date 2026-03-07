package integrations

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validIntegration(id string) *Integration {
	return &Integration{
		ID:          id,
		Name:        "Test " + id,
		Category:    "testing",
		Description: "A test integration",
		AuthType:    AuthTypeOAuth2,
		OAuth: &OAuthConfig{
			AuthorizationURL: "https://example.com/auth",
			TokenURL:         "https://example.com/token",
			Scopes:           []string{"read", "write"},
			UsePKCE:          true,
		},
		Tools: []ToolManifest{
			{
				Name:        id + "_list",
				Description: "List items",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"limit": map[string]any{"type": "integer"},
					},
				},
			},
			{
				Name:        id + "_get",
				Description: "Get item by ID",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{"type": "string"},
					},
				},
			},
		},
		RequiredPlan: "starter",
		Status:       IntegrationStatusActive,
		Version:      "1.0.0",
	}
}

// --- Validation tests ---

func TestIntegration_Validate_Success(t *testing.T) {
	i := validIntegration("test")
	assert.NoError(t, i.Validate())
}

func TestIntegration_Validate_MissingID(t *testing.T) {
	i := validIntegration("test")
	i.ID = ""
	assert.ErrorContains(t, i.Validate(), "ID is required")
}

func TestIntegration_Validate_MissingName(t *testing.T) {
	i := validIntegration("test")
	i.Name = ""
	assert.ErrorContains(t, i.Validate(), "name is required")
}

func TestIntegration_Validate_MissingCategory(t *testing.T) {
	i := validIntegration("test")
	i.Category = ""
	assert.ErrorContains(t, i.Validate(), "category is required")
}

func TestIntegration_Validate_MissingDescription(t *testing.T) {
	i := validIntegration("test")
	i.Description = ""
	assert.ErrorContains(t, i.Validate(), "description is required")
}

func TestIntegration_Validate_InvalidAuthType(t *testing.T) {
	i := validIntegration("test")
	i.AuthType = "magic"
	assert.ErrorContains(t, i.Validate(), "invalid auth type")
}

func TestIntegration_Validate_InvalidStatus(t *testing.T) {
	i := validIntegration("test")
	i.Status = "bad"
	assert.ErrorContains(t, i.Validate(), "invalid status")
}

func TestIntegration_Validate_OAuth_MissingConfig(t *testing.T) {
	i := validIntegration("test")
	i.OAuth = nil
	assert.ErrorContains(t, i.Validate(), "oauth config is required")
}

func TestIntegration_Validate_OAuth_MissingAuthURL(t *testing.T) {
	i := validIntegration("test")
	i.OAuth.AuthorizationURL = ""
	assert.ErrorContains(t, i.Validate(), "authorization_url is required")
}

func TestIntegration_Validate_OAuth_MissingTokenURL(t *testing.T) {
	i := validIntegration("test")
	i.OAuth.TokenURL = ""
	assert.ErrorContains(t, i.Validate(), "token_url is required")
}

func TestIntegration_Validate_APIKey_MissingConfig(t *testing.T) {
	i := &Integration{
		ID:          "test",
		Name:        "Test",
		Category:    "testing",
		Description: "A test",
		AuthType:    AuthTypeAPIKey,
	}
	assert.ErrorContains(t, i.Validate(), "api_key_config is required")
}

func TestIntegration_Validate_APIKey_MissingHeader(t *testing.T) {
	i := &Integration{
		ID:          "test",
		Name:        "Test",
		Category:    "testing",
		Description: "A test",
		AuthType:    AuthTypeAPIKey,
		APIKeyConfig: &APIKeyConfig{
			Header: "",
		},
	}
	assert.ErrorContains(t, i.Validate(), "header is required")
}

func TestIntegration_Validate_APIKey_Success(t *testing.T) {
	i := &Integration{
		ID:          "test",
		Name:        "Test",
		Category:    "testing",
		Description: "A test",
		AuthType:    AuthTypeAPIKey,
		APIKeyConfig: &APIKeyConfig{
			Header: "X-API-Key",
		},
	}
	assert.NoError(t, i.Validate())
}

func TestIntegration_Validate_NoneAuth(t *testing.T) {
	i := &Integration{
		ID:          "test",
		Name:        "Test",
		Category:    "testing",
		Description: "A test",
		AuthType:    AuthTypeNone,
	}
	assert.NoError(t, i.Validate())
}

func TestIntegration_Validate_EmptyToolName(t *testing.T) {
	i := validIntegration("test")
	i.Tools = append(i.Tools, ToolManifest{Name: "", Description: "no name"})
	assert.ErrorContains(t, i.Validate(), "tool name is required")
}

func TestIntegration_Validate_EmptyToolDescription(t *testing.T) {
	i := validIntegration("test")
	i.Tools = append(i.Tools, ToolManifest{Name: "foo", Description: ""})
	assert.ErrorContains(t, i.Validate(), "description is required")
}

func TestIntegration_Validate_DuplicateToolName(t *testing.T) {
	i := validIntegration("test")
	i.Tools = append(i.Tools, ToolManifest{Name: "test_list", Description: "duplicate"})
	assert.ErrorContains(t, i.Validate(), "duplicate tool name")
}

func TestIntegration_Validate_DefaultStatus(t *testing.T) {
	i := validIntegration("test")
	i.Status = "" // empty should be valid
	assert.NoError(t, i.Validate())
}

func TestIntegration_ToolNames(t *testing.T) {
	i := validIntegration("test")
	names := i.ToolNames()
	assert.Equal(t, []string{"test_list", "test_get"}, names)
}

// --- ValidAuthType / ValidIntegrationStatus ---

func TestValidAuthType(t *testing.T) {
	assert.True(t, ValidAuthType(AuthTypeOAuth2))
	assert.True(t, ValidAuthType(AuthTypeAPIKey))
	assert.True(t, ValidAuthType(AuthTypeNone))
	assert.False(t, ValidAuthType("magic"))
	assert.False(t, ValidAuthType(""))
}

func TestValidIntegrationStatus(t *testing.T) {
	assert.True(t, ValidIntegrationStatus(IntegrationStatusActive))
	assert.True(t, ValidIntegrationStatus(IntegrationStatusBeta))
	assert.True(t, ValidIntegrationStatus(IntegrationStatusDeprecated))
	assert.True(t, ValidIntegrationStatus("")) // empty is valid (default)
	assert.False(t, ValidIntegrationStatus("bad"))
}

// --- Registry tests ---

func TestRegistry_NewEmpty(t *testing.T) {
	r := NewIntegrationRegistry()
	assert.Equal(t, 0, r.Count())
	assert.Empty(t, r.List())
}

func TestRegistry_Register_Success(t *testing.T) {
	r := NewIntegrationRegistry()
	err := r.Register(validIntegration("google"))
	require.NoError(t, err)
	assert.Equal(t, 1, r.Count())
}

func TestRegistry_Register_Nil(t *testing.T) {
	r := NewIntegrationRegistry()
	err := r.Register(nil)
	assert.ErrorContains(t, err, "nil")
}

func TestRegistry_Register_Invalid(t *testing.T) {
	r := NewIntegrationRegistry()
	err := r.Register(&Integration{ID: "bad"})
	assert.Error(t, err)
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))
	err := r.Register(validIntegration("google"))
	assert.ErrorContains(t, err, "already registered")
}

func TestRegistry_Replace(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))
	updated := validIntegration("google")
	updated.Name = "Google Updated"
	require.NoError(t, r.Replace(updated))
	assert.Equal(t, "Google Updated", r.Get("google").Name)
	assert.Equal(t, 1, r.Count())
}

func TestRegistry_Replace_Nil(t *testing.T) {
	r := NewIntegrationRegistry()
	assert.ErrorContains(t, r.Replace(nil), "nil")
}

func TestRegistry_Replace_Invalid(t *testing.T) {
	r := NewIntegrationRegistry()
	assert.Error(t, r.Replace(&Integration{ID: "bad"}))
}

func TestRegistry_Get(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))
	got := r.Get("google")
	assert.NotNil(t, got)
	assert.Equal(t, "google", got.ID)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewIntegrationRegistry()
	assert.Nil(t, r.Get("nonexistent"))
}

func TestRegistry_List_Sorted(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("shopify")))
	require.NoError(t, r.Register(validIntegration("google")))
	require.NoError(t, r.Register(validIntegration("github")))
	list := r.List()
	require.Len(t, list, 3)
	assert.Equal(t, "github", list[0].ID)
	assert.Equal(t, "google", list[1].ID)
	assert.Equal(t, "shopify", list[2].ID)
}

func TestRegistry_ListByCategory(t *testing.T) {
	r := NewIntegrationRegistry()
	g := validIntegration("google")
	g.Category = "email"
	s := validIntegration("shopify")
	s.Category = "ecommerce"
	require.NoError(t, r.Register(g))
	require.NoError(t, r.Register(s))

	email := r.ListByCategory("email")
	assert.Len(t, email, 1)
	assert.Equal(t, "google", email[0].ID)

	ecom := r.ListByCategory("ecommerce")
	assert.Len(t, ecom, 1)
	assert.Equal(t, "shopify", ecom[0].ID)

	none := r.ListByCategory("crm")
	assert.Empty(t, none)
}

func TestRegistry_ListByCategory_CaseInsensitive(t *testing.T) {
	r := NewIntegrationRegistry()
	g := validIntegration("google")
	g.Category = "Email"
	require.NoError(t, r.Register(g))

	list := r.ListByCategory("email")
	assert.Len(t, list, 1)
}

func TestRegistry_ListActive(t *testing.T) {
	r := NewIntegrationRegistry()
	active := validIntegration("active")
	active.Status = IntegrationStatusActive
	beta := validIntegration("beta")
	beta.Status = IntegrationStatusBeta
	dep := validIntegration("deprecated")
	dep.Status = IntegrationStatusDeprecated
	noStatus := validIntegration("nostatus")
	noStatus.Status = ""

	require.NoError(t, r.Register(active))
	require.NoError(t, r.Register(beta))
	require.NoError(t, r.Register(dep))
	require.NoError(t, r.Register(noStatus))

	list := r.ListActive()
	assert.Len(t, list, 2) // active + nostatus
	ids := make([]string, len(list))
	for i, l := range list {
		ids[i] = l.ID
	}
	assert.Contains(t, ids, "active")
	assert.Contains(t, ids, "nostatus")
}

func TestRegistry_Remove(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))
	assert.True(t, r.Remove("google"))
	assert.Equal(t, 0, r.Count())
	assert.Nil(t, r.Get("google"))
}

func TestRegistry_Remove_NotFound(t *testing.T) {
	r := NewIntegrationRegistry()
	assert.False(t, r.Remove("nonexistent"))
}

func TestRegistry_Categories(t *testing.T) {
	r := NewIntegrationRegistry()
	g := validIntegration("google")
	g.Category = "Email"
	s := validIntegration("shopify")
	s.Category = "ecommerce"
	h := validIntegration("github")
	h.Category = "Developer"

	require.NoError(t, r.Register(g))
	require.NoError(t, r.Register(s))
	require.NoError(t, r.Register(h))

	cats := r.Categories()
	assert.Equal(t, []string{"developer", "ecommerce", "email"}, cats)
}

// --- JSON loading tests ---

func TestRegistry_LoadFromJSON_Success(t *testing.T) {
	r := NewIntegrationRegistry()
	i := validIntegration("test")
	data, err := json.Marshal(i)
	require.NoError(t, err)

	err = r.LoadFromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, 1, r.Count())
	assert.Equal(t, "test", r.Get("test").ID)
}

func TestRegistry_LoadFromJSON_InvalidJSON(t *testing.T) {
	r := NewIntegrationRegistry()
	err := r.LoadFromJSON([]byte("{invalid"))
	assert.ErrorContains(t, err, "failed to parse")
}

func TestRegistry_LoadFromJSON_InvalidManifest(t *testing.T) {
	r := NewIntegrationRegistry()
	data, _ := json.Marshal(&Integration{ID: "bad"})
	err := r.LoadFromJSON(data)
	assert.Error(t, err)
}

func TestRegistry_LoadMultipleFromJSON_Success(t *testing.T) {
	r := NewIntegrationRegistry()
	integrations := []*Integration{
		validIntegration("google"),
		validIntegration("shopify"),
	}
	data, err := json.Marshal(integrations)
	require.NoError(t, err)

	err = r.LoadMultipleFromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, 2, r.Count())
}

func TestRegistry_LoadMultipleFromJSON_InvalidJSON(t *testing.T) {
	r := NewIntegrationRegistry()
	err := r.LoadMultipleFromJSON([]byte("not json"))
	assert.ErrorContains(t, err, "failed to parse")
}

func TestRegistry_LoadMultipleFromJSON_StopsOnError(t *testing.T) {
	r := NewIntegrationRegistry()
	data := `[
		{"id": "good", "name": "Good", "category": "test", "description": "ok", "auth_type": "none"},
		{"id": "", "name": "Bad"}
	]`
	err := r.LoadMultipleFromJSON([]byte(data))
	assert.Error(t, err)
	// First one should not have been registered either (stops on second)
	assert.Equal(t, 1, r.Count())
}

// --- Tool lookup tests ---

func TestRegistry_GetToolManifest(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))

	tm, integID := r.GetToolManifest("google_list")
	assert.NotNil(t, tm)
	assert.Equal(t, "google", integID)
	assert.Equal(t, "google_list", tm.Name)
}

func TestRegistry_GetToolManifest_NotFound(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))
	tm, integID := r.GetToolManifest("nonexistent")
	assert.Nil(t, tm)
	assert.Empty(t, integID)
}

func TestRegistry_AllToolNames(t *testing.T) {
	r := NewIntegrationRegistry()
	require.NoError(t, r.Register(validIntegration("google")))
	require.NoError(t, r.Register(validIntegration("shopify")))

	names := r.AllToolNames()
	assert.Len(t, names, 4)
	// Sorted
	assert.Equal(t, "google_get", names[0])
	assert.Equal(t, "google_list", names[1])
	assert.Equal(t, "shopify_get", names[2])
	assert.Equal(t, "shopify_list", names[3])
}

// --- JSON roundtrip ---

func TestIntegration_JSONRoundtrip(t *testing.T) {
	orig := validIntegration("google")
	orig.OAuth.DynamicParams = map[string]DynamicParam{
		"shop": {Label: "Shop URL", Placeholder: "mystore.myshopify.com", Required: true},
	}
	orig.OAuth.ExtraAuthParams = map[string]string{"access_type": "offline"}
	orig.SuggestedTemplates = []string{"gmail-assistant"}

	data, err := json.Marshal(orig)
	require.NoError(t, err)

	var restored Integration
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, orig.ID, restored.ID)
	assert.Equal(t, orig.Name, restored.Name)
	assert.Equal(t, orig.Category, restored.Category)
	assert.Equal(t, orig.OAuth.DynamicParams["shop"].Label, restored.OAuth.DynamicParams["shop"].Label)
	assert.Equal(t, orig.OAuth.ExtraAuthParams["access_type"], restored.OAuth.ExtraAuthParams["access_type"])
	assert.Equal(t, orig.SuggestedTemplates, restored.SuggestedTemplates)
	assert.Len(t, restored.Tools, 2)
}
