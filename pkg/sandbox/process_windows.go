//go:build windows

package sandbox

import (
	"os/exec"
)

// applyPlatformLimits is a no-op on Windows — resource limits are
// applied at the container level when available.
func applyPlatformLimits(cmd *exec.Cmd, policy *Policy) {
	// Windows Job Objects could be used here in the future.
}
