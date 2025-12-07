// ABOUTME: Integration test helpers for setting up test fixtures
// ABOUTME: Creates fake Claude installations for testing
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/mcp"
)

// TestEnv represents a test environment with a fake Claude installation
type TestEnv struct {
	ClaudeDir string
	t         *testing.T
}

// SetupTestEnv creates a temporary Claude installation for testing
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "claudeup-e2e-*")
	if err != nil {
		t.Fatal(err)
	}

	env := &TestEnv{
		ClaudeDir: tempDir,
		t:         t,
	}

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tempDir, "plugins"), 0755); err != nil {
		t.Fatal(err)
	}

	return env
}

// Cleanup removes the test environment
func (e *TestEnv) Cleanup() {
	os.RemoveAll(e.ClaudeDir)
}

// CreateMarketplace creates a fake marketplace
func (e *TestEnv) CreateMarketplace(name, repo string) {
	e.t.Helper()

	marketplaceDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", name)
	if err := os.MkdirAll(marketplaceDir, 0755); err != nil {
		e.t.Fatal(err)
	}

	// Initialize git repo
	gitDir := filepath.Join(marketplaceDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		e.t.Fatal(err)
	}
}

// CreatePlugin creates a fake plugin with optional MCP servers
func (e *TestEnv) CreatePlugin(name, marketplace, version string, mcpServers map[string]mcp.ServerDefinition) {
	e.t.Helper()

	pluginDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", marketplace, "plugins", name)
	if err := os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0755); err != nil {
		e.t.Fatal(err)
	}

	// Create plugin.json
	pluginJSON := mcp.PluginJSON{
		Name:       name,
		Version:    version,
		MCPServers: mcpServers,
	}

	data, err := json.MarshalIndent(pluginJSON, "", "  ")
	if err != nil {
		e.t.Fatal(err)
	}

	pluginJSONPath := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	if err := os.WriteFile(pluginJSONPath, data, 0644); err != nil {
		e.t.Fatal(err)
	}
}

// CreatePluginRegistry creates installed_plugins.json
func (e *TestEnv) CreatePluginRegistry(plugins map[string]claude.PluginMetadata) {
	e.t.Helper()

	registry := &claude.PluginRegistry{
		Version: 1,
		Plugins: plugins,
	}

	if err := claude.SavePlugins(e.ClaudeDir, registry); err != nil {
		e.t.Fatal(err)
	}
}

// CreateMarketplaceRegistry creates known_marketplaces.json
func (e *TestEnv) CreateMarketplaceRegistry(marketplaces map[string]claude.MarketplaceMetadata) {
	e.t.Helper()

	registry := claude.MarketplaceRegistry(marketplaces)

	if err := claude.SaveMarketplaces(e.ClaudeDir, registry); err != nil {
		e.t.Fatal(err)
	}
}

// LoadPluginRegistry loads the current plugin registry
func (e *TestEnv) LoadPluginRegistry() *claude.PluginRegistry {
	e.t.Helper()

	registry, err := claude.LoadPlugins(e.ClaudeDir)
	if err != nil {
		e.t.Fatal(err)
	}
	return registry
}

// LoadMarketplaceRegistry loads the current marketplace registry
func (e *TestEnv) LoadMarketplaceRegistry() claude.MarketplaceRegistry {
	e.t.Helper()

	registry, err := claude.LoadMarketplaces(e.ClaudeDir)
	if err != nil {
		e.t.Fatal(err)
	}
	return registry
}

// PluginExists checks if a plugin exists in the registry
func (e *TestEnv) PluginExists(name string) bool {
	e.t.Helper()

	registry := e.LoadPluginRegistry()
	return registry.PluginExists(name)
}

// PluginCount returns the number of installed plugins
func (e *TestEnv) PluginCount() int {
	e.t.Helper()

	registry := e.LoadPluginRegistry()
	return len(registry.Plugins)
}

// MarketplaceCount returns the number of installed marketplaces
func (e *TestEnv) MarketplaceCount() int {
	e.t.Helper()

	registry := e.LoadMarketplaceRegistry()
	return len(registry)
}
