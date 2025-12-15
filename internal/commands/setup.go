// ABOUTME: Setup command for first-time Claude Code configuration
// ABOUTME: Installs Claude CLI, applies profile, handles existing installations
package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/internal/secrets"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

// stdinReader is a shared bufio.Reader for os.Stdin to avoid buffering issues
// when multiple prompts are used in sequence
var stdinReader = bufio.NewReader(os.Stdin)

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
	Args: cobra.NoArgs,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().StringVar(&setupProfile, "profile", "default", "Profile to apply")
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.RenderHeader("claudeup Setup"))
	fmt.Println()

	// Step 1: Check for Claude CLI
	if err := ensureClaudeCLI(); err != nil {
		return err
	}

	// Step 2: Check if claude directory exists
	if err := ensureClaudeDir(); err != nil {
		return err
	}

	// Step 3: Ensure profiles directory and default profiles exist
	profilesDir := getProfilesDir()
	if err := profile.EnsureDefaultProfiles(profilesDir); err != nil {
		return fmt.Errorf("failed to set up profiles: %w", err)
	}

	// Step 4: Check for existing installation
	// Use the global claudeDir from root.go (set via --claude-dir flag)
	// Derive claudeJSONPath: when using custom dir, .claude.json is inside it
	claudeJSONPath := filepath.Join(claudeDir, ".claude.json")

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

	fmt.Println(ui.RenderDetail("Using profile", ui.Bold(p.Name)))
	if p.Description != "" {
		fmt.Printf("  %s\n", ui.Muted(p.Description))
	}
	fmt.Println()

	showProfileSummary(p)

	// Step 6: Confirm (unless --yes)
	if !confirmProceed() {
		ui.PrintMuted("Setup cancelled.")
		return nil
	}

	// Step 7: Apply the profile
	fmt.Println()
	ui.PrintInfo("Applying profile...")

	chain := buildSecretChain()
	result, err := profile.Apply(p, claudeDir, claudeJSONPath, chain)
	if err != nil {
		return fmt.Errorf("failed to apply profile: %w", err)
	}

	// Step 8: Show results
	showApplyResults(result)

	// Step 9: Run doctor
	fmt.Println()
	ui.PrintInfo("Running health check...")
	if err := runDoctor(cmd, nil); err != nil {
		ui.PrintWarning(fmt.Sprintf("Health check encountered issues: %v", err))
	}

	fmt.Println()
	if len(result.Errors) > 0 {
		ui.PrintWarning("Setup completed with errors. Review the issues above.")
	} else {
		ui.PrintSuccess("Setup complete!")
	}

	return nil
}

// ensureClaudeDir checks if the claude directory exists and prompts to create it if not
func ensureClaudeDir() error {
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		ui.PrintWarning(fmt.Sprintf("Directory %s does not exist.", claudeDir))
		fmt.Println()
		fmt.Println(ui.Bold("Options:"))
		fmt.Println("  [c] Create it and continue")
		fmt.Println("  [a] Abort")
		fmt.Println()

		choice := promptChoice("Choice", "c")

		switch strings.ToLower(choice) {
		case "c":
			// Create the directory structure
			if err := os.MkdirAll(filepath.Join(claudeDir, "plugins"), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			ui.PrintSuccess(fmt.Sprintf("Created %s", claudeDir))
			fmt.Println()
		case "a":
			fmt.Println("Setup aborted.")
			return fmt.Errorf("setup aborted by user")
		default:
			return fmt.Errorf("invalid choice: %s", choice)
		}
	}
	return nil
}

// Minimum Claude CLI version required for proper functionality
// Versions before 1.0.80 have Ink raw mode issues when stdin is not properly connected
const minClaudeVersion = "1.0.80"

func ensureClaudeCLI() error {
	fmt.Print("Checking for Claude CLI... ")

	if _, err := exec.LookPath("claude"); err == nil {
		version := getClaudeVersion()
		if version != "unknown" && isVersionOutdated(version, minClaudeVersion) {
			fmt.Printf("%s outdated (%s)\n", ui.Warning(ui.SymbolWarning), version)
			fmt.Println()
			ui.PrintWarning(fmt.Sprintf("Claude CLI version %s is installed, but version %s or newer is required.", version, minClaudeVersion))
			fmt.Println("Older versions have known issues with terminal handling that cause setup to fail.")
			fmt.Println()
			return promptClaudeUpgrade(version)
		}
		fmt.Printf("%s found (%s)\n", ui.Success(ui.SymbolSuccess), version)
		return nil
	}

	fmt.Printf("%s not found\n", ui.Warning(ui.SymbolWarning))
	fmt.Println()
	ui.PrintWarning("Claude CLI is required but not installed.")
	fmt.Println()

	// Auto-install with --yes, otherwise ask
	if !config.YesFlag {
		fmt.Println("Would you like to install it now using the official installer?")
		fmt.Println()
		ui.PrintWarning("Warning: This will download and execute code from the internet.")
		fmt.Println("     Command: curl -fsSL https://claude.ai/install.sh | bash")
		fmt.Println()
		choice := promptChoice("Install Claude CLI?", "y")
		if strings.ToLower(choice) != "y" && strings.ToLower(choice) != "yes" {
			fmt.Println()
			fmt.Println("To install manually, visit: https://docs.anthropic.com/en/docs/claude-code/getting-started")
			fmt.Println()
			fmt.Println("Then run 'claudeup setup' again.")
			return fmt.Errorf("Claude CLI not installed")
		}
	}

	fmt.Println()
	ui.PrintInfo("Installing Claude CLI...")

	if err := runClaudeInstaller(); err != nil {
		return fmt.Errorf("failed to install Claude CLI: %w", err)
	}

	ui.PrintSuccess("Claude CLI installed")
	return nil
}

