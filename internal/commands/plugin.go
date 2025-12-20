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
	pluginListSummary bool
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
}

func runPluginList(cmd *cobra.Command, args []string) error {
	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Load settings
	settings, err := claude.LoadSettings(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Get all plugins (user-scoped)
	allPlugins := plugins.GetAllPlugins()

	// Sort plugin names for consistent output
	names := make([]string, 0, len(allPlugins))
	for name := range allPlugins {
		names = append(names, name)
	}
	sort.Strings(names)

	// Calculate statistics
	cachedCount := 0
	localCount := 0
	enabledCount := 0
	disabledCount := 0
	staleCount := 0

	for name, plugin := range allPlugins {
		if plugin.IsLocal {
			localCount++
		} else {
			cachedCount++
		}

		if !plugin.PathExists() {
			staleCount++
		} else if settings.IsPluginEnabled(name) {
			enabledCount++
		} else {
			disabledCount++
		}
	}

	// If summary only, just show stats
	if pluginListSummary {
		fmt.Println(ui.RenderHeader("Plugin Summary"))
		fmt.Println()
		fmt.Println(ui.RenderDetail("Total", fmt.Sprintf("%d plugins", len(names))))
		fmt.Println(ui.RenderDetail("Enabled", fmt.Sprintf("%d", enabledCount)))
		if disabledCount > 0 {
			fmt.Println(ui.RenderDetail("Disabled", fmt.Sprintf("%d", disabledCount)))
		}
		if staleCount > 0 {
			fmt.Println(ui.RenderDetail("Stale", fmt.Sprintf("%d", staleCount)))
		}
		fmt.Println()
		fmt.Println(ui.Bold("By Type:"))
		fmt.Println(ui.Indent(fmt.Sprintf("Cached: %d %s", cachedCount, ui.Muted("(copied to ~/.claude/plugins/cache/)")), 1))
		fmt.Println(ui.Indent(fmt.Sprintf("Local:  %d %s", localCount, ui.Muted("(referenced from marketplace)")), 1))
		return nil
	}

	// Print header
	fmt.Println(ui.RenderSection("Installed Plugins", len(names)))
	fmt.Println()

	// Print each plugin
	for _, name := range names {
		plugin := allPlugins[name]
		var statusSymbol, statusText string

		if !plugin.PathExists() {
			statusSymbol = ui.Error(ui.SymbolError)
			statusText = ui.Error("stale (path not found)")
		} else if settings.IsPluginEnabled(name) {
			statusSymbol = ui.Success(ui.SymbolSuccess)
			statusText = "enabled"
		} else {
			statusSymbol = ui.Error(ui.SymbolError)
			statusText = "disabled"
		}

		fmt.Printf("%s %s\n", statusSymbol, ui.Bold(name))
		fmt.Println(ui.Indent(ui.RenderDetail("Version", plugin.Version), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Status", statusText), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Path", plugin.InstallPath), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Installed", plugin.InstalledAt), 1))

		pluginType := "cached"
		if plugin.IsLocal {
			pluginType = "local"
		}
		fmt.Println(ui.Indent(ui.RenderDetail("Type", pluginType), 1))
		fmt.Println()
	}

	// Print summary at the end
	fmt.Println(ui.RenderSection("Summary", -1))
	fmt.Printf("Total: %d plugins (%d cached, %d local)\n", len(names), cachedCount, localCount)
	if staleCount > 0 {
		ui.PrintWarning(fmt.Sprintf("%d stale plugins detected", staleCount))
	}

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	claudeJSONPath := filepath.Join(filepath.Dir(claudeDir), ".claude.json")
	snapshot, err := profile.Snapshot(profileName, claudeDir, claudeJSONPath)
	if err != nil {
		return err
	}

	profilesDir := filepath.Join(homeDir, ".claudeup", "profiles")
	return profile.Save(profilesDir, snapshot)
}
