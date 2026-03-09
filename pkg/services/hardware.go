package services

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/operatoronline/Operator-OS/pkg/logger"
)

// DetectHardwareProfile reads system memory and returns the appropriate profile.
func DetectHardwareProfile() HardwareProfile {
	// Allow override via environment variable
	if override := os.Getenv("OPERATOR_PROFILE"); override != "" {
		switch HardwareProfile(override) {
		case ProfileNano, ProfileStandard, ProfilePro, ProfileScale:
			logger.InfoCF("services", "Hardware profile overridden",
				map[string]any{"profile": override})
			return HardwareProfile(override)
		}
	}

	totalMB := getTotalMemoryMB()
	cpus := runtime.NumCPU()

	logger.InfoCF("services", "Hardware detected",
		map[string]any{
			"total_memory_mb": totalMB,
			"cpus":            cpus,
			"os":              runtime.GOOS,
			"arch":            runtime.GOARCH,
		})

	switch {
	case totalMB >= 32768: // 32GB+
		return ProfileScale
	case totalMB >= 8192: // 8GB+
		return ProfilePro
	case totalMB >= 2048: // 2GB+
		return ProfileStandard
	default:
		return ProfileNano
	}
}

// getTotalMemoryMB reads /proc/meminfo on Linux, falls back to runtime estimate.
func getTotalMemoryMB() int64 {
	// Try /proc/meminfo (Linux)
	data, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.ParseInt(fields[1], 10, 64)
					if err == nil {
						return kb / 1024
					}
				}
			}
		}
	}

	// Fallback: estimate from GOMAXPROCS (rough, but better than nothing)
	// Assume 2GB per CPU as a conservative default
	return int64(runtime.NumCPU()) * 2048
}

// ProfileDescription returns a human-readable description of the profile.
func ProfileDescription(p HardwareProfile) string {
	switch p {
	case ProfileNano:
		return "Nano (≤1GB) — Core agent only"
	case ProfileStandard:
		return "Standard (2-4GB) — Agent + Browser + Sandbox"
	case ProfilePro:
		return "Pro (8-16GB) — Full stack with Repo"
	case ProfileScale:
		return "Scale (32GB+) — Multi-tenant, pooled services"
	default:
		return "Unknown profile"
	}
}
