package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ContainerSandbox provides full container isolation using an OCI runtime.
// It supports gVisor (runsc) for kernel-level isolation and runc as a fallback.
// Requires Linux. On other platforms, Available() returns false.
type ContainerSandbox struct {
	config      *sandboxConfig
	runtimePath string
	runtimeName string // "runsc" or "runc"
	mu          sync.Mutex
}

// NewContainerSandbox creates a container-level sandbox. It auto-detects
// the OCI runtime if RuntimePath is not specified in the config.
func NewContainerSandbox(cfg *sandboxConfig) *ContainerSandbox {
	if cfg == nil {
		cfg = &sandboxConfig{}
	}
	cs := &ContainerSandbox{config: cfg}
	cs.detectRuntime()
	return cs
}

func (s *ContainerSandbox) Level() IsolationLevel {
	return IsolationContainer
}

func (s *ContainerSandbox) Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return s.runtimePath != ""
}

func (s *ContainerSandbox) Close() error {
	return nil
}

// detectRuntime finds the OCI runtime binary, preferring gVisor (runsc)
// over runc for stronger isolation.
func (s *ContainerSandbox) detectRuntime() {
	if s.config.RuntimePath != "" {
		s.runtimePath = s.config.RuntimePath
		s.runtimeName = filepath.Base(s.runtimePath)
		return
	}

	// Prefer runsc (gVisor) for kernel-level isolation.
	if path, err := exec.LookPath("runsc"); err == nil {
		s.runtimePath = path
		s.runtimeName = "runsc"
		return
	}

	// Fall back to runc.
	if path, err := exec.LookPath("runc"); err == nil {
		s.runtimePath = path
		s.runtimeName = "runc"
		return
	}
}

// RuntimeName returns the detected runtime name ("runsc", "runc", or "").
func (s *ContainerSandbox) RuntimeName() string {
	return s.runtimeName
}

func (s *ContainerSandbox) Execute(ctx context.Context, command string, policy *Policy) (*ExecutionResult, error) {
	if strings.TrimSpace(command) == "" {
		return nil, ErrEmptyCommand
	}
	if !s.Available() {
		return nil, ErrUnavailable
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

	baseName := extractBaseCommand(command)
	if !policy.IsCommandAllowed(baseName) {
		return nil, ErrCommandDenied
	}

	// Create container ID.
	containerID, err := generateContainerID()
	if err != nil {
		return nil, fmt.Errorf("sandbox: failed to generate container ID: %w", err)
	}

	// Create bundle directory.
	rootDir := s.config.RootDir
	if rootDir == "" {
		rootDir = os.TempDir()
	}
	bundleDir := filepath.Join(rootDir, "sandbox-"+containerID)
	rootfsDir := filepath.Join(bundleDir, "rootfs")

	if err := os.MkdirAll(rootfsDir, 0o700); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSandboxSetupFailed, err)
	}
	defer os.RemoveAll(bundleDir)

	// Create minimal rootfs.
	if err := s.setupRootfs(rootfsDir, policy); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSandboxSetupFailed, err)
	}

	// Generate OCI config.
	spec := s.generateSpec(command, policy)
	configPath := filepath.Join(bundleDir, "config.json")
	configData, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("sandbox: failed to marshal OCI config: %w", err)
	}
	if err := os.WriteFile(configPath, configData, 0o600); err != nil {
		return nil, fmt.Errorf("%w: failed to write OCI config: %v", ErrSandboxSetupFailed, err)
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

	// Run the container.
	args := []string{"run", "--bundle", bundleDir, containerID}
	cmd := exec.CommandContext(execCtx, s.runtimePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	duration := time.Since(start)

	result := &ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	// Enforce output size limit.
	maxOutput := policy.MaxOutputBytes
	if maxOutput == 0 {
		maxOutput = 10 * 1024 * 1024
	}
	result.Stdout = truncate(result.Stdout, maxOutput)
	result.Stderr = truncate(result.Stderr, maxOutput)

	if runErr != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			// Kill the container on timeout.
			s.killContainer(containerID)
			result.Killed = true
			result.KillReason = "timeout"
			result.ExitCode = -1
			return result, nil
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// Exit code 137 typically means SIGKILL (OOM).
			if result.ExitCode == 137 {
				result.Killed = true
				result.KillReason = "OOM killed"
			}
		} else {
			return nil, fmt.Errorf("sandbox: container execution failed: %w", runErr)
		}
	}

	// Clean up the container.
	s.deleteContainer(containerID)

	return result, nil
}

