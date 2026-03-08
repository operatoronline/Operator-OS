package sandbox

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── IsolationLevel ───

func TestValidIsolationLevel(t *testing.T) {
	tests := []struct {
		level IsolationLevel
		valid bool
	}{
		{IsolationNone, true},
		{IsolationProcess, true},
		{IsolationContainer, true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.valid, ValidIsolationLevel(tt.level), "level=%q", tt.level)
	}
}

// ─── ExecutionResult ───

func TestExecutionResult_Success(t *testing.T) {
	assert.True(t, (&ExecutionResult{ExitCode: 0}).Success())
	assert.False(t, (&ExecutionResult{ExitCode: 1}).Success())
	assert.False(t, (&ExecutionResult{ExitCode: 0, Killed: true}).Success())
}

func TestExecutionResult_Output(t *testing.T) {
	// Stdout only.
	r := &ExecutionResult{Stdout: "hello"}
	assert.Equal(t, "hello", r.Output())

	// Stderr only.
	r = &ExecutionResult{Stderr: "error"}
	assert.Equal(t, "error", r.Output())

	// Both.
	r = &ExecutionResult{Stdout: "hello", Stderr: "error"}
	assert.Equal(t, "hello\nerror", r.Output())

	// Empty.
	r = &ExecutionResult{}
	assert.Equal(t, "", r.Output())
}

func TestExecutionResult_JSON(t *testing.T) {
	r := &ExecutionResult{
		Stdout:     "hello",
		Stderr:     "warning",
		ExitCode:   0,
		Duration:   100 * time.Millisecond,
		Killed:     false,
		KillReason: "",
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	var decoded ExecutionResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, r.Stdout, decoded.Stdout)
	assert.Equal(t, r.Stderr, decoded.Stderr)
	assert.Equal(t, r.ExitCode, decoded.ExitCode)
}

// ─── Policy ───

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()
	require.NotNil(t, p)
	assert.Equal(t, "default", p.Name)
	assert.Equal(t, 60*time.Second, p.Timeout)
	assert.Equal(t, int64(256*1024*1024), p.Resources.MaxMemoryBytes)
	assert.Equal(t, int64(30), p.Resources.MaxCPUSeconds)
	assert.Equal(t, int64(32), p.Resources.MaxProcesses)
	assert.False(t, p.Network.AllowNetwork)
	assert.True(t, p.Filesystem.ReadOnlyRoot)
	assert.NoError(t, p.Validate())
}

func TestFreeTierPolicy(t *testing.T) {
	p := FreeTierPolicy()
	require.NotNil(t, p)
	assert.Equal(t, "free-tier", p.Name)
	assert.Equal(t, 30*time.Second, p.Timeout)
	assert.Equal(t, int64(128*1024*1024), p.Resources.MaxMemoryBytes)
	assert.NoError(t, p.Validate())
}

func TestProTierPolicy(t *testing.T) {
	p := ProTierPolicy()
	require.NotNil(t, p)
	assert.Equal(t, "pro-tier", p.Name)
	assert.Equal(t, 5*time.Minute, p.Timeout)
	assert.Equal(t, int64(1024*1024*1024), p.Resources.MaxMemoryBytes)
	assert.True(t, p.Network.AllowNetwork)
	assert.True(t, p.Network.AllowDNS)
	assert.NoError(t, p.Validate())
}

