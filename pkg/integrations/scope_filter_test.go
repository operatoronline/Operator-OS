package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AgentIntegrationScope.Validate ---

func TestAgentIntegrationScope_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		s := AgentIntegrationScope{IntegrationID: "google-gmail"}
		assert.NoError(t, s.Validate())
	})

	t.Run("empty integration_id", func(t *testing.T) {
		s := AgentIntegrationScope{}
		assert.Error(t, s.Validate())
	})

	t.Run("with tools and scopes", func(t *testing.T) {
		s := AgentIntegrationScope{
			IntegrationID: "google-gmail",
			AllowedTools:  []string{"gmail_list_messages", "gmail_send_message"},
			AllowedScopes: []string{"gmail.readonly"},
		}
		assert.NoError(t, s.Validate())
	})
}

// --- NewAgentScopeFilter ---

func TestNewAgentScopeFilter_OpenAccess(t *testing.T) {
	t.Run("nil scopes", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		assert.True(t, f.IsOpenAccess())
	})

	t.Run("empty scopes", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{})
		assert.True(t, f.IsOpenAccess())
	})
}

func TestNewAgentScopeFilter_Restricted(t *testing.T) {
	f := NewAgentScopeFilter([]AgentIntegrationScope{
		{IntegrationID: "google-gmail"},
		{IntegrationID: "shopify-products", AllowedTools: []string{"shopify_list_products"}},
	})
	assert.False(t, f.IsOpenAccess())
}

// --- IsIntegrationAllowed ---

func TestAgentScopeFilter_IsIntegrationAllowed(t *testing.T) {
	t.Run("open access allows everything", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		assert.True(t, f.IsIntegrationAllowed("anything"))
	})

	t.Run("allowed integration", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.True(t, f.IsIntegrationAllowed("google-gmail"))
	})

	t.Run("disallowed integration", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.False(t, f.IsIntegrationAllowed("shopify-products"))
	})
}

// --- IsToolAllowed ---

func TestAgentScopeFilter_IsToolAllowed(t *testing.T) {
	t.Run("open access allows all tools", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		assert.True(t, f.IsToolAllowed("any-integration", "any_tool"))
	})

	t.Run("integration not allowed", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.False(t, f.IsToolAllowed("shopify-products", "shopify_list_products"))
	})

	t.Run("all tools allowed when empty list", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"}, // empty AllowedTools = all tools
		})
		assert.True(t, f.IsToolAllowed("google-gmail", "gmail_list_messages"))
		assert.True(t, f.IsToolAllowed("google-gmail", "gmail_send_message"))
	})

	t.Run("specific tools allowed", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list_messages", "gmail_get_message"}},
		})
		assert.True(t, f.IsToolAllowed("google-gmail", "gmail_list_messages"))
		assert.True(t, f.IsToolAllowed("google-gmail", "gmail_get_message"))
		assert.False(t, f.IsToolAllowed("google-gmail", "gmail_send_message"))
	})
}

// --- IsScopeAllowed ---

func TestAgentScopeFilter_IsScopeAllowed(t *testing.T) {
	t.Run("open access allows all scopes", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		assert.True(t, f.IsScopeAllowed("any", "any.scope"))
	})

	t.Run("integration not allowed", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.False(t, f.IsScopeAllowed("shopify", "read_products"))
	})

	t.Run("all scopes when empty list", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.True(t, f.IsScopeAllowed("google-gmail", "gmail.readonly"))
	})

	t.Run("specific scopes", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedScopes: []string{"gmail.readonly"}},
		})
		assert.True(t, f.IsScopeAllowed("google-gmail", "gmail.readonly"))
		assert.False(t, f.IsScopeAllowed("google-gmail", "gmail.send"))
	})
}

// --- FilterTools ---

func TestAgentScopeFilter_FilterTools(t *testing.T) {
	tools := []*IntegrationTool{
		NewIntegrationTool("google-gmail", ToolManifest{Name: "gmail_list_messages", Description: "List"}, nil),
		NewIntegrationTool("google-gmail", ToolManifest{Name: "gmail_send_message", Description: "Send"}, nil),
		NewIntegrationTool("shopify-products", ToolManifest{Name: "shopify_list_products", Description: "List"}, nil),
	}

	t.Run("open access returns all", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		result := f.FilterTools(tools)
		assert.Len(t, result, 3)
	})

	t.Run("filter by integration", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		result := f.FilterTools(tools)
		assert.Len(t, result, 2)
		assert.Equal(t, "gmail_list_messages", result[0].Name())
		assert.Equal(t, "gmail_send_message", result[1].Name())
	})

	t.Run("filter by specific tools", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list_messages"}},
		})
		result := f.FilterTools(tools)
		assert.Len(t, result, 1)
		assert.Equal(t, "gmail_list_messages", result[0].Name())
	})

	t.Run("no integrations allowed returns empty", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "unknown-integration"},
		})
		result := f.FilterTools(tools)
		assert.Empty(t, result)
	})
}

