// ABOUTME: Unit tests for global configuration management
// ABOUTME: Tests config loading, saving, and MCP enable/disable operations
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DisabledMCPServers == nil {
		t.Error("DisabledMCPServers slice should be initialized")
	}

	if cfg.Preferences.ActiveProfile != "" {
		t.Error("ActiveProfile should default to empty string")
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
	if !loadedCfg.IsMCPServerDisabled("test-server") {
		t.Error("Loaded config should have test-server disabled")
	}
}