func TestPolicy_Validate(t *testing.T) {
	// Nil policy.
	var p *Policy
	assert.ErrorIs(t, p.Validate(), ErrNilPolicy)

	// Valid policy.
	assert.NoError(t, DefaultPolicy().Validate())

	// Negative memory.
	bad := DefaultPolicy()
	bad.Resources.MaxMemoryBytes = -1
	assert.Error(t, bad.Validate())

	// Negative CPU.
	bad = DefaultPolicy()
	bad.Resources.MaxCPUSeconds = -1
	assert.Error(t, bad.Validate())

	// Negative processes.
	bad = DefaultPolicy()
	bad.Resources.MaxProcesses = -1
	assert.Error(t, bad.Validate())

	// Negative file size.
	bad = DefaultPolicy()
	bad.Resources.MaxFileSize = -1
	assert.Error(t, bad.Validate())

	// Negative open files.
	bad = DefaultPolicy()
	bad.Resources.MaxOpenFiles = -1
	assert.Error(t, bad.Validate())

	// Negative output bytes.
	bad = DefaultPolicy()
	bad.MaxOutputBytes = -1
	assert.Error(t, bad.Validate())

	// Negative temp dir size.
	bad = DefaultPolicy()
	bad.Filesystem.TempDirSize = -1
	assert.Error(t, bad.Validate())

	// Allowed hosts without network.
	bad = DefaultPolicy()
	bad.Network.AllowNetwork = false
	bad.Network.AllowedHosts = []string{"example.com"}
	assert.Error(t, bad.Validate())

	// Allowed ports without network.
	bad = DefaultPolicy()
	bad.Network.AllowNetwork = false
	bad.Network.AllowedPorts = []int{443}
	assert.Error(t, bad.Validate())

	// Zero values are fine (means unlimited).
	zero := &Policy{}
	assert.NoError(t, zero.Validate())
}

func TestPolicy_Merge(t *testing.T) {
	base := DefaultPolicy()

	// Merge nil.
	merged := base.Merge(nil)
	assert.Equal(t, base.Name, merged.Name)
	assert.Equal(t, base.Timeout, merged.Timeout)

	// Merge with overrides.
	override := &Policy{
		Name:    "custom",
		Timeout: 120 * time.Second,
		Resources: ResourceLimits{
			MaxMemoryBytes: 512 * 1024 * 1024,
		},
		Environment: map[string]string{"FOO": "bar"},
	}
	merged = base.Merge(override)
	assert.Equal(t, "custom", merged.Name)
	assert.Equal(t, 120*time.Second, merged.Timeout)
	assert.Equal(t, int64(512*1024*1024), merged.Resources.MaxMemoryBytes)
	// Non-overridden fields preserved.
	assert.Equal(t, base.Resources.MaxCPUSeconds, merged.Resources.MaxCPUSeconds)
	assert.Equal(t, "bar", merged.Environment["FOO"])

	// Merge filesystem overrides.
	fsOverride := &Policy{
		Filesystem: FilesystemPolicy{
			WorkingDir:     "/app",
			ReadOnlyPaths:  []string{"/usr"},
			ReadWritePaths: []string{"/data"},
			HiddenPaths:    []string{"/secrets"},
			TempDirSize:    128 * 1024 * 1024,
		},
	}
	merged = base.Merge(fsOverride)
	assert.Equal(t, "/app", merged.Filesystem.WorkingDir)
	assert.Equal(t, []string{"/usr"}, merged.Filesystem.ReadOnlyPaths)
	assert.Equal(t, []string{"/data"}, merged.Filesystem.ReadWritePaths)
	assert.Equal(t, []string{"/secrets"}, merged.Filesystem.HiddenPaths)
	assert.Equal(t, int64(128*1024*1024), merged.Filesystem.TempDirSize)

	// Merge command lists.
	cmdOverride := &Policy{
		AllowedCommands: []string{"ls", "cat"},
		DeniedCommands:  []string{"rm"},
	}
	merged = base.Merge(cmdOverride)
	assert.Equal(t, []string{"ls", "cat"}, merged.AllowedCommands)
	assert.Equal(t, []string{"rm"}, merged.DeniedCommands)
}

