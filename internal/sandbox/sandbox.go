// ABOUTME: Core sandbox types and orchestration for running Claude in containers.
// ABOUTME: Provides Options struct and Manager interface for sandbox lifecycle.
package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
)

// Options configures a sandbox session
type Options struct {
	// Profile name for persistent state (empty = ephemeral)
	Profile string

	// WorkDir is the host directory to mount at /workspace
	// Empty string means no mount
	WorkDir string

	// Mounts are additional host:container path mappings
	Mounts []Mount

	// Credentials are credential type names to mount (git, ssh, gh)
	Credentials []string

	// Secrets are environment variable names to resolve and inject
	Secrets []string

	// ExcludeSecrets are secret names to exclude (overrides profile)
	ExcludeSecrets []string

	// Env are static environment variables to set
	Env map[string]string

	// Shell drops to bash instead of Claude CLI
	Shell bool

	// Image overrides the default sandbox image
	Image string

	// Sync forces re-application of profile settings
	Sync bool
}

// Mount represents a host-to-container path mapping
type Mount struct {
	Host      string
	Container string
	ReadOnly  bool
}

// Runner executes sandbox sessions
type Runner interface {
	// Run starts a sandbox session with the given options
	// It blocks until the session ends
	Run(opts Options) error

	// Available returns true if this runner can be used
	Available() error
}

// StateDir returns the sandbox state directory for a profile
// Creates the directory if it doesn't exist
func StateDir(claudeUpDir, profile string) (string, error) {
	if profile == "" {
		return "", fmt.Errorf("profile name required for persistent state")
	}

	dir := filepath.Join(claudeUpDir, "sandboxes", profile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create sandbox state dir: %w", err)
	}

	return dir, nil
}

// CleanState removes the sandbox state directory for a profile
func CleanState(claudeUpDir, profile string) error {
	if profile == "" {
		return fmt.Errorf("profile name required")
	}

	dir := filepath.Join(claudeUpDir, "sandboxes", profile)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove sandbox state: %w", err)
	}

	return nil
}

// DefaultImage returns the default sandbox image name
func DefaultImage() string {
	return "ghcr.io/claudeup/claudeup-sandbox:latest"
}

// CopyAuthFile copies the user's .claude.json file to the sandbox state directory
// This allows sandboxes to use the user's existing authentication without interactive prompts
func CopyAuthFile(homeDir, claudeUpDir, profile string) error {
	if profile == "" {
		return fmt.Errorf("profile name required")
	}

	// Source: user's .claude.json in home directory
	sourceFile := filepath.Join(homeDir, ".claude.json")
	if _, err := os.Stat(sourceFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("auth file not found at %s", sourceFile)
		}
		return fmt.Errorf("failed to check auth file: %w", err)
	}

	// Destination: sandbox state directory
	stateDir, err := StateDir(claudeUpDir, profile)
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}

	destFile := filepath.Join(stateDir, ".claude.json")

	// Copy the file
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to read auth file: %w", err)
	}

	if err := os.WriteFile(destFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	return nil
}
