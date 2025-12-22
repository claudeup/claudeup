// ABOUTME: Scope command for managing and viewing Claude Code settings across scopes
// ABOUTME: Provides list and clear subcommands for scope-level configuration management
package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/claudeup/claudeup/internal/backup"
	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	scopeListScope    string
	scopeClearForce   bool
	scopeClearBackup  bool
	scopeRestoreForce bool
)

var scopeCmd = &cobra.Command{
	Use:   "scope",
	Short: "Manage Claude Code settings scopes",
	Long: `View and manage Claude Code settings across different scopes.

Claude Code uses three scope levels:
  - user: Global settings (~/.claude/settings.json)
  - project: Project settings (.claude/settings.json)
  - local: Local overrides (.claude/settings.local.json)

Settings are merged with local > project > user precedence.`,
}

var scopeListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show plugins enabled at each scope",
	Long: `Display which plugins are enabled at each scope level.

Shows a hierarchical view of all scopes with enabled plugins.
Use --scope to filter to a specific scope level.

Examples:
  claudeup scope list                    # Show all scopes
  claudeup scope list --scope user       # Show only user scope
  claudeup scope list --scope project    # Show only project scope`,
	Args: cobra.NoArgs,
	RunE: runScopeList,
}

var scopeClearCmd = &cobra.Command{
	Use:   "clear <user|project|local>",
	Short: "Clear settings at a specific scope",
	Long: `Remove settings at the specified scope level.

This is a destructive operation that removes configuration files.
You will be prompted for confirmation unless --force is used.

User scope:
  - Resets ~/.claude/settings.json to empty configuration
  - Does not remove plugins (use 'claudeup plugin' commands)

Project scope:
  - Removes .claude/settings.json from project
  - Team members will be affected on next pull

Local scope:
  - Removes .claude/settings.local.json
  - Only affects this machine

Examples:
  claudeup scope clear user              # Clear user scope with confirmation
  claudeup scope clear project --force   # Clear project scope without confirmation
  claudeup scope clear local             # Clear local scope with confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runScopeClear,
}

var scopeRestoreCmd = &cobra.Command{
	Use:   "restore <user|local>",
	Short: "Restore settings from a backup",
	Long: `Restore settings for a scope from the most recent backup.

Backups are created when using 'scope clear' or 'profile use --reset'.
Only user and local scopes support restore; for project scope, use git.

Examples:
  claudeup scope restore user          # Restore user scope from backup
  claudeup scope restore local         # Restore local scope from backup`,
	Args: cobra.ExactArgs(1),
	RunE: runScopeRestore,
}

func init() {
	rootCmd.AddCommand(scopeCmd)
	scopeCmd.AddCommand(scopeListCmd)
	scopeCmd.AddCommand(scopeClearCmd)
	scopeCmd.AddCommand(scopeRestoreCmd)

	scopeListCmd.Flags().StringVar(&scopeListScope, "scope", "", "Filter to scope: user, project, or local (default: show all)")
	scopeClearCmd.Flags().BoolVar(&scopeClearForce, "force", false, "Skip confirmation prompts")
	scopeClearCmd.Flags().BoolVar(&scopeClearBackup, "backup", false, "Create backup before clearing")
	scopeRestoreCmd.Flags().BoolVar(&scopeRestoreForce, "force", false, "Skip confirmation prompts")
}