// --- FilterToolNames ---

func TestAgentScopeFilter_FilterToolNames(t *testing.T) {
	reg := NewIntegrationRegistry()
	err := reg.Register(&Integration{
		ID: "google-gmail", Name: "Gmail", Category: "email",
		Description: "Gmail integration", AuthType: AuthTypeNone,
		Tools: []ToolManifest{
			{Name: "gmail_list", Description: "List messages"},
			{Name: "gmail_send", Description: "Send message"},
		},
	})
	require.NoError(t, err)

	t.Run("open access returns all", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		result := f.FilterToolNames([]string{"gmail_list", "gmail_send"}, reg)
		assert.Len(t, result, 2)
	})

	t.Run("filtered by integration", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		result := f.FilterToolNames([]string{"gmail_list", "gmail_send"}, reg)
		assert.Len(t, result, 2)
	})

	t.Run("filtered by specific tools", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list"}},
		})
		result := f.FilterToolNames([]string{"gmail_list", "gmail_send"}, reg)
		assert.Len(t, result, 1)
		assert.Equal(t, "gmail_list", result[0])
	})

	t.Run("unknown tools skipped", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		result := f.FilterToolNames([]string{"gmail_list", "unknown_tool"}, reg)
		assert.Len(t, result, 1)
	})

	t.Run("nil registry returns nil", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		result := f.FilterToolNames([]string{"gmail_list"}, nil)
		assert.Nil(t, result)
	})
}

// --- AllowedIntegrationIDs ---

func TestAgentScopeFilter_AllowedIntegrationIDs(t *testing.T) {
	t.Run("open access returns nil", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		assert.Nil(t, f.AllowedIntegrationIDs())
	})

	t.Run("returns sorted IDs", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "shopify-products"},
			{IntegrationID: "google-gmail"},
		})
		ids := f.AllowedIntegrationIDs()
		assert.Equal(t, []string{"google-gmail", "shopify-products"}, ids)
	})
}

// --- GetScope ---

func TestAgentScopeFilter_GetScope(t *testing.T) {
	t.Run("open access returns nil", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		assert.Nil(t, f.GetScope("google-gmail"))
	})

	t.Run("returns scope for allowed integration", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list"}},
		})
		scope := f.GetScope("google-gmail")
		require.NotNil(t, scope)
		assert.Equal(t, "google-gmail", scope.IntegrationID)
		assert.Equal(t, []string{"gmail_list"}, scope.AllowedTools)
	})

	t.Run("returns nil for disallowed integration", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.Nil(t, f.GetScope("shopify"))
	})
}

// --- EffectiveScopes ---

func TestAgentScopeFilter_EffectiveScopes(t *testing.T) {
	defaults := []string{"gmail.readonly", "gmail.send", "gmail.modify"}

	t.Run("open access returns defaults", func(t *testing.T) {
		f := NewAgentScopeFilter(nil)
		result := f.EffectiveScopes("google-gmail", defaults)
		assert.Equal(t, defaults, result)
	})

	t.Run("integration not allowed returns nil", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "shopify"},
		})
		result := f.EffectiveScopes("google-gmail", defaults)
		assert.Nil(t, result)
	})

	t.Run("empty allowed scopes returns defaults", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		result := f.EffectiveScopes("google-gmail", defaults)
		assert.Equal(t, defaults, result)
	})

	t.Run("intersection of allowed and default scopes", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedScopes: []string{"gmail.readonly", "gmail.send"}},
		})
		result := f.EffectiveScopes("google-gmail", defaults)
		assert.Equal(t, []string{"gmail.readonly", "gmail.send"}, result)
	})

	t.Run("allowed scope not in defaults is excluded", func(t *testing.T) {
		f := NewAgentScopeFilter([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedScopes: []string{"gmail.readonly", "gmail.admin"}},
		})
		result := f.EffectiveScopes("google-gmail", defaults)
		assert.Equal(t, []string{"gmail.readonly"}, result)
	})
}

// --- ValidateAgentScopes ---

