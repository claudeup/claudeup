// ABOUTME: Unit tests for MCP server discovery functionality
// ABOUTME: Tests MCP server detection and parsing from plugin.json files
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v5/internal/claude"
)

func TestDiscoverEnabledMCPServers(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin1 with MCP servers (will be enabled)
	pluginPath1 := filepath.Join(tempDir, "plugin1")
	if err := os.MkdirAll(filepath.Join(pluginPath1, ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	pluginJSON1 := PluginJSON{
		Name:    "test-plugin-1",
		Version: "1.0.0",
		MCPServers: map[string]ServerDefinition{
			"server1": {
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
	}

	data1, _ := json.Marshal(pluginJSON1)
	if err := os.WriteFile(filepath.Join(pluginPath1, ".claude-plugin", "plugin.json"), data1, 0644); err != nil {
		t.Fatal(err)
	}

	// Create plugin2 with MCP servers (will be disabled)
	pluginPath2 := filepath.Join(tempDir, "plugin2")
	if err := os.MkdirAll(filepath.Join(pluginPath2, ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	pluginJSON2 := PluginJSON{
		Name:    "test-plugin-2",
		Version: "1.0.0",
		MCPServers: map[string]ServerDefinition{
			"server2": {
				Command: "python",
				Args:    []string{"server.py"},
			},
		},
	}

	data2, _ := json.Marshal(pluginJSON2)
	if err := os.WriteFile(filepath.Join(pluginPath2, ".claude-plugin", "plugin.json"), data2, 0644); err != nil {
		t.Fatal(err)
	}

	// Create plugin registry
	registry := &claude.PluginRegistry{
		Version: 2,
		Plugins: map[string][]claude.PluginMetadata{
			"plugin1@marketplace": {{
				Scope:       "user",
				InstallPath: pluginPath1,
			}},
			"plugin2@marketplace": {{
				Scope:       "user",
				InstallPath: pluginPath2,
			}},
		},
	}

	// Create settings with only plugin1 enabled
	settings := &claude.Settings{
		EnabledPlugins: map[string]bool{
			"plugin1@marketplace": true,
			"plugin2@marketplace": false,
		},
	}

	// Discover MCP servers with filtering
	servers, err := DiscoverEnabledMCPServers(registry, settings)
	if err != nil {
		t.Fatal(err)
	}

	// Should only find plugin1 (plugin2 is disabled)
	if len(servers) != 1 {
		t.Errorf("Expected 1 plugin with MCP servers, got %d", len(servers))
	}

	if servers[0].PluginName != "plugin1@marketplace" {
		t.Errorf("Expected plugin1@marketplace, got %s", servers[0].PluginName)
	}

	// Verify we got server1 but not server2
	if _, exists := servers[0].Servers["server1"]; !exists {
		t.Error("server1 should exist")
	}

	// Verify server2 is NOT in the results
	for _, server := range servers {
		if _, exists := server.Servers["server2"]; exists {
			t.Error("server2 should not exist (plugin2 is disabled)")
		}
	}
}

func TestDiscoverEnabledMCPServersWithMissingPluginInSettings(t *testing.T) {
	// Test that plugins not in enabledPlugins map are treated as disabled
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin with MCP servers
	pluginPath := filepath.Join(tempDir, "plugin")
	if err := os.MkdirAll(filepath.Join(pluginPath, ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	pluginJSON := PluginJSON{
		Name:    "test-plugin",
		Version: "1.0.0",
		MCPServers: map[string]ServerDefinition{
			"server": {
				Command: "node",
			},
		},
	}

	data, _ := json.Marshal(pluginJSON)
	if err := os.WriteFile(filepath.Join(pluginPath, ".claude-plugin", "plugin.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Create plugin registry
	registry := &claude.PluginRegistry{
		Version: 2,
		Plugins: map[string][]claude.PluginMetadata{
			"plugin@marketplace": {{
				Scope:       "user",
				InstallPath: pluginPath,
			}},
		},
	}

	// Create settings without the plugin (should be treated as disabled)
	settings := &claude.Settings{
		EnabledPlugins: map[string]bool{},
	}

	// Discover MCP servers with filtering
	servers, err := DiscoverEnabledMCPServers(registry, settings)
	if err != nil {
		t.Fatal(err)
	}

	// Should find nothing (plugin not in enabledPlugins)
	if len(servers) != 0 {
		t.Errorf("Expected 0 plugins (plugin not in enabledPlugins map), got %d", len(servers))
	}
}

func TestDiscoverEnabledMCPServersFromMCPJSON(t *testing.T) {
	// Test that .mcp.json format is read correctly for enabled plugins
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin with .mcp.json (newer format)
	pluginPath := filepath.Join(tempDir, "plugin")
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		t.Fatal(err)
	}

	mcpJSON := struct {
		MCPServers map[string]ServerDefinition `json:"mcpServers"`
	}{
		MCPServers: map[string]ServerDefinition{
			"server": {
				Command: "${CLAUDE_PLUGIN_ROOT}/scripts/mcp-server.cjs",
			},
		},
	}

	mcpData, _ := json.Marshal(mcpJSON)
	if err := os.WriteFile(filepath.Join(pluginPath, ".mcp.json"), mcpData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create plugin registry
	registry := &claude.PluginRegistry{
		Version: 2,
		Plugins: map[string][]claude.PluginMetadata{
			"plugin@marketplace": {{
				Scope:       "user",
				InstallPath: pluginPath,
			}},
		},
	}

	// Create settings with plugin enabled
	settings := &claude.Settings{
		EnabledPlugins: map[string]bool{
			"plugin@marketplace": true,
		},
	}

	// Discover MCP servers
	servers, err := DiscoverEnabledMCPServers(registry, settings)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 1 {
		t.Fatalf("Expected 1 plugin with MCP servers, got %d", len(servers))
	}

	server := servers[0].Servers["server"]
	if server.Command != "${CLAUDE_PLUGIN_ROOT}/scripts/mcp-server.cjs" {
		t.Errorf("Expected command with variable, got '%s'", server.Command)
	}
}
