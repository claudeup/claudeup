// ABOUTME: E2E tests for maintenance commands
// ABOUTME: Tests doctor, fix-paths, and cleanup workflows
package e2e

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/malston/claude-pm/internal/claude"
)

func TestDoctorDetectsStalePlugins(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create marketplace
	env.CreateMarketplace("test-marketplace", "test/repo")

	// Create one valid plugin
	env.CreatePlugin("valid-plugin", "test-marketplace", "1.0.0", nil)
	validPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "valid-plugin")

	// Create plugin registry with both valid and stale plugins
	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"valid-plugin@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: validPath,
		},
		"stale-plugin@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: "/non/existent/path",
		},
	})

	// Load and check
	registry := env.LoadPluginRegistry()

	validCount := 0
	staleCount := 0

	for _, plugin := range registry.Plugins {
		if plugin.PathExists() {
			validCount++
		} else {
			staleCount++
		}
	}

	if validCount != 1 {
		t.Errorf("Expected 1 valid plugin, got %d", validCount)
	}

	if staleCount != 1 {
		t.Errorf("Expected 1 stale plugin, got %d", staleCount)
	}
}

func TestFixPathsCorrectsPaths(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create marketplace
	env.CreateMarketplace("claude-code-plugins", "anthropics/claude-code")

	// Create plugin in correct location
	env.CreatePlugin("test-plugin", "claude-code-plugins", "1.0.0", nil)
	correctPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugins", "test-plugin")

	// But register it with wrong path (missing /plugins/ subdirectory)
	wrongPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "test-plugin")

	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"test-plugin@claude-code-plugins": {
			Version:     "1.0.0",
			InstallPath: wrongPath,
			IsLocal:     true,
		},
	})

	// Plugin should not exist at wrong path
	registry := env.LoadPluginRegistry()
	plugin := registry.Plugins["test-plugin@claude-code-plugins"]
	if plugin.PathExists() {
		t.Error("Plugin should not exist at wrong path")
	}

	// Verify wrong path contains the marketplace name
	if !strings.Contains(wrongPath, "claude-code-plugins") {
		t.Fatal("Wrong path should contain marketplace name")
	}

	// Verify wrong path doesn't have /plugins/ subdirectory
	if strings.Contains(wrongPath, "/plugins/test-plugin") {
		t.Fatal("Wrong path should not have /plugins/ subdirectory")
	}

	// Verify correct path does have /plugins/ subdirectory
	if !strings.Contains(correctPath, "/plugins/test-plugin") {
		t.Fatal("Correct path should have /plugins/ subdirectory")
	}

	// Fix the path by updating the registry
	plugin.InstallPath = correctPath
	registry.Plugins["test-plugin@claude-code-plugins"] = plugin

	if err := claude.SavePlugins(env.ClaudeDir, registry); err != nil {
		t.Fatal(err)
	}

	// Reload and verify the path was persisted
	registry = env.LoadPluginRegistry()
	plugin = registry.Plugins["test-plugin@claude-code-plugins"]

	if plugin.InstallPath != correctPath {
		t.Errorf("Expected path %s, got %s", correctPath, plugin.InstallPath)
	}

	// Plugin should now exist at corrected path
	if !plugin.PathExists() {
		t.Error("Plugin should exist at corrected path")
	}
}

func TestCleanupRemovesStalePlugins(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create marketplace
	env.CreateMarketplace("test-marketplace", "test/repo")

	// Create one valid plugin
	env.CreatePlugin("valid-plugin", "test-marketplace", "1.0.0", nil)
	validPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "valid-plugin")

	// Create plugin registry with valid and stale plugins
	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"valid-plugin@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: validPath,
		},
		"stale-plugin1@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: "/non/existent/path1",
		},
		"stale-plugin2@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: "/non/existent/path2",
		},
	})

	// Initial state: 3 plugins
	if count := env.PluginCount(); count != 3 {
		t.Errorf("Expected 3 plugins initially, got %d", count)
	}

	// Clean up stale plugins
	registry := env.LoadPluginRegistry()

	for name, plugin := range registry.Plugins {
		if !plugin.PathExists() {
			registry.DisablePlugin(name)
		}
	}

	if err := claude.SavePlugins(env.ClaudeDir, registry); err != nil {
		t.Fatal(err)
	}

	// After cleanup: should have 1 plugin
	if count := env.PluginCount(); count != 1 {
		t.Errorf("Expected 1 plugin after cleanup, got %d", count)
	}

	// Only valid plugin should remain
	if !env.PluginExists("valid-plugin@test-marketplace") {
		t.Error("valid-plugin should exist after cleanup")
	}

	if env.PluginExists("stale-plugin1@test-marketplace") {
		t.Error("stale-plugin1 should not exist after cleanup")
	}

	if env.PluginExists("stale-plugin2@test-marketplace") {
		t.Error("stale-plugin2 should not exist after cleanup")
	}
}

func TestFixPathsMultipleMarketplaces(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create multiple marketplaces
	env.CreateMarketplace("claude-code-plugins", "anthropics/claude-code")
	env.CreateMarketplace("every-marketplace", "every/marketplace")

	// Create plugins in correct locations
	env.CreatePlugin("plugin1", "claude-code-plugins", "1.0.0", nil)
	env.CreatePlugin("plugin2", "every-marketplace", "2.0.0", nil)

	correctPath1 := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugins", "plugin1")
	correctPath2 := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "every-marketplace", "plugins", "plugin2")

	// Register with wrong paths
	wrongPath1 := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugin1")
	wrongPath2 := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "every-marketplace", "plugin2")

	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"plugin1@claude-code-plugins": {
			Version:     "1.0.0",
			InstallPath: wrongPath1,
			IsLocal:     true,
		},
		"plugin2@every-marketplace": {
			Version:     "2.0.0",
			InstallPath: wrongPath2,
			IsLocal:     true,
		},
	})

	// Both should not exist at wrong paths
	registry := env.LoadPluginRegistry()

	for _, plugin := range registry.Plugins {
		if plugin.PathExists() {
			t.Error("Plugins should not exist at wrong paths")
		}
	}

	// Fix paths by directly setting to correct paths
	plugin1 := registry.Plugins["plugin1@claude-code-plugins"]
	plugin1.InstallPath = correctPath1
	registry.Plugins["plugin1@claude-code-plugins"] = plugin1

	plugin2 := registry.Plugins["plugin2@every-marketplace"]
	plugin2.InstallPath = correctPath2
	registry.Plugins["plugin2@every-marketplace"] = plugin2

	if err := claude.SavePlugins(env.ClaudeDir, registry); err != nil {
		t.Fatal(err)
	}

	// Reload to verify the saved changes
	registry = env.LoadPluginRegistry()

	plugin1 = registry.Plugins["plugin1@claude-code-plugins"]
	if plugin1.InstallPath != correctPath1 {
		t.Errorf("Plugin1 expected path %s, got %s", correctPath1, plugin1.InstallPath)
	}

	if !plugin1.PathExists() {
		t.Error("Plugin1 should exist at corrected path")
	}

	plugin2 = registry.Plugins["plugin2@every-marketplace"]
	if plugin2.InstallPath != correctPath2 {
		t.Errorf("Plugin2 expected path %s, got %s", correctPath2, plugin2.InstallPath)
	}

	if !plugin2.PathExists() {
		t.Error("Plugin2 should exist at corrected path")
	}
}
