// ABOUTME: Unit tests for MCP server discovery functionality
// ABOUTME: Tests MCP server detection and parsing from plugin.json files
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v3/internal/claude"
)

func TestDiscoverMCPServers(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin with MCP servers
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

	// Create plugin without MCP servers
	pluginPath2 := filepath.Join(tempDir, "plugin2")
	if err := os.MkdirAll(filepath.Join(pluginPath2, ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	pluginJSON2 := PluginJSON{
		Name:       "test-plugin-2",
		Version:    "1.0.0",
		MCPServers: map[string]ServerDefinition{},
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
			"plugin3@marketplace": {{
				Scope:       "user",
				InstallPath: filepath.Join(tempDir, "non-existent"),
			}},
		},
	}

	// Discover MCP servers
	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	// Should only find plugin1 with MCP servers
	if len(servers) != 1 {
		t.Errorf("Expected 1 plugin with MCP servers, got %d", len(servers))
	}

	if servers[0].PluginName != "plugin1@marketplace" {
		t.Errorf("Expected plugin1@marketplace, got %s", servers[0].PluginName)
	}

	if len(servers[0].Servers) != 1 {
		t.Errorf("Expected 1 MCP server, got %d", len(servers[0].Servers))
	}

	server1, exists := servers[0].Servers["server1"]
	if !exists {
		t.Error("server1 should exist")
	}

	if server1.Command != "node" {
		t.Errorf("Expected command 'node', got '%s'", server1.Command)
	}

	if len(server1.Args) != 1 || server1.Args[0] != "server.js" {
		t.Errorf("Expected args [server.js], got %v", server1.Args)
	}
}

func TestDiscoverMCPServersWithEnv(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin with MCP server that has env vars
	pluginPath := filepath.Join(tempDir, "plugin")
	if err := os.MkdirAll(filepath.Join(pluginPath, ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	pluginJSON := PluginJSON{
		Name:    "test-plugin",
		Version: "1.0.0",
		MCPServers: map[string]ServerDefinition{
			"server": {
				Command: "python",
				Args:    []string{"-m", "server"},
				Env: map[string]string{
					"API_KEY": "test-key",
					"DEBUG":   "true",
				},
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

	// Discover MCP servers
	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 1 {
		t.Fatalf("Expected 1 plugin with MCP servers, got %d", len(servers))
	}

	server := servers[0].Servers["server"]
	if server.Command != "python" {
		t.Errorf("Expected command 'python', got '%s'", server.Command)
	}

	if len(server.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(server.Env))
	}

	if server.Env["API_KEY"] != "test-key" {
		t.Errorf("Expected API_KEY=test-key, got %s", server.Env["API_KEY"])
	}
}

func TestDiscoverMCPServersInvalidJSON(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin with invalid JSON
	pluginPath := filepath.Join(tempDir, "plugin")
	if err := os.MkdirAll(filepath.Join(pluginPath, ".claude-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(pluginPath, ".claude-plugin", "plugin.json"), []byte("invalid json"), 0644); err != nil {
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

	// Discover MCP servers - should skip invalid plugin
	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 plugins with MCP servers (invalid JSON should be skipped), got %d", len(servers))
	}
}

func TestDiscoverMCPServersNoPluginJSON(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin directory but no plugin.json
	pluginPath := filepath.Join(tempDir, "plugin")
	if err := os.MkdirAll(filepath.Join(pluginPath, ".claude-plugin"), 0755); err != nil {
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

	// Discover MCP servers - should skip plugin without plugin.json
	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 plugins with MCP servers (no plugin.json), got %d", len(servers))
	}
}

func TestDiscoverMCPServersEmptyRegistry(t *testing.T) {
	registry := &claude.PluginRegistry{
		Version: 2,
		Plugins: make(map[string][]claude.PluginMetadata),
	}

	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 plugins with MCP servers (empty registry), got %d", len(servers))
	}
}

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

func TestDiscoverMCPServersFromMCPJSON(t *testing.T) {
	// Create temp directory
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

	// Discover MCP servers
	servers, err := DiscoverMCPServers(registry)
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

func TestDiscoverMCPServersFallbackToPluginJSON(t *testing.T) {
	// Test that when .mcp.json doesn't exist, we fall back to plugin.json
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create plugin with only plugin.json (older format)
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
				Args:    []string{"server.js"},
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

	// Discover MCP servers
	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 1 {
		t.Fatalf("Expected 1 plugin with MCP servers, got %d", len(servers))
	}

	server := servers[0].Servers["server"]
	if server.Command != "node" {
		t.Errorf("Expected command 'node', got '%s'", server.Command)
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

func TestFilterDisabledMCPServers(t *testing.T) {
	// Test that disabled MCP servers are filtered out
	servers := []PluginMCPServers{
		{
			PluginName: "plugin1@marketplace",
			PluginPath: "/path/to/plugin1",
			Servers: map[string]ServerDefinition{
				"server-a": {Command: "node", Args: []string{"a.js"}},
				"server-b": {Command: "node", Args: []string{"b.js"}},
			},
		},
		{
			PluginName: "plugin2@marketplace",
			PluginPath: "/path/to/plugin2",
			Servers: map[string]ServerDefinition{
				"server-c": {Command: "python", Args: []string{"c.py"}},
			},
		},
	}

	// Disable server-b from plugin1
	disabledServers := []string{"plugin1@marketplace:server-b"}

	filtered := FilterDisabledMCPServers(servers, disabledServers)

	// Should have 2 plugins still
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 plugins, got %d", len(filtered))
	}

	// plugin1 should only have server-a now
	var plugin1 *PluginMCPServers
	for i := range filtered {
		if filtered[i].PluginName == "plugin1@marketplace" {
			plugin1 = &filtered[i]
			break
		}
	}

	if plugin1 == nil {
		t.Fatal("plugin1@marketplace not found")
	}

	if len(plugin1.Servers) != 1 {
		t.Errorf("Expected 1 server in plugin1, got %d", len(plugin1.Servers))
	}

	if _, exists := plugin1.Servers["server-a"]; !exists {
		t.Error("server-a should exist")
	}

	if _, exists := plugin1.Servers["server-b"]; exists {
		t.Error("server-b should NOT exist (disabled)")
	}

	// plugin2 should still have server-c
	var plugin2 *PluginMCPServers
	for i := range filtered {
		if filtered[i].PluginName == "plugin2@marketplace" {
			plugin2 = &filtered[i]
			break
		}
	}

	if plugin2 == nil {
		t.Fatal("plugin2@marketplace not found")
	}

	if len(plugin2.Servers) != 1 {
		t.Errorf("Expected 1 server in plugin2, got %d", len(plugin2.Servers))
	}

	if _, exists := plugin2.Servers["server-c"]; !exists {
		t.Error("server-c should exist")
	}
}

func TestFilterDisabledMCPServers_AllServersDisabled(t *testing.T) {
	// Test that plugins with all servers disabled are removed from results
	servers := []PluginMCPServers{
		{
			PluginName: "plugin1@marketplace",
			PluginPath: "/path/to/plugin1",
			Servers: map[string]ServerDefinition{
				"server-a": {Command: "node"},
			},
		},
	}

	// Disable the only server
	disabledServers := []string{"plugin1@marketplace:server-a"}

	filtered := FilterDisabledMCPServers(servers, disabledServers)

	// Should have 0 plugins since all servers were disabled
	if len(filtered) != 0 {
		t.Errorf("Expected 0 plugins (all servers disabled), got %d", len(filtered))
	}
}

func TestFilterDisabledMCPServers_EmptyDisabledList(t *testing.T) {
	// Test that empty disabled list returns all servers
	servers := []PluginMCPServers{
		{
			PluginName: "plugin1@marketplace",
			PluginPath: "/path/to/plugin1",
			Servers: map[string]ServerDefinition{
				"server-a": {Command: "node"},
			},
		},
	}

	filtered := FilterDisabledMCPServers(servers, []string{})

	if len(filtered) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(filtered))
	}

	if len(filtered[0].Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(filtered[0].Servers))
	}
}

func TestFilterDisabledMCPServers_NilDisabledList(t *testing.T) {
	// Test that nil disabled list returns all servers
	servers := []PluginMCPServers{
		{
			PluginName: "plugin1@marketplace",
			PluginPath: "/path/to/plugin1",
			Servers: map[string]ServerDefinition{
				"server-a": {Command: "node"},
			},
		},
	}

	filtered := FilterDisabledMCPServers(servers, nil)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(filtered))
	}
}
