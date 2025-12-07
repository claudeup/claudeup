// ABOUTME: Setup command for first-time Claude Code configuration
// ABOUTME: Installs Claude CLI, applies profile, handles existing installations
package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/malston/claude-pm/internal/config"
	"github.com/malston/claude-pm/internal/profile"
	"github.com/malston/claude-pm/internal/secrets"
	"github.com/spf13/cobra"
)

var (
	setupProfile string
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Claude Code with a profile",
	Long: `First-time setup or reset of Claude Code configuration.

Installs Claude CLI if missing, then applies the specified profile.
If an existing installation is detected, offers to save current state
as a profile before applying the new one.`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().StringVar(&setupProfile, "profile", "default", "Profile to apply")
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("━━━ Claude PM Setup ━━━")
	fmt.Println()

	// Step 1: Check for Claude CLI
	if err := ensureClaudeCLI(); err != nil {
		return err
	}

	// Step 2: Ensure profiles directory and default profiles exist
	profilesDir := getProfilesDir()
	if err := profile.EnsureDefaultProfiles(profilesDir); err != nil {
		return fmt.Errorf("failed to set up profiles: %w", err)
	}

	// Step 4: Check for existing installation
	claudeDir := profile.DefaultClaudeDir()
	claudeJSONPath := profile.DefaultClaudeJSONPath()

	existing, err := profile.Snapshot("existing", claudeDir, claudeJSONPath)
	if err == nil && hasContent(existing) {
		if err := handleExistingInstallation(existing, profilesDir); err != nil {
			return err
		}
	}

	// Step 5: Load and show the profile
	p, err := profile.Load(profilesDir, setupProfile)
	if err != nil {
		return fmt.Errorf("failed to load profile %q: %w", setupProfile, err)
	}

	fmt.Printf("Using profile: %s\n", p.Name)
	if p.Description != "" {
		fmt.Printf("  %s\n", p.Description)
	}
	fmt.Println()

	showProfileSummary(p)

	// Step 6: Confirm (unless --yes)
	if !confirmProceed() {
		fmt.Println("Setup cancelled.")
		return nil
	}

	// Step 7: Apply the profile
	fmt.Println()
	fmt.Println("Applying profile...")

	chain := buildSecretChain()
	result, err := profile.Apply(p, claudeDir, claudeJSONPath, chain)
	if err != nil {
		return fmt.Errorf("failed to apply profile: %w", err)
	}

	// Step 8: Show results
	showApplyResults(result)

	// Step 9: Run doctor
	fmt.Println()
	fmt.Println("Running health check...")
	if err := runDoctor(cmd, nil); err != nil {
		fmt.Printf("  ⚠ Health check encountered issues: %v\n", err)
	}

	fmt.Println()
	fmt.Println("✓ Setup complete!")

	return nil
}

func ensureClaudeCLI() error {
	fmt.Print("Checking for Claude CLI... ")

	if _, err := exec.LookPath("claude"); err == nil {
		version := getClaudeVersion()
		fmt.Printf("✓ found (%s)\n", version)
		return nil
	}

	fmt.Println("not found")
	fmt.Println()
	fmt.Println("Claude CLI is required but not installed.")
	fmt.Println()

	// Auto-install with --yes, otherwise ask
	if !config.YesFlag {
		fmt.Println("Would you like to install it now using the official installer?")
		fmt.Println()
		fmt.Println("  ⚠️  Warning: This will download and execute code from the internet.")
		fmt.Println("     Command: curl -fsSL https://claude.ai/install.sh | bash")
		fmt.Println()
		choice := promptChoice("Install Claude CLI?", "y")
		if strings.ToLower(choice) != "y" && strings.ToLower(choice) != "yes" {
			fmt.Println()
			fmt.Println("To install manually, visit: https://docs.anthropic.com/en/docs/claude-code/getting-started")
			fmt.Println()
			fmt.Println("Then run 'claude-pm setup' again.")
			return fmt.Errorf("Claude CLI not installed")
		}
	}

	fmt.Println()
	fmt.Println("Installing Claude CLI...")

	cmd := exec.Command("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Claude CLI: %w", err)
	}

	fmt.Println("  ✓ Claude CLI installed")
	return nil
}

