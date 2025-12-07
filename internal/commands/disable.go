// ABOUTME: Disable command implementation for plugins and MCP servers
// ABOUTME: Removes plugins from installed_plugins.json or tracks disabled MCP servers
package commands

import (
	"fmt"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable <plugin-name>",
	Short: "Disable a plugin",
	Long: `Disable a plugin by removing it from the installed plugins registry.

The plugin's metadata is saved so it can be re-enabled later without reinstalling.

Example:
  claudeup disable hookify@claude-code-plugins
  claudeup disable compound-engineering`,
	Args: cobra.ExactArgs(1),
	RunE: runDisable,
}

func init() {
	rootCmd.AddCommand(disableCmd)
}

func runDisable(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if already disabled
	if cfg.IsPluginDisabled(pluginName) {
		fmt.Printf("✓ Plugin %s is already disabled\n", pluginName)
		return nil
	}

	// Load plugins registry
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Check if plugin exists
	pluginMeta, exists := plugins.Plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Save plugin metadata to config
	disabledPlugin := config.DisabledPlugin{
		Version:      pluginMeta.Version,
		InstalledAt:  pluginMeta.InstalledAt,
		LastUpdated:  pluginMeta.LastUpdated,
		InstallPath:  pluginMeta.InstallPath,
		GitCommitSha: pluginMeta.GitCommitSha,
		IsLocal:      pluginMeta.IsLocal,
	}
	cfg.DisablePlugin(pluginName, disabledPlugin)

	// Remove from plugins registry
	plugins.DisablePlugin(pluginName)

	// Save both config and plugins registry
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if err := claude.SavePlugins(claudeDir, plugins); err != nil {
		return fmt.Errorf("failed to save plugins: %w", err)
	}

	fmt.Printf("✓ Disabled %s\n\n", pluginName)
	fmt.Println("Plugin commands, agents, skills, and MCP servers are now unavailable")
	fmt.Println("Run 'claudeup enable", pluginName+"' to re-enable")

	return nil
}
