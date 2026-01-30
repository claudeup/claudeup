// ABOUTME: Scope command for managing and viewing Claude Code settings across scopes
// ABOUTME: Provides list and clear subcommands for scope-level configuration management
package commands

import (
	"bufio"
	"fmt"
	"os"

	"github.com/claudeup/claudeup/v4/internal/backup"
	"github.com/claudeup/claudeup/v4/internal/claude"
	"github.com/claudeup/claudeup/v4/internal/config"
	"github.com/claudeup/claudeup/v4/internal/ui"
	"github.com/spf13/cobra"
)

var (
	scopeListScope    string
	scopeListUser     bool
	scopeListProject  bool
	scopeListLocal    bool
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
Use --user, --project, or --local to filter to a specific scope.

Examples:
  claudeup scope list           # Show all scopes
  claudeup scope list --user    # Show only user scope
  claudeup scope list --project # Show only project scope`,
	Args: cobra.NoArgs,
	RunE: runScopeList,
}

var scopeClearCmd = &cobra.Command{
	Use:   "clear <user|project|local>",
	Short: "Clear settings at a specific scope",
	Long: `Remove settings at the specified scope level.

This is a destructive operation that removes configuration files.
You will be prompted for confirmation unless --force is used.

For user scope, you must type 'yes' to confirm (extra safety).
Use --backup to save a backup before clearing.

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
  claudeup scope clear user              # Clear with typed confirmation
  claudeup scope clear user --backup     # Clear with backup
  claudeup scope clear project --force   # Clear without confirmation
  claudeup scope restore user            # Restore from backup`,
	Args: cobra.ExactArgs(1),
	RunE: runScopeClear,
}

var scopeRestoreCmd = &cobra.Command{
	Use:   "restore <user|local>",
	Short: "Restore settings from a backup",
	Long: `Restore settings for a scope from the most recent backup.

Backups are created when using 'scope clear' or 'profile apply --replace'.
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
	scopeListCmd.Flags().BoolVar(&scopeListUser, "user", false, "Show only user scope")
	scopeListCmd.Flags().BoolVar(&scopeListProject, "project", false, "Show only project scope")
	scopeListCmd.Flags().BoolVar(&scopeListLocal, "local", false, "Show only local scope")
	scopeClearCmd.Flags().BoolVar(&scopeClearForce, "force", false, "Skip confirmation prompts")
	scopeClearCmd.Flags().BoolVar(&scopeClearBackup, "backup", false, "Create backup before clearing")
	scopeRestoreCmd.Flags().BoolVar(&scopeRestoreForce, "force", false, "Skip confirmation prompts")
}

func runScopeList(cmd *cobra.Command, args []string) error {
	// Resolve scope from --scope or boolean aliases
	resolvedScope, err := resolveScopeFlags(scopeListScope, scopeListUser, scopeListProject, scopeListLocal)
	if err != nil {
		return err
	}
	scopeListScope = resolvedScope

	// Get current directory for project/local scopes
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	return RenderPluginsByScope(claudeDir, projectDir, scopeListScope)
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
		if scope == "user" {
			// User scope requires typing "yes" for safety
			fmt.Print(ui.Warning("Type 'yes' to clear user scope: "))
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			if !ui.ValidateTypedConfirmation(input, "yes") {
				fmt.Println("Cancelled.")
				return nil
			}
		} else {
			confirmed := ui.PromptYesNo(fmt.Sprintf("Clear %s scope settings?", scope), false)
			if !confirmed {
				fmt.Println("Cancelled.")
				return nil
			}
		}
	}

	// Create backup if requested
	if scopeClearBackup {
		claudeupHome := config.MustClaudeupHome()

		var backupPath string
		if scope == "local" {
			backupPath, err = backup.SaveLocalScopeBackup(claudeupHome, projectDir, settingsPath)
		} else {
			backupPath, err = backup.SaveScopeBackup(claudeupHome, scope, settingsPath)
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

	claudeupHome := config.MustClaudeupHome()

	// Get current directory for local scope
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get backup info (local scope uses project-specific naming)
	var backupInfo *backup.BackupInfo
	if scope == "local" {
		backupInfo, err = backup.GetLocalBackupInfo(claudeupHome, projectDir)
	} else {
		backupInfo, err = backup.GetBackupInfo(claudeupHome, scope)
	}
	if err != nil {
		return fmt.Errorf("failed to check backup: %w", err)
	}

	if !backupInfo.Exists {
		return fmt.Errorf("no backup found for %s scope\nHint: Backups are created when using 'scope clear' or 'profile apply --replace'", scope)
	}

	// Get settings path
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

	// Restore (local scope uses project-specific naming)
	if scope == "local" {
		err = backup.RestoreLocalScopeBackup(claudeupHome, projectDir, settingsPath)
	} else {
		err = backup.RestoreScopeBackup(claudeupHome, scope, settingsPath)
	}
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Restored %s scope from backup", scope))

	return nil
}
