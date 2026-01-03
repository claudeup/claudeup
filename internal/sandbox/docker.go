// ABOUTME: Docker-specific sandbox runner implementation.
// ABOUTME: Handles container lifecycle, mounts, TTY attachment, and cleanup.
package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/internal/profile"
)

// DockerRunner implements Runner using Docker
type DockerRunner struct {
	// ClaudeUpDir is the claudeup config directory (~/.claudeup)
	ClaudeUpDir string
}

// NewDockerRunner creates a new Docker runner
func NewDockerRunner(claudeUpDir string) *DockerRunner {
	return &DockerRunner{ClaudeUpDir: claudeUpDir}
}

// Available checks if Docker is installed and running
func (r *DockerRunner) Available() error {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}

// Run starts a sandbox session
func (r *DockerRunner) Run(opts Options) error {
	if err := r.Available(); err != nil {
		return err
	}

	// Bootstrap profile settings on first run or sync
	if opts.Profile != "" {
		stateDir, err := StateDir(r.ClaudeUpDir, opts.Profile)
		if err != nil {
			return fmt.Errorf("failed to get state directory: %w", err)
		}

		if IsFirstRun(stateDir) || opts.Sync {
			// Load profile and bootstrap
			profilesDir := filepath.Join(r.ClaudeUpDir, "profiles")
			p, err := profile.Load(profilesDir, opts.Profile)
			if err != nil {
				return fmt.Errorf("failed to load profile for bootstrap: %w", err)
			}
			if err := BootstrapFromProfile(p, stateDir); err != nil {
				return fmt.Errorf("failed to bootstrap sandbox: %w", err)
			}
		}
	}

	args := r.buildArgs(opts)

	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// buildArgs constructs the docker run command arguments
func (r *DockerRunner) buildArgs(opts Options) []string {
	args := []string{"run", "-it", "--rm"}

	// Image
	image := opts.Image
	if image == "" {
		image = DefaultImage()
	}

	// Working directory mount
	if opts.WorkDir != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/workspace", opts.WorkDir))
	}

	// Persistent state mount (if using a profile)
	if opts.Profile != "" {
		stateDir, err := StateDir(r.ClaudeUpDir, opts.Profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not get state directory: %v\n", err)
		} else {
			args = append(args, "-v", fmt.Sprintf("%s:/root/.claude", stateDir))
		}
		// Mount profiles directory for sync access (read-only)
		profilesDir := filepath.Join(r.ClaudeUpDir, "profiles")
		args = append(args, "-v", fmt.Sprintf("%s:/root/.claudeup/profiles:ro", profilesDir))
	}

	// Additional mounts
	for _, m := range opts.Mounts {
		mountArg := fmt.Sprintf("%s:%s", m.Host, m.Container)
		if m.ReadOnly {
			mountArg += ":ro"
		}
		args = append(args, "-v", mountArg)
	}

	// Credential mounts
	if len(opts.Credentials) > 0 {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			credMounts, warnings := ResolveCredentialMounts(opts.Credentials, homeDir, "")
			for _, w := range warnings {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
			}
			for _, m := range credMounts {
				mountArg := fmt.Sprintf("%s:%s:ro", m.Host, m.Container)
				args = append(args, "-v", mountArg)
			}
		}
	}

	// Environment variables
	for key, value := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Tell entrypoint to skip sync in ephemeral mode (no profile means no profiles dir mounted)
	if opts.Profile == "" {
		args = append(args, "-e", "CLAUDEUP_SKIP_SYNC=1")
	}

	// Secrets (already resolved to values)
	// Note: In the actual integration, secrets will be resolved before calling Run

	// Network (default bridge is fine)
	args = append(args, "--network", "bridge")

	// Image
	args = append(args, image)

	// Pass "bash" to entrypoint if shell mode
	if opts.Shell {
		args = append(args, "bash")
	}

	return args
}

// PullImage pulls the sandbox image
func (r *DockerRunner) PullImage(image string) error {
	if image == "" {
		image = DefaultImage()
	}

	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ImageExists checks if the sandbox image exists locally
func (r *DockerRunner) ImageExists(image string) bool {
	if image == "" {
		image = DefaultImage()
	}

	cmd := exec.Command("docker", "image", "inspect", image)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run() == nil
}

// ParseMount parses a mount string in host:container[:ro] format
func ParseMount(s string) (Mount, error) {
	parts := strings.Split(s, ":")

	if len(parts) < 2 || len(parts) > 3 {
		return Mount{}, fmt.Errorf("invalid mount format: %s (expected host:container[:ro])", s)
	}

	host := expandHome(parts[0])
	container := parts[1]

	if host == "" {
		return Mount{}, fmt.Errorf("invalid mount: host path cannot be empty")
	}
	if container == "" {
		return Mount{}, fmt.Errorf("invalid mount: container path cannot be empty")
	}

	m := Mount{
		Host:      host,
		Container: container,
	}

	if len(parts) == 3 {
		if parts[2] == "ro" {
			m.ReadOnly = true
		} else {
			return Mount{}, fmt.Errorf("invalid mount option: %s (expected 'ro')", parts[2])
		}
	}

	return m, nil
}

// expandHome expands ~ to the user's home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}
