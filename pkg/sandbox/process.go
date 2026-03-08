package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ProcessSandbox provides OS-level process isolation with resource limits.
// It uses setrlimit (via SysProcAttr) on Unix systems and falls back to
// basic process execution on other platforms. This sandbox does NOT provide
// namespace isolation — use ContainerSandbox for full isolation.
type ProcessSandbox struct {
	config *sandboxConfig
}

// NewProcessSandbox creates a process-level sandbox.
func NewProcessSandbox(cfg *sandboxConfig) *ProcessSandbox {
	if cfg == nil {
		cfg = &sandboxConfig{}
	}
	return &ProcessSandbox{config: cfg}
}

func (s *ProcessSandbox) Level() IsolationLevel {
	return IsolationProcess
}

func (s *ProcessSandbox) Available() bool {
	return true
}

func (s *ProcessSandbox) Close() error {
	return nil
}

func (s *ProcessSandbox) Execute(ctx context.Context, command string, policy *Policy) (*ExecutionResult, error) {
	if strings.TrimSpace(command) == "" {
		return nil, ErrEmptyCommand
	}
	if policy == nil {
		policy = s.config.DefaultPolicy
	}
	if policy == nil {
		policy = DefaultPolicy()
	}
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("sandbox: invalid policy: %w", err)
	}

	// Check command against policy.
	// Extract the base command name for allow/deny checks.
	baseName := extractBaseCommand(command)
	if !policy.IsCommandAllowed(baseName) {
		return nil, ErrCommandDenied
	}

	// Apply timeout.
	var execCtx context.Context
	var cancel context.CancelFunc
	if policy.Timeout > 0 {
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

	if policy.Filesystem.WorkingDir != "" {
		cmd.Dir = policy.Filesystem.WorkingDir
	}

	// Configure environment.
	if !policy.InheritEnv {
		env := buildRestrictedEnv(policy.Environment)
		cmd.Env = env
	} else if len(policy.Environment) > 0 {
		cmd.Env = append(cmd.Environ(), envSlice(policy.Environment)...)
	}

	// Apply OS-level resource limits.
	s.applyResourceLimits(cmd, policy)

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
	maxOutput := policy.MaxOutputBytes
	if maxOutput == 0 {
		maxOutput = 10 * 1024 * 1024 // 10 MB default
	}
	result.Stdout = truncate(result.Stdout, maxOutput)
	result.Stderr = truncate(result.Stderr, maxOutput)

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Killed = true
			result.KillReason = "timeout"
			result.ExitCode = -1
			return result, nil
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()

			// Check for OOM kill signal on Unix.
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() && status.Signal() == syscall.SIGKILL {
					result.Killed = true
					result.KillReason = "killed (possible OOM)"
				}
			}
		} else {
			return nil, fmt.Errorf("sandbox: execution failed: %w", err)
		}
	}

	return result, nil
}

// applyResourceLimits configures OS-level resource limits on the command.
// This is platform-specific and uses build-tag files for the actual implementation.
func (s *ProcessSandbox) applyResourceLimits(cmd *exec.Cmd, policy *Policy) {
	applyPlatformLimits(cmd, policy)
}

// buildRestrictedEnv creates a minimal environment with only the specified vars
// plus essential system variables.
func buildRestrictedEnv(env map[string]string) []string {
	result := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/tmp",
		"LANG=C.UTF-8",
	}
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// extractBaseCommand extracts the first command name from a shell command string.
func extractBaseCommand(command string) string {
	cmd := strings.TrimSpace(command)
	// Handle env prefixes like "FOO=bar cmd".
	for strings.Contains(cmd, "=") && !strings.HasPrefix(cmd, "=") {
		parts := strings.SplitN(cmd, " ", 2)
		if len(parts) < 2 {
			break
		}
		if !strings.Contains(parts[0], "=") {
			break
		}
		cmd = strings.TrimSpace(parts[1])
	}
	// Take the first word.
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	// Strip path prefixes.
	base := parts[0]
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	if idx := strings.LastIndex(base, "\\"); idx >= 0 {
		base = base[idx+1:]
	}
	return base
}
