// ABOUTME: Unit tests for MCP server discovery functionality
// ABOUTME: Tests MCP server detection and parsing from plugin.json files
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
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
		Version: 1,
		Plugins: map[string]claude.PluginMetadata{
			"plugin1@marketplace": {
				InstallPath: pluginPath1,
			},
			"plugin2@marketplace": {
				InstallPath: pluginPath2,
			},
			"plugin3@marketplace": {
				InstallPath: filepath.Join(tempDir, "non-existent"),
			},
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
		Version: 1,
		Plugins: map[string]claude.PluginMetadata{
			"plugin@marketplace": {
				InstallPath: pluginPath,
			},
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
		Version: 1,
		Plugins: map[string]claude.PluginMetadata{
			"plugin@marketplace": {
				InstallPath: pluginPath,
			},
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
		Version: 1,
		Plugins: map[string]claude.PluginMetadata{
			"plugin@marketplace": {
				InstallPath: pluginPath,
			},
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
		Version: 1,
		Plugins: map[string]claude.PluginMetadata{},
	}

	servers, err := DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 plugins with MCP servers (empty registry), got %d", len(servers))
	}
}
