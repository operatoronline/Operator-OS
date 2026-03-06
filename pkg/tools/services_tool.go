package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/standardws/operator/pkg/services"
)

// ServicesTool lets agents inspect and manage the managed services infrastructure.
type ServicesTool struct {
	manager *services.Manager
}

// NewServicesTool creates a services management tool.
func NewServicesTool(manager *services.Manager) *ServicesTool {
	return &ServicesTool{manager: manager}
}

func (t *ServicesTool) Name() string { return "services" }

func (t *ServicesTool) Description() string {
	return "Manage Operator-OS infrastructure services. Check which services are available " +
		"on the current hardware, view service status, start or stop services, " +
		"and get the hardware profile."
}

func (t *ServicesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"status", "start", "stop", "profile"},
				"description": "The management action to perform",
			},
			"service": map[string]any{
				"type":        "string",
				"enum":        []string{"browser", "sandbox", "repo"},
				"description": "Target service (required for start/stop)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *ServicesTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)

	switch action {
	case "profile":
		profile := t.manager.Profile()
		available := t.manager.AvailableServices()
		names := make([]string, len(available))
		for i, s := range available {
			names[i] = string(s)
		}
		result := map[string]any{
			"profile":            string(profile),
			"description":        services.ProfileDescription(profile),
			"available_services": names,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return NewToolResult(string(data))

	case "status":
		all := t.manager.StatusAll()
		data, _ := json.MarshalIndent(all, "", "  ")
		return NewToolResult(string(data))

	case "start":
		svc, ok := args["service"].(string)
		if !ok || svc == "" {
			return ErrorResult("'service' parameter required for start action")
		}
		stype := services.ServiceType(svc)
		info, err := t.manager.EnsureRunning(ctx, stype)
		if err != nil {
			return ErrorResult(fmt.Sprintf("Failed to start %s: %s", svc, err))
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		return NewToolResult(string(data))

	case "stop":
		svc, ok := args["service"].(string)
		if !ok || svc == "" {
			return ErrorResult("'service' parameter required for stop action")
		}
		stype := services.ServiceType(svc)
		if err := t.manager.Stop(ctx, stype); err != nil {
			return ErrorResult(fmt.Sprintf("Failed to stop %s: %s", svc, err))
		}
		return NewToolResult(fmt.Sprintf("Service %s stopped", svc))

	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s", action))
	}
}
