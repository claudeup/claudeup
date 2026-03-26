//go:build !windows

// ABOUTME: Unix process-group management for test command execution.
// ABOUTME: Ensures child processes (e.g. gum) are killed when the parent times out.
package helpers

import (
	"os/exec"
	"syscall"
	"time"
)

// configureProcessGroup sets up the command to run in its own process group
// and configures the cancel function to kill the entire group. This ensures
// child processes (like gum) are also terminated on timeout.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Safe to access cmd.Process here: Go's exec.CommandContext guarantees
		// Cancel is only invoked after Start() succeeds, so Process is non-nil.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	// Give pipes a moment to flush after the process is killed
	cmd.WaitDelay = 2 * time.Second
}
