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
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	// WaitDelay bounds how long Go waits for I/O pipes to drain after the
	// process is killed; prevents test hangs if a child leaves pipes open.
	cmd.WaitDelay = 2 * time.Second
}
