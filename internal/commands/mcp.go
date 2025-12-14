// ABOUTME: MCP command implementation for managing MCP servers
// ABOUTME: Lists and shows information about MCP servers provided by plugins
package commands

import (
	"fmt"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/mcp"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long:  `List and manage MCP servers provided by Claude Code plugins.`,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MCP servers",
	Long:  `Display MCP servers grouped by the plugin that provides them.`,
	RunE:  runMCPList,
}

var mcpDisableCmd = &cobra.Command{
	Use:   "disable <plugin>:<server>",
	Short: "Disable a specific MCP server",
	Long: `Disable a specific MCP server without disabling the entire plugin.

The server reference must be in the format: plugin-name:server-name`,
	Example: `  claudeup mcp disable my-plugin@acme-marketplace:database
  claudeup mcp disable tools@example-marketplace:browser`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPDisable,
}

var mcpEnableCmd = &cobra.Command{
	Use:   "enable <plugin>:<server>",
	Short: "Enable a previously disabled MCP server",
	Long: `Enable a specific MCP server that was previously disabled.

The server reference must be in the format: plugin-name:server-name`,
	Example: `  claudeup mcp enable my-plugin@acme-marketplace:database
  claudeup mcp enable tools@example-marketplace:browser`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPEnable,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpDisableCmd)
	mcpCmd.AddCommand(mcpEnableCmd)
}

func runMCPList(cmd *cobra.Command, args []string) error {
	// Load plugins
	plugins, err := claude.LoadPlugins(claudeDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	// Discover MCP servers
	mcpServers, err := mcp.DiscoverMCPServers(plugins)
	if err != nil {
		return fmt.Errorf("failed to discover MCP servers: %w", err)
	}

	if len(mcpServers) == 0 {
		fmt.Println("No MCP servers found in installed plugins.")
		return nil
	}

	// Sort by plugin name for consistent output
	sort.Slice(mcpServers, func(i, j int) bool {
		return mcpServers[i].PluginName < mcpServers[j].PluginName
	})

	// Count total servers
	totalServers := 0
	for _, pluginServers := range mcpServers {
		totalServers += len(pluginServers.Servers)
	}

	// Print header
	fmt.Println(ui.RenderSection("MCP Servers", totalServers))
	fmt.Println()

	// Print each plugin's MCP servers
	for _, pluginServers := range mcpServers {
		fmt.Printf("%s %s\n", ui.Success(ui.SymbolSuccess), ui.Bold(pluginServers.PluginName))

		// Sort server names
		serverNames := make([]string, 0, len(pluginServers.Servers))
		for name := range pluginServers.Servers {
			serverNames = append(serverNames, name)
		}
		sort.Strings(serverNames)

		// Print each server
		for _, serverName := range serverNames {
			server := pluginServers.Servers[serverName]
			fmt.Println(ui.Indent(fmt.Sprintf("%s %s", ui.Success(ui.SymbolSuccess), ui.Bold(serverName)), 1))
			fmt.Println(ui.Indent(ui.RenderDetail("Command", server.Command), 2))
			if len(server.Args) > 0 {
				fmt.Println(ui.Indent(ui.RenderDetail("Args", fmt.Sprintf("%v", server.Args)), 2))
			}
			if len(server.Env) > 0 {
				fmt.Println(ui.Indent(ui.RenderDetail("Env", fmt.Sprintf("%d variables", len(server.Env))), 2))
			}
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d MCP servers from %d plugins\n", totalServers, len(mcpServers))

	return nil
}

func runMCPDisable(cmd *cobra.Command, args []string) error {
	serverRef := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if already disabled
	if cfg.IsMCPServerDisabled(serverRef) {
		ui.PrintSuccess(fmt.Sprintf("MCP server %s is already disabled", serverRef))
		return nil
	}

	// Disable the MCP server
	cfg.DisableMCPServer(serverRef)

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Disabled MCP server %s", serverRef))
	fmt.Println()
	fmt.Println("This MCP server will no longer be loaded")
	fmt.Printf("Run 'claudeup mcp enable %s' to re-enable\n", serverRef)
	fmt.Println("\nNote: You may need to restart Claude Code for changes to take effect")

	return nil
}

func runMCPEnable(cmd *cobra.Command, args []string) error {
	serverRef := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if it's disabled
	if !cfg.IsMCPServerDisabled(serverRef) {
		ui.PrintSuccess(fmt.Sprintf("MCP server %s is already enabled", serverRef))
		return nil
	}

	// Enable the MCP server
	cfg.EnableMCPServer(serverRef)

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Enabled MCP server %s", serverRef))
	fmt.Println()
	fmt.Println("This MCP server will now be loaded")
	fmt.Printf("Run 'claudeup mcp disable %s' to disable again\n", serverRef)
	fmt.Println("\nNote: You may need to restart Claude Code for changes to take effect")

	return nil
}
