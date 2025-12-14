// ABOUTME: Disable command implementation for plugins and MCP servers
// ABOUTME: Removes plugins from installed_plugins.json or tracks disabled MCP servers
package commands

import (
	"fmt"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/ui"
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
		ui.PrintSuccess(fmt.Sprintf("Plugin %s is already disabled", pluginName))
		return nil
	}

	// Load plugins registry
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Check if plugin exists
	pluginMeta, exists := plugins.GetPlugin(pluginName)
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

	ui.PrintSuccess(fmt.Sprintf("Disabled %s", pluginName))
	fmt.Println()
	ui.PrintInfo("Plugin commands, agents, skills, and MCP servers are now unavailable")
	fmt.Printf("%s Run 'claudeup enable %s' to re-enable\n", ui.Muted(ui.SymbolArrow), pluginName)

	return nil
}
