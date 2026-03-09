// Package services manages Operator-OS managed services (browser, sandbox, repo).
//
// Services are Docker containers that spin up on-demand when an agent first needs them.
// The manager handles lifecycle, health checks, and resource allocation based on
// the detected hardware profile (Nano/Standard/Pro/Scale).
package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/operatoronline/Operator-OS/pkg/logger"
)

// ServiceState represents the current state of a managed service.
type ServiceState string

const (
	StateStopped  ServiceState = "stopped"
	StateStarting ServiceState = "starting"
	StateRunning  ServiceState = "running"
	StateDegraded ServiceState = "degraded"
	StateStopping ServiceState = "stopping"
	StateError    ServiceState = "error"
)

// ServiceType identifies which managed service this is.
type ServiceType string

const (
	ServiceBrowser ServiceType = "browser"
	ServiceSandbox ServiceType = "sandbox"
	ServiceRepo    ServiceType = "repo"
)

// ServiceInfo describes a running managed service.
type ServiceInfo struct {
	Type        ServiceType  `json:"type"`
	State       ServiceState `json:"state"`
	ContainerID string       `json:"container_id,omitempty"`
	Port        int          `json:"port,omitempty"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	LastHealth  *time.Time   `json:"last_health,omitempty"`
	Error       string       `json:"error,omitempty"`
	Image       string       `json:"image"`
}

// HardwareProfile determines which services can run on the current hardware.
type HardwareProfile string

const (
	ProfileNano     HardwareProfile = "nano"     // ≤1GB  — agent only
	ProfileStandard HardwareProfile = "standard" // 2-4GB — + browser + sandbox
	ProfilePro      HardwareProfile = "pro"      // 8-16GB — + repo + multi-instance
	ProfileScale    HardwareProfile = "scale"    // 32GB+ — pooled multi-tenant
)

// Manager controls the lifecycle of all managed services.
type Manager struct {
	mu       sync.RWMutex
	services map[ServiceType]*ServiceInfo
	profile  HardwareProfile
	config   *Config
	ctx      context.Context
	cancel   context.CancelFunc
}

// Config holds the configuration for managed services.
type Config struct {
	// DataDir is the persistent storage directory for service data.
	DataDir string `json:"data_dir"`

	// Network is the Docker network services communicate on.
	Network string `json:"network"`

	// Browser settings
	BrowserImage string `json:"browser_image"`
	BrowserPort  int    `json:"browser_port"`

	// Sandbox settings
	SandboxImage string `json:"sandbox_image"`
	SandboxPort  int    `json:"sandbox_port"`

	// Repo settings
	RepoImage string `json:"repo_image"`
	RepoPort  int    `json:"repo_port"`

	// AutoStart controls whether services start automatically on boot.
	AutoStart bool `json:"auto_start"`
}

// DefaultConfig returns the default service configuration.
func DefaultConfig() *Config {
	return &Config{
		DataDir:      "/var/lib/operator/services",
		Network:      "operator-net",
		BrowserImage: "ghcr.io/operatoronline/go-browser:latest",
		BrowserPort:  18800,
		SandboxImage: "ghcr.io/operatoronline/go-sandbox:latest",
		SandboxPort:  18801,
		RepoImage:    "ghcr.io/operatoronline/go-repo:latest",
		RepoPort:     18802,
		AutoStart:    false,
	}
}

// NewManager creates a new service manager.
func NewManager(cfg *Config) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	profile := DetectHardwareProfile()

	m := &Manager{
		services: make(map[ServiceType]*ServiceInfo),
		profile:  profile,
		config:   cfg,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize service info for all service types
	for _, svc := range []struct {
		stype ServiceType
		image string
	}{
		{ServiceBrowser, cfg.BrowserImage},
		{ServiceSandbox, cfg.SandboxImage},
		{ServiceRepo, cfg.RepoImage},
	} {
		m.services[svc.stype] = &ServiceInfo{
			Type:  svc.stype,
			State: StateStopped,
			Image: svc.image,
		}
	}

	logger.InfoCF("services", "Service manager initialized",
		map[string]any{
			"profile":  profile,
			"data_dir": cfg.DataDir,
		})

	return m
}

// Profile returns the detected hardware profile.
func (m *Manager) Profile() HardwareProfile {
	return m.profile
}

// AvailableServices returns which services can run on the current hardware profile.
func (m *Manager) AvailableServices() []ServiceType {
	switch m.profile {
	case ProfileNano:
		return nil // No managed services on nano
	case ProfileStandard:
		return []ServiceType{ServiceBrowser, ServiceSandbox}
	case ProfilePro, ProfileScale:
		return []ServiceType{ServiceBrowser, ServiceSandbox, ServiceRepo}
	default:
		return nil
	}
}

// IsAvailable checks if a service type can run on the current hardware.
func (m *Manager) IsAvailable(stype ServiceType) bool {
	for _, available := range m.AvailableServices() {
		if available == stype {
			return true
		}
	}
	return false
}

// Status returns the current status of a service.
func (m *Manager) Status(stype ServiceType) *ServiceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if info, ok := m.services[stype]; ok {
		return info
	}
	return nil
}

// StatusAll returns the status of all services.
func (m *Manager) StatusAll() map[ServiceType]*ServiceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[ServiceType]*ServiceInfo, len(m.services))
	for k, v := range m.services {
		result[k] = v
	}
	return result
}

// EnsureRunning starts a service if it's not already running.
// This is the primary method agents call — lazy initialization.
func (m *Manager) EnsureRunning(ctx context.Context, stype ServiceType) (*ServiceInfo, error) {
	if !m.IsAvailable(stype) {
		return nil, fmt.Errorf("service %s not available on %s profile (need more RAM)", stype, m.profile)
	}

	m.mu.Lock()
	info := m.services[stype]

	if info.State == StateRunning {
		m.mu.Unlock()
		return info, nil
	}

	if info.State == StateStarting {
		m.mu.Unlock()
		// Wait for startup to complete
		return m.waitForReady(ctx, stype, 60*time.Second)
	}

	info.State = StateStarting
	m.mu.Unlock()

	logger.InfoCF("services", "Starting managed service",
		map[string]any{"service": stype, "image": info.Image})

	if err := m.startContainer(ctx, stype); err != nil {
		m.mu.Lock()
		info.State = StateError
		info.Error = err.Error()
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to start %s: %w", stype, err)
	}

	now := time.Now()
	m.mu.Lock()
	info.State = StateRunning
	info.StartedAt = &now
	info.LastHealth = &now
	info.Error = ""
	m.mu.Unlock()

	logger.InfoCF("services", "Service started successfully",
		map[string]any{"service": stype})

	return info, nil
}

// Stop stops a running service.
func (m *Manager) Stop(ctx context.Context, stype ServiceType) error {
	m.mu.Lock()
	info, ok := m.services[stype]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown service: %s", stype)
	}

	if info.State != StateRunning && info.State != StateDegraded {
		m.mu.Unlock()
		return nil // Already stopped
	}

	info.State = StateStopping
	containerID := info.ContainerID
	m.mu.Unlock()

	containerName := m.containerName(stype)
	if containerID != "" {
		containerName = containerID
	}

	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", "10", containerName)
	if err := cmd.Run(); err != nil {
		logger.WarnCF("services", "Error stopping container",
			map[string]any{"service": stype, "error": err.Error()})
	}

	// Remove the container
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerName)
	rmCmd.Run() // Best effort

	m.mu.Lock()
	info.State = StateStopped
	info.ContainerID = ""
	m.mu.Unlock()

	logger.InfoCF("services", "Service stopped", map[string]any{"service": stype})
	return nil
}

// StopAll stops all running services.
func (m *Manager) StopAll(ctx context.Context) {
	m.cancel()
	for stype := range m.services {
		m.Stop(ctx, stype)
	}
}

// startContainer launches a Docker container for the given service.
func (m *Manager) startContainer(ctx context.Context, stype ServiceType) error {
	name := m.containerName(stype)
	image := m.services[stype].Image

	// Remove any existing container with the same name
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", name)
	rmCmd.Run() // Ignore error — may not exist

	var port int
	var envVars []string

	switch stype {
	case ServiceBrowser:
		port = m.config.BrowserPort
		envVars = []string{
			"OPERATOR_POOL_SIZE=1",
			"OPERATOR_IDLE_TIMEOUT=300",
		}
	case ServiceSandbox:
		port = m.config.SandboxPort
		envVars = []string{
			"OPERATOR_MAX_SANDBOXES=5",
			"OPERATOR_EXEC_TIMEOUT=300",
		}
	case ServiceRepo:
		port = m.config.RepoPort
		envVars = []string{
			"OPERATOR_MAX_REPOS=50",
			"GITEA__server__ROOT_URL=http://localhost:" + fmt.Sprintf("%d", port),
		}
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--restart", "unless-stopped",
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", port, port),
	}

	// Add environment variables
	for _, env := range envVars {
		args = append(args, "-e", env)
	}

	// Add volume for persistent data
	dataVolume := fmt.Sprintf("operator-%s-data", stype)
	switch stype {
	case ServiceBrowser:
		args = append(args, "-v", dataVolume+":/home/operator")
		// Browser needs special capabilities
		args = append(args, "--shm-size=2g")
	case ServiceSandbox:
		args = append(args, "-v", dataVolume+":/var/lib/sandbox")
		// Sandbox needs Docker socket for spawning containers
		args = append(args, "-v", "/var/run/docker.sock:/var/run/docker.sock")
	case ServiceRepo:
		args = append(args, "-v", dataVolume+":/data")
	}

	// Add health check label
	args = append(args, "--label", "operator.managed=true")
	args = append(args, "--label", fmt.Sprintf("operator.service=%s", stype))

	args = append(args, image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run failed: %w — %s", err, strings.TrimSpace(string(output)))
	}

	containerID := strings.TrimSpace(string(output))

	m.mu.Lock()
	m.services[stype].ContainerID = containerID[:12] // Short ID
	m.mu.Unlock()

	return nil
}

// waitForReady polls a service until it's healthy or the timeout expires.
func (m *Manager) waitForReady(ctx context.Context, stype ServiceType, timeout time.Duration) (*ServiceInfo, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("service %s failed to become ready within %s", stype, timeout)
		case <-ticker.C:
			m.mu.RLock()
			info := m.services[stype]
			state := info.State
			m.mu.RUnlock()

			if state == StateRunning {
				return info, nil
			}
			if state == StateError {
				return nil, fmt.Errorf("service %s failed to start: %s", stype, info.Error)
			}
		}
	}
}

// containerName returns the Docker container name for a service.
func (m *Manager) containerName(stype ServiceType) string {
	return fmt.Sprintf("operator-%s", stype)
}

// HealthCheck performs a health check on a running service.
func (m *Manager) HealthCheck(ctx context.Context, stype ServiceType) error {
	m.mu.RLock()
	info := m.services[stype]
	m.mu.RUnlock()

	if info.State != StateRunning {
		return fmt.Errorf("service %s is not running (state: %s)", stype, info.State)
	}

	name := m.containerName(stype)
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Health.Status}}", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	status := strings.TrimSpace(string(output))
	now := time.Now()

	m.mu.Lock()
	info.LastHealth = &now
	if status != "healthy" && status != "" {
		info.State = StateDegraded
	}
	m.mu.Unlock()

	return nil
}

// StartHealthMonitor begins periodic health checking for all running services.
func (m *Manager) StartHealthMonitor() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				for stype, info := range m.services {
					if info.State == StateRunning || info.State == StateDegraded {
						if err := m.HealthCheck(m.ctx, stype); err != nil {
							logger.WarnCF("services", "Health check failed",
								map[string]any{"service": stype, "error": err.Error()})
						}
					}
				}
			}
		}
	}()
}
