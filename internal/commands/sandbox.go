// ABOUTME: Sandbox command for running Claude Code in a Docker container.
// ABOUTME: Provides security isolation with TTY passthrough and profile-based persistence.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/internal/sandbox"
	"github.com/claudeup/claudeup/internal/secrets"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	sandboxProfile    string
	sandboxMounts     []string
	sandboxNoMount    bool
	sandboxSecrets    []string
	sandboxNoSecrets  []string
	sandboxShell      bool
	sandboxClean      bool
	sandboxImage      string
	sandboxEphemeral  bool
	sandboxCopyAuth   bool
)

var sandboxCmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Run Claude Code in a Docker container",
	Long: `Run Claude Code in an isolated Docker container for security.

By default, runs an ephemeral session where nothing persists after exit.
Use --profile to persist state between sessions.

The current working directory is mounted at /workspace unless --no-mount is used.`,
	Example: `  # Ephemeral session
  claudeup sandbox

  # Persistent session using a profile
  claudeup sandbox --profile untrusted

  # Drop to bash instead of Claude CLI
  claudeup sandbox --shell

  # Add extra mount
  claudeup sandbox --mount ~/data:/data

  # Reset a profile's sandbox state
  claudeup sandbox --clean --profile untrusted`,
	RunE: runSandbox,
}

func init() {
	rootCmd.AddCommand(sandboxCmd)

	sandboxCmd.Flags().StringVar(&sandboxProfile, "profile", "", "Profile for persistent state")
	sandboxCmd.Flags().StringSliceVar(&sandboxMounts, "mount", nil, "Additional mounts (host:container[:ro])")
	sandboxCmd.Flags().BoolVar(&sandboxNoMount, "no-mount", false, "Don't mount working directory")
	sandboxCmd.Flags().StringSliceVar(&sandboxSecrets, "secret", nil, "Additional secrets to inject")
	sandboxCmd.Flags().StringSliceVar(&sandboxNoSecrets, "no-secret", nil, "Secrets to exclude")
	sandboxCmd.Flags().BoolVar(&sandboxShell, "shell", false, "Drop to bash instead of Claude CLI")
	sandboxCmd.Flags().BoolVar(&sandboxClean, "clean", false, "Reset sandbox state for profile")
	sandboxCmd.Flags().StringVar(&sandboxImage, "image", "", "Override sandbox image")
	sandboxCmd.Flags().BoolVar(&sandboxEphemeral, "ephemeral", false, "Force ephemeral mode (no persistence)")
	sandboxCmd.Flags().BoolVar(&sandboxCopyAuth, "copy-auth", false, "Copy authentication from ~/.claude.json")
}

func runSandbox(cmd *cobra.Command, args []string) error {
	claudePMDir := config.MustClaudeupHome()

	// Handle --clean
	if sandboxClean {
		if sandboxProfile == "" {
			return fmt.Errorf("--clean requires --profile")
		}
		if err := sandbox.CleanState(claudePMDir, sandboxProfile); err != nil {
			return err
		}
		ui.PrintSuccess(fmt.Sprintf("Cleaned sandbox state for profile %q", sandboxProfile))
		return nil
	}

	// Check Docker availability
	runner := sandbox.NewDockerRunner(claudePMDir)
	if err := runner.Available(); err != nil {
		return fmt.Errorf("docker is required: %w", err)
	}

	// Build options
	opts := sandbox.Options{
		Shell: sandboxShell,
		Image: sandboxImage,
		Env:   make(map[string]string),
	}

	// Profile handling
	if sandboxProfile != "" && !sandboxEphemeral {
		opts.Profile = sandboxProfile

		// Load profile for sandbox config
		profilesDir := filepath.Join(claudePMDir, "profiles")
		p, err := profile.Load(profilesDir, sandboxProfile)
		if err != nil {
			return fmt.Errorf("failed to load profile %q: %w", sandboxProfile, err)
		}
		// Apply profile's sandbox config (may be empty, that's fine)
		applyProfileSandboxConfig(&opts, p)
	}

	// Working directory mount
	if !sandboxNoMount {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		opts.WorkDir = wd
	}

	// Parse additional mounts
	for _, m := range sandboxMounts {
		mount, err := sandbox.ParseMount(m)
		if err != nil {
			return err
		}
		opts.Mounts = append(opts.Mounts, mount)
	}

	// CLI secret overrides
	opts.Secrets = append(opts.Secrets, sandboxSecrets...)
	opts.ExcludeSecrets = append(opts.ExcludeSecrets, sandboxNoSecrets...)

	// Resolve secrets
	if err := resolveSecrets(&opts); err != nil {
		return fmt.Errorf("failed to resolve secrets: %w", err)
	}

	// Copy authentication if requested
	if shouldCopyAuth(opts.Profile) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		if err := sandbox.CopyAuthFile(homeDir, claudePMDir, opts.Profile); err != nil {
			// If user explicitly requested --copy-auth, fail hard
			if sandboxCopyAuth {
				return fmt.Errorf("--copy-auth failed: %w", err)
			}
			// Otherwise just warn (config setting default)
			ui.PrintWarning(fmt.Sprintf("Failed to copy authentication: %v", err))
		}
	}

	// Ensure image exists
	if !runner.ImageExists(opts.Image) {
		image := opts.Image
		if image == "" {
			image = sandbox.DefaultImage()
		}
		ui.PrintInfo(fmt.Sprintf("Pulling sandbox image %s...", image))
		if err := runner.PullImage(opts.Image); err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
	}

	// Show what we're doing
	printSandboxInfo(opts)

	// Run the sandbox
	return runner.Run(opts)
}

