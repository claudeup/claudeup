// ABOUTME: MCP server discovery functionality
// ABOUTME: Scans plugins for mcp.json files and parses server definitions
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/claude"
)

// ServerDefinition represents an MCP server configuration
type ServerDefinition struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// PluginJSON represents the plugin.json file structure
type PluginJSON struct {
	Name       string                      `json:"name"`
	Version    string                      `json:"version"`
	MCPServers map[string]ServerDefinition `json:"mcpServers"`
}

// PluginMCPServers represents MCP servers provided by a plugin
type PluginMCPServers struct {
	PluginName string
	PluginPath string
	Servers    map[string]ServerDefinition
}

// DiscoverMCPServers scans all plugins and discovers their MCP servers
func DiscoverMCPServers(pluginRegistry *claude.PluginRegistry) ([]PluginMCPServers, error) {
	return discoverMCPServers(pluginRegistry, nil)
}

// DiscoverEnabledMCPServers scans plugins and discovers MCP servers from enabled plugins only
func DiscoverEnabledMCPServers(pluginRegistry *claude.PluginRegistry, settings *claude.Settings) ([]PluginMCPServers, error) {
	return discoverMCPServers(pluginRegistry, settings)
}

// discoverMCPServers is the internal implementation that optionally filters by enabled plugins
func discoverMCPServers(pluginRegistry *claude.PluginRegistry, settings *claude.Settings) ([]PluginMCPServers, error) {
	var results []PluginMCPServers

	for name, plugin := range pluginRegistry.GetAllPlugins() {
		// If settings provided, skip disabled plugins
		if settings != nil && !settings.IsPluginEnabled(name) {
			continue
		}

		// Skip plugins with non-existent paths
		if !plugin.PathExists() {
			continue
		}

		var mcpServers map[string]ServerDefinition

		// Try .mcp.json first (newer format)
		mcpJSONPath := filepath.Join(plugin.InstallPath, ".mcp.json")
		if data, err := os.ReadFile(mcpJSONPath); err == nil {
			var mcpFile struct {
				MCPServers map[string]ServerDefinition `json:"mcpServers"`
			}
			if err := json.Unmarshal(data, &mcpFile); err == nil && len(mcpFile.MCPServers) > 0 {
				mcpServers = mcpFile.MCPServers
			}
		}

		// Fall back to plugin.json (older format)
		if mcpServers == nil {
			pluginJSONPath := filepath.Join(plugin.InstallPath, ".claude-plugin", "plugin.json")
			if data, err := os.ReadFile(pluginJSONPath); err == nil {
				var pluginJSON PluginJSON
				if err := json.Unmarshal(data, &pluginJSON); err == nil && len(pluginJSON.MCPServers) > 0 {
					mcpServers = pluginJSON.MCPServers
				}
			}
		}

		// Only add if the plugin actually has MCP servers
		if len(mcpServers) > 0 {
			results = append(results, PluginMCPServers{
				PluginName: name,
				PluginPath: plugin.InstallPath,
				Servers:    mcpServers,
			})
		}
	}

	return results, nil
}
