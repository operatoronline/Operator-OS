package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/standardws/operator/pkg/services"
)

// BrowserTool gives agents access to a real browser via the go-browser service.
// On first call, the browser service starts automatically (lazy initialization).
type BrowserTool struct {
	manager *services.Manager
}

// NewBrowserTool creates a browser tool backed by the service manager.
func NewBrowserTool(manager *services.Manager) *BrowserTool {
	return &BrowserTool{manager: manager}
}

func (t *BrowserTool) Name() string { return "browser" }

func (t *BrowserTool) Description() string {
	return "Control a real browser to navigate websites, click elements, fill forms, take screenshots, and extract content. " +
		"This is a full browser (not headless) — undetectable by anti-bot systems. " +
		"Use for web scraping, form filling, authentication, and any task that requires real browser interaction."
}

func (t *BrowserTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"navigate", "click", "type", "screenshot",
					"get_dom", "execute_js", "wait_for", "scroll",
					"get_text", "get_links", "back", "forward", "refresh",
				},
				"description": "The browser action to perform",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL to navigate to (for 'navigate' action)",
			},
			"selector": map[string]any{
				"type":        "string",
				"description": "CSS selector for the target element",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "Text to type (for 'type' action) or JS code (for 'execute_js')",
			},
			"timeout_ms": map[string]any{
				"type":        "integer",
				"description": "Timeout in milliseconds for wait operations",
				"default":     10000,
			},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Ensure browser service is running (lazy start)
	info, err := t.manager.EnsureRunning(ctx, services.ServiceBrowser)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Browser service unavailable: %s", err))
	}

	endpoint := t.manager.BrowserEndpoint()

	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("'action' parameter is required")
	}

	// Build the request payload for the browser service
	payload := map[string]any{
		"action": action,
	}

	// Forward all relevant parameters
	for _, key := range []string{"url", "selector", "text", "timeout_ms"} {
		if v, ok := args[key]; ok {
			payload[key] = v
		}
	}

	payloadBytes, _ := json.Marshal(payload)
	result, err := endpoint.Post(ctx, "/api/v1/action", "application/json", string(payloadBytes))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Browser action failed: %s", err))
	}

	_ = info // used for future metrics

	return NewToolResult(string(result))
}