func applyProfileSandboxConfig(opts *sandbox.Options, p *profile.Profile) {
	// Add profile secrets
	opts.Secrets = append(opts.Secrets, p.Sandbox.Secrets...)

	// Add profile mounts
	for _, m := range p.Sandbox.Mounts {
		opts.Mounts = append(opts.Mounts, sandbox.Mount{
			Host:      m.Host,
			Container: m.Container,
			ReadOnly:  m.ReadOnly,
		})
	}

	// Add profile env
	for k, v := range p.Sandbox.Env {
		opts.Env[k] = v
	}
}

func resolveSecrets(opts *sandbox.Options) error {
	if len(opts.Secrets) == 0 {
		return nil
	}

	chain := secrets.NewChain(
		secrets.NewEnvResolver(),
		secrets.NewOnePasswordResolver(),
		secrets.NewKeychainResolver(),
	)

	// Build exclusion set
	excluded := make(map[string]bool)
	for _, s := range opts.ExcludeSecrets {
		excluded[s] = true
	}

	// Resolve each secret
	for _, secretName := range opts.Secrets {
		if excluded[secretName] {
			continue
		}

		value, source, err := chain.Resolve(secretName)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("could not resolve secret %q: %v", secretName, err))
			continue
		}

		opts.Env[secretName] = value
		_ = source // Could log which source was used
	}

	return nil
}

func shouldCopyAuth(profile string) bool {
	// Only copy auth for profile-based sandboxes
	if profile == "" {
		return false
	}

	// Check if flag is explicitly set
	if sandboxCopyAuth {
		return true
	}

	// Check config setting
	cfg, err := config.Load()
	if err != nil {
		// If config can't be loaded, default to not copying
		return false
	}

	return cfg.Sandbox.CopyAuth
}

func printSandboxInfo(opts sandbox.Options) {
	fmt.Println(ui.RenderSection("Sandbox", -1))
	fmt.Println()

	if opts.Profile != "" {
		fmt.Println(ui.RenderDetail("Profile", ui.Bold(opts.Profile)+" "+ui.Muted("(persistent)")))
	} else {
		fmt.Println(ui.RenderDetail("Mode", "ephemeral"))
	}

	if opts.WorkDir != "" {
		fmt.Println(ui.RenderDetail("Workdir", fmt.Sprintf("%s %s /workspace", ui.Muted(opts.WorkDir), ui.SymbolArrow)))
	} else {
		fmt.Println(ui.RenderDetail("Workdir", ui.Muted("(none)")))
	}

	if len(opts.Mounts) > 0 {
		fmt.Println(ui.RenderDetail("Mounts", fmt.Sprintf("%d additional", len(opts.Mounts))))
	}

	secretCount := 0
	for range opts.Env {
		secretCount++
	}
	if secretCount > 0 {
		fmt.Println(ui.RenderDetail("Secrets", fmt.Sprintf("%d injected", secretCount)))
	}

	if opts.Shell {
		fmt.Println(ui.RenderDetail("Entry", "bash"))
	} else {
		fmt.Println(ui.RenderDetail("Entry", "claude"))
	}

	fmt.Println()
}
