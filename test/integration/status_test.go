// ABOUTME: Integration tests for status and list commands
// ABOUTME: Tests internal functions with file-based fixtures
package integration

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v2/internal/claude"
	"github.com/claudeup/claudeup/v2/internal/mcp"
)

var _ = Describe("StatusCommand", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

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
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("verifies initial state", func() {
		Expect(env.MarketplaceCount()).To(Equal(1))
		Expect(env.PluginCount()).To(Equal(2))
	})

	It("verifies plugin exists", func() {
		Expect(env.PluginExists("plugin1@test-marketplace")).To(BeTrue())
		Expect(env.PluginExists("stale-plugin@test-marketplace")).To(BeTrue())
	})
})

var _ = Describe("PluginsListCommand", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

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
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("lists plugins correctly", func() {
		registry := env.LoadPluginRegistry()

		Expect(registry.Plugins).To(HaveLen(2))

		plugin1, _ := registry.GetPlugin("plugin1@test-marketplace")
		Expect(plugin1.Version).To(Equal("1.0.0"))
		Expect(plugin1.IsLocal).To(BeFalse())

		plugin2, _ := registry.GetPlugin("plugin2@test-marketplace")
		Expect(plugin2.Version).To(Equal("2.0.0"))
		Expect(plugin2.IsLocal).To(BeTrue())
	})
})

var _ = Describe("MarketplacesCommand", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

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
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("lists marketplaces correctly", func() {
		registry := env.LoadMarketplaceRegistry()

		Expect(registry).To(HaveLen(2))

		m1 := registry["marketplace1"]
		Expect(m1.Source.Repo).To(Equal("test/repo1"))

		m2 := registry["marketplace2"]
		Expect(m2.Source.Repo).To(Equal("test/repo2"))
	})
})

var _ = Describe("MCPListCommand", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

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
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("discovers MCP servers", func() {
		registry := env.LoadPluginRegistry()
		servers, err := mcp.DiscoverMCPServers(registry)
		Expect(err).NotTo(HaveOccurred())

		Expect(servers).To(HaveLen(1))
		Expect(servers[0].PluginName).To(Equal("plugin-with-mcp@test-marketplace"))

		testServer := servers[0].Servers["test-server"]
		Expect(testServer.Command).To(Equal("node"))
		Expect(testServer.Env).To(HaveLen(1))
	})
})
