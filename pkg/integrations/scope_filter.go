// Package integrations — scope_filter.go provides per-agent integration
// scope narrowing. An AgentScopeFilter restricts which integration tools
// are available to a specific agent based on its AllowedIntegrations config.
package integrations

import (
	"fmt"
	"sort"
	"strings"
)

// AgentIntegrationScope defines which parts of an integration an agent
// is allowed to access. An empty AllowedTools slice means "all tools"
// for that integration. An empty AllowedScopes slice means "all scopes".
type AgentIntegrationScope struct {
	// IntegrationID is the integration manifest ID (e.g. "google-gmail", "shopify-products").
	IntegrationID string `json:"integration_id"`
	// AllowedTools restricts the agent to specific tool names. Empty means all tools.
	AllowedTools []string `json:"allowed_tools,omitempty"`
	// AllowedScopes restricts the OAuth scopes the agent can use. Empty means all scopes.
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

// Validate checks the scope configuration for required fields.
func (s *AgentIntegrationScope) Validate() error {
	if s.IntegrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	return nil
}

// AgentScopeFilter filters integration tools available to a specific agent.
// If no scopes are configured (nil or empty), all tools from all connected
// integrations are available (open access). Once at least one scope is set,
// only tools from the listed integrations (and optionally specific tools
// within them) are available.
type AgentScopeFilter struct {
	// scopes maps integration_id → scope config.
	scopes map[string]*AgentIntegrationScope
	// openAccess is true when no restrictions are configured.
	openAccess bool
}

// NewAgentScopeFilter creates a filter from agent integration scopes.
// A nil or empty slice means open access (all integrations allowed).
func NewAgentScopeFilter(scopes []AgentIntegrationScope) *AgentScopeFilter {
	if len(scopes) == 0 {
		return &AgentScopeFilter{openAccess: true}
	}
	m := make(map[string]*AgentIntegrationScope, len(scopes))
	for i := range scopes {
		s := scopes[i]
		m[s.IntegrationID] = &s
	}
	return &AgentScopeFilter{scopes: m}
}

// IsOpenAccess returns true if no restrictions are configured.
func (f *AgentScopeFilter) IsOpenAccess() bool {
	return f.openAccess
}

// IsIntegrationAllowed checks if an integration is in the allowed set.
func (f *AgentScopeFilter) IsIntegrationAllowed(integrationID string) bool {
	if f.openAccess {
		return true
	}
	_, ok := f.scopes[integrationID]
	return ok
}

// IsToolAllowed checks if a specific tool from an integration is allowed.
func (f *AgentScopeFilter) IsToolAllowed(integrationID, toolName string) bool {
	if f.openAccess {
		return true
	}
	scope, ok := f.scopes[integrationID]
	if !ok {
		return false
	}
	// Empty AllowedTools means all tools from this integration are allowed.
	if len(scope.AllowedTools) == 0 {
		return true
	}
	for _, t := range scope.AllowedTools {
		if t == toolName {
			return true
		}
	}
	return false
}

// IsScopeAllowed checks if an OAuth scope is allowed for the integration.
func (f *AgentScopeFilter) IsScopeAllowed(integrationID, scopeName string) bool {
	if f.openAccess {
		return true
	}
	scope, ok := f.scopes[integrationID]
	if !ok {
		return false
	}
	// Empty AllowedScopes means all scopes are allowed.
	if len(scope.AllowedScopes) == 0 {
		return true
	}
	for _, s := range scope.AllowedScopes {
		if s == scopeName {
			return true
		}
	}
	return false
}

// FilterTools takes a list of integration tools and returns only those
// the agent is allowed to use. Each tool must implement IntegrationTooler
// (i.e. expose IntegrationID()).
func (f *AgentScopeFilter) FilterTools(tools []*IntegrationTool) []*IntegrationTool {
	if f.openAccess {
		return tools
	}
	var allowed []*IntegrationTool
	for _, t := range tools {
		if f.IsToolAllowed(t.IntegrationID(), t.Name()) {
			allowed = append(allowed, t)
		}
	}
	return allowed
}

// FilterToolNames takes integration tool names with a registry to resolve
// them, and returns only the allowed tool names.
func (f *AgentScopeFilter) FilterToolNames(toolNames []string, registry *IntegrationRegistry) []string {
	if f.openAccess {
		return toolNames
	}
	if registry == nil {
		return nil
	}
	var allowed []string
	for _, name := range toolNames {
		tm, integrationID := registry.GetToolManifest(name)
		if tm == nil {
			continue
		}
		if f.IsToolAllowed(integrationID, name) {
			allowed = append(allowed, name)
		}
	}
	return allowed
}

// AllowedIntegrationIDs returns the list of allowed integration IDs, sorted.
// Returns nil for open access.
func (f *AgentScopeFilter) AllowedIntegrationIDs() []string {
	if f.openAccess {
		return nil
	}
	ids := make([]string, 0, len(f.scopes))
	for id := range f.scopes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetScope returns the scope config for an integration, or nil if not allowed.
func (f *AgentScopeFilter) GetScope(integrationID string) *AgentIntegrationScope {
	if f.openAccess {
		return nil
	}
	return f.scopes[integrationID]
}

// EffectiveScopes returns the scopes the agent should request when connecting
// to an integration. If AllowedScopes is empty, returns the default scopes
// from the integration manifest. Otherwise, returns the intersection.
func (f *AgentScopeFilter) EffectiveScopes(integrationID string, defaultScopes []string) []string {
	if f.openAccess {
		return defaultScopes
	}
	scope, ok := f.scopes[integrationID]
	if !ok {
		return nil
	}
	if len(scope.AllowedScopes) == 0 {
		return defaultScopes
	}
	// Return only the scopes that are both in allowed and default.
	defaultSet := make(map[string]bool, len(defaultScopes))
	for _, s := range defaultScopes {
		defaultSet[s] = true
	}
	var effective []string
	for _, s := range scope.AllowedScopes {
		if defaultSet[s] {
			effective = append(effective, s)
		}
	}
	return effective
}

// ValidateAgentScopes validates a slice of agent integration scopes.
// Checks for required fields, duplicate integration IDs, and optionally
// validates tool names and scopes against the integration registry.
func ValidateAgentScopes(scopes []AgentIntegrationScope, registry *IntegrationRegistry) error {
	if len(scopes) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(scopes))
	for i, s := range scopes {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("scope[%d]: %w", i, err)
		}
		if seen[s.IntegrationID] {
			return fmt.Errorf("scope[%d]: duplicate integration_id %q", i, s.IntegrationID)
		}
		seen[s.IntegrationID] = true

		// Validate against registry if provided.
		if registry != nil {
			integ := registry.Get(s.IntegrationID)
			if integ == nil {
				return fmt.Errorf("scope[%d]: integration %q not found in registry", i, s.IntegrationID)
			}
			// Validate tool names.
			if len(s.AllowedTools) > 0 {
				toolSet := make(map[string]bool, len(integ.Tools))
				for _, t := range integ.Tools {
					toolSet[t.Name] = true
				}
				for _, toolName := range s.AllowedTools {
					if !toolSet[toolName] {
						return fmt.Errorf("scope[%d]: tool %q not found in integration %q", i, toolName, s.IntegrationID)
					}
				}
			}
			// Validate scope names against OAuth config.
			if len(s.AllowedScopes) > 0 && integ.OAuth != nil {
				validScopes := make(map[string]bool, len(integ.OAuth.Scopes))
				for _, sc := range integ.OAuth.Scopes {
					validScopes[sc] = true
				}
				for _, scopeName := range s.AllowedScopes {
					if !validScopes[scopeName] {
						return fmt.Errorf("scope[%d]: scope %q not found in integration %q", i, scopeName, s.IntegrationID)
					}
				}
			}
		}
	}
	return nil
}

// ScopeSummary returns a human-readable summary of the scope restrictions.
func ScopeSummary(scopes []AgentIntegrationScope) string {
	if len(scopes) == 0 {
		return "open access (all integrations)"
	}
	var parts []string
	for _, s := range scopes {
		if len(s.AllowedTools) == 0 {
			parts = append(parts, s.IntegrationID+" (all tools)")
		} else {
			parts = append(parts, fmt.Sprintf("%s (%d tools)", s.IntegrationID, len(s.AllowedTools)))
		}
	}
	return strings.Join(parts, ", ")
}
