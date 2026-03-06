package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/standardws/operator/pkg/services"
)

// SandboxTool gives agents secure, isolated code execution via the go-sandbox service.
// Supports Python, Node.js, Go, Rust, and Java in sandboxed containers.
type SandboxTool struct {
	manager *services.Manager
}

// NewSandboxTool creates a sandbox tool backed by the service manager.
func NewSandboxTool(manager *services.Manager) *SandboxTool {
	return &SandboxTool{manager: manager}
}

func (t *SandboxTool) Name() string { return "sandbox" }

func (t *SandboxTool) Description() string {
	return "Execute code in a secure, isolated sandbox. Supports Python, Node.js, Go, Rust, and Java. " +
		"Use for running scripts, data analysis, testing code, installing packages, and any task " +
		"requiring code execution in a safe environment. Each execution runs in its own container."
}

func (t *SandboxTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"run_code", "run_command", "write_file", "read_file",
					"install_package", "list_files", "create", "destroy",
				},
				"description": "The sandbox action to perform",
			},
			"language": map[string]any{
				"type":        "string",
				"enum":        []string{"python", "nodejs", "go", "rust", "java"},
				"description": "Programming language for code execution",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Code to execute (for 'run_code' action)",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Shell command to run (for 'run_command' action)",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File path within the sandbox",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "File content to write",
			},
			"package": map[string]any{
				"type":        "string",
				"description": "Package name to install (pip, npm, etc.)",
			},
			"sandbox_id": map[string]any{
				"type":        "string",
				"description": "ID of an existing sandbox to reuse (omit to create ephemeral)",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Execution timeout in seconds",
				"default":     30,
			},
		},
		"required": []string{"action"},
	}
}

func (t *SandboxTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Ensure sandbox service is running
	info, err := t.manager.EnsureRunning(ctx, services.ServiceSandbox)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Sandbox service unavailable: %s", err))
	}

	endpoint := t.manager.SandboxEndpoint()

	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("'action' parameter is required")
	}

	payload := map[string]any{
		"action": action,
	}

	// Forward all relevant parameters
	for _, key := range []string{"language", "code", "command", "path", "content", "package", "sandbox_id", "timeout_seconds"} {
		if v, ok := args[key]; ok {
			payload[key] = v
		}
	}

	payloadBytes, _ := json.Marshal(payload)
	result, err := endpoint.Post(ctx, "/api/v1/execute", "application/json", string(payloadBytes))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Sandbox execution failed: %s", err))
	}

	_ = info

	return NewToolResult(string(result))
}