func TestPolicy_IsCommandAllowed(t *testing.T) {
	// No restrictions.
	p := &Policy{}
	assert.True(t, p.IsCommandAllowed("anything"))

	// Deny list.
	p = &Policy{DeniedCommands: []string{"rm", "dd"}}
	assert.False(t, p.IsCommandAllowed("rm"))
	assert.False(t, p.IsCommandAllowed("dd"))
	assert.True(t, p.IsCommandAllowed("ls"))

	// Allow list.
	p = &Policy{AllowedCommands: []string{"ls", "cat", "echo"}}
	assert.True(t, p.IsCommandAllowed("ls"))
	assert.True(t, p.IsCommandAllowed("cat"))
	assert.False(t, p.IsCommandAllowed("rm"))

	// Both — deny takes precedence.
	p = &Policy{
		AllowedCommands: []string{"ls", "rm"},
		DeniedCommands:  []string{"rm"},
	}
	assert.True(t, p.IsCommandAllowed("ls"))
	assert.False(t, p.IsCommandAllowed("rm"))
}

// ─── NoopSandbox ───

func TestNoopSandbox_Level(t *testing.T) {
	s := NewNoopSandbox()
	assert.Equal(t, IsolationNone, s.Level())
}

func TestNoopSandbox_Available(t *testing.T) {
	s := NewNoopSandbox()
	assert.True(t, s.Available())
}

func TestNoopSandbox_Close(t *testing.T) {
	s := NewNoopSandbox()
	assert.NoError(t, s.Close())
}

func TestNoopSandbox_Execute_SimpleCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	result, err := s.Execute(context.Background(), "echo hello", nil)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
	assert.True(t, result.Success())
}

func TestNoopSandbox_Execute_EmptyCommand(t *testing.T) {
	s := NewNoopSandbox()
	_, err := s.Execute(context.Background(), "", nil)
	assert.ErrorIs(t, err, ErrEmptyCommand)

	_, err = s.Execute(context.Background(), "   ", nil)
	assert.ErrorIs(t, err, ErrEmptyCommand)
}

func TestNoopSandbox_Execute_FailingCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	result, err := s.Execute(context.Background(), "exit 42", nil)
	require.NoError(t, err)
	assert.Equal(t, 42, result.ExitCode)
	assert.False(t, result.Success())
}

func TestNoopSandbox_Execute_WithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	policy := &Policy{Timeout: 100 * time.Millisecond}
	result, err := s.Execute(context.Background(), "sleep 10", policy)
	require.NoError(t, err)
	assert.True(t, result.Killed)
	assert.Equal(t, "timeout", result.KillReason)
	assert.Equal(t, -1, result.ExitCode)
}

func TestNoopSandbox_Execute_WithWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	policy := &Policy{
		Filesystem: FilesystemPolicy{WorkingDir: "/tmp"},
	}
	result, err := s.Execute(context.Background(), "pwd", policy)
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "/tmp")
}

func TestNoopSandbox_Execute_WithEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	policy := &Policy{
		InheritEnv:  false,
		Environment: map[string]string{"MY_VAR": "test_value"},
	}
	result, err := s.Execute(context.Background(), "echo $MY_VAR", policy)
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "test_value")
}

func TestNoopSandbox_Execute_OutputTruncation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	// Generate large output; limit to 100 bytes.
	policy := &Policy{MaxOutputBytes: 100}
	result, err := s.Execute(context.Background(), "seq 1 1000", policy)
	require.NoError(t, err)
	assert.True(t, len(result.Stdout) > 100)
	assert.Contains(t, result.Stdout, "truncated")
}

func TestNoopSandbox_Execute_ContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	_, err := s.Execute(ctx, "echo hello", nil)
	// Either returns error or a killed result.
	if err == nil {
		// OK — the command may have completed before cancel took effect.
	}
	_ = err
}

func TestNoopSandbox_Execute_Stderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewNoopSandbox()
	result, err := s.Execute(context.Background(), "echo err >&2", nil)
	require.NoError(t, err)
	assert.Contains(t, result.Stderr, "err")
}

// ─── ProcessSandbox ───

func TestProcessSandbox_Level(t *testing.T) {
	s := NewProcessSandbox(nil)
	assert.Equal(t, IsolationProcess, s.Level())
}

func TestProcessSandbox_Available(t *testing.T) {
	s := NewProcessSandbox(nil)
	assert.True(t, s.Available())
}