func TestValidateAgentScopes(t *testing.T) {
	t.Run("nil is valid", func(t *testing.T) {
		assert.NoError(t, ValidateAgentScopes(nil, nil))
	})

	t.Run("empty is valid", func(t *testing.T) {
		assert.NoError(t, ValidateAgentScopes([]AgentIntegrationScope{}, nil))
	})

	t.Run("valid scopes without registry", func(t *testing.T) {
		scopes := []AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
			{IntegrationID: "shopify-products"},
		}
		assert.NoError(t, ValidateAgentScopes(scopes, nil))
	})

	t.Run("missing integration_id", func(t *testing.T) {
		scopes := []AgentIntegrationScope{
			{IntegrationID: ""},
		}
		err := ValidateAgentScopes(scopes, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integration_id is required")
	})

	t.Run("duplicate integration_id", func(t *testing.T) {
		scopes := []AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
			{IntegrationID: "google-gmail"},
		}
		err := ValidateAgentScopes(scopes, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("integration not found in registry", func(t *testing.T) {
		reg := NewIntegrationRegistry()
		scopes := []AgentIntegrationScope{
			{IntegrationID: "nonexistent"},
		}
		err := ValidateAgentScopes(scopes, reg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in registry")
	})

	t.Run("valid with registry", func(t *testing.T) {
		reg := NewIntegrationRegistry()
		reg.Register(&Integration{
			ID: "google-gmail", Name: "Gmail", Category: "email",
			Description: "Gmail", AuthType: AuthTypeNone,
			Tools: []ToolManifest{
				{Name: "gmail_list", Description: "List"},
				{Name: "gmail_send", Description: "Send"},
			},
		})
		scopes := []AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list"}},
		}
		assert.NoError(t, ValidateAgentScopes(scopes, reg))
	})

	t.Run("invalid tool name in registry", func(t *testing.T) {
		reg := NewIntegrationRegistry()
		reg.Register(&Integration{
			ID: "google-gmail", Name: "Gmail", Category: "email",
			Description: "Gmail", AuthType: AuthTypeNone,
			Tools: []ToolManifest{
				{Name: "gmail_list", Description: "List"},
			},
		})
		scopes := []AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_nonexistent"}},
		}
		err := ValidateAgentScopes(scopes, reg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid scope name in registry", func(t *testing.T) {
		reg := NewIntegrationRegistry()
		reg.Register(&Integration{
			ID: "google-gmail", Name: "Gmail", Category: "email",
			Description: "Gmail", AuthType: AuthTypeOAuth2,
			OAuth: &OAuthConfig{
				AuthorizationURL: "https://auth.example.com",
				TokenURL:         "https://token.example.com",
				Scopes:           []string{"gmail.readonly", "gmail.send"},
			},
			Tools: []ToolManifest{
				{Name: "gmail_list", Description: "List"},
			},
		})
		scopes := []AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedScopes: []string{"gmail.admin"}},
		}
		err := ValidateAgentScopes(scopes, reg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scope")
		assert.Contains(t, err.Error(), "not found")
	})
}

// --- ScopeSummary ---

func TestScopeSummary(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := ScopeSummary(nil)
		assert.Contains(t, result, "open access")
	})

	t.Run("all tools", func(t *testing.T) {
		result := ScopeSummary([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
		})
		assert.Contains(t, result, "google-gmail (all tools)")
	})

	t.Run("specific tools", func(t *testing.T) {
		result := ScopeSummary([]AgentIntegrationScope{
			{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list", "gmail_send"}},
		})
		assert.Contains(t, result, "google-gmail (2 tools)")
	})

	t.Run("multiple integrations", func(t *testing.T) {
		result := ScopeSummary([]AgentIntegrationScope{
			{IntegrationID: "google-gmail"},
			{IntegrationID: "shopify-products", AllowedTools: []string{"shopify_list"}},
		})
		assert.Contains(t, result, "google-gmail")
		assert.Contains(t, result, "shopify-products")
	})
}

// --- Multi-integration filtering ---

func TestAgentScopeFilter_MultiIntegration(t *testing.T) {
	f := NewAgentScopeFilter([]AgentIntegrationScope{
		{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list_messages"}},
		{IntegrationID: "shopify-products"}, // all tools
	})

	// Gmail: only gmail_list_messages allowed
	assert.True(t, f.IsToolAllowed("google-gmail", "gmail_list_messages"))
	assert.False(t, f.IsToolAllowed("google-gmail", "gmail_send_message"))

	// Shopify: all tools allowed
	assert.True(t, f.IsToolAllowed("shopify-products", "shopify_list_products"))
	assert.True(t, f.IsToolAllowed("shopify-products", "shopify_get_product"))

	// Unknown: not allowed
	assert.False(t, f.IsToolAllowed("github", "github_list_repos"))
}

// --- Duplicate scope handling ---

func TestNewAgentScopeFilter_DuplicateOverwrites(t *testing.T) {
	// When duplicate integration IDs are provided, last one wins.
	f := NewAgentScopeFilter([]AgentIntegrationScope{
		{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_list"}},
		{IntegrationID: "google-gmail", AllowedTools: []string{"gmail_send"}},
	})
	// Last one should win
	assert.True(t, f.IsToolAllowed("google-gmail", "gmail_send"))
	assert.False(t, f.IsToolAllowed("google-gmail", "gmail_list"))
}
