package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/standardws/operator/pkg/tools"
)

// IntegrationToolExecutor is the function signature for executing an integration tool.
// It receives the integration ID, tool name, arguments, and user context,
// and returns the result string and any error.
type IntegrationToolExecutor func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error)

// IntegrationTool adapts a ToolManifest into a tools.Tool.
// It delegates execution to an IntegrationToolExecutor which handles
// credential retrieval, API calls, etc.
type IntegrationTool struct {
	manifest      ToolManifest
	integrationID string
	executor      IntegrationToolExecutor
}

var _ tools.Tool = (*IntegrationTool)(nil)

// NewIntegrationTool creates a tool adapter from a manifest.
func NewIntegrationTool(integrationID string, manifest ToolManifest, executor IntegrationToolExecutor) *IntegrationTool {
	return &IntegrationTool{
		manifest:      manifest,
		integrationID: integrationID,
		executor:      executor,
	}
}

func (t *IntegrationTool) Name() string        { return t.manifest.Name }
func (t *IntegrationTool) Description() string  { return t.manifest.Description }
func (t *IntegrationTool) Parameters() map[string]any { return t.manifest.Parameters }

func (t *IntegrationTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	if t.executor == nil {
		return tools.ErrorResult(fmt.Sprintf("integration %q tool %q has no executor configured", t.integrationID, t.manifest.Name))
	}
	result, err := t.executor(ctx, t.integrationID, t.manifest.Name, args)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("integration tool %q failed: %v", t.manifest.Name, err))
	}
	return tools.NewToolResult(result)
}

// IntegrationID returns the parent integration ID.
func (t *IntegrationTool) IntegrationID() string { return t.integrationID }

// RequiredScopes returns the OAuth scopes this tool needs.
func (t *IntegrationTool) RequiredScopes() []string { return t.manifest.RequiredScopes }

// ToolRegistrar registers integration tools into a tools.ToolRegistry.
type ToolRegistrar struct {
	registry    *IntegrationRegistry
	toolReg     *tools.ToolRegistry
	executor    IntegrationToolExecutor
}

// NewToolRegistrar creates a registrar that bridges integration manifests to the tool registry.
func NewToolRegistrar(integrationRegistry *IntegrationRegistry, toolRegistry *tools.ToolRegistry, executor IntegrationToolExecutor) *ToolRegistrar {
	return &ToolRegistrar{
		registry: integrationRegistry,
		toolReg:  toolRegistry,
		executor: executor,
	}
}

// RegisterAll registers tools from all integrations into the tool registry.
// Returns the number of tools registered.
func (r *ToolRegistrar) RegisterAll() int {
	count := 0
	for _, integ := range r.registry.List() {
		count += r.RegisterIntegration(integ.ID)
	}
	return count
}

// RegisterIntegration registers tools from a single integration.
// Returns the number of tools registered.
func (r *ToolRegistrar) RegisterIntegration(integrationID string) int {
	integ := r.registry.Get(integrationID)
	if integ == nil {
		return 0
	}
	count := 0
	for _, tm := range integ.Tools {
		tool := NewIntegrationTool(integrationID, tm, r.executor)
		r.toolReg.Register(tool)
		count++
	}
	return count
}

// --- HTTP API Tool Executor ---

// HTTPToolExecutorConfig configures the HTTP-based tool executor.
type HTTPToolExecutorConfig struct {
	// BaseURLResolver resolves the base URL for an integration's API.
	BaseURLResolver func(integrationID string) string
	// TokenResolver retrieves the access token for a user+integration.
	TokenResolver func(ctx context.Context, userID, integrationID string) (string, error)
	// EndpointResolver resolves a tool name to an HTTP method + path.
	EndpointResolver func(integrationID, toolName string) (method, path string, err error)
	// HTTPClient is the HTTP client to use. Defaults to http.DefaultClient.
	HTTPClient *http.Client
}

// NewHTTPToolExecutor creates an IntegrationToolExecutor that makes HTTP API calls.
// This is a generic executor for REST-based integrations.
func NewHTTPToolExecutor(cfg HTTPToolExecutorConfig) IntegrationToolExecutor {
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error) {
		// Resolve user from context
		userID := userIDFromContext(ctx)
		if userID == "" {
			return "", fmt.Errorf("no user ID in context")
		}

		// Get access token
		if cfg.TokenResolver == nil {
			return "", fmt.Errorf("no token resolver configured")
		}
		token, err := cfg.TokenResolver(ctx, userID, integrationID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve token: %w", err)
		}

		// Resolve endpoint
		if cfg.EndpointResolver == nil {
			return "", fmt.Errorf("no endpoint resolver configured")
		}
		method, path, err := cfg.EndpointResolver(integrationID, toolName)
		if err != nil {
			return "", fmt.Errorf("failed to resolve endpoint: %w", err)
		}

		// Build URL
		baseURL := ""
		if cfg.BaseURLResolver != nil {
			baseURL = cfg.BaseURLResolver(integrationID)
		}
		url := baseURL + path

		// Build request
		var body *strings.Reader
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
			bodyBytes, err := json.Marshal(args)
			if err != nil {
				return "", fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = strings.NewReader(string(bodyBytes))
		}

		var req *http.Request
		if body != nil {
			req, err = http.NewRequestWithContext(ctx, method, url, body)
		} else {
			req, err = http.NewRequestWithContext(ctx, method, url, nil)
		}
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		var respBody strings.Builder
		_, err = fmt.Fscanf(resp.Body, "%s", &respBody)
		// Use io.ReadAll instead for proper reading
		respBytes := make([]byte, 0, 4096)
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				respBytes = append(respBytes, buf[:n]...)
			}
			if readErr != nil {
				break
			}
		}

		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBytes))
		}

		return string(respBytes), nil
	}
}

// userIDFromContext extracts user_id from context.
// This mirrors the pattern from pkg/users.
type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID stores a user ID in the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func userIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}