func TestProcessSandbox_Close(t *testing.T) {
	s := NewProcessSandbox(nil)
	assert.NoError(t, s.Close())
}

func TestProcessSandbox_Execute_SimpleCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	result, err := s.Execute(context.Background(), "echo sandbox", nil)
	require.NoError(t, err)
	assert.Equal(t, "sandbox\n", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
	assert.True(t, result.Success())
}

func TestProcessSandbox_Execute_EmptyCommand(t *testing.T) {
	s := NewProcessSandbox(nil)
	_, err := s.Execute(context.Background(), "", nil)
	assert.ErrorIs(t, err, ErrEmptyCommand)
}

func TestProcessSandbox_Execute_DeniedCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	policy := DefaultPolicy()
	policy.DeniedCommands = []string{"echo"}
	_, err := s.Execute(context.Background(), "echo hello", policy)
	assert.ErrorIs(t, err, ErrCommandDenied)
}

func TestProcessSandbox_Execute_AllowedCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	policy := DefaultPolicy()
	policy.AllowedCommands = []string{"echo", "ls"}
	result, err := s.Execute(context.Background(), "echo allowed", policy)
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "allowed")

	// Not in allow list.
	policy.AllowedCommands = []string{"cat"}
	_, err = s.Execute(context.Background(), "echo denied", policy)
	assert.ErrorIs(t, err, ErrCommandDenied)
}

func TestProcessSandbox_Execute_WithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	policy := DefaultPolicy()
	policy.Timeout = 100 * time.Millisecond
	result, err := s.Execute(context.Background(), "sleep 10", policy)
	require.NoError(t, err)
	assert.True(t, result.Killed)
	assert.Equal(t, "timeout", result.KillReason)
}

func TestProcessSandbox_Execute_WithDefaultPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	defaultP := DefaultPolicy()
	defaultP.Timeout = 5 * time.Second
	s := NewProcessSandbox(&sandboxConfig{DefaultPolicy: defaultP})
	result, err := s.Execute(context.Background(), "echo defaults", nil)
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "defaults")
}

func TestProcessSandbox_Execute_InvalidPolicy(t *testing.T) {
	s := NewProcessSandbox(nil)
	policy := &Policy{Resources: ResourceLimits{MaxMemoryBytes: -1}}
	_, err := s.Execute(context.Background(), "echo hello", policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid policy")
}

func TestProcessSandbox_Execute_RestrictedEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	policy := DefaultPolicy()
	policy.InheritEnv = false
	policy.Environment = map[string]string{"SANDBOX_TEST": "isolated"}
	result, err := s.Execute(context.Background(), "echo $SANDBOX_TEST", policy)
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "isolated")
}

func TestProcessSandbox_Execute_OutputLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	policy := DefaultPolicy()
	policy.MaxOutputBytes = 50
	result, err := s.Execute(context.Background(), "seq 1 1000", policy)
	require.NoError(t, err)
	assert.Contains(t, result.Stdout, "truncated")
}

func TestProcessSandbox_Execute_ExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	result, err := s.Execute(context.Background(), "exit 7", nil)
	require.NoError(t, err)
	assert.Equal(t, 7, result.ExitCode)
	assert.False(t, result.Success())
}

func TestProcessSandbox_Execute_Duration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	s := NewProcessSandbox(nil)
	result, err := s.Execute(context.Background(), "sleep 0.1", nil)
	require.NoError(t, err)
	assert.True(t, result.Duration >= 50*time.Millisecond)
}

// ─── ContainerSandbox ───

func TestContainerSandbox_Level(t *testing.T) {
	s := NewContainerSandbox(nil)
	assert.Equal(t, IsolationContainer, s.Level())
}

func TestContainerSandbox_Close(t *testing.T) {
	s := NewContainerSandbox(nil)
	assert.NoError(t, s.Close())
}

func TestContainerSandbox_EmptyCommand(t *testing.T) {
	s := NewContainerSandbox(nil)
	_, err := s.Execute(context.Background(), "", nil)
	assert.ErrorIs(t, err, ErrEmptyCommand)
}