func runScopeList(cmd *cobra.Command, args []string) error {
	// Get current directory for project/local scopes
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Validate scope if specified
	if scopeListScope != "" {
		if err := claude.ValidateScope(scopeListScope); err != nil {
			return err
		}
	}

	// Determine which scopes to show
	scopesToShow := []string{}
	if scopeListScope != "" {
		scopesToShow = append(scopesToShow, scopeListScope)
	} else {
		scopesToShow = []string{"user", "project", "local"}
	}

	// Track if we're in a project directory
	inProjectDir := false
	if _, err := os.Stat(projectDir + "/.claude"); err == nil {
		inProjectDir = true
	}

	// Load settings for each scope and display
	effectivePlugins := make(map[string]bool)
	totalEnabled := 0

	for _, scope := range scopesToShow {
		// Skip project/local if not in project directory and showing all scopes
		if scopeListScope == "" && !inProjectDir && (scope == "project" || scope == "local") {
			continue
		}

		// Get settings path for this scope
		settingsPath, err := claude.SettingsPathForScope(scope, claudeDir, projectDir)
		if err != nil {
			return err
		}

		// Load settings for this scope
		settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
		if err != nil {
			// If file doesn't exist, show appropriate message
			if os.IsNotExist(err) || (scope != "user" && err != nil) {
				if scopeListScope == "" {
					// When showing all scopes, mention it's not configured
					fmt.Println(ui.RenderSection(fmt.Sprintf("Scope: %s (%s)", formatScopeName(scope), settingsPath), -1))
					fmt.Printf("  %s Not configured\n", ui.Muted("—"))
					fmt.Println()
					continue
				} else {
					// When showing specific scope, it's an error if it doesn't exist
					return fmt.Errorf("%s scope not configured", scope)
				}
			}
			return err
		}

		// Get enabled plugins for this scope
		enabledPlugins := []string{}
		for plugin, enabled := range settings.EnabledPlugins {
			if enabled {
				enabledPlugins = append(enabledPlugins, plugin)
				effectivePlugins[plugin] = true
			}
		}
		sort.Strings(enabledPlugins)

		// Display scope header
		fmt.Println(ui.RenderSection(fmt.Sprintf("Scope: %s (%s)", formatScopeName(scope), settingsPath), len(enabledPlugins)))

		// Display plugins
		if len(enabledPlugins) > 0 {
			for _, plugin := range enabledPlugins {
				fmt.Printf("  %s %s\n", ui.Success(ui.SymbolSuccess), plugin)
				totalEnabled++
			}
		} else {
			fmt.Printf("  %s No plugins enabled\n", ui.Muted("—"))
		}

		fmt.Println()
	}

	// Show messages for non-project directory if showing all scopes
	if scopeListScope == "" && !inProjectDir {
		fmt.Println(ui.Muted("Project scope: Not in a project directory"))
		fmt.Println(ui.Muted("Local scope: Not configured for this directory"))
		fmt.Println()
	}

	// Show effective configuration when showing all scopes
	if scopeListScope == "" {
		fmt.Println(ui.Bold(fmt.Sprintf("Effective Configuration: %d unique plugins enabled", len(effectivePlugins))))
	}

	return nil
}

// formatScopeName returns a capitalized scope name for display
func formatScopeName(scope string) string {
	switch scope {
	case "user":
		return "User"
	case "project":
		return "Project"
	case "local":
		return "Local"
	default:
		return scope
	}
}

