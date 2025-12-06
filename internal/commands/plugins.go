// ABOUTME: Plugins command implementation for listing and managing plugins
// ABOUTME: Shows detailed information about installed Claude Code plugins
package commands

import (
	"fmt"
	"sort"

	"github.com/malston/claude-pm/internal/claude"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "List installed plugins",
	Long:  `Display detailed information about all installed plugins.`,
	RunE:  runPluginsList,
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
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

	return nil
}
