package integrations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Integration is a declarative manifest describing an external service integration.
// It defines the OAuth provider config, the tools it exposes, and metadata for discovery.
type Integration struct {
	// ID is the unique identifier (e.g. "google", "shopify", "github").
	ID string `json:"id"`
	// Name is a human-readable display name.
	Name string `json:"name"`
	// Icon is a path or URL to the integration's icon.
	Icon string `json:"icon,omitempty"`
	// Category groups related integrations (e.g. "email", "ecommerce", "crm").
	Category string `json:"category"`
	// Description explains what this integration does.
	Description string `json:"description"`
	// AuthType is the authentication method ("oauth2", "api_key", "none").
	AuthType string `json:"auth_type"`
	// OAuth contains OAuth 2.0 provider configuration (when auth_type is "oauth2").
	OAuth *OAuthConfig `json:"oauth,omitempty"`
	// APIKeyConfig describes API key auth (when auth_type is "api_key").
	APIKeyConfig *APIKeyConfig `json:"api_key_config,omitempty"`
	// Tools are the tool definitions this integration exposes to the agent.
	Tools []ToolManifest `json:"tools"`
	// RequiredPlan is the minimum plan required to use this integration (e.g. "free", "starter", "pro").
	RequiredPlan string `json:"required_plan,omitempty"`
	// SuggestedTemplates lists agent template IDs that work well with this integration.
	SuggestedTemplates []string `json:"suggested_templates,omitempty"`
	// Status controls whether this integration is available ("active", "beta", "deprecated").
	Status string `json:"status,omitempty"`
	// Version is the manifest version for tracking changes.
	Version string `json:"version,omitempty"`
}

// OAuthConfig holds OAuth 2.0 provider settings within an integration manifest.
type OAuthConfig struct {
	// AuthorizationURL is the OAuth authorization endpoint.
	AuthorizationURL string `json:"authorization_url"`
	// TokenURL is the OAuth token exchange endpoint.
	TokenURL string `json:"token_url"`
	// Scopes are the default OAuth scopes requested.
	Scopes []string `json:"scopes"`
	// UsePKCE indicates whether PKCE (RFC 7636) should be used.
	UsePKCE bool `json:"use_pkce"`
	// DynamicParams are user-provided values that get substituted into URLs.
	// For example, Shopify needs a {shop} parameter in its URLs.
	DynamicParams map[string]DynamicParam `json:"dynamic_params,omitempty"`
	// ExtraAuthParams are additional query parameters for the authorization request.
	ExtraAuthParams map[string]string `json:"extra_auth_params,omitempty"`
}

// DynamicParam describes a user-configurable parameter (e.g. a shop URL).
type DynamicParam struct {
	// Label is the human-readable field label.
	Label string `json:"label"`
	// Placeholder is hint text for the input field.
	Placeholder string `json:"placeholder,omitempty"`
	// Required indicates whether this parameter must be provided.
	Required bool `json:"required"`
	// Validation is an optional regex pattern for client-side validation.
	Validation string `json:"validation,omitempty"`
}

// APIKeyConfig describes how to authenticate via API key.
type APIKeyConfig struct {
	// Header is the HTTP header name for the key (e.g. "Authorization", "X-API-Key").
	Header string `json:"header"`
	// Prefix is prepended to the key value (e.g. "Bearer ", "Token ").
	Prefix string `json:"prefix,omitempty"`
	// Label is a human-readable description of the key.
	Label string `json:"label,omitempty"`
}

// ToolManifest describes a single tool that an integration provides.
type ToolManifest struct {
	// Name is the tool name as seen by the LLM (e.g. "shopify_get_orders").
	Name string `json:"name"`
	// Description explains what the tool does.
	Description string `json:"description"`
	// Parameters is the JSON Schema for the tool's input.
	Parameters map[string]any `json:"parameters"`
	// RequiredScopes lists OAuth scopes this tool needs.
	RequiredScopes []string `json:"required_scopes,omitempty"`
	// RateLimit is the max calls per minute for this tool (0 = unlimited).
	RateLimit int `json:"rate_limit,omitempty"`
}

