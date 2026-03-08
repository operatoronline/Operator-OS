package sandbox

import (
	"fmt"
	"time"
)

// Policy defines the security and resource constraints for a sandboxed execution.
type Policy struct {
	// Name is a human-readable identifier for this policy (e.g., "free-tier", "pro-agent").
	Name string `json:"name"`

	// Timeout is the maximum execution time. Zero means no timeout.
	Timeout time.Duration `json:"timeout"`

	// Resources defines CPU, memory, and I/O limits.
	Resources ResourceLimits `json:"resources"`

	// Network controls network access within the sandbox.
	Network NetworkPolicy `json:"network"`

	// Filesystem controls file access within the sandbox.
	Filesystem FilesystemPolicy `json:"filesystem"`

	// AllowedCommands restricts which commands can be executed.
	// Empty means all commands are allowed (subject to deny list).
	AllowedCommands []string `json:"allowed_commands,omitempty"`

	// DeniedCommands lists commands that are never allowed.
	DeniedCommands []string `json:"denied_commands,omitempty"`

	// Environment variables to set (overrides inherited env).
	Environment map[string]string `json:"environment,omitempty"`

	// InheritEnv controls whether the sandbox inherits the parent environment.
	// When false, only Environment variables are available.
	InheritEnv bool `json:"inherit_env"`

	// MaxOutputBytes limits the size of captured stdout+stderr.
	// Zero means use default (10MB).
	MaxOutputBytes int64 `json:"max_output_bytes"`
}

// ResourceLimits defines the resource constraints for a sandbox.
type ResourceLimits struct {
	// MaxMemoryBytes is the maximum memory the process can use.
	// Zero means no limit.
	MaxMemoryBytes int64 `json:"max_memory_bytes"`

	// MaxCPUSeconds is the maximum CPU time (user + system).
	// Zero means no limit.
	MaxCPUSeconds int64 `json:"max_cpu_seconds"`

	// MaxProcesses is the maximum number of child processes.
	// Zero means no limit.
	MaxProcesses int64 `json:"max_processes"`

	// MaxFileSize is the maximum size of any file created by the process.
	// Zero means no limit.
	MaxFileSize int64 `json:"max_file_size"`

	// MaxOpenFiles is the maximum number of open file descriptors.
	// Zero means no limit.
	MaxOpenFiles int64 `json:"max_open_files"`
}

// NetworkPolicy controls network access within the sandbox.
type NetworkPolicy struct {
	// AllowNetwork enables network access. Default is false (no network).
	AllowNetwork bool `json:"allow_network"`

	// AllowedHosts restricts network access to specific hostnames/IPs.
	// Empty means all hosts are allowed (when AllowNetwork is true).
	AllowedHosts []string `json:"allowed_hosts,omitempty"`

	// AllowedPorts restricts network access to specific ports.
	// Empty means all ports are allowed.
	AllowedPorts []int `json:"allowed_ports,omitempty"`

	// AllowDNS allows DNS resolution. Defaults to true when AllowNetwork is true.
	AllowDNS bool `json:"allow_dns"`
}

// FilesystemPolicy controls file access within the sandbox.
type FilesystemPolicy struct {
	// WorkingDir is the working directory for the command.
	WorkingDir string `json:"working_dir"`

	// ReadOnlyPaths are mounted read-only in the sandbox.
	ReadOnlyPaths []string `json:"readonly_paths,omitempty"`

	// ReadWritePaths are mounted read-write in the sandbox.
	ReadWritePaths []string `json:"readwrite_paths,omitempty"`

	// HiddenPaths are not visible in the sandbox.
	HiddenPaths []string `json:"hidden_paths,omitempty"`

	// TempDirSize is the maximum size of /tmp in the sandbox.
	// Zero means default (64MB).
	TempDirSize int64 `json:"temp_dir_size"`

	// ReadOnlyRoot makes the root filesystem read-only.
	// Only ReadWritePaths and /tmp are writable.
	ReadOnlyRoot bool `json:"readonly_root"`
}

// DefaultPolicy returns a reasonable default policy for sandboxed agent execution.
func DefaultPolicy() *Policy {
	return &Policy{
		Name:    "default",
		Timeout: 60 * time.Second,
		Resources: ResourceLimits{
			MaxMemoryBytes: 256 * 1024 * 1024, // 256 MB
			MaxCPUSeconds:  30,
			MaxProcesses:   32,
			MaxFileSize:    50 * 1024 * 1024, // 50 MB
			MaxOpenFiles:   256,
		},
		Network: NetworkPolicy{
			AllowNetwork: false,
			AllowDNS:     false,
		},
		Filesystem: FilesystemPolicy{
			ReadOnlyRoot: true,
			TempDirSize:  64 * 1024 * 1024, // 64 MB
		},
		InheritEnv:     false,
		MaxOutputBytes: 10 * 1024 * 1024, // 10 MB
	}
}

// FreeTierPolicy returns a restrictive policy suitable for free-tier users.
func FreeTierPolicy() *Policy {
	p := DefaultPolicy()
	p.Name = "free-tier"
	p.Timeout = 30 * time.Second
	p.Resources.MaxMemoryBytes = 128 * 1024 * 1024 // 128 MB
	p.Resources.MaxCPUSeconds = 10
	p.Resources.MaxProcesses = 8
	p.Resources.MaxFileSize = 10 * 1024 * 1024 // 10 MB
	p.Resources.MaxOpenFiles = 64
	p.MaxOutputBytes = 1 * 1024 * 1024 // 1 MB
	return p
}

