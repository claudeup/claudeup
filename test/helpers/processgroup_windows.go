//go:build windows

// ABOUTME: Windows no-op for process-group management in test command execution.
// ABOUTME: On Windows, exec.CommandContext already kills the process on timeout;
// ABOUTME: child process cleanup is best-effort without Unix process groups.
package helpers

import (
	"os/exec"
	"time"
)

// configureProcessGroup is a no-op on Windows. exec.CommandContext will still
// kill the main process on timeout; child processes may outlive the parent.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.WaitDelay = 2 * time.Second
}
