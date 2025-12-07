// ABOUTME: Plugins command implementation for listing and managing plugins
// ABOUTME: Shows detailed information about installed Claude Code plugins
package commands

import (
	"fmt"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
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

	// Sort plugin names for consistent output
	names := make([]string, 0, len(plugins.Plugins))
	for name := range plugins.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	// Calculate statistics
	cachedCount := 0
	localCount := 0
	enabledCount := 0
	staleCount := 0

	for _, plugin := range plugins.Plugins {
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
		fmt.Println("=== Plugin Summary ===")
		fmt.Printf("\nTotal:   %d plugins\n", len(names))
		fmt.Printf("Enabled: %d\n", enabledCount)
		if staleCount > 0 {
			fmt.Printf("Stale:   %d\n", staleCount)
		}
		fmt.Printf("\nBy Type:\n")
		fmt.Printf("  Cached: %d (copied to ~/.claude/plugins/cache/)\n", cachedCount)
		fmt.Printf("  Local:  %d (referenced from marketplace)\n", localCount)
		return nil
	}

	// Print header
	fmt.Printf("=== Installed Plugins (%d) ===\n\n", len(names))

	// Print each plugin
	for _, name := range names {
		plugin := plugins.Plugins[name]
		status := "✓"
		statusText := "enabled"

		if !plugin.PathExists() {
			status = "✗"
			statusText = "stale (path not found)"
		}

		fmt.Printf("%s %s\n", status, name)
		fmt.Printf("   Version:    %s\n", plugin.Version)
		fmt.Printf("   Status:     %s\n", statusText)
		fmt.Printf("   Path:       %s\n", plugin.InstallPath)
		fmt.Printf("   Installed:  %s\n", plugin.InstalledAt)
		if plugin.IsLocal {
			fmt.Printf("   Type:       local\n")
		} else {
			fmt.Printf("   Type:       cached\n")
		}
		fmt.Println()
	}

	// Print summary at the end
	fmt.Println("━━━ Summary ━━━")
	fmt.Printf("Total: %d plugins (%d cached, %d local)\n", len(names), cachedCount, localCount)
	if staleCount > 0 {
		fmt.Printf("⚠ %d stale plugins detected\n", staleCount)
	}

	return nil
}