// Valid auth types.
const (
	AuthTypeOAuth2 = "oauth2"
	AuthTypeAPIKey = "api_key"
	AuthTypeNone   = "none"
)

// Valid integration statuses.
const (
	IntegrationStatusActive     = "active"
	IntegrationStatusBeta       = "beta"
	IntegrationStatusDeprecated = "deprecated"
)

// ValidAuthType checks if the auth type is recognized.
func ValidAuthType(t string) bool {
	switch t {
	case AuthTypeOAuth2, AuthTypeAPIKey, AuthTypeNone:
		return true
	}
	return false
}

// ValidIntegrationStatus checks if the integration status is recognized.
func ValidIntegrationStatus(s string) bool {
	switch s {
	case IntegrationStatusActive, IntegrationStatusBeta, IntegrationStatusDeprecated, "":
		return true
	}
	return false
}

// Validate checks the integration manifest for required fields and consistency.
func (i *Integration) Validate() error {
	if i.ID == "" {
		return fmt.Errorf("integration ID is required")
	}
	if i.Name == "" {
		return fmt.Errorf("integration name is required")
	}
	if i.Category == "" {
		return fmt.Errorf("integration category is required")
	}
	if i.Description == "" {
		return fmt.Errorf("integration description is required")
	}
	if !ValidAuthType(i.AuthType) {
		return fmt.Errorf("invalid auth type %q: must be oauth2, api_key, or none", i.AuthType)
	}
	if !ValidIntegrationStatus(i.Status) {
		return fmt.Errorf("invalid status %q: must be active, beta, or deprecated", i.Status)
	}
	if i.AuthType == AuthTypeOAuth2 {
		if i.OAuth == nil {
			return fmt.Errorf("oauth config is required when auth_type is oauth2")
		}
		if i.OAuth.AuthorizationURL == "" {
			return fmt.Errorf("oauth authorization_url is required")
		}
		if i.OAuth.TokenURL == "" {
			return fmt.Errorf("oauth token_url is required")
		}
	}
	if i.AuthType == AuthTypeAPIKey {
		if i.APIKeyConfig == nil {
			return fmt.Errorf("api_key_config is required when auth_type is api_key")
		}
		if i.APIKeyConfig.Header == "" {
			return fmt.Errorf("api_key_config header is required")
		}
	}
	// Validate tool names are unique within this integration.
	seen := make(map[string]bool, len(i.Tools))
	for _, t := range i.Tools {
		if t.Name == "" {
			return fmt.Errorf("tool name is required")
		}
		if t.Description == "" {
			return fmt.Errorf("tool %q: description is required", t.Name)
		}
		if seen[t.Name] {
			return fmt.Errorf("duplicate tool name %q", t.Name)
		}
		seen[t.Name] = true
	}
	return nil
}

// ToolNames returns the names of all tools in this integration.
func (i *Integration) ToolNames() []string {
	names := make([]string, len(i.Tools))
	for idx, t := range i.Tools {
		names[idx] = t.Name
	}
	return names
}

// IntegrationRegistry manages available integration manifests.
// It is concurrency-safe.
type IntegrationRegistry struct {
	mu           sync.RWMutex
	integrations map[string]*Integration
}

// NewIntegrationRegistry creates an empty integration registry.
func NewIntegrationRegistry() *IntegrationRegistry {
	return &IntegrationRegistry{
		integrations: make(map[string]*Integration),
	}
}