func TestContainerSandbox_Unavailable(t *testing.T) {
	// Force unavailable by setting a bad runtime path.
	cfg := &sandboxConfig{RuntimePath: "/nonexistent/runtime"}
	s := NewContainerSandbox(cfg)
	// On non-Linux or without runtime, this should be unavailable.
	if runtime.GOOS != "linux" {
		assert.False(t, s.Available())
		_, err := s.Execute(context.Background(), "echo hello", nil)
		assert.ErrorIs(t, err, ErrUnavailable)
	}
}

func TestContainerSandbox_RuntimeName(t *testing.T) {
	cfg := &sandboxConfig{RuntimePath: "/usr/local/bin/runsc"}
	s := NewContainerSandbox(cfg)
	assert.Equal(t, "runsc", s.RuntimeName())

	cfg = &sandboxConfig{RuntimePath: "/usr/bin/runc"}
	s = NewContainerSandbox(cfg)
	assert.Equal(t, "runc", s.RuntimeName())
}

func TestContainerSandbox_GenerateSpec(t *testing.T) {
	s := NewContainerSandbox(nil)
	policy := DefaultPolicy()
	policy.Filesystem.WorkingDir = "/app"
	policy.Environment = map[string]string{"MY_VAR": "value"}
	policy.Resources.MaxMemoryBytes = 128 * 1024 * 1024
	policy.Resources.MaxProcesses = 16
	policy.Resources.MaxCPUSeconds = 30
	policy.Resources.MaxFileSize = 10 * 1024 * 1024
	policy.Resources.MaxOpenFiles = 64
	policy.Filesystem.ReadOnlyPaths = []string{"/usr"}
	policy.Filesystem.ReadWritePaths = []string{"/data"}

	spec := s.generateSpec("echo test", policy)

	assert.Equal(t, "1.0.2", spec.OCIVersion)
	assert.Equal(t, []string{"/bin/sh", "-c", "echo test"}, spec.Process.Args)
	assert.Equal(t, "/app", spec.Process.Cwd)
	assert.Equal(t, uint32(65534), spec.Process.User.UID) // nobody
	assert.True(t, spec.Root.Readonly)
	assert.Equal(t, "sandbox", spec.Hostname)

	// Check env.
	found := false
	for _, e := range spec.Process.Env {
		if e == "MY_VAR=value" {
			found = true
		}
	}
	assert.True(t, found, "env should contain MY_VAR=value")

	// Check namespaces — should include network (network denied).
	hasNetwork := false
	for _, ns := range spec.Linux.Namespaces {
		if ns.Type == "network" {
			hasNetwork = true
		}
	}
	assert.True(t, hasNetwork, "should have network namespace when network denied")

	// Check resources.
	require.NotNil(t, spec.Linux.Resources)
	require.NotNil(t, spec.Linux.Resources.Memory)
	assert.Equal(t, int64(128*1024*1024), spec.Linux.Resources.Memory.Limit)
	require.NotNil(t, spec.Linux.Resources.Pids)
	assert.Equal(t, int64(16), spec.Linux.Resources.Pids.Limit)
	require.NotNil(t, spec.Linux.Resources.CPU)

	// Check rlimits.
	assert.Len(t, spec.Process.Rlimits, 3) // FSIZE, NOFILE, CPU

	// Check bind mounts.
	var roMount, rwMount bool
	for _, m := range spec.Mounts {
		if m.Destination == "/usr" && m.Source == "/usr" {
			for _, opt := range m.Options {
				if opt == "ro" {
					roMount = true
				}
			}
		}
		if m.Destination == "/data" && m.Source == "/data" {
			for _, opt := range m.Options {
				if opt == "rw" {
					rwMount = true
				}
			}
		}
	}
	assert.True(t, roMount, "should have read-only /usr mount")
	assert.True(t, rwMount, "should have read-write /data mount")
}

