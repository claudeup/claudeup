// ABOUTME: Status command implementation showing overview of Claude installation
// ABOUTME: Displays marketplaces, plugins, and detected issues
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/config"
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/internal/ui"
	"github.com/spf13/cobra"
)

var (
	statusScope   string
	statusUser    bool
	statusProject bool
	statusLocal   bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show an overview of Claude Code installation",
	Long: `Display the current state of your Claude Code installation.

Shows:
  - Active profile
  - Installed marketplaces
  - Plugin counts and status
  - Any detected issues

For detailed plugin information, use 'claudeup plugins'.
For diagnostics, use 'claudeup doctor'.`,
	Args: cobra.NoArgs,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusScope, "scope", "", "Filter to scope: user, project, or local (default: show all)")
	statusCmd.Flags().BoolVar(&statusUser, "user", false, "Show only user scope")
	statusCmd.Flags().BoolVar(&statusProject, "project", false, "Show only project scope")
	statusCmd.Flags().BoolVar(&statusLocal, "local", false, "Show only local scope")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Resolve scope from --scope or boolean aliases
	resolvedScope, err := resolveScopeFlags(statusScope, statusUser, statusProject, statusLocal)
	if err != nil {
		return err
	}
	statusScope = resolvedScope

	// Get current directory for scope-aware settings
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load marketplaces
	marketplaces, err := claude.LoadMarketplaces(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load marketplaces: %w", err)
	}

	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Validate scope if specified
	if statusScope != "" {
		if err := claude.ValidateScope(statusScope); err != nil {
			return err
		}
	}

	// Print header
	fmt.Println(ui.RenderSection("claudeup Status", -1))

	// Determine active profile using scope hierarchy: project > local > user
	activeProfile, profileScope := getActiveProfile(projectDir)
	if activeProfile == "" {
		activeProfile = "none"
	}

	fmt.Println()
	if profileScope != "" {
		fmt.Println(ui.RenderDetail("Active Profile", fmt.Sprintf("%s %s", ui.Bold(activeProfile), ui.Muted(fmt.Sprintf("(%s scope)", profileScope)))))
	} else {
		fmt.Println(ui.RenderDetail("Active Profile", ui.Bold(activeProfile)))
	}

	// Check if active profile exists
	if activeProfile != "none" && activeProfile != "" {
		profilesDir := filepath.Join(config.MustClaudeupHome(), "profiles")

		// Check if profile file exists (both disk and embedded)
		_, diskErr := profile.Load(profilesDir, activeProfile)
		_, embeddedErr := profile.GetEmbeddedProfile(activeProfile)

		if diskErr != nil && embeddedErr != nil {
			// Profile doesn't exist anywhere - show warning
			ui.PrintWarning(fmt.Sprintf("Active profile '%s' not found.", activeProfile))
		}
	}

	// Print marketplaces
	fmt.Println()
	fmt.Println(ui.RenderSection("Marketplaces", len(marketplaces)))
	for name := range marketplaces {
		fmt.Printf("  %s %s\n", ui.Success(ui.SymbolSuccess), name)
	}

	// Load settings from each scope to determine where plugins are enabled
	var scopes []string
	if statusScope != "" {
		// Only check specified scope when --scope flag is used
		scopes = []string{statusScope}
	} else {
		// Check all scopes in precedence order
		scopes = []string{"local", "project", "user"}
	}

	scopeSettings := make(map[string]*claude.Settings)
	for _, scope := range scopes {
		scopeSettings[scope], _ = claude.LoadSettingsForScope(scope, claudeDir, projectDir)
	}

	// Build map of plugin -> scope (highest precedence wins)
	pluginScopes := make(map[string]string)
	for _, sp := range plugins.GetPluginsAtScopes(scopes) {
		name := sp.Name
		// Check scopes in precedence order (local > project > user)
		for _, scope := range scopes {
			if scopeSettings[scope] != nil && scopeSettings[scope].IsPluginEnabled(name) {
				pluginScopes[name] = scope
				break // Found at highest precedence scope
			}
		}
	}

	// Count enabled plugins and detect issues
	enabledCount := 0
	stalePlugins := []string{}   // Installed but path missing
	missingPlugins := []string{} // Enabled in settings but not installed

	// First, collect all plugins enabled in settings (across all scopes)
	enabledInSettings := make(map[string]bool)
	for _, scope := range scopes {
		if scopeSettings[scope] != nil {
			for name, enabled := range scopeSettings[scope].EnabledPlugins {
				if enabled {
					enabledInSettings[name] = true
				}
			}
		}
	}

	// Check installed plugins for issues (deduplicate by name to avoid
	// over-counting when a plugin is installed at multiple scopes)
	seen := make(map[string]bool)
	for _, sp := range plugins.GetPluginsAtScopes(scopes) {
		name := sp.Name
		if seen[name] {
			continue
		}
		seen[name] = true
		plugin := sp.PluginMetadata
		// Check if plugin is enabled in any scope
		if _, enabled := pluginScopes[name]; enabled {
			enabledCount++
			// Also check if enabled plugin has stale path
			if !plugin.PathExists() {
				stalePlugins = append(stalePlugins, name)
			}
		}
	}

	// Find plugins enabled in settings but not installed
	for name := range enabledInSettings {
		if !plugins.PluginExistsAtAnyScope(name) {
			missingPlugins = append(missingPlugins, name)
		}
	}

	// Sort for consistent output
	sort.Strings(stalePlugins)
	sort.Strings(missingPlugins)

	// Print plugins summary with scope information
	fmt.Println()
	fmt.Println(ui.RenderSection("Plugins", enabledCount))
	if enabledCount > 0 {
		// Sort plugin names for consistent output
		pluginNames := make([]string, 0, len(pluginScopes))
		for name := range pluginScopes {
			pluginNames = append(pluginNames, name)
		}
		sort.Strings(pluginNames)

		for _, name := range pluginNames {
			scope := pluginScopes[name]
			fmt.Printf("  %s %s %s\n", ui.Success(ui.SymbolSuccess), name, ui.Muted(fmt.Sprintf("(%s)", scope)))
		}
	}
	// Only show stale plugins if there are any
	if len(stalePlugins) > 0 {
		fmt.Printf("  %s %d stale\n", ui.Warning(ui.SymbolWarning), len(stalePlugins))
	}

	// Print issues if any
	hasIssues := len(stalePlugins) > 0 || len(missingPlugins) > 0
	if hasIssues {
		fmt.Println()
		fmt.Println(ui.RenderSection("Issues", -1))

		if len(missingPlugins) > 0 {
			// Check which missing plugins are in the saved profile
			pluginsInProfile := make(map[string]bool)
			if activeProfile != "" && activeProfile != "none" {
				profilesDir := getProfilesDir()
				savedProfile, err := loadProfileWithFallback(profilesDir, activeProfile)
				if err == nil {
					for _, p := range savedProfile.Plugins {
						pluginsInProfile[p] = true
					}
				}
			}

			fmt.Println()
			fmt.Printf("  %s %d plugin%s enabled but not installed:\n",
				ui.Warning(ui.SymbolWarning), len(missingPlugins), pluralS(len(missingPlugins)))
			for _, name := range missingPlugins {
				suffix := ""
				if pluginsInProfile[name] {
					suffix = ui.Muted(" (in profile)")
				}
				fmt.Printf("    - %s%s\n", name, suffix)
			}
			fmt.Println()
			if activeProfile != "" && activeProfile != "none" {
				fmt.Printf("  %s Reinstall from profile: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold(fmt.Sprintf("claudeup profile apply %s --reinstall", activeProfile)))
				fmt.Printf("  %s Or remove from settings: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold("claudeup profile clean <plugin-name>"))
			} else {
				fmt.Printf("  %s Install manually: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold("claude plugin install <name>"))
				fmt.Printf("  %s Or remove from settings: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold("claudeup profile clean <plugin-name>"))
			}
		}

		if len(stalePlugins) > 0 {
			fmt.Println()
			fmt.Printf("  %s %d plugin%s installed but path missing:\n",
				ui.Warning(ui.SymbolWarning), len(stalePlugins), pluralS(len(stalePlugins)))
			for _, name := range stalePlugins {
				fmt.Printf("    - %s\n", name)
			}
			fmt.Println()
			if activeProfile != "" && activeProfile != "none" {
				fmt.Printf("  %s Reinstall from profile: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold(fmt.Sprintf("claudeup profile apply %s --reinstall", activeProfile)))
			} else {
				fmt.Printf("  %s Reinstall manually: %s\n",
					ui.Muted(ui.SymbolArrow), ui.Bold("claude plugin install <name> --reinstall"))
			}
			fmt.Printf("  %s Run %s for full diagnostics\n",
				ui.Muted(ui.SymbolArrow), ui.Bold("claudeup doctor"))
		}

	}

	return nil
}