// Register adds an integration manifest. Returns error if validation fails
// or an integration with the same ID is already registered.
func (r *IntegrationRegistry) Register(i *Integration) error {
	if i == nil {
		return fmt.Errorf("integration is nil")
	}
	if err := i.Validate(); err != nil {
		return fmt.Errorf("invalid integration %q: %w", i.ID, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.integrations[i.ID]; exists {
		return fmt.Errorf("integration %q already registered", i.ID)
	}
	r.integrations[i.ID] = i
	return nil
}

// Replace replaces (or adds) an integration, skipping the duplicate check.
// Useful for hot-reloading manifests. Validates before replacing.
func (r *IntegrationRegistry) Replace(i *Integration) error {
	if i == nil {
		return fmt.Errorf("integration is nil")
	}
	if err := i.Validate(); err != nil {
		return fmt.Errorf("invalid integration %q: %w", i.ID, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.integrations[i.ID] = i
	return nil
}

// Get returns an integration by ID, or nil if not found.
func (r *IntegrationRegistry) Get(id string) *Integration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.integrations[id]
}

// List returns all registered integrations sorted by ID.
func (r *IntegrationRegistry) List() []*Integration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Integration, 0, len(r.integrations))
	for _, i := range r.integrations {
		result = append(result, i)
	}
	sort.Slice(result, func(a, b int) bool {
		return result[a].ID < result[b].ID
	})
	return result
}

// ListByCategory returns integrations in a given category, sorted by ID.
func (r *IntegrationRegistry) ListByCategory(category string) []*Integration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Integration
	for _, i := range r.integrations {
		if strings.EqualFold(i.Category, category) {
			result = append(result, i)
		}
	}
	sort.Slice(result, func(a, b int) bool {
		return result[a].ID < result[b].ID
	})
	return result
}

// ListActive returns only integrations with status "active" or "" (default), sorted by ID.
func (r *IntegrationRegistry) ListActive() []*Integration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Integration
	for _, i := range r.integrations {
		if i.Status == IntegrationStatusActive || i.Status == "" {
			result = append(result, i)
		}
	}
	sort.Slice(result, func(a, b int) bool {
		return result[a].ID < result[b].ID
	})
	return result
}

// Remove removes an integration by ID. Returns true if it was found and removed.
func (r *IntegrationRegistry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.integrations[id]; !exists {
		return false
	}
	delete(r.integrations, id)
	return true
}

// Count returns the number of registered integrations.
func (r *IntegrationRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.integrations)
}

// Categories returns all unique categories, sorted.
func (r *IntegrationRegistry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	catSet := make(map[string]bool)
	for _, i := range r.integrations {
		catSet[strings.ToLower(i.Category)] = true
	}
	cats := make([]string, 0, len(catSet))
	for c := range catSet {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// LoadFromJSON parses a JSON manifest and registers the integration.
func (r *IntegrationRegistry) LoadFromJSON(data []byte) error {
	var i Integration
	if err := json.Unmarshal(data, &i); err != nil {
		return fmt.Errorf("failed to parse integration manifest: %w", err)
	}
	return r.Register(&i)
}

// LoadMultipleFromJSON parses a JSON array of manifests and registers them all.
// Stops on first error.
func (r *IntegrationRegistry) LoadMultipleFromJSON(data []byte) error {
	var integrations []Integration
	if err := json.Unmarshal(data, &integrations); err != nil {
		return fmt.Errorf("failed to parse integration manifests: %w", err)
	}
	for idx := range integrations {
		if err := r.Register(&integrations[idx]); err != nil {
			return fmt.Errorf("integration[%d] %q: %w", idx, integrations[idx].ID, err)
		}
	}
	return nil
}

// GetToolManifest returns a specific tool manifest from any integration.
// Returns the tool and its parent integration ID, or nil if not found.
func (r *IntegrationRegistry) GetToolManifest(toolName string) (*ToolManifest, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, integ := range r.integrations {
		for idx := range integ.Tools {
			if integ.Tools[idx].Name == toolName {
				return &integ.Tools[idx], integ.ID
			}
		}
	}
	return nil, ""
}

// AllToolNames returns every tool name across all integrations, sorted.
func (r *IntegrationRegistry) AllToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for _, integ := range r.integrations {
		for _, t := range integ.Tools {
			names = append(names, t.Name)
		}
	}
	sort.Strings(names)
	return names
}
