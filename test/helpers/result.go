// ABOUTME: Result type for capturing CLI command output
// ABOUTME: Used by TestEnv to return stdout, stderr, and exit code
package helpers

// Result captures the output of a CLI command execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Success returns true if the command exited with code 0
func (r *Result) Success() bool {
	return r.ExitCode == 0
}
