// ABOUTME: Unit tests for global configuration management
// ABOUTME: Tests config loading, saving, and plugin/MCP enable/disable operations
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DisabledPlugins == nil {
		t.Error("DisabledPlugins map should be initialized")
	}

	if cfg.DisabledMCPServers == nil {
		t.Error("DisabledMCPServers slice should be initialized")
	}

	if cfg.Preferences.AutoUpdate != false {
		t.Error("AutoUpdate should default to false")
	}

	if cfg.Preferences.VerboseOutput != false {
		t.Error("VerboseOutput should default to false")
	}
}

func TestIsPluginDisabled(t *testing.T) {
	cfg := DefaultConfig()
	pluginName := "test-plugin@test-marketplace"

	// Should not be disabled initially
	if cfg.IsPluginDisabled(pluginName) {
		t.Error("Plugin should not be disabled initially")
	}

	// Add to disabled map
	cfg.DisabledPlugins[pluginName] = DisabledPlugin{
		Version: "1.0.0",
	}

	// Should be disabled now
	if !cfg.IsPluginDisabled(pluginName) {
		t.Error("Plugin should be disabled after adding to map")
	}
}

func TestDisablePlugin(t *testing.T) {
	cfg := DefaultConfig()
	pluginName := "test-plugin@test-marketplace"
	metadata := DisabledPlugin{
		Version:      "1.0.0",
		InstallPath:  "/path/to/plugin",
		GitCommitSha: "abc123",
		IsLocal:      true,
	}

	// First disable should return true
	if !cfg.DisablePlugin(pluginName, metadata) {
		t.Error("First disable should return true")
	}

	// Second disable should return false (already disabled)
	if cfg.DisablePlugin(pluginName, metadata) {
		t.Error("Second disable should return false")
	}

	// Verify metadata was stored
	stored, exists := cfg.GetDisabledPlugin(pluginName)
	if !exists {
		t.Error("Plugin metadata should exist")
	}

	if stored.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", stored.Version)
	}

	if stored.GitCommitSha != "abc123" {
		t.Errorf("Expected commit abc123, got %s", stored.GitCommitSha)
	}
}

func TestEnablePlugin(t *testing.T) {
	cfg := DefaultConfig()
	pluginName := "test-plugin@test-marketplace"
	metadata := DisabledPlugin{
		Version: "1.0.0",
	}

	// Enable non-disabled plugin should return false
	_, enabled := cfg.EnablePlugin(pluginName)
	if enabled {
		t.Error("Enabling non-disabled plugin should return false")
	}

	// Disable then enable
	cfg.DisablePlugin(pluginName, metadata)
	retrieved, enabled := cfg.EnablePlugin(pluginName)

	if !enabled {
		t.Error("Enabling disabled plugin should return true")
	}

	if retrieved.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", retrieved.Version)
	}

	// Should no longer be in disabled map
	if cfg.IsPluginDisabled(pluginName) {
		t.Error("Plugin should not be disabled after enabling")
	}
}

func TestIsMCPServerDisabled(t *testing.T) {
	cfg := DefaultConfig()
	serverRef := "plugin@marketplace:server"

	// Should not be disabled initially
	if cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should not be disabled initially")
	}

	// Add to disabled list
	cfg.DisabledMCPServers = append(cfg.DisabledMCPServers, serverRef)

	// Should be disabled now
	if !cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should be disabled after adding to list")
	}
}

func TestDisableMCPServer(t *testing.T) {
	cfg := DefaultConfig()
	serverRef := "plugin@marketplace:server"

	// First disable should return true
	if !cfg.DisableMCPServer(serverRef) {
		t.Error("First disable should return true")
	}

	// Should be in disabled list
	if !cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should be in disabled list")
	}

	// Second disable should return false (already disabled)
	if cfg.DisableMCPServer(serverRef) {
		t.Error("Second disable should return false")
	}
}

func TestEnableMCPServer(t *testing.T) {
	cfg := DefaultConfig()
	serverRef := "plugin@marketplace:server"

	// Enable non-disabled server should return false
	if cfg.EnableMCPServer(serverRef) {
		t.Error("Enabling non-disabled MCP server should return false")
	}

	// Disable then enable
	cfg.DisableMCPServer(serverRef)
	if !cfg.EnableMCPServer(serverRef) {
		t.Error("Enabling disabled MCP server should return true")
	}

	// Should no longer be in disabled list
	if cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should not be disabled after enabling")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create config
	cfg := DefaultConfig()
	cfg.DisablePlugin("test-plugin", DisabledPlugin{Version: "1.0.0"})
	cfg.DisableMCPServer("test-server")

	// Save to temp file
	configFile := filepath.Join(tempDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load it back
	loadedData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	var loadedCfg GlobalConfig
	if err := json.Unmarshal(loadedData, &loadedCfg); err != nil {
		t.Fatal(err)
	}

	// Verify loaded config matches
	if !loadedCfg.IsPluginDisabled("test-plugin") {
		t.Error("Loaded config should have test-plugin disabled")
	}

	if !loadedCfg.IsMCPServerDisabled("test-server") {
		t.Error("Loaded config should have test-server disabled")
	}

	plugin, exists := loadedCfg.GetDisabledPlugin("test-plugin")
	if !exists {
		t.Error("Loaded config should have test-plugin metadata")
	}

	if plugin.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", plugin.Version)
	}
}

func TestGetDisabledPlugin(t *testing.T) {
	cfg := DefaultConfig()
	pluginName := "test-plugin@test-marketplace"

	// Get non-existent plugin
	_, exists := cfg.GetDisabledPlugin(pluginName)
	if exists {
		t.Error("Non-existent plugin should not exist")
	}

	// Add plugin
	metadata := DisabledPlugin{
		Version:     "2.0.0",
		InstallPath: "/test/path",
	}
	cfg.DisablePlugin(pluginName, metadata)

	// Get existing plugin
	retrieved, exists := cfg.GetDisabledPlugin(pluginName)
	if !exists {
		t.Error("Plugin should exist after disabling")
	}

	if retrieved.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", retrieved.Version)
	}

	if retrieved.InstallPath != "/test/path" {
		t.Errorf("Expected path /test/path, got %s", retrieved.InstallPath)
	}
}
