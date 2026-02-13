// ABOUTME: Integration test helpers for setting up test fixtures
// ABOUTME: Creates fake Claude installations for testing
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/internal/mcp"
)

// TestEnv represents a test environment with a fake Claude installation
type TestEnv struct {
	ClaudeDir string
}

// SetupTestEnv creates a temporary Claude installation for testing
func SetupTestEnv() *TestEnv {
	tempDir, err := os.MkdirTemp("", "claudeup-e2e-*")
	Expect(err).NotTo(HaveOccurred())

	env := &TestEnv{
		ClaudeDir: tempDir,
	}

	// Create directory structure
	err = os.MkdirAll(filepath.Join(tempDir, "plugins"), 0755)
	Expect(err).NotTo(HaveOccurred())

	return env
}

// Cleanup removes the test environment
func (e *TestEnv) Cleanup() {
	os.RemoveAll(e.ClaudeDir)
}

// CreateMarketplace creates a fake marketplace
func (e *TestEnv) CreateMarketplace(name, repo string) {
	marketplaceDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", name)
	err := os.MkdirAll(marketplaceDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	// Initialize git repo
	gitDir := filepath.Join(marketplaceDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	Expect(err).NotTo(HaveOccurred())
}

// CreatePlugin creates a fake plugin with optional MCP servers
func (e *TestEnv) CreatePlugin(name, marketplace, version string, mcpServers map[string]mcp.ServerDefinition) {
	pluginDir := filepath.Join(e.ClaudeDir, "plugins", "marketplaces", marketplace, "plugins", name)
	err := os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0755)
	Expect(err).NotTo(HaveOccurred())

	// Create plugin.json
	pluginJSON := mcp.PluginJSON{
		Name:       name,
		Version:    version,
		MCPServers: mcpServers,
	}

	data, err := json.MarshalIndent(pluginJSON, "", "  ")
	Expect(err).NotTo(HaveOccurred())

	pluginJSONPath := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	err = os.WriteFile(pluginJSONPath, data, 0644)
	Expect(err).NotTo(HaveOccurred())
}

// CreatePluginRegistry creates installed_plugins.json
func (e *TestEnv) CreatePluginRegistry(plugins map[string]claude.PluginMetadata) {
	// Convert to V2 format
	pluginsV2 := make(map[string][]claude.PluginMetadata)
	for name, meta := range plugins {
		// Ensure scope is set
		if meta.Scope == "" {
			meta.Scope = "user"
		}
		pluginsV2[name] = []claude.PluginMetadata{meta}
	}

	registry := &claude.PluginRegistry{
		Version: 2,
		Plugins: pluginsV2,
	}

	err := claude.SavePlugins(e.ClaudeDir, registry)
	Expect(err).NotTo(HaveOccurred())
}

// CreateMarketplaceRegistry creates known_marketplaces.json
func (e *TestEnv) CreateMarketplaceRegistry(marketplaces map[string]claude.MarketplaceMetadata) {
	registry := claude.MarketplaceRegistry(marketplaces)

	// Write marketplace registry directly (SaveMarketplaces was removed as dead code)
	marketplacesPath := filepath.Join(e.ClaudeDir, "plugins", "known_marketplaces.json")
	data, err := json.MarshalIndent(registry, "", "  ")
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(marketplacesPath, data, 0644)
	Expect(err).NotTo(HaveOccurred())
}

// LoadPluginRegistry loads the current plugin registry
func (e *TestEnv) LoadPluginRegistry() *claude.PluginRegistry {
	registry, err := claude.LoadPlugins(e.ClaudeDir)
	Expect(err).NotTo(HaveOccurred())
	return registry
}

// LoadMarketplaceRegistry loads the current marketplace registry
func (e *TestEnv) LoadMarketplaceRegistry() claude.MarketplaceRegistry {
	registry, err := claude.LoadMarketplaces(e.ClaudeDir)
	Expect(err).NotTo(HaveOccurred())
	return registry
}

// PluginExists checks if a plugin exists in the registry at any scope
func (e *TestEnv) PluginExists(name string) bool {
	registry := e.LoadPluginRegistry()
	return registry.PluginExistsAtAnyScope(name)
}

// PluginCount returns the number of installed plugins
func (e *TestEnv) PluginCount() int {
	registry := e.LoadPluginRegistry()
	return len(registry.Plugins)
}

// MarketplaceCount returns the number of installed marketplaces
func (e *TestEnv) MarketplaceCount() int {
	registry := e.LoadMarketplaceRegistry()
	return len(registry)
}