// setupRootfs creates a minimal filesystem for the container.
func (s *ContainerSandbox) setupRootfs(rootfsDir string, policy *Policy) error {
	// Create essential directories.
	dirs := []string{"bin", "dev", "etc", "proc", "sys", "tmp", "usr/bin", "usr/lib"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(rootfsDir, d), 0o755); err != nil {
			return err
		}
	}

	// Create working directory if specified.
	if policy.Filesystem.WorkingDir != "" {
		wd := strings.TrimPrefix(policy.Filesystem.WorkingDir, "/")
		if err := os.MkdirAll(filepath.Join(rootfsDir, wd), 0o755); err != nil {
			return err
		}
	}

	return nil
}

// ociSpec represents a minimal OCI runtime spec.
type ociSpec struct {
	OCIVersion string      `json:"ociVersion"`
	Process    ociProcess  `json:"process"`
	Root       ociRoot     `json:"root"`
	Hostname   string      `json:"hostname"`
	Mounts     []ociMount  `json:"mounts,omitempty"`
	Linux      *ociLinux   `json:"linux,omitempty"`
}

type ociProcess struct {
	Terminal bool        `json:"terminal"`
	User     ociUser     `json:"user"`
	Args     []string    `json:"args"`
	Env      []string    `json:"env,omitempty"`
	Cwd      string      `json:"cwd"`
	Rlimits  []ociRlimit `json:"rlimits,omitempty"`
}

type ociUser struct {
	UID uint32 `json:"uid"`
	GID uint32 `json:"gid"`
}

type ociRoot struct {
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
}

type ociMount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Options     []string `json:"options,omitempty"`
}

type ociLinux struct {
	Resources   *ociResources   `json:"resources,omitempty"`
	Namespaces  []ociNamespace  `json:"namespaces,omitempty"`
}

type ociResources struct {
	Memory *ociMemory `json:"memory,omitempty"`
	CPU    *ociCPU    `json:"cpu,omitempty"`
	Pids   *ociPids   `json:"pids,omitempty"`
}

type ociMemory struct {
	Limit int64 `json:"limit,omitempty"`
}

type ociCPU struct {
	Quota  int64 `json:"quota,omitempty"`
	Period int64 `json:"period,omitempty"`
}

type ociPids struct {
	Limit int64 `json:"limit"`
}

type ociNamespace struct {
	Type string `json:"type"`
}

type ociRlimit struct {
	Type string `json:"type"`
	Hard uint64 `json:"hard"`
	Soft uint64 `json:"soft"`
}

