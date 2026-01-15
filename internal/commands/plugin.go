// ABOUTME: Plugin subcommand group for managing Claude Code plugins
// ABOUTME: Provides list, add, disable, enable, and remove subcommands
package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/claudeup/claudeup/v2/internal/claude"
	"github.com/claudeup/claudeup/v2/internal/config"
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/internal/ui"
	"github.com/spf13/cobra"
)

var (
	pluginListSummary    bool
	pluginFilterEnabled  bool
	pluginFilterDisabled bool
	pluginListFormat     string
	pluginListByScope    bool
	pluginBrowseFormat   string
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

var pluginBrowseCmd = &cobra.Command{
	Use:   "browse <marketplace>",
	Short: "Browse available plugins in a marketplace",
	Long: `Display plugins available in a marketplace before installing.

Accepts marketplace name, repo (user/repo), or URL as identifier.`,
	Example: `  claudeup plugin browse claude-code-workflows
  claudeup plugin browse wshobson/agents
  claudeup plugin browse --format table my-marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginBrowse,
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginBrowseCmd)

	pluginListCmd.Flags().BoolVar(&pluginListSummary, "summary", false, "Show only summary statistics")
	pluginListCmd.Flags().BoolVar(&pluginFilterEnabled, "enabled", false, "Show only enabled plugins")
	pluginListCmd.Flags().BoolVar(&pluginFilterDisabled, "disabled", false, "Show only disabled plugins")
	pluginListCmd.Flags().StringVar(&pluginListFormat, "format", "", "Output format (table)")
	pluginListCmd.Flags().BoolVar(&pluginListByScope, "by-scope", false, "Group enabled plugins by scope")
	pluginBrowseCmd.Flags().StringVar(&pluginBrowseFormat, "format", "", "Output format (table)")
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

	// Handle --by-scope: show plugins grouped by scope
	if pluginListByScope {
		return RenderPluginsByScope(claudeDir, projectDir, "")
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

func runPluginBrowse(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Find the marketplace
	meta, marketplaceName, err := claude.FindMarketplace(claudeDir, identifier)
	if err != nil {
		// Build helpful error message
		registry, loadErr := claude.LoadMarketplaces(claudeDir)
		if loadErr != nil {
			return fmt.Errorf("marketplace %q not found\n\nTo add a marketplace:\n  claude marketplace add <repo-or-url>", identifier)
		}

		var installed []string
		for name, m := range registry {
			if m.Source.Repo != "" {
				installed = append(installed, fmt.Sprintf("  %s (%s)", name, m.Source.Repo))
			} else {
				installed = append(installed, fmt.Sprintf("  %s", name))
			}
		}
		sort.Strings(installed)

		msg := fmt.Sprintf("marketplace %q not found\n\nTo add a marketplace:\n  claude marketplace add <repo-or-url>", identifier)
		if len(installed) > 0 {
			msg += "\n\nInstalled marketplaces:\n" + strings.Join(installed, "\n")
		}
		return errors.New(msg)
	}

	// Load the marketplace index
	index, err := claude.LoadMarketplaceIndex(meta.InstallLocation)
	if err != nil {
		return fmt.Errorf("marketplace %q has no plugin index\n\nThe marketplace at %s is missing .claude-plugin/marketplace.json", marketplaceName, meta.InstallLocation)
	}

	// Handle empty marketplace
	if len(index.Plugins) == 0 {
		fmt.Printf("No plugins available in %s\n", index.Name)
		return nil
	}

	// Load installed plugins to check status (non-fatal if unavailable)
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		// Can still show marketplace plugins, just without installation status
		plugins = nil
	}

	// Sort plugins alphabetically
	sortedPlugins := make([]claude.MarketplacePluginInfo, len(index.Plugins))
	copy(sortedPlugins, index.Plugins)
	sort.Slice(sortedPlugins, func(i, j int) bool {
		return sortedPlugins[i].Name < sortedPlugins[j].Name
	})

	// Display based on format
	switch pluginBrowseFormat {
	case "json":
		printBrowseJSON(sortedPlugins, index.Name, marketplaceName, plugins)
	case "table":
		printBrowseTable(sortedPlugins, index.Name, marketplaceName, plugins)
	default:
		printBrowseDefault(sortedPlugins, index.Name, marketplaceName, plugins)
	}

	return nil
}

func printBrowseDefault(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	fmt.Println(ui.RenderSection("Available in "+indexName, len(plugins)))
	fmt.Println()

	// Calculate max name width for alignment
	nameWidth := 20
	for _, p := range plugins {
		if len(p.Name) > nameWidth {
			nameWidth = len(p.Name)
		}
	}
	nameWidth += 2

	for _, p := range plugins {
		// Check if installed
		fullName := p.Name + "@" + marketplaceName
		var status string
		if installed != nil && installed.PluginExists(fullName) {
			status = ui.Success(ui.SymbolSuccess)
		}

		// Truncate description if needed (rune-safe for UTF-8)
		desc := p.Description
		descRunes := []rune(desc)
		if len(descRunes) > 60 {
			desc = string(descRunes[:57]) + "..."
		}

		// Format with styling - fixed width columns
		nameFmt := fmt.Sprintf("%%-%ds", nameWidth)
		nameCol := fmt.Sprintf(nameFmt, p.Name)
		descCol := fmt.Sprintf("%-60s", desc)
		versionCol := fmt.Sprintf("%-8s", p.Version)

		fmt.Printf("%s %s  %s %s\n",
			ui.Bold(nameCol),
			ui.Muted(descCol),
			ui.Muted(versionCol),
			status)
	}
}

func printBrowseTable(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	// Calculate max name width for alignment
	nameWidth := 6 // minimum "PLUGIN" length
	for _, p := range plugins {
		if len(p.Name) > nameWidth {
			nameWidth = len(p.Name)
		}
	}
	nameWidth += 2 // add padding

	descWidth := 60

	// Print header with bold styling
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-10s %%s", nameWidth, descWidth)
	header := fmt.Sprintf(headerFmt, "PLUGIN", "DESCRIPTION", "VERSION", "STATUS")
	fmt.Println(ui.Bold(header))
	fmt.Println(ui.Muted(strings.Repeat("─", nameWidth+descWidth+10+12)))

	// Print rows
	for _, p := range plugins {
		fullName := p.Name + "@" + marketplaceName

		// Truncate description (rune-safe for UTF-8)
		desc := p.Description
		descRunes := []rune(desc)
		if len(descRunes) > descWidth {
			desc = string(descRunes[:descWidth-3]) + "..."
		}

		// Format columns with padding first (before applying ANSI styles)
		nameFmt := fmt.Sprintf("%%-%ds", nameWidth)
		nameCol := fmt.Sprintf(nameFmt, p.Name)
		descCol := fmt.Sprintf("%-*s", descWidth, desc)
		versionCol := fmt.Sprintf("%-10s", p.Version)

		// Check installed status
		var statusCol string
		if installed != nil && installed.PluginExists(fullName) {
			statusCol = ui.Success("installed")
		}

		fmt.Printf("%s %s %s %s\n",
			ui.Bold(nameCol),
			ui.Muted(descCol),
			ui.Muted(versionCol),
			statusCol)
	}
}

func printBrowseJSON(plugins []claude.MarketplacePluginInfo, indexName, marketplaceName string, installed *claude.PluginRegistry) {
	type pluginOutput struct {
		Name        string `json:"name"`
		FullName    string `json:"fullName"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Installed   bool   `json:"installed"`
	}

	output := struct {
		Marketplace string         `json:"marketplace"`
		Count       int            `json:"count"`
		Plugins     []pluginOutput `json:"plugins"`
	}{
		Marketplace: indexName,
		Count:       len(plugins),
		Plugins:     make([]pluginOutput, len(plugins)),
	}

	for i, p := range plugins {
		fullName := p.Name + "@" + marketplaceName
		output.Plugins[i] = pluginOutput{
			Name:        p.Name,
			FullName:    fullName,
			Description: p.Description,
			Version:     p.Version,
			Installed:   installed != nil && installed.PluginExists(fullName),
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}