// ProTierPolicy returns a generous policy for paid-tier users.
func ProTierPolicy() *Policy {
	p := DefaultPolicy()
	p.Name = "pro-tier"
	p.Timeout = 5 * time.Minute
	p.Resources.MaxMemoryBytes = 1024 * 1024 * 1024 // 1 GB
	p.Resources.MaxCPUSeconds = 120
	p.Resources.MaxProcesses = 128
	p.Resources.MaxFileSize = 500 * 1024 * 1024 // 500 MB
	p.Resources.MaxOpenFiles = 1024
	p.Network = NetworkPolicy{
		AllowNetwork: true,
		AllowDNS:     true,
	}
	p.MaxOutputBytes = 50 * 1024 * 1024 // 50 MB
	return p
}

// Validate checks that the policy has no contradictory or invalid settings.
func (p *Policy) Validate() error {
	if p == nil {
		return ErrNilPolicy
	}
	if p.Resources.MaxMemoryBytes < 0 {
		return fmt.Errorf("sandbox: max_memory_bytes must be non-negative, got %d", p.Resources.MaxMemoryBytes)
	}
	if p.Resources.MaxCPUSeconds < 0 {
		return fmt.Errorf("sandbox: max_cpu_seconds must be non-negative, got %d", p.Resources.MaxCPUSeconds)
	}
	if p.Resources.MaxProcesses < 0 {
		return fmt.Errorf("sandbox: max_processes must be non-negative, got %d", p.Resources.MaxProcesses)
	}
	if p.Resources.MaxFileSize < 0 {
		return fmt.Errorf("sandbox: max_file_size must be non-negative, got %d", p.Resources.MaxFileSize)
	}
	if p.Resources.MaxOpenFiles < 0 {
		return fmt.Errorf("sandbox: max_open_files must be non-negative, got %d", p.Resources.MaxOpenFiles)
	}
	if p.MaxOutputBytes < 0 {
		return fmt.Errorf("sandbox: max_output_bytes must be non-negative, got %d", p.MaxOutputBytes)
	}
	if p.Filesystem.TempDirSize < 0 {
		return fmt.Errorf("sandbox: temp_dir_size must be non-negative, got %d", p.Filesystem.TempDirSize)
	}
	if !p.Network.AllowNetwork && len(p.Network.AllowedHosts) > 0 {
		return fmt.Errorf("sandbox: allowed_hosts specified but network is disabled")
	}
	if !p.Network.AllowNetwork && len(p.Network.AllowedPorts) > 0 {
		return fmt.Errorf("sandbox: allowed_ports specified but network is disabled")
	}
	return nil
}

// Merge returns a new policy that applies overrides from other on top of the base.
// Zero values in other do not override the base.
func (p *Policy) Merge(other *Policy) *Policy {
	if other == nil {
		cp := *p
		return &cp
	}
	result := *p
	if other.Name != "" {
		result.Name = other.Name
	}
	if other.Timeout > 0 {
		result.Timeout = other.Timeout
	}
	if other.Resources.MaxMemoryBytes > 0 {
		result.Resources.MaxMemoryBytes = other.Resources.MaxMemoryBytes
	}
	if other.Resources.MaxCPUSeconds > 0 {
		result.Resources.MaxCPUSeconds = other.Resources.MaxCPUSeconds
	}
	if other.Resources.MaxProcesses > 0 {
		result.Resources.MaxProcesses = other.Resources.MaxProcesses
	}
	if other.Resources.MaxFileSize > 0 {
		result.Resources.MaxFileSize = other.Resources.MaxFileSize
	}
	if other.Resources.MaxOpenFiles > 0 {
		result.Resources.MaxOpenFiles = other.Resources.MaxOpenFiles
	}
	if other.MaxOutputBytes > 0 {
		result.MaxOutputBytes = other.MaxOutputBytes
	}
	if other.Filesystem.WorkingDir != "" {
		result.Filesystem.WorkingDir = other.Filesystem.WorkingDir
	}
	if other.Filesystem.TempDirSize > 0 {
		result.Filesystem.TempDirSize = other.Filesystem.TempDirSize
	}
	if len(other.Filesystem.ReadOnlyPaths) > 0 {
		result.Filesystem.ReadOnlyPaths = other.Filesystem.ReadOnlyPaths
	}
	if len(other.Filesystem.ReadWritePaths) > 0 {
		result.Filesystem.ReadWritePaths = other.Filesystem.ReadWritePaths
	}
	if len(other.Filesystem.HiddenPaths) > 0 {
		result.Filesystem.HiddenPaths = other.Filesystem.HiddenPaths
	}
	if len(other.AllowedCommands) > 0 {
		result.AllowedCommands = other.AllowedCommands
	}
	if len(other.DeniedCommands) > 0 {
		result.DeniedCommands = other.DeniedCommands
	}
	if len(other.Environment) > 0 {
		if result.Environment == nil {
			result.Environment = make(map[string]string)
		}
		for k, v := range other.Environment {
			result.Environment[k] = v
		}
	}
	return &result
}

// IsCommandAllowed checks whether a command is allowed by this policy.
func (p *Policy) IsCommandAllowed(command string) bool {
	// Check deny list first.
	for _, denied := range p.DeniedCommands {
		if command == denied {
			return false
		}
	}
	// If allow list is specified, command must be in it.
	if len(p.AllowedCommands) > 0 {
		for _, allowed := range p.AllowedCommands {
			if command == allowed {
				return true
			}
		}
		return false
	}
	return true
}
