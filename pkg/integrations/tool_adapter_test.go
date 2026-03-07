package integrations

import (
	"context"
	"testing"

	"github.com/standardws/operator/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationTool_Implements(t *testing.T) {
	var _ tools.Tool = (*IntegrationTool)(nil)
}

func TestIntegrationTool_Name(t *testing.T) {
	tm := ToolManifest{Name: "google_list_emails", Description: "List emails"}
	tool := NewIntegrationTool("google", tm, nil)
	assert.Equal(t, "google_list_emails", tool.Name())
}

func TestIntegrationTool_Description(t *testing.T) {
	tm := ToolManifest{Name: "test", Description: "Test description"}
	tool := NewIntegrationTool("test", tm, nil)
	assert.Equal(t, "Test description", tool.Description())
}

func TestIntegrationTool_Parameters(t *testing.T) {
	params := map[string]any{"type": "object"}
	tm := ToolManifest{Name: "test", Description: "Test", Parameters: params}
	tool := NewIntegrationTool("test", tm, nil)
	assert.Equal(t, params, tool.Parameters())
}

func TestIntegrationTool_IntegrationID(t *testing.T) {
	tm := ToolManifest{Name: "test", Description: "Test"}
	tool := NewIntegrationTool("google", tm, nil)
	assert.Equal(t, "google", tool.IntegrationID())
}

func TestIntegrationTool_RequiredScopes(t *testing.T) {
	tm := ToolManifest{Name: "test", Description: "Test", RequiredScopes: []string{"read", "write"}}
	tool := NewIntegrationTool("google", tm, nil)
	assert.Equal(t, []string{"read", "write"}, tool.RequiredScopes())
}

func TestIntegrationTool_Execute_NoExecutor(t *testing.T) {
	tm := ToolManifest{Name: "test", Description: "Test"}
	tool := NewIntegrationTool("google", tm, nil)
	result := tool.Execute(context.Background(), nil)
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "no executor configured")
}

func TestIntegrationTool_Execute_Success(t *testing.T) {
	executor := func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error) {
		return `{"emails": []}`, nil
	}
	tm := ToolManifest{Name: "google_list", Description: "List"}
	tool := NewIntegrationTool("google", tm, executor)

	result := tool.Execute(context.Background(), map[string]any{"limit": 10})
	assert.False(t, result.IsError)
	assert.Equal(t, `{"emails": []}`, result.ForLLM)
}

func TestIntegrationTool_Execute_Error(t *testing.T) {
	executor := func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error) {
		return "", assert.AnError
	}
	tm := ToolManifest{Name: "google_list", Description: "List"}
	tool := NewIntegrationTool("google", tm, executor)

	result := tool.Execute(context.Background(), nil)
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "failed")
}

func TestIntegrationTool_Execute_PassesArgs(t *testing.T) {
	var capturedInteg, capturedTool string
	var capturedArgs map[string]any
	executor := func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error) {
		capturedInteg = integrationID
		capturedTool = toolName
		capturedArgs = args
		return "ok", nil
	}
	tm := ToolManifest{Name: "shopify_orders", Description: "Get orders"}
	tool := NewIntegrationTool("shopify", tm, executor)

	args := map[string]any{"status": "open", "limit": float64(25)}
	tool.Execute(context.Background(), args)

	assert.Equal(t, "shopify", capturedInteg)
	assert.Equal(t, "shopify_orders", capturedTool)
	assert.Equal(t, args, capturedArgs)
}

// --- ToolRegistrar ---

func TestToolRegistrar_RegisterAll(t *testing.T) {
	integReg := NewIntegrationRegistry()
	require.NoError(t, integReg.Register(validIntegration("google")))
	require.NoError(t, integReg.Register(validIntegration("shopify")))

	toolReg := tools.NewToolRegistry()
	executor := func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error) {
		return "ok", nil
	}

	registrar := NewToolRegistrar(integReg, toolReg, executor)
	count := registrar.RegisterAll()
	assert.Equal(t, 4, count) // 2 tools per integration

	// Verify tools are registered
	assert.Equal(t, 4, toolReg.Count())
	_, ok := toolReg.Get("google_list")
	assert.True(t, ok)
	_, ok = toolReg.Get("shopify_get")
	assert.True(t, ok)
}

func TestToolRegistrar_RegisterIntegration(t *testing.T) {
	integReg := NewIntegrationRegistry()
	require.NoError(t, integReg.Register(validIntegration("google")))

	toolReg := tools.NewToolRegistry()
	executor := func(ctx context.Context, integrationID, toolName string, args map[string]any) (string, error) {
		return "ok", nil
	}

	registrar := NewToolRegistrar(integReg, toolReg, executor)
	count := registrar.RegisterIntegration("google")
	assert.Equal(t, 2, count)
	assert.Equal(t, 2, toolReg.Count())
}

func TestToolRegistrar_RegisterIntegration_NotFound(t *testing.T) {
	integReg := NewIntegrationRegistry()
	toolReg := tools.NewToolRegistry()
	registrar := NewToolRegistrar(integReg, toolReg, nil)
	count := registrar.RegisterIntegration("nonexistent")
	assert.Equal(t, 0, count)
}

// --- Context helpers ---

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user123")
	assert.Equal(t, "user123", userIDFromContext(ctx))
}

func TestUserIDFromContext_Empty(t *testing.T) {
	assert.Equal(t, "", userIDFromContext(context.Background()))
}
