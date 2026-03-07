package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/standardws/operator/pkg/auth"
)

// Service manages OAuth 2.0 authorization flows with PKCE support.
type Service struct {
	registry   *ProviderRegistry
	stateStore StateStore
	httpClient *http.Client
}

// ServiceConfig configures the OAuth service.
type ServiceConfig struct {
	Registry   *ProviderRegistry
	StateStore StateStore
	HTTPClient *http.Client // optional, defaults to http.DefaultClient
}

// NewService creates a new OAuth service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Registry == nil {
		return nil, fmt.Errorf("provider registry is required")
	}
	if cfg.StateStore == nil {
		return nil, fmt.Errorf("state store is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Service{
		registry:   cfg.Registry,
		stateStore: cfg.StateStore,
		httpClient: client,
	}, nil
}

// StartFlowResult contains the authorization URL and state for an OAuth flow.
type StartFlowResult struct {
	AuthURL  string `json:"auth_url"`
	State    string `json:"state"`
	Provider string `json:"provider"`
}

// StartFlow initiates an OAuth authorization flow for a user.
// It generates a state token and (if PKCE is enabled) a code verifier,
// persists them, and returns the authorization URL.
func (s *Service) StartFlow(userID, providerID string, scopes []string, redirectAfter string) (*StartFlowResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if providerID == "" {
		return nil, fmt.Errorf("provider ID is required")
	}

	provider := s.registry.Get(providerID)
	if provider == nil {
		return nil, fmt.Errorf("provider %q not found", providerID)
	}

	stateToken, err := generateStateToken()
	if err != nil {
		return nil, fmt.Errorf("generating state token: %w", err)
	}

	// Merge default and requested scopes.
	scopeSet := make(map[string]bool)
	for _, sc := range provider.Scopes {
		scopeSet[sc] = true
	}
	for _, sc := range scopes {
		scopeSet[sc] = true
	}
	scopeList := make([]string, 0, len(scopeSet))
	for sc := range scopeSet {
		scopeList = append(scopeList, sc)
	}
	scopeStr := strings.Join(scopeList, " ")

	// Generate PKCE if enabled.
	var codeVerifier, codeChallenge string
	if provider.UsePKCE {
		pkce, err := auth.GeneratePKCE()
		if err != nil {
			return nil, fmt.Errorf("generating PKCE: %w", err)
		}
		codeVerifier = pkce.CodeVerifier
		codeChallenge = pkce.CodeChallenge
	}

	// Persist state.
	oauthState := &OAuthState{
		UserID:       userID,
		ProviderID:   providerID,
		State:        stateToken,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectAfter,
		Scopes:       scopeStr,
	}
	if err := s.stateStore.Create(oauthState); err != nil {
		return nil, fmt.Errorf("persisting state: %w", err)
	}

	// Build authorization URL.
	authURL := s.buildAuthURL(provider, stateToken, scopeStr, codeChallenge)

	return &StartFlowResult{
		AuthURL:  authURL,
		State:    stateToken,
		Provider: providerID,
	}, nil
}

// HandleCallback processes the OAuth callback, exchanging the authorization code
// for tokens. Returns the token response.
func (s *Service) HandleCallback(stateToken, code string) (*TokenResponse, error) {
	if stateToken == "" {
		return nil, fmt.Errorf("state token is required")
	}
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	// Look up state.
	oauthState, err := s.stateStore.GetByState(stateToken)
	if err != nil {
		return nil, fmt.Errorf("looking up state: %w", err)
	}
	if oauthState == nil {
		return nil, fmt.Errorf("invalid or unknown state token")
	}

	// Check expiry.
	if time.Now().UTC().After(oauthState.ExpiresAt) {
		return nil, fmt.Errorf("state token expired")
	}

	// Check if already used.
	if oauthState.Used {
		return nil, fmt.Errorf("state token already used")
	}

	// Mark as used immediately (prevent replay).
	if err := s.stateStore.MarkUsed(oauthState.ID); err != nil {
		return nil, fmt.Errorf("marking state used: %w", err)
	}

	// Look up provider.
	provider := s.registry.Get(oauthState.ProviderID)
	if provider == nil {
		return nil, fmt.Errorf("provider %q not found", oauthState.ProviderID)
	}

	// Exchange code for tokens.
	tokenResp, err := s.exchangeCode(provider, code, oauthState.CodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	tokenResp.ProviderID = oauthState.ProviderID
	tokenResp.UserID = oauthState.UserID

	return tokenResp, nil
}

// RefreshToken exchanges a refresh token for a new access token.
func (s *Service) RefreshToken(providerID, refreshToken string) (*TokenResponse, error) {
	if providerID == "" {
		return nil, fmt.Errorf("provider ID is required")
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	provider := s.registry.Get(providerID)
	if provider == nil {
		return nil, fmt.Errorf("provider %q not found", providerID)
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {provider.ClientID},
	}
	if provider.ClientSecret != "" {
		data.Set("client_secret", provider.ClientSecret)
	}

	resp, err := s.httpClient.PostForm(provider.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	tokenResp, err := parseTokenResponse(body)
	if err != nil {
		return nil, err
	}

	// Preserve refresh token if not returned.
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = refreshToken
	}
	tokenResp.ProviderID = providerID

	return tokenResp, nil
}

// GetRegistry returns the provider registry.
func (s *Service) GetRegistry() *ProviderRegistry {
	return s.registry
}

// buildAuthURL constructs the authorization URL with all required parameters.
func (s *Service) buildAuthURL(provider *Provider, state, scopes, codeChallenge string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {provider.ClientID},
		"redirect_uri":  {provider.RedirectURL},
		"scope":         {scopes},
		"state":         {state},
	}

	if provider.UsePKCE && codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}

	// Add extra parameters.
	for k, v := range provider.ExtraAuthParams {
		params.Set(k, v)
	}

	return provider.AuthURL + "?" + params.Encode()
}

// exchangeCode exchanges an authorization code for tokens at the provider's token endpoint.
func (s *Service) exchangeCode(provider *Provider, code, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {provider.RedirectURL},
		"client_id":    {provider.ClientID},
	}
	if provider.ClientSecret != "" {
		data.Set("client_secret", provider.ClientSecret)
	}
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	resp, err := s.httpClient.PostForm(provider.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return parseTokenResponse(body)
}

// parseTokenResponse parses a standard OAuth 2.0 token response.
func parseTokenResponse(body []byte) (*TokenResponse, error) {
	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		IDToken      string `json:"id_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	if raw.Error != "" {
		desc := raw.ErrorDesc
		if desc == "" {
			desc = raw.Error
		}
		return nil, fmt.Errorf("oauth error: %s", desc)
	}

	if raw.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	resp := &TokenResponse{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		TokenType:    raw.TokenType,
		ExpiresIn:    raw.ExpiresIn,
		Scope:        raw.Scope,
		IDToken:      raw.IDToken,
	}

	if raw.ExpiresIn > 0 {
		resp.ExpiresAt = time.Now().UTC().Add(time.Duration(raw.ExpiresIn) * time.Second)
	}

	return resp, nil
}
