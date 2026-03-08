package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// NoopSandbox executes commands directly without any isolation.
// Suitable for development and self-hosted single-user deployments.
type NoopSandbox struct{}

// NewNoopSandbox creates a sandbox that provides no isolation.
func NewNoopSandbox() *NoopSandbox {
	return &NoopSandbox{}
}

func (s *NoopSandbox) Level() IsolationLevel {
	return IsolationNone
}

func (s *NoopSandbox) Available() bool {
	return true
}

func (s *NoopSandbox) Close() error {
	return nil
}

func (s *NoopSandbox) Execute(ctx context.Context, command string, policy *Policy) (*ExecutionResult, error) {
	if strings.TrimSpace(command) == "" {
		return nil, ErrEmptyCommand
	}

	// Apply timeout from policy if set.
	var execCtx context.Context
	var cancel context.CancelFunc
	if policy != nil && policy.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, policy.Timeout)
	} else {
		execCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	start := time.Now()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", command)
	}

	if policy != nil && policy.Filesystem.WorkingDir != "" {
		cmd.Dir = policy.Filesystem.WorkingDir
	}

	// Set environment if policy specifies it.
	if policy != nil && !policy.InheritEnv {
		env := make([]string, 0, len(policy.Environment))
		for k, v := range policy.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		if len(env) > 0 {
			cmd.Env = env
		}
	} else if policy != nil && len(policy.Environment) > 0 {
		cmd.Env = append(cmd.Environ(), envSlice(policy.Environment)...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := &ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	// Enforce output size limit.
	if policy != nil && policy.MaxOutputBytes > 0 {
		result.Stdout = truncate(result.Stdout, policy.MaxOutputBytes)
		result.Stderr = truncate(result.Stderr, policy.MaxOutputBytes)
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Killed = true
			result.KillReason = "timeout"
			result.ExitCode = -1
			return result, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("sandbox: execution failed: %w", err)
		}
	}

	return result, nil
}

// envSlice converts a map to a slice of KEY=VALUE strings.
func envSlice(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// truncate limits a string to maxBytes, appending a truncation notice.
func truncate(s string, maxBytes int64) string {
	if int64(len(s)) <= maxBytes {
		return s
	}
	return s[:maxBytes] + fmt.Sprintf("\n... (truncated, %d bytes total)", len(s))
}
