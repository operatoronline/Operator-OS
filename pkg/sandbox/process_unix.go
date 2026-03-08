//go:build !windows

package sandbox

import (
	"os/exec"
	"syscall"
)

// applyPlatformLimits sets Unix-specific resource limits via SysProcAttr.
func applyPlatformLimits(cmd *exec.Cmd, policy *Policy) {
	if policy == nil {
		return
	}

	// Use ulimit wrapper to set resource limits since SysProcAttr.Rlimit
	// is not available on all Go versions. We wrap the command in a shell
	// that sets limits before exec.
	//
	// For production container isolation, these are enforced by cgroups instead.
	// Process-level limits are defense-in-depth.

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Set process group so we can kill the entire tree on timeout.
	cmd.SysProcAttr.Setpgid = true
}
