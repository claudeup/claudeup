// ABOUTME: Enable command implementation for plugins and MCP servers
// ABOUTME: Restores plugins to installed_plugins.json from saved metadata
package commands

import (
	"fmt"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable <plugin-name>",
	Short: "Enable a previously disabled plugin",
	Long: `Enable a plugin by restoring it to the installed plugins registry.

This only works for plugins that were disabled using 'claudeup disable'.
If the plugin was never installed, you'll need to install it first using the claude CLI.

Example:
  claudeup enable hookify@claude-code-plugins
  claudeup enable compound-engineering`,
	Args: cobra.ExactArgs(1),
	RunE: runEnable,
}

func init() {
	rootCmd.AddCommand(enableCmd)
}

func runEnable(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if plugin is disabled
	disabledMeta, exists := cfg.EnablePlugin(pluginName)
	if !exists {
		return fmt.Errorf("plugin %s is not disabled (or was never installed via claudeup)", pluginName)
	}

	// Load plugins registry
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Check if already enabled
	if plugins.PluginExists(pluginName) {
		return fmt.Errorf("plugin %s is already enabled", pluginName)
	}

	// Restore plugin to registry
	pluginMeta := claude.PluginMetadata{
		Version:      disabledMeta.Version,
		InstalledAt:  disabledMeta.InstalledAt,
		LastUpdated:  disabledMeta.LastUpdated,
		InstallPath:  disabledMeta.InstallPath,
		GitCommitSha: disabledMeta.GitCommitSha,
		IsLocal:      disabledMeta.IsLocal,
	}
	plugins.EnablePlugin(pluginName, pluginMeta)

	// Save both config and plugins registry
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if err := claude.SavePlugins(claudeDir, plugins); err != nil {
		return fmt.Errorf("failed to save plugins: %w", err)
	}

	fmt.Printf("✓ Enabled %s\n\n", pluginName)
	fmt.Println("Plugin commands, agents, skills, and MCP servers are now available")
	fmt.Println("Run 'claudeup disable", pluginName+"' to disable again")

	return nil
}
