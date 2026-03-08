// Package sandbox provides agent execution isolation with configurable
// security policies. It supports multiple isolation levels:
//
//   - None: direct execution (development/self-hosted single-user)
//   - Process: OS-level resource limits and restricted environment
//   - Container: full container isolation via OCI runtime (gVisor runsc, runc)
//
// The package is designed for CGO_ENABLED=0 builds and works cross-platform
// with graceful degradation on non-Linux systems.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// IsolationLevel defines the strength of sandbox isolation.
type IsolationLevel string

const (
	// IsolationNone provides no isolation — commands run directly.
	// Suitable for self-hosted single-user deployments.
	IsolationNone IsolationLevel = "none"

	// IsolationProcess provides OS-level process isolation with resource
	// limits (memory, CPU time, file size, process count) and restricted
	// environment variables. Works on all platforms.
	IsolationProcess IsolationLevel = "process"

	// IsolationContainer provides full container isolation using an OCI
	// runtime (gVisor runsc preferred, runc as fallback). Requires Linux
	// and an installed runtime. Provides PID/network/mount namespace
	// isolation, seccomp filtering, and filesystem restrictions.
	IsolationContainer IsolationLevel = "container"
)

// ValidIsolationLevel returns true if the level is recognized.
func ValidIsolationLevel(level IsolationLevel) bool {
	switch level {
	case IsolationNone, IsolationProcess, IsolationContainer:
		return true
	}
	return false
}

// ExecutionResult contains the output and metadata from a sandboxed execution.
type ExecutionResult struct {
	// Stdout is the standard output from the command.
	Stdout string `json:"stdout"`
	// Stderr is the standard error from the command.
	Stderr string `json:"stderr"`
	// ExitCode is the process exit code (0 = success).
	ExitCode int `json:"exit_code"`
	// Duration is how long the command took to execute.
	Duration time.Duration `json:"duration_ms"`
	// Killed is true if the process was killed (timeout, OOM, etc.).
	Killed bool `json:"killed"`
	// KillReason describes why the process was killed, if applicable.
	KillReason string `json:"kill_reason,omitempty"`
}

// Success returns true if the command exited with code 0 and was not killed.
func (r *ExecutionResult) Success() bool {
	return r.ExitCode == 0 && !r.Killed
}

// Output returns combined stdout and stderr similar to exec.CombinedOutput.
func (r *ExecutionResult) Output() string {
	if r.Stderr == "" {
		return r.Stdout
	}
	if r.Stdout == "" {
		return r.Stderr
	}
	return r.Stdout + "\n" + r.Stderr
}

// Sandbox defines the interface for executing commands in an isolated
// environment. Implementations must be safe for concurrent use.
type Sandbox interface {
	// Execute runs a command within the sandbox according to the given policy.
	// The context controls cancellation and deadline; the policy defines
	// resource limits and access restrictions.
	Execute(ctx context.Context, command string, policy *Policy) (*ExecutionResult, error)

	// Level returns the isolation level this sandbox provides.
	Level() IsolationLevel

	// Available returns true if this sandbox type can operate on the current
	// system. For example, ContainerSandbox requires an OCI runtime.
	Available() bool

	// Close releases any resources held by the sandbox.
	Close() error
}

// Errors returned by sandbox implementations.
var (
	ErrNilPolicy         = errors.New("sandbox: policy is nil")
	ErrEmptyCommand      = errors.New("sandbox: command is empty")
	ErrUnavailable       = errors.New("sandbox: isolation level not available on this system")
	ErrRuntimeNotFound   = errors.New("sandbox: OCI runtime not found")
	ErrResourceExceeded  = errors.New("sandbox: resource limit exceeded")
	ErrNetworkDenied     = errors.New("sandbox: network access denied by policy")
	ErrPathDenied        = errors.New("sandbox: path access denied by policy")
	ErrCommandDenied     = errors.New("sandbox: command denied by policy")
	ErrTimeout           = errors.New("sandbox: execution timed out")
	ErrOOMKilled         = errors.New("sandbox: process killed (out of memory)")
	ErrSandboxSetupFailed = errors.New("sandbox: failed to set up isolation")
)

// New creates a sandbox at the requested isolation level. If the requested
// level is not available, it returns an error rather than silently downgrading.
// Use NewWithFallback for automatic fallback behavior.
func New(level IsolationLevel, opts ...Option) (Sandbox, error) {
	if !ValidIsolationLevel(level) {
		return nil, fmt.Errorf("sandbox: unknown isolation level %q", level)
	}

	cfg := &sandboxConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	switch level {
	case IsolationNone:
		return NewNoopSandbox(), nil
	case IsolationProcess:
		return NewProcessSandbox(cfg), nil
	case IsolationContainer:
		cs := NewContainerSandbox(cfg)
		if !cs.Available() {
			return nil, ErrRuntimeNotFound
		}
		return cs, nil
	default:
		return nil, fmt.Errorf("sandbox: unsupported isolation level %q", level)
	}
}

// NewWithFallback creates a sandbox at the requested isolation level, falling
// back to lower levels if the requested one is not available. The fallback
// order is: container → process → none.
func NewWithFallback(preferred IsolationLevel, opts ...Option) Sandbox {
	cfg := &sandboxConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	switch preferred {
	case IsolationContainer:
		cs := NewContainerSandbox(cfg)
		if cs.Available() {
			return cs
		}
		return NewProcessSandbox(cfg)
	case IsolationProcess:
		return NewProcessSandbox(cfg)
	default:
		return NewNoopSandbox()
	}
}

// sandboxConfig holds common configuration for sandbox constructors.
type sandboxConfig struct {
	// RuntimePath is the path to the OCI runtime binary (for container sandbox).
	RuntimePath string
	// RootDir is the root directory for sandbox filesystems.
	RootDir string
	// DefaultPolicy is applied when Execute is called with nil policy.
	DefaultPolicy *Policy
}

// Option configures a sandbox.
type Option func(*sandboxConfig)

// WithRuntimePath sets the OCI runtime binary path.
func WithRuntimePath(path string) Option {
	return func(c *sandboxConfig) {
		c.RuntimePath = path
	}
}

// WithRootDir sets the root directory for sandbox filesystems.
func WithRootDir(dir string) Option {
	return func(c *sandboxConfig) {
		c.RootDir = dir
	}
}

// WithDefaultPolicy sets the default policy for executions.
func WithDefaultPolicy(p *Policy) Option {
	return func(c *sandboxConfig) {
		c.DefaultPolicy = p
	}
}
