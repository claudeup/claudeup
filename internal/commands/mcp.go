// ABOUTME: MCP command implementation for managing MCP servers
// ABOUTME: Lists and shows information about MCP servers provided by plugins
package commands

import (
	"fmt"
	"sort"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/mcp"
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

The server reference must be in the format: plugin-name:server-name

Example:
  claudeup mcp disable compound-engineering@every-marketplace:playwright
  claudeup mcp disable superpowers-chrome@superpowers-marketplace:chrome`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPDisable,
}

var mcpEnableCmd = &cobra.Command{
	Use:   "enable <plugin>:<server>",
	Short: "Enable a previously disabled MCP server",
	Long: `Enable a specific MCP server that was previously disabled.

The server reference must be in the format: plugin-name:server-name

Example:
  claudeup mcp enable compound-engineering@every-marketplace:playwright
  claudeup mcp enable superpowers-chrome@superpowers-marketplace:chrome`,
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

	// Print header
	fmt.Println("=== MCP Servers by Plugin ===")

	// Count total servers
	totalServers := 0
	for _, pluginServers := range mcpServers {
		totalServers += len(pluginServers.Servers)
	}

	// Print each plugin's MCP servers
	for _, pluginServers := range mcpServers {
		fmt.Printf("✓ %s\n", pluginServers.PluginName)

		// Sort server names
		serverNames := make([]string, 0, len(pluginServers.Servers))
		for name := range pluginServers.Servers {
			serverNames = append(serverNames, name)
		}
		sort.Strings(serverNames)

		// Print each server
		for _, serverName := range serverNames {
			server := pluginServers.Servers[serverName]
			fmt.Printf("   ✓ %s\n", serverName)
			fmt.Printf("      Command: %s\n", server.Command)
			if len(server.Args) > 0 {
				fmt.Printf("      Args:    %v\n", server.Args)
			}
			if len(server.Env) > 0 {
				fmt.Printf("      Env:     %d variables\n", len(server.Env))
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
		fmt.Printf("✓ MCP server %s is already disabled\n", serverRef)
		return nil
	}

	// Disable the MCP server
	cfg.DisableMCPServer(serverRef)

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Disabled MCP server %s\n\n", serverRef)
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
		fmt.Printf("✓ MCP server %s is already enabled\n", serverRef)
		return nil
	}

	// Enable the MCP server
	cfg.EnableMCPServer(serverRef)

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Enabled MCP server %s\n\n", serverRef)
	fmt.Println("This MCP server will now be loaded")
	fmt.Printf("Run 'claudeup mcp disable %s' to disable again\n", serverRef)
	fmt.Println("\nNote: You may need to restart Claude Code for changes to take effect")

	return nil
}