func runScopeClear(cmd *cobra.Command, args []string) error {
	scope := args[0]

	// Validate scope
	if err := claude.ValidateScope(scope); err != nil {
		return err
	}

	// Get current directory for project/local scopes
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get settings path for this scope
	settingsPath, err := claude.SettingsPathForScope(scope, claudeDir, projectDir)
	if err != nil {
		return err
	}

	// Check if settings file exists
	_, err = os.Stat(settingsPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s scope is not configured\nHint: No settings file found at %s", scope, settingsPath)
	}

	// Load current settings to show what will be cleared
	settings, err := claude.LoadSettingsForScope(scope, claudeDir, projectDir)
	if err != nil {
		return fmt.Errorf("failed to load %s scope settings: %w", scope, err)
	}

	// Count enabled plugins
	enabledCount := 0
	for _, enabled := range settings.EnabledPlugins {
		if enabled {
			enabledCount++
		}
	}

	// Show what will be cleared
	fmt.Println(ui.RenderSection(fmt.Sprintf("Clear %s scope (%s)", formatScopeName(scope), settingsPath), -1))
	fmt.Println()

	if enabledCount > 0 {
		fmt.Printf("This will reset:\n")
		fmt.Printf("  • %d enabled plugins\n", enabledCount)
		fmt.Printf("  • %s-level settings\n", formatScopeName(scope))
	} else {
		fmt.Printf("This will remove the settings file (no plugins currently enabled)\n")
	}

	fmt.Println()

	// Scope-specific warnings
	switch scope {
	case "user":
		fmt.Println(ui.Muted("This does NOT:"))
		fmt.Println(ui.Muted("  • Remove plugins from disk (use 'claudeup plugin uninstall')"))
		fmt.Println(ui.Muted("  • Remove marketplaces"))
		fmt.Println(ui.Muted("  • Affect project or local scopes"))
	case "project":
		ui.PrintWarning("Team Impact Warning:")
		fmt.Println("  Team members will lose project configuration on next pull.")
		fmt.Println("  Consider committing this change if intentional.")
	case "local":
		fmt.Println(ui.Muted("This only affects this machine."))
		fmt.Println(ui.Muted("Project and user scopes remain unchanged."))
	}

	fmt.Println()

	// Prompt for confirmation unless --force
	if !scopeClearForce {
		confirmed := ui.PromptYesNo(fmt.Sprintf("Clear %s scope settings?", scope), false)
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Create backup if requested
	if scopeClearBackup {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		var backupPath string
		if scope == "local" {
			backupPath, err = backup.SaveLocalScopeBackup(homeDir, projectDir, settingsPath)
		} else {
			backupPath, err = backup.SaveScopeBackup(homeDir, scope, settingsPath)
		}
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("  Backup saved: %s\n", backupPath)
	}

	// Clear the scope
	if err := clearScope(scope, settingsPath, claudeDir); err != nil {
		return fmt.Errorf("failed to clear %s scope: %w", scope, err)
	}

	ui.PrintSuccess(fmt.Sprintf("Cleared %s scope settings", scope))

	// Show next steps
	fmt.Println()
	fmt.Printf("To restore settings, use:\n")
	fmt.Printf("  claudeup scope list --scope %s   # Verify it's cleared\n", scope)

	switch scope {
	case "user", "project":
		fmt.Printf("  claude plugin install <plugin>  # Re-enable plugins\n")
	case "local":
		fmt.Printf("  claude plugin install <plugin> --scope local  # Re-enable local plugins\n")
	}

	return nil
}

// clearScope removes settings at the specified scope
func clearScope(scope string, settingsPath string, claudeDir string) error {
	switch scope {
	case "user":
		// Load existing settings and only clear enabledPlugins
		settings, err := claude.LoadSettings(claudeDir)
		if err != nil {
			// If settings don't exist, create minimal settings
			settings = &claude.Settings{
				EnabledPlugins: make(map[string]bool),
			}
		} else {
			// Clear only the enabledPlugins field, preserve everything else
			settings.EnabledPlugins = make(map[string]bool)
		}
		return claude.SaveSettings(claudeDir, settings)

	case "project":
		// Remove project settings file
		if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil

	case "local":
		// Remove local settings file
		if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil

	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}
}

func runScopeRestore(cmd *cobra.Command, args []string) error {
	scope := args[0]

	// Reject project scope
	if scope == "project" {
		return fmt.Errorf("project scope restore not supported\nHint: Use 'git checkout' to restore project configuration files")
	}

	// Validate scope
	if err := claude.ValidateScope(scope); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Get backup info
	backupInfo, err := backup.GetBackupInfo(homeDir, scope)
	if err != nil {
		return fmt.Errorf("failed to check backup: %w", err)
	}

	if !backupInfo.Exists {
		return fmt.Errorf("no backup found for %s scope\nHint: Backups are created when using 'scope clear' or 'profile use --reset'", scope)
	}

	// Get settings path
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	settingsPath, err := claude.SettingsPathForScope(scope, claudeDir, projectDir)
	if err != nil {
		return err
	}

	// Show backup info
	fmt.Println(ui.RenderSection(fmt.Sprintf("Restore %s scope from backup", formatScopeName(scope)), -1))
	fmt.Println()
	fmt.Printf("Backup created: %s\n", backupInfo.ModTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Contents:\n")
	fmt.Printf("  • %d plugins enabled\n", backupInfo.PluginCount)
	fmt.Println()

	// Confirm unless --force
	if !scopeRestoreForce {
		confirmed := ui.PromptYesNo("Restore this backup?", false)
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Restore
	if err := backup.RestoreScopeBackup(homeDir, scope, settingsPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Restored %s scope from backup", scope))

	return nil
}