func getClaudeVersion() string {
	cmd := exec.Command("claude", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(strings.Split(string(output), "\n")[0])
}

func getProfilesDir() string {
	return filepath.Join(profile.MustHomeDir(), ".claude-pm", "profiles")
}

func hasContent(p *profile.Profile) bool {
	return len(p.Plugins) > 0 || len(p.MCPServers) > 0 || len(p.Marketplaces) > 0
}

func handleExistingInstallation(existing *profile.Profile, profilesDir string) error {
	fmt.Println("Existing Claude Code installation detected:")
	fmt.Printf("  → %d MCP servers, %d marketplaces, %d plugins\n",
		len(existing.MCPServers), len(existing.Marketplaces), len(existing.Plugins))
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [s] Save current setup as a profile, then continue")
	fmt.Println("  [c] Continue anyway (will replace current setup)")
	fmt.Println("  [a] Abort")
	fmt.Println()

	choice := promptChoice("Choice", "s")

	switch strings.ToLower(choice) {
	case "s":
		name := promptString("Profile name", "current")
		existing.Name = name
		existing.Description = "Saved from existing installation"
		if err := profile.Save(profilesDir, existing); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}
		fmt.Printf("  ✓ Saved as '%s'\n", name)
		fmt.Println()
	case "c":
		fmt.Println("  Continuing without saving...")
		fmt.Println()
	case "a":
		return fmt.Errorf("setup aborted by user")
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}

	return nil
}

func showProfileSummary(p *profile.Profile) {
	fmt.Println("Profile contents:")
	if len(p.MCPServers) > 0 {
		fmt.Printf("  MCP Servers:   %d\n", len(p.MCPServers))
		for _, m := range p.MCPServers {
			fmt.Printf("    - %s\n", m.Name)
		}
	}
	if len(p.Marketplaces) > 0 {
		fmt.Printf("  Marketplaces:  %d\n", len(p.Marketplaces))
		for _, m := range p.Marketplaces {
			fmt.Printf("    - %s\n", m.Repo)
		}
	}
	if len(p.Plugins) > 0 {
		fmt.Printf("  Plugins:       %d\n", len(p.Plugins))
		for _, plug := range p.Plugins {
			fmt.Printf("    - %s\n", plug)
		}
	}
	fmt.Println()
}

func confirmProceed() bool {
	if config.YesFlag {
		return true
	}

	choice := promptChoice("Proceed?", "y")
	return strings.ToLower(choice) == "y" || strings.ToLower(choice) == "yes"
}

func promptChoice(prompt, defaultValue string) string {
	if config.YesFlag {
		return defaultValue
	}

	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func promptString(prompt, defaultValue string) string {
	if config.YesFlag {
		return defaultValue
	}

	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

func buildSecretChain() *secrets.Chain {
	return secrets.NewChain(
		secrets.NewEnvResolver(),
		secrets.NewOnePasswordResolver(),
		secrets.NewKeychainResolver(),
	)
}

func showApplyResults(result *profile.ApplyResult) {
	if len(result.PluginsRemoved) > 0 {
		fmt.Printf("  Removed %d plugins\n", len(result.PluginsRemoved))
	}
	if len(result.PluginsInstalled) > 0 {
		fmt.Printf("  Installed %d plugins\n", len(result.PluginsInstalled))
	}
	if len(result.MCPServersRemoved) > 0 {
		fmt.Printf("  Removed %d MCP servers\n", len(result.MCPServersRemoved))
	}
	if len(result.MCPServersInstalled) > 0 {
		fmt.Printf("  Installed %d MCP servers\n", len(result.MCPServersInstalled))
	}
	if len(result.MarketplacesAdded) > 0 {
		fmt.Printf("  Added %d marketplaces\n", len(result.MarketplacesAdded))
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("  ⚠ Some operations had errors:")
		for _, err := range result.Errors {
			fmt.Printf("    - %v\n", err)
		}
	}
}
