package oauth

import (
	"fmt"
	"strings"
)

// Provider describes an OAuth 2.0 provider configuration.
type Provider struct {
	// ID is the unique identifier (e.g. "google", "github", "shopify").
	ID string `json:"id"`
	// Name is a human-readable display name.
	Name string `json:"name"`
	// AuthURL is the authorization endpoint.
	AuthURL string `json:"auth_url"`
	// TokenURL is the token exchange endpoint.
	TokenURL string `json:"token_url"`
	// ClientID is the OAuth client ID.
	ClientID string `json:"client_id"`
	// ClientSecret is the OAuth client secret (empty for public clients).
	ClientSecret string `json:"client_secret,omitempty"`
	// Scopes are the default scopes requested.
	Scopes []string `json:"scopes"`
	// RedirectURL is the callback URL registered with the provider.
	RedirectURL string `json:"redirect_url"`
	// UsePKCE indicates whether PKCE (RFC 7636) should be used.
	// Defaults to true for security.
	UsePKCE bool `json:"use_pkce"`
	// ExtraAuthParams are additional query parameters sent during authorization.
	ExtraAuthParams map[string]string `json:"extra_auth_params,omitempty"`
}

// Validate checks that the provider configuration has required fields.
func (p *Provider) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("provider ID is required")
	}
	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if p.AuthURL == "" {
		return fmt.Errorf("auth URL is required")
	}
	if p.TokenURL == "" {
		return fmt.Errorf("token URL is required")
	}
	if p.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if p.RedirectURL == "" {
		return fmt.Errorf("redirect URL is required")
	}
	return nil
}

// ScopeString returns scopes as a space-separated string.
func (p *Provider) ScopeString() string {
	return strings.Join(p.Scopes, " ")
}

// ProviderRegistry holds registered OAuth providers.
type ProviderRegistry struct {
	providers map[string]*Provider
}

// NewProviderRegistry creates an empty provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]*Provider),
	}
}

// Register adds a provider to the registry. Returns an error if validation fails
// or if a provider with the same ID is already registered.
func (r *ProviderRegistry) Register(p *Provider) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid provider %q: %w", p.ID, err)
	}
	if _, exists := r.providers[p.ID]; exists {
		return fmt.Errorf("provider %q already registered", p.ID)
	}
	r.providers[p.ID] = p
	return nil
}

// Get returns a provider by ID, or nil if not found.
func (r *ProviderRegistry) Get(id string) *Provider {
	return r.providers[id]
}

// List returns all registered providers.
func (r *ProviderRegistry) List() []*Provider {
	result := make([]*Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// Remove removes a provider from the registry.
func (r *ProviderRegistry) Remove(id string) {
	delete(r.providers, id)
}