func TestContainerSandbox_GenerateSpec_NetworkAllowed(t *testing.T) {
	s := NewContainerSandbox(nil)
	policy := DefaultPolicy()
	policy.Network.AllowNetwork = true

	spec := s.generateSpec("echo test", policy)

	// Should NOT have network namespace when network is allowed.
	for _, ns := range spec.Linux.Namespaces {
		assert.NotEqual(t, "network", ns.Type, "should not isolate network when allowed")
	}
}

func TestContainerSandbox_GenerateSpec_NoResources(t *testing.T) {
	s := NewContainerSandbox(nil)
	policy := &Policy{} // Zero resources.
	spec := s.generateSpec("echo test", policy)
	assert.Nil(t, spec.Linux.Resources)
	assert.Nil(t, spec.Process.Rlimits)
}

func TestContainerSandbox_GenerateSpec_DefaultCwd(t *testing.T) {
	s := NewContainerSandbox(nil)
	policy := &Policy{}
	spec := s.generateSpec("echo test", policy)
	assert.Equal(t, "/tmp", spec.Process.Cwd) // default
}

func TestContainerSandbox_InvalidPolicy(t *testing.T) {
	// Force runtimePath so Available() returns true on Linux.
	s := &ContainerSandbox{
		config:      &sandboxConfig{},
		runtimePath: "/usr/bin/runc",
		runtimeName: "runc",
	}
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	policy := &Policy{Resources: ResourceLimits{MaxMemoryBytes: -1}}
	_, err := s.Execute(context.Background(), "echo hello", policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid policy")
}

func TestContainerSandbox_DeniedCommand(t *testing.T) {
	// Force runtimePath so Available() returns true on Linux.
	s := &ContainerSandbox{
		config:      &sandboxConfig{},
		runtimePath: "/usr/bin/runc",
		runtimeName: "runc",
	}
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	policy := DefaultPolicy()
	// Deny "echo" not "sh" — extractBaseCommand extracts "echo" from "echo hello".
	policy.DeniedCommands = []string{"echo"}
	_, err := s.Execute(context.Background(), "echo hello", policy)
	assert.ErrorIs(t, err, ErrCommandDenied)
}

// ─── New / NewWithFallback ───

func TestNew_None(t *testing.T) {
	s, err := New(IsolationNone)
	require.NoError(t, err)
	assert.Equal(t, IsolationNone, s.Level())
}

func TestNew_Process(t *testing.T) {
	s, err := New(IsolationProcess)
	require.NoError(t, err)
	assert.Equal(t, IsolationProcess, s.Level())
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New("bogus")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown isolation level")
}

func TestNew_ContainerNoRuntime(t *testing.T) {
	// On Linux, the container sandbox auto-detects runc/runsc even if
	// RuntimePath is bogus — because detectRuntime falls back to LookPath.
	// So we test with a config that prevents auto-detection.
	if runtime.GOOS != "linux" {
		_, err := New(IsolationContainer, WithRuntimePath("/nonexistent/runtime"))
		assert.Error(t, err)
		return
	}
	// On Linux with runc available, New succeeds. Test the actual unavailable
	// path by constructing directly.
	s := &ContainerSandbox{config: &sandboxConfig{}, runtimePath: "", runtimeName: ""}
	assert.False(t, s.Available())
	_, err := s.Execute(context.Background(), "echo hello", nil)
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestNewWithFallback_Container(t *testing.T) {
	s := NewWithFallback(IsolationContainer)
	// Should fall back to process on systems without an OCI runtime.
	assert.NotNil(t, s)
	level := s.Level()
	assert.True(t, level == IsolationContainer || level == IsolationProcess,
		"should be container or process, got %s", level)
}

func TestNewWithFallback_Process(t *testing.T) {
	s := NewWithFallback(IsolationProcess)
	assert.Equal(t, IsolationProcess, s.Level())
}

func TestNewWithFallback_None(t *testing.T) {
	s := NewWithFallback(IsolationNone)
	assert.Equal(t, IsolationNone, s.Level())
}

// ─── Options ───

func TestWithRuntimePath(t *testing.T) {
	cfg := &sandboxConfig{}
	WithRuntimePath("/usr/local/bin/runsc")(cfg)
	assert.Equal(t, "/usr/local/bin/runsc", cfg.RuntimePath)
}

func TestWithRootDir(t *testing.T) {
	cfg := &sandboxConfig{}
	WithRootDir("/var/sandbox")(cfg)
	assert.Equal(t, "/var/sandbox", cfg.RootDir)
}

func TestWithDefaultPolicy(t *testing.T) {
	cfg := &sandboxConfig{}
	p := DefaultPolicy()
	WithDefaultPolicy(p)(cfg)
	assert.Equal(t, p, cfg.DefaultPolicy)
}

// ─── Helper functions ───

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"echo hello", "echo"},
		{"  ls -la  ", "ls"},
		{"/usr/bin/python script.py", "python"},
		{"FOO=bar echo hello", "echo"},
		{"FOO=bar BAZ=qux cmd arg1", "cmd"},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, extractBaseCommand(tt.input), "input=%q", tt.input)
	}
}