// generateSpec creates an OCI runtime specification from the policy.
func (s *ContainerSandbox) generateSpec(command string, policy *Policy) *ociSpec {
	cwd := "/tmp"
	if policy.Filesystem.WorkingDir != "" {
		cwd = policy.Filesystem.WorkingDir
	}

	env := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=/tmp",
		"LANG=C.UTF-8",
		"TERM=xterm",
	}
	for k, v := range policy.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	spec := &ociSpec{
		OCIVersion: "1.0.2",
		Process: ociProcess{
			Terminal: false,
			User:     ociUser{UID: 65534, GID: 65534}, // nobody
			Args:     []string{"/bin/sh", "-c", command},
			Env:      env,
			Cwd:      cwd,
		},
		Root: ociRoot{
			Path:     "rootfs",
			Readonly: policy.Filesystem.ReadOnlyRoot,
		},
		Hostname: "sandbox",
		Mounts: []ociMount{
			{Destination: "/proc", Type: "proc", Source: "proc"},
			{Destination: "/dev", Type: "tmpfs", Source: "tmpfs", Options: []string{"nosuid", "strictatime", "mode=755", "size=65536k"}},
			{Destination: "/tmp", Type: "tmpfs", Source: "tmpfs", Options: []string{"nosuid", "nodev", "mode=1777"}},
		},
		Linux: &ociLinux{
			Namespaces: []ociNamespace{
				{Type: "pid"},
				{Type: "ipc"},
				{Type: "uts"},
				{Type: "mount"},
			},
		},
	}

	// Add network namespace isolation if network is denied.
	if !policy.Network.AllowNetwork {
		spec.Linux.Namespaces = append(spec.Linux.Namespaces, ociNamespace{Type: "network"})
	}

	// Configure resource limits.
	resources := &ociResources{}
	hasResources := false

	if policy.Resources.MaxMemoryBytes > 0 {
		resources.Memory = &ociMemory{Limit: policy.Resources.MaxMemoryBytes}
		hasResources = true
	}

	if policy.Resources.MaxProcesses > 0 {
		resources.Pids = &ociPids{Limit: policy.Resources.MaxProcesses}
		hasResources = true
	}

	if policy.Resources.MaxCPUSeconds > 0 {
		// Convert CPU seconds to CPU quota (microseconds per period).
		// Use 100ms period, allow proportional CPU time.
		period := int64(100000) // 100ms in microseconds
		quota := period         // 100% of one CPU core
		resources.CPU = &ociCPU{Quota: quota, Period: period}
		hasResources = true
	}

	if hasResources {
		spec.Linux.Resources = resources
	}

	// Add rlimits.
	var rlimits []ociRlimit
	if policy.Resources.MaxFileSize > 0 {
		rlimits = append(rlimits, ociRlimit{
			Type: "RLIMIT_FSIZE",
			Hard: uint64(policy.Resources.MaxFileSize),
			Soft: uint64(policy.Resources.MaxFileSize),
		})
	}
	if policy.Resources.MaxOpenFiles > 0 {
		rlimits = append(rlimits, ociRlimit{
			Type: "RLIMIT_NOFILE",
			Hard: uint64(policy.Resources.MaxOpenFiles),
			Soft: uint64(policy.Resources.MaxOpenFiles),
		})
	}
	if policy.Resources.MaxCPUSeconds > 0 {
		rlimits = append(rlimits, ociRlimit{
			Type: "RLIMIT_CPU",
			Hard: uint64(policy.Resources.MaxCPUSeconds),
			Soft: uint64(policy.Resources.MaxCPUSeconds),
		})
	}
	if len(rlimits) > 0 {
		spec.Process.Rlimits = rlimits
	}

	// Add read-only bind mounts.
	for _, path := range policy.Filesystem.ReadOnlyPaths {
		spec.Mounts = append(spec.Mounts, ociMount{
			Destination: path,
			Type:        "bind",
			Source:      path,
			Options:     []string{"bind", "ro"},
		})
	}

	// Add read-write bind mounts.
	for _, path := range policy.Filesystem.ReadWritePaths {
		spec.Mounts = append(spec.Mounts, ociMount{
			Destination: path,
			Type:        "bind",
			Source:      path,
			Options:     []string{"bind", "rw"},
		})
	}

	return spec
}

// killContainer forcefully kills a running container.
func (s *ContainerSandbox) killContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.runtimePath, "kill", containerID, "KILL")
	_ = cmd.Run()
}

// deleteContainer removes a container's state.
func (s *ContainerSandbox) deleteContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.runtimePath, "delete", "--force", containerID)
	_ = cmd.Run()
}

// generateContainerID creates a unique container ID.
func generateContainerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "op-" + hex.EncodeToString(b), nil
}