// runClaudeInstaller runs the official Claude CLI installer script
func runClaudeInstaller() error {
	cmd := exec.Command("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getClaudeVersion() string {
	cmd := exec.Command("claude", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(strings.Split(string(output), "\n")[0])
}

// isVersionOutdated returns true if current version is older than minimum version
// Uses simple string comparison of semver-like versions (e.g., "1.0.72" vs "1.0.80")
func isVersionOutdated(current, minimum string) bool {
	currentParts := parseVersion(current)
	minimumParts := parseVersion(minimum)

	for i := 0; i < len(minimumParts); i++ {
		if i >= len(currentParts) {
			return true // Current has fewer parts, treat as older
		}
		if currentParts[i] < minimumParts[i] {
			return true
		}
		if currentParts[i] > minimumParts[i] {
			return false
		}
	}
	return false
}

// parseVersion extracts numeric parts from a version string
// Handles formats like "1.0.72", "claude 1.0.72", "v1.0.72"
func parseVersion(version string) []int {
	// Remove common prefixes
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "claude ")

	parts := strings.Split(version, ".")
	nums := make([]int, 0, len(parts))

	for _, part := range parts {
		// Extract numeric portion (handles cases like "72-beta")
		numStr := ""
		for _, c := range part {
			if c >= '0' && c <= '9' {
				numStr += string(c)
			} else {
				break
			}
		}
		if numStr != "" {
			if num, err := strconv.Atoi(numStr); err == nil {
				nums = append(nums, num)
			}
		}
	}
	return nums
}

// promptClaudeUpgrade asks the user if they want to upgrade Claude CLI
func promptClaudeUpgrade(currentVersion string) error {
	if !config.YesFlag {
		fmt.Println("Would you like to upgrade Claude CLI now using the official installer?")
		fmt.Println()
		ui.PrintWarning("Warning: This will download and execute code from the internet.")
		fmt.Println("     Command: curl -fsSL https://claude.ai/install.sh | bash")
		fmt.Println()
		choice := promptChoice("Upgrade Claude CLI?", "y")
		if strings.ToLower(choice) != "y" && strings.ToLower(choice) != "yes" {
			fmt.Println()
			fmt.Println("To upgrade manually, run:")
			fmt.Println("  curl -fsSL https://claude.ai/install.sh | bash")
			fmt.Println()
			fmt.Println("Then run 'claudeup setup' again.")
			return fmt.Errorf("Claude CLI version %s is outdated (minimum: %s)", currentVersion, minClaudeVersion)
		}
	}

	fmt.Println()
	ui.PrintInfo("Upgrading Claude CLI...")

	if err := runClaudeInstaller(); err != nil {
		return fmt.Errorf("failed to upgrade Claude CLI: %w", err)
	}

	// Verify the upgrade succeeded
	newVersion := getClaudeVersion()
	if newVersion != "unknown" && isVersionOutdated(newVersion, minClaudeVersion) {
		return fmt.Errorf("Claude CLI upgrade did not resolve version issue (still %s, need %s)", newVersion, minClaudeVersion)
	}

	ui.PrintSuccess(fmt.Sprintf("Claude CLI upgraded to %s", newVersion))
	return nil
}

func getProfilesDir() string {
	return filepath.Join(profile.MustHomeDir(), ".claudeup", "profiles")
}

func hasContent(p *profile.Profile) bool {
	return len(p.Plugins) > 0 || len(p.MCPServers) > 0 || len(p.Marketplaces) > 0
}

func handleExistingInstallation(existing *profile.Profile, profilesDir string) error {
	ui.PrintInfo("Existing Claude Code installation detected:")
	fmt.Printf("  %s %d MCP servers, %d marketplaces, %d plugins\n",
		ui.Muted(ui.SymbolArrow), len(existing.MCPServers), len(existing.Marketplaces), len(existing.Plugins))
	fmt.Println()
	fmt.Println(ui.Bold("Options:"))
	fmt.Println("  [s] Save current setup as a profile, then continue")
	fmt.Println("  [b] Create backup of directory, then continue")
	fmt.Println("  [c] Continue anyway (will replace current setup)")
	fmt.Println("  [a] Abort")
	fmt.Println()

	choice := promptChoice("Choice", "s")

	switch strings.ToLower(choice) {
	case "s":
		name := promptProfileName("Profile name", "saved")
		existing.Name = name
		existing.Description = "Saved from existing installation"
		if err := profile.Save(profilesDir, existing); err != nil {
			return fmt.Errorf("failed to save profile: %w", err)
		}
		ui.PrintSuccess(fmt.Sprintf("Saved as '%s'", name))
		fmt.Println()
	case "b":
		backupPath, err := createBackup(claudeDir)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		ui.PrintSuccess(fmt.Sprintf("Created backup at %s", backupPath))
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

// createBackup creates a backup copy of the claude directory
// Returns the path to the backup directory
func createBackup(srcDir string) (string, error) {
	// Find an available backup path
	backupBase := srcDir + ".backup"
	backupPath := backupBase

	// If backup already exists, add a number suffix
	const maxBackups = 100
	counter := 1
	for counter <= maxBackups {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s.%d", backupBase, counter)
		counter++
	}
	if counter > maxBackups {
		return "", fmt.Errorf("too many existing backups (max %d), please clean up old backups", maxBackups)
	}

	// Copy the directory recursively
	if err := copyDir(srcDir, backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}

func showProfileSummary(p *profile.Profile) {
	fmt.Println(ui.Bold("Profile contents:"))
	if len(p.MCPServers) > 0 {
		fmt.Println(ui.Indent(ui.RenderDetail("MCP Servers", fmt.Sprintf("%d", len(p.MCPServers))), 1))
		for _, m := range p.MCPServers {
			fmt.Printf("    %s %s\n", ui.Muted(ui.SymbolBullet), m.Name)
		}
	}
	if len(p.Marketplaces) > 0 {
		fmt.Println(ui.Indent(ui.RenderDetail("Marketplaces", fmt.Sprintf("%d", len(p.Marketplaces))), 1))
		for _, m := range p.Marketplaces {
			fmt.Printf("    %s %s\n", ui.Muted(ui.SymbolBullet), m.Repo)
		}
	}
	if len(p.Plugins) > 0 {
		fmt.Println(ui.Indent(ui.RenderDetail("Plugins", fmt.Sprintf("%d", len(p.Plugins))), 1))
		for _, plug := range p.Plugins {
			fmt.Printf("    %s %s\n", ui.Muted(ui.SymbolBullet), plug)
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
	input, _ := stdinReader.ReadString('\n')
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
	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

// promptProfileName prompts for a profile name and validates that it doesn't conflict with embedded profiles
func promptProfileName(prompt, defaultValue string) string {
	for {
		name := promptString(prompt, defaultValue)

		// Check if this is an embedded profile
		if profile.IsEmbeddedProfile(name) {
			ui.PrintError(fmt.Sprintf("Cannot overwrite built-in profile '%s'", name))
			fmt.Println()
			continue
		}

		return name
	}
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
		fmt.Printf("  %s Removed %d plugins\n", ui.Success(ui.SymbolSuccess), len(result.PluginsRemoved))
	}
	if len(result.PluginsAlreadyRemoved) > 0 {
		fmt.Printf("  %s %d plugins were already uninstalled\n", ui.Muted(ui.SymbolSuccess), len(result.PluginsAlreadyRemoved))
	}
	if len(result.PluginsInstalled) > 0 {
		fmt.Printf("  %s Installed %d plugins\n", ui.Success(ui.SymbolSuccess), len(result.PluginsInstalled))
	}
	if len(result.PluginsAlreadyPresent) > 0 {
		fmt.Printf("  %s %d plugins were already installed\n", ui.Muted(ui.SymbolSuccess), len(result.PluginsAlreadyPresent))
	}
	if len(result.MCPServersRemoved) > 0 {
		fmt.Printf("  %s Removed %d MCP servers\n", ui.Success(ui.SymbolSuccess), len(result.MCPServersRemoved))
	}
	if len(result.MCPServersInstalled) > 0 {
		fmt.Printf("  %s Installed %d MCP servers\n", ui.Success(ui.SymbolSuccess), len(result.MCPServersInstalled))
	}
	if len(result.MarketplacesAdded) > 0 {
		fmt.Printf("  %s Added %d marketplaces\n", ui.Success(ui.SymbolSuccess), len(result.MarketplacesAdded))
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		ui.PrintWarning("Some operations had errors:")
		for _, err := range result.Errors {
			fmt.Printf("    %s %v\n", ui.Error(ui.SymbolBullet), err)
		}
	}
}
