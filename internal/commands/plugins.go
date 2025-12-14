// ABOUTME: Plugins command implementation for listing and managing plugins
// ABOUTME: Shows detailed information about installed Claude Code plugins
package commands

import (
	"fmt"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	pluginsSummary bool
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "List installed plugins",
	Long:  `Display detailed information about all installed plugins.`,
	RunE:  runPluginsList,
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.Flags().BoolVar(&pluginsSummary, "summary", false, "Show only summary statistics")
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
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
	staleCount := 0

	for _, plugin := range allPlugins {
		if plugin.IsLocal {
			localCount++
		} else {
			cachedCount++
		}

		if plugin.PathExists() {
			enabledCount++
		} else {
			staleCount++
		}
	}

	// If summary only, just show stats
	if pluginsSummary {
		fmt.Println(ui.RenderHeader("Plugin Summary"))
		fmt.Println()
		fmt.Println(ui.RenderDetail("Total", fmt.Sprintf("%d plugins", len(names))))
		fmt.Println(ui.RenderDetail("Enabled", fmt.Sprintf("%d", enabledCount)))
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

		if plugin.PathExists() {
			statusSymbol = ui.Success(ui.SymbolSuccess)
			statusText = "enabled"
		} else {
			statusSymbol = ui.Error(ui.SymbolError)
			statusText = ui.Error("stale (path not found)")
		}

		fmt.Printf("%s %s\n", statusSymbol, ui.Bold(name))
		fmt.Println(ui.Indent(ui.RenderDetail("Version", plugin.Version), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Status", statusText), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Path", ui.Muted(plugin.InstallPath)), 1))
		fmt.Println(ui.Indent(ui.RenderDetail("Installed", ui.Muted(plugin.InstalledAt)), 1))

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
