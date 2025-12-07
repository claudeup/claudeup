// ABOUTME: Integration tests for status and list commands
// ABOUTME: Tests internal functions with file-based fixtures
package integration

import (
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/mcp"
)

func TestStatusCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create test marketplaces
	env.CreateMarketplaceRegistry(map[string]claude.MarketplaceMetadata{
		"test-marketplace": {
			Source: claude.MarketplaceSource{
				Source: "github",
				Repo:   "test/repo",
			},
			InstallLocation: filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace"),
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
	})

	// Create test plugins
	env.CreateMarketplace("test-marketplace", "test/repo")
	env.CreatePlugin("plugin1", "test-marketplace", "1.0.0", map[string]mcp.ServerDefinition{
		"test-server": {
			Command: "node",
			Args:    []string{"server.js"},
		},
	})

	pluginPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin1")
	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"plugin1@test-marketplace": {
			Version:      "1.0.0",
			InstallPath:  pluginPath,
			GitCommitSha: "abc123",
			IsLocal:      true,
		},
		"stale-plugin@test-marketplace": {
			Version:      "1.0.0",
			InstallPath:  "/non/existent/path",
			GitCommitSha: "def456",
			IsLocal:      true,
		},
	})

	// Verify initial state
	if count := env.MarketplaceCount(); count != 1 {
		t.Errorf("Expected 1 marketplace, got %d", count)
	}

	if count := env.PluginCount(); count != 2 {
		t.Errorf("Expected 2 plugins, got %d", count)
	}

	// Verify plugin exists
	if !env.PluginExists("plugin1@test-marketplace") {
		t.Error("plugin1@test-marketplace should exist")
	}

	if !env.PluginExists("stale-plugin@test-marketplace") {
		t.Error("stale-plugin@test-marketplace should exist")
	}
}

func TestPluginsListCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create test marketplace
	env.CreateMarketplace("test-marketplace", "test/repo")

	// Create plugins
	env.CreatePlugin("plugin1", "test-marketplace", "1.0.0", nil)
	env.CreatePlugin("plugin2", "test-marketplace", "2.0.0", nil)

	plugin1Path := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin1")
	plugin2Path := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin2")

	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"plugin1@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: plugin1Path,
			IsLocal:     false,
		},
		"plugin2@test-marketplace": {
			Version:     "2.0.0",
			InstallPath: plugin2Path,
			IsLocal:     true,
		},
	})

	// Load and verify
	registry := env.LoadPluginRegistry()

	if len(registry.Plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(registry.Plugins))
	}

	plugin1 := registry.Plugins["plugin1@test-marketplace"]
	if plugin1.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", plugin1.Version)
	}

	if plugin1.IsLocal {
		t.Error("plugin1 should not be local")
	}

	plugin2 := registry.Plugins["plugin2@test-marketplace"]
	if plugin2.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", plugin2.Version)
	}

	if !plugin2.IsLocal {
		t.Error("plugin2 should be local")
	}
}

func TestMarketplacesCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create test marketplaces
	env.CreateMarketplace("marketplace1", "test/repo1")
	env.CreateMarketplace("marketplace2", "test/repo2")

	env.CreateMarketplaceRegistry(map[string]claude.MarketplaceMetadata{
		"marketplace1": {
			Source: claude.MarketplaceSource{
				Source: "github",
				Repo:   "test/repo1",
			},
			InstallLocation: filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "marketplace1"),
			LastUpdated:     "2024-01-01T00:00:00Z",
		},
		"marketplace2": {
			Source: claude.MarketplaceSource{
				Source: "git",
				Repo:   "test/repo2",
			},
			InstallLocation: filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "marketplace2"),
			LastUpdated:     "2024-01-02T00:00:00Z",
		},
	})

	// Load and verify
	registry := env.LoadMarketplaceRegistry()

	if len(registry) != 2 {
		t.Errorf("Expected 2 marketplaces, got %d", len(registry))
	}

	m1 := registry["marketplace1"]
	if m1.Source.Repo != "test/repo1" {
		t.Errorf("Expected repo test/repo1, got %s", m1.Source.Repo)
	}

	m2 := registry["marketplace2"]
	if m2.Source.Repo != "test/repo2" {
		t.Errorf("Expected repo test/repo2, got %s", m2.Source.Repo)
	}
}

func TestMCPListCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer env.Cleanup()

	// Create marketplace
	env.CreateMarketplace("test-marketplace", "test/repo")

	// Create plugins with and without MCP servers
	env.CreatePlugin("plugin-with-mcp", "test-marketplace", "1.0.0", map[string]mcp.ServerDefinition{
		"test-server": {
			Command: "node",
			Args:    []string{"server.js"},
			Env: map[string]string{
				"DEBUG": "true",
			},
		},
	})

	env.CreatePlugin("plugin-without-mcp", "test-marketplace", "1.0.0", nil)

	pluginWithMCPPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin-with-mcp")
	pluginWithoutMCPPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "plugin-without-mcp")

	env.CreatePluginRegistry(map[string]claude.PluginMetadata{
		"plugin-with-mcp@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: pluginWithMCPPath,
		},
		"plugin-without-mcp@test-marketplace": {
			Version:     "1.0.0",
			InstallPath: pluginWithoutMCPPath,
		},
	})

	// Discover MCP servers
	registry := env.LoadPluginRegistry()
	servers, err := mcp.DiscoverMCPServers(registry)
	if err != nil {
		t.Fatal(err)
	}

	// Should only find plugin-with-mcp
	if len(servers) != 1 {
		t.Errorf("Expected 1 plugin with MCP servers, got %d", len(servers))
	}

	if servers[0].PluginName != "plugin-with-mcp@test-marketplace" {
		t.Errorf("Expected plugin-with-mcp@test-marketplace, got %s", servers[0].PluginName)
	}

	testServer := servers[0].Servers["test-server"]
	if testServer.Command != "node" {
		t.Errorf("Expected command node, got %s", testServer.Command)
	}

	if len(testServer.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(testServer.Env))
	}
}
