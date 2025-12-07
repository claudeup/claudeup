// ABOUTME: Integration tests for enable/disable commands
// ABOUTME: Tests plugin and MCP server enable/disable workflows
package integration

import (
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/config"
)

func TestPluginDisableEnable(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create test marketplace and plugin
	env.CreateMarketplace("test-marketplace", "test/repo")
	env.CreatePlugin("test-plugin", "test-marketplace", "1.0.0", nil)

	pluginPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "test-plugin")
	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"test-plugin@test-marketplace": {
			Version:      "1.0.0",
			InstallPath:  pluginPath,
			GitCommitSha: "abc123",
			IsLocal:      true,
		},
	})

	// Initial state: plugin should exist
	if !env.PluginExists("test-plugin@test-marketplace") {
		t.Error("Plugin should exist initially")
	}

	if count := env.PluginCount(); count != 1 {
		t.Errorf("Expected 1 plugin, got %d", count)
	}

	// Disable the plugin
	registry := env.LoadPluginRegistry()
	metadata := registry.Plugins["test-plugin@test-marketplace"]

	cfg := config.DefaultConfig()
	cfg.DisablePlugin("test-plugin@test-marketplace", config.DisabledPlugin{
		Version:      metadata.Version,
		InstallPath:  metadata.InstallPath,
		GitCommitSha: metadata.GitCommitSha,
		IsLocal:      metadata.IsLocal,
	})

	registry.DisablePlugin("test-plugin@test-marketplace")
	if err := claude.SavePlugins(env.ClaudeDir, registry); err != nil {
		t.Fatal(err)
	}

	// Plugin should no longer be in registry
	if env.PluginExists("test-plugin@test-marketplace") {
		t.Error("Plugin should not exist after disable")
	}

	if count := env.PluginCount(); count != 0 {
		t.Errorf("Expected 0 plugins after disable, got %d", count)
	}

	// Plugin should be in disabled list
	if !cfg.IsPluginDisabled("test-plugin@test-marketplace") {
		t.Error("Plugin should be in disabled list")
	}

	// Re-enable the plugin
	disabledMeta, exists := cfg.EnablePlugin("test-plugin@test-marketplace")
	if !exists {
		t.Error("Plugin should exist in disabled list")
	}

	registry = env.LoadPluginRegistry()
	registry.EnablePlugin("test-plugin@test-marketplace", claude.PluginMetadata{
		Version:      disabledMeta.Version,
		InstallPath:  disabledMeta.InstallPath,
		GitCommitSha: disabledMeta.GitCommitSha,
		IsLocal:      disabledMeta.IsLocal,
	})
	if err := claude.SavePlugins(env.ClaudeDir, registry); err != nil {
		t.Fatal(err)
	}

	// Plugin should be back in registry
	if !env.PluginExists("test-plugin@test-marketplace") {
		t.Error("Plugin should exist after re-enable")
	}

	if count := env.PluginCount(); count != 1 {
		t.Errorf("Expected 1 plugin after re-enable, got %d", count)
	}

	// Plugin should not be in disabled list
	if cfg.IsPluginDisabled("test-plugin@test-marketplace") {
		t.Error("Plugin should not be in disabled list after enable")
	}
}

func TestMCPServerDisableEnable(t *testing.T) {
	cfg := config.DefaultConfig()
	serverRef := "test-plugin@test-marketplace:test-server"

	// Initial state: server should not be disabled
	if cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should not be disabled initially")
	}

	// Disable the MCP server
	if !cfg.DisableMCPServer(serverRef) {
		t.Error("DisableMCPServer should return true for first disable")
	}

	// Server should be disabled now
	if !cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should be disabled")
	}

	// Disabling again should return false
	if cfg.DisableMCPServer(serverRef) {
		t.Error("DisableMCPServer should return false for already disabled server")
	}

	// Enable the MCP server
	if !cfg.EnableMCPServer(serverRef) {
		t.Error("EnableMCPServer should return true")
	}

	// Server should not be disabled now
	if cfg.IsMCPServerDisabled(serverRef) {
		t.Error("MCP server should not be disabled after enable")
	}

	// Enabling again should return false
	if cfg.EnableMCPServer(serverRef) {
		t.Error("EnableMCPServer should return false for already enabled server")
	}
}

func TestMultiplePluginsDisable(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create test marketplace and plugins
	env.CreateMarketplace("test-marketplace", "test/repo")
	env.CreatePlugin("plugin1", "test-marketplace", "1.0.0", nil)
	env.CreatePlugin("plugin2", "test-marketplace", "2.0.0", nil)
	env.CreatePlugin("plugin3", "test-marketplace", "3.0.0", nil)

	plugin1Path := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin1")
	plugin2Path := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin2")
	plugin3Path := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin3")

	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"plugin1@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: plugin1Path,
		},
		"plugin2@test-marketplace": {
			Version:     "2.0.0",
			InstallPath: plugin2Path,
		},
		"plugin3@test-marketplace": {
			Version:     "3.0.0",
			InstallPath: plugin3Path,
		},
	})

	// Initial state: 3 plugins
	if count := env.PluginCount(); count != 3 {
		t.Errorf("Expected 3 plugins, got %d", count)
	}

	// Disable plugin1 and plugin3
	registry := env.LoadPluginRegistry()
	cfg := config.DefaultConfig()

	for _, name := range []string{"plugin1@test-marketplace", "plugin3@test-marketplace"} {
		metadata := registry.Plugins[name]
		cfg.DisablePlugin(name, config.DisabledPlugin{
			Version:     metadata.Version,
			InstallPath: metadata.InstallPath,
		})
		registry.DisablePlugin(name)
	}

	if err := claude.SavePlugins(env.ClaudeDir, registry); err != nil {
		t.Fatal(err)
	}

	// Should have 1 plugin left
	if count := env.PluginCount(); count != 1 {
		t.Errorf("Expected 1 plugin after disabling 2, got %d", count)
	}

	// plugin2 should still exist
	if !env.PluginExists("plugin2@test-marketplace") {
		t.Error("plugin2 should still exist")
	}

	// plugin1 and plugin3 should not exist
	if env.PluginExists("plugin1@test-marketplace") {
		t.Error("plugin1 should not exist after disable")
	}

	if env.PluginExists("plugin3@test-marketplace") {
		t.Error("plugin3 should not exist after disable")
	}

	// Both should be in disabled list
	if !cfg.IsPluginDisabled("plugin1@test-marketplace") {
		t.Error("plugin1 should be in disabled list")
	}

	if !cfg.IsPluginDisabled("plugin3@test-marketplace") {
		t.Error("plugin3 should be in disabled list")
	}
}
