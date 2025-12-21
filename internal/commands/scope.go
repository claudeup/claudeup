// ABOUTME: Scope command for managing and viewing Claude Code settings across scopes
// ABOUTME: Provides list and clear subcommands for scope-level configuration management
package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	scopeListScope string
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

func init() {
	rootCmd.AddCommand(scopeCmd)
	scopeCmd.AddCommand(scopeListCmd)
	scopeListCmd.Flags().StringVar(&scopeListScope, "scope", "", "Show only specified scope (user, project, or local)")
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