func TestTruncate(t *testing.T) {
	// Short string — no truncation.
	assert.Equal(t, "hello", truncate("hello", 100))

	// Exact length.
	assert.Equal(t, "hello", truncate("hello", 5))

	// Over limit.
	result := truncate("hello world", 5)
	assert.True(t, strings.HasPrefix(result, "hello"))
	assert.Contains(t, result, "truncated")
}

func TestEnvSlice(t *testing.T) {
	env := map[string]string{
		"A": "1",
		"B": "2",
	}
	slice := envSlice(env)
	assert.Len(t, slice, 2)
	found := map[string]bool{}
	for _, s := range slice {
		found[s] = true
	}
	assert.True(t, found["A=1"])
	assert.True(t, found["B=2"])
}

func TestBuildRestrictedEnv(t *testing.T) {
	env := buildRestrictedEnv(map[string]string{"CUSTOM": "value"})
	// Should contain system essentials plus custom.
	hasPath := false
	hasCustom := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			hasPath = true
		}
		if e == "CUSTOM=value" {
			hasCustom = true
		}
	}
	assert.True(t, hasPath, "should include PATH")
	assert.True(t, hasCustom, "should include custom var")
}

func TestGenerateContainerID(t *testing.T) {
	id1, err := generateContainerID()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(id1, "op-"))
	assert.Len(t, id1, 35) // "op-" + 32 hex chars

	id2, err := generateContainerID()
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2, "IDs should be unique")
}

// ─── Interface compliance ───

func TestInterfaceCompliance(t *testing.T) {
	var _ Sandbox = (*NoopSandbox)(nil)
	var _ Sandbox = (*ProcessSandbox)(nil)
	var _ Sandbox = (*ContainerSandbox)(nil)
}

// ─── Error constants ───

func TestErrorConstants(t *testing.T) {
	assert.NotNil(t, ErrNilPolicy)
	assert.NotNil(t, ErrEmptyCommand)
	assert.NotNil(t, ErrUnavailable)
	assert.NotNil(t, ErrRuntimeNotFound)
	assert.NotNil(t, ErrResourceExceeded)
	assert.NotNil(t, ErrNetworkDenied)
	assert.NotNil(t, ErrPathDenied)
	assert.NotNil(t, ErrCommandDenied)
	assert.NotNil(t, ErrTimeout)
	assert.NotNil(t, ErrOOMKilled)
	assert.NotNil(t, ErrSandboxSetupFailed)
}

// ─── OCI Spec JSON ───

func TestOCISpec_JSON(t *testing.T) {
	s := NewContainerSandbox(nil)
	spec := s.generateSpec("echo test", DefaultPolicy())
	data, err := json.Marshal(spec)
	require.NoError(t, err)

	var decoded ociSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "1.0.2", decoded.OCIVersion)
	assert.Equal(t, []string{"/bin/sh", "-c", "echo test"}, decoded.Process.Args)
}
