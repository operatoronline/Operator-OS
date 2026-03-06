package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/standardws/operator/pkg/services"
)

// RepoTool gives agents Git version control via the go-repo service.
// Supports creating repos, committing, branching, PRs, and file operations.
type RepoTool struct {
	manager *services.Manager
}

// NewRepoTool creates a repo tool backed by the service manager.
func NewRepoTool(manager *services.Manager) *RepoTool {
	return &RepoTool{manager: manager}
}

func (t *RepoTool) Name() string { return "repo" }

func (t *RepoTool) Description() string {
	return "Manage Git repositories with a built-in Git service. Create repos, commit changes, " +
		"create branches and pull requests, read and write files, and view diffs. " +
		"Use for version control, code management, and collaborative development workflows."
}

func (t *RepoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"create_repo", "list_repos", "delete_repo",
					"write_file", "read_file", "list_files", "delete_file",
					"commit", "diff",
					"create_branch", "list_branches", "delete_branch",
					"create_pr", "merge_pr", "list_prs",
				},
				"description": "The repository action to perform",
			},
			"repo": map[string]any{
				"type":        "string",
				"description": "Repository name",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File path within the repository",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "File content to write",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Commit message or PR description",
			},
			"branch": map[string]any{
				"type":        "string",
				"description": "Branch name",
			},
			"base_branch": map[string]any{
				"type":        "string",
				"description": "Base branch for PRs or new branches",
				"default":     "main",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Title for pull requests",
			},
			"pr_id": map[string]any{
				"type":        "integer",
				"description": "Pull request ID (for merge_pr)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *RepoTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Ensure repo service is running
	info, err := t.manager.EnsureRunning(ctx, services.ServiceRepo)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Repo service unavailable: %s", err))
	}

	endpoint := t.manager.RepoEndpoint()

	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("'action' parameter is required")
	}

	payload := map[string]any{
		"action": action,
	}

	for _, key := range []string{"repo", "path", "content", "message", "branch", "base_branch", "title", "pr_id"} {
		if v, ok := args[key]; ok {
			payload[key] = v
		}
	}

	payloadBytes, _ := json.Marshal(payload)
	result, err := endpoint.Post(ctx, "/api/v1/agent", "application/json", string(payloadBytes))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Repo operation failed: %s", err))
	}

	_ = info

	return NewToolResult(string(result))
}
