// ABOUTME: Plugin subcommand group for managing Claude Code plugins
// ABOUTME: Provides list, add, disable, enable, and remove subcommands
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	pluginListSummary    bool
	pluginFilterEnabled  bool
	pluginFilterDisabled bool
	pluginListFormat     string
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long: `Manage Claude Code plugins - list, enable, or disable.

Use 'claude plugin install' and 'claude plugin uninstall' to add or remove plugins.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `Display detailed information about all installed plugins.

Shows each plugin's version, status, install path, and type (cached or local).
Use --summary for a quick overview without individual plugin details.`,
	Args: cobra.NoArgs,
	RunE: runPluginList,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <plugin-name>",
	Short: "Disable a plugin",
	Long: `Disable a plugin without removing it.

The plugin will remain installed but won't be loaded by Claude Code.`,
	Example: `  claudeup plugin disable my-plugin@acme-marketplace`,
	Args:    cobra.ExactArgs(1),
	RunE:    runPluginDisable,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <plugin-name>",
	Short: "Enable a previously disabled plugin",
	Long:  `Enable a plugin that was previously disabled.`,
	Example: `  claudeup plugin enable my-plugin@acme-marketplace`,
	Args:    cobra.ExactArgs(1),
	RunE:    runPluginEnable,
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginEnableCmd)

	pluginListCmd.Flags().BoolVar(&pluginListSummary, "summary", false, "Show only summary statistics")
	pluginListCmd.Flags().BoolVar(&pluginFilterEnabled, "enabled", false, "Show only enabled plugins")
	pluginListCmd.Flags().BoolVar(&pluginFilterDisabled, "disabled", false, "Show only disabled plugins")
	pluginListCmd.Flags().StringVar(&pluginListFormat, "format", "", "Output format (table)")
}

func runPluginList(cmd *cobra.Command, args []string) error {
	// Validate mutually exclusive flags
	if pluginFilterEnabled && pluginFilterDisabled {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}

	// Get current directory for project scope
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Analyze plugins across all scopes
	analysis, err := claude.AnalyzePluginScopes(claudeDir, projectDir)
	if err != nil {
		return fmt.Errorf("failed to analyze plugins: %w", err)
	}

	// Sort plugin names for consistent output
	names := make([]string, 0, len(analysis))
	for name := range analysis {
		names = append(names, name)
	}
	sort.Strings(names)

	// Calculate statistics (before filtering)
	stats := calculatePluginStatistics(analysis)

	// Apply filters
	totalCount := len(names)
	filterLabel := ""
	if pluginFilterEnabled {
		filtered := make([]string, 0)
		for _, name := range names {
			if analysis[name].IsEnabled() {
				filtered = append(filtered, name)
			}
		}
		names = filtered
		filterLabel = "enabled"
	} else if pluginFilterDisabled {
		filtered := make([]string, 0)
		for _, name := range names {
			if !analysis[name].IsEnabled() {
				filtered = append(filtered, name)
			}
		}
		names = filtered
		filterLabel = "disabled"
	}

	// Display based on output mode
	if pluginListSummary {
		printPluginSummary(stats)
		return nil
	}

	if pluginListFormat == "table" {
		printPluginTable(names, analysis)
		return nil
	}

	printPluginDetails(names, analysis)
	printPluginListFooterFiltered(stats, len(names), totalCount, filterLabel)

	return nil
}

func runPluginDisable(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Load plugins to verify it exists
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	if !plugins.PluginExists(pluginName) {
		return fmt.Errorf("plugin %q is not installed", pluginName)
	}

	// Load settings
	settings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Check if already disabled
	if !settings.IsPluginEnabled(pluginName) {
		ui.PrintSuccess(fmt.Sprintf("Plugin %s is already disabled", pluginName))
		return nil
	}

	// Disable the plugin
	settings.DisablePlugin(pluginName)

	// Save settings
	if err := claude.SaveSettings(claudeDir, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Disabled plugin %s", pluginName))
	fmt.Println()
	fmt.Printf("Run 'claudeup plugin enable %s' to re-enable\n", pluginName)
	fmt.Println("\nNote: You may need to restart Claude Code for changes to take effect")

	return nil
}

func runPluginEnable(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Load plugins to verify it exists
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	if !plugins.PluginExists(pluginName) {
		return fmt.Errorf("plugin %q is not installed", pluginName)
	}

	// Load settings
	settings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Check if already enabled
	if settings.IsPluginEnabled(pluginName) {
		ui.PrintSuccess(fmt.Sprintf("Plugin %s is already enabled", pluginName))
		return nil
	}

	// Enable the plugin
	settings.EnablePlugin(pluginName)

	// Save settings
	if err := claude.SaveSettings(claudeDir, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Enabled plugin %s", pluginName))
	fmt.Println()
	fmt.Printf("Run 'claudeup plugin disable %s' to disable\n", pluginName)
	fmt.Println("\nNote: You may need to restart Claude Code for changes to take effect")

	// Offer to save to profile
	cfg, err := config.Load()
	if err == nil && cfg.Preferences.ActiveProfile != "" {
		shouldSave := config.YesFlag // Auto-save with --yes
		if !config.YesFlag {
			fmt.Println()
			shouldSave = ui.PromptYesNo("Save to current profile?", true)
		}

		if shouldSave {
			if err := saveCurrentStateToProfile(cfg.Preferences.ActiveProfile); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to save profile: %v", err))
			} else {
				ui.PrintSuccess(fmt.Sprintf("Updated profile %q", cfg.Preferences.ActiveProfile))
			}
		}
	}

	return nil
}

// saveCurrentStateToProfile saves the current Claude state to a profile
func saveCurrentStateToProfile(profileName string) error {
	claudeJSONPath := filepath.Join(filepath.Dir(claudeDir), ".claude.json")
	snapshot, err := profile.Snapshot(profileName, claudeDir, claudeJSONPath)
	if err != nil {
		return err
	}

	profilesDir := filepath.Join(config.MustClaudeupHome(), "profiles")
	return profile.Save(profilesDir, snapshot)
}

// formatScopeList formats a list of scopes as a comma-separated string
func formatScopeList(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	if len(scopes) == 1 {
		return scopes[0]
	}

	// Join with commas
	result := ""
	for i, scope := range scopes {
		if i > 0 {
			result += ", "
		}
		result += scope
	}
	return result
}
