package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderValidate(t *testing.T) {
	valid := &Provider{
		ID:          "test",
		Name:        "Test Provider",
		AuthURL:     "https://example.com/auth",
		TokenURL:    "https://example.com/token",
		ClientID:    "client-123",
		RedirectURL: "https://app.example.com/callback",
		UsePKCE:     true,
	}
	assert.NoError(t, valid.Validate())

	tests := []struct {
		name    string
		modify  func(p *Provider)
		wantErr string
	}{
		{"empty ID", func(p *Provider) { p.ID = "" }, "provider ID is required"},
		{"empty name", func(p *Provider) { p.Name = "" }, "provider name is required"},
		{"empty auth URL", func(p *Provider) { p.AuthURL = "" }, "auth URL is required"},
		{"empty token URL", func(p *Provider) { p.TokenURL = "" }, "token URL is required"},
		{"empty client ID", func(p *Provider) { p.ClientID = "" }, "client ID is required"},
		{"empty redirect URL", func(p *Provider) { p.RedirectURL = "" }, "redirect URL is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				ID:          valid.ID,
				Name:        valid.Name,
				AuthURL:     valid.AuthURL,
				TokenURL:    valid.TokenURL,
				ClientID:    valid.ClientID,
				RedirectURL: valid.RedirectURL,
			}
			tt.modify(p)
			err := p.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestProviderScopeString(t *testing.T) {
	p := &Provider{Scopes: []string{"openid", "email", "profile"}}
	assert.Equal(t, "openid email profile", p.ScopeString())

	empty := &Provider{}
	assert.Equal(t, "", empty.ScopeString())
}

func TestProviderRegistry(t *testing.T) {
	reg := NewProviderRegistry()
	require.NotNil(t, reg)

	// List empty.
	assert.Empty(t, reg.List())

	// Register valid provider.
	p := &Provider{
		ID:          "google",
		Name:        "Google",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		ClientID:    "client-123",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid", "email"},
		UsePKCE:     true,
	}
	require.NoError(t, reg.Register(p))

	// Get.
	got := reg.Get("google")
	assert.Equal(t, "Google", got.Name)

	// Get not found.
	assert.Nil(t, reg.Get("nonexistent"))

	// List.
	list := reg.List()
	assert.Len(t, list, 1)

	// Duplicate registration.
	err := reg.Register(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Invalid provider.
	err = reg.Register(&Provider{ID: "bad"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid provider")

	// Remove.
	reg.Remove("google")
	assert.Nil(t, reg.Get("google"))
	assert.Empty(t, reg.List())
}

func TestRegistryMultipleProviders(t *testing.T) {
	reg := NewProviderRegistry()

	providers := []*Provider{
		{ID: "google", Name: "Google", AuthURL: "https://google.com/auth", TokenURL: "https://google.com/token", ClientID: "g1", RedirectURL: "https://app.com/cb"},
		{ID: "github", Name: "GitHub", AuthURL: "https://github.com/auth", TokenURL: "https://github.com/token", ClientID: "h1", RedirectURL: "https://app.com/cb"},
		{ID: "shopify", Name: "Shopify", AuthURL: "https://shopify.com/auth", TokenURL: "https://shopify.com/token", ClientID: "s1", RedirectURL: "https://app.com/cb"},
	}

	for _, p := range providers {
		require.NoError(t, reg.Register(p))
	}

	assert.Len(t, reg.List(), 3)
	assert.NotNil(t, reg.Get("github"))
	assert.Equal(t, "Shopify", reg.Get("shopify").Name)
}
