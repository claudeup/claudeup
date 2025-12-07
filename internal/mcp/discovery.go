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
	var results []PluginMCPServers

	for name, plugin := range pluginRegistry.Plugins {
		// Skip plugins with non-existent paths
		if !plugin.PathExists() {
			continue
		}

		// Check for plugin.json in the .claude-plugin directory
		pluginJSONPath := filepath.Join(plugin.InstallPath, ".claude-plugin", "plugin.json")
		if _, err := os.Stat(pluginJSONPath); os.IsNotExist(err) {
			continue
		}

		// Read and parse plugin.json
		data, err := os.ReadFile(pluginJSONPath)
		if err != nil {
			// Skip plugins where we can't read plugin.json
			continue
		}

		var pluginJSON PluginJSON
		if err := json.Unmarshal(data, &pluginJSON); err != nil {
			// Skip plugins with invalid plugin.json
			continue
		}

		// Only add if the plugin actually has MCP servers
		if len(pluginJSON.MCPServers) > 0 {
			results = append(results, PluginMCPServers{
				PluginName: name,
				PluginPath: plugin.InstallPath,
				Servers:    pluginJSON.MCPServers,
			})
		}
	}

	return results, nil
}
