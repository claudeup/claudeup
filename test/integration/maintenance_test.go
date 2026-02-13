// ABOUTME: Integration tests for maintenance commands
// ABOUTME: Tests doctor, fix-paths, and cleanup workflows
package integration

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/claude"
)

var _ = Describe("DoctorDetectsStalePlugins", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

		env.CreateMarketplace("test-marketplace", "test/repo")
		env.CreatePlugin("valid-plugin", "test-marketplace", "1.0.0", nil)
		validPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "valid-plugin")

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
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("detects valid and stale plugins", func() {
		registry := env.LoadPluginRegistry()

		validCount := 0
		staleCount := 0

		for _, sp := range registry.GetPluginsAtScopes(claude.ValidScopes) {
			if sp.PathExists() {
				validCount++
			} else {
				staleCount++
			}
		}

		Expect(validCount).To(Equal(1))
		Expect(staleCount).To(Equal(1))
	})
})

var _ = Describe("FixPathsCorrectsPaths", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

		env.CreateMarketplace("claude-code-plugins", "anthropics/claude-code")
		env.CreatePlugin("test-plugin", "claude-code-plugins", "1.0.0", nil)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("corrects plugin paths", func() {
		correctPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugins", "test-plugin")
		wrongPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "test-plugin")

		env.CreatePluginRegistry(map[string]claude.PluginMetadata{
			"test-plugin@claude-code-plugins": {
				Version:     "1.0.0",
				InstallPath: wrongPath,
				IsLocal:     true,
			},
		})

		registry := env.LoadPluginRegistry()
		plugin, _ := registry.GetPluginAtScope("test-plugin@claude-code-plugins", "user")
		Expect(plugin.PathExists()).To(BeFalse(), "Plugin should not exist at wrong path")

		Expect(wrongPath).To(ContainSubstring("claude-code-plugins"), "Wrong path should contain marketplace name")
		Expect(wrongPath).NotTo(ContainSubstring("/plugins/test-plugin"), "Wrong path should not have /plugins/ subdirectory")
		Expect(correctPath).To(ContainSubstring("/plugins/test-plugin"), "Correct path should have /plugins/ subdirectory")

		plugin.InstallPath = correctPath
		registry.SetPlugin("test-plugin@claude-code-plugins", plugin)

		err := claude.SavePlugins(env.ClaudeDir, registry)
		Expect(err).NotTo(HaveOccurred())

		registry = env.LoadPluginRegistry()
		plugin, _ = registry.GetPluginAtScope("test-plugin@claude-code-plugins", "user")

		Expect(plugin.InstallPath).To(Equal(correctPath))
		Expect(plugin.PathExists()).To(BeTrue(), "Plugin should exist at corrected path")
	})
})

var _ = Describe("CleanupRemovesStalePlugins", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

		env.CreateMarketplace("test-marketplace", "test/repo")
		env.CreatePlugin("valid-plugin", "test-marketplace", "1.0.0", nil)
		validPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace", "plugins", "valid-plugin")

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
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("removes stale plugins", func() {
		Expect(env.PluginCount()).To(Equal(3))

		registry := env.LoadPluginRegistry()

		for _, sp := range registry.GetPluginsAtScopes(claude.ValidScopes) {
			if !sp.PathExists() {
				registry.DisablePlugin(sp.Name)
			}
		}

		err := claude.SavePlugins(env.ClaudeDir, registry)
		Expect(err).NotTo(HaveOccurred())

		Expect(env.PluginCount()).To(Equal(1))
		Expect(env.PluginExists("valid-plugin@test-marketplace")).To(BeTrue(), "valid-plugin should exist after cleanup")
		Expect(env.PluginExists("stale-plugin1@test-marketplace")).To(BeFalse(), "stale-plugin1 should not exist after cleanup")
		Expect(env.PluginExists("stale-plugin2@test-marketplace")).To(BeFalse(), "stale-plugin2 should not exist after cleanup")
	})
})

var _ = Describe("FixPathsMultipleMarketplaces", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

		env.CreateMarketplace("claude-code-plugins", "anthropics/claude-code")
		env.CreateMarketplace("every-marketplace", "every/marketplace")

		env.CreatePlugin("plugin1", "claude-code-plugins", "1.0.0", nil)
		env.CreatePlugin("plugin2", "every-marketplace", "2.0.0", nil)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("fixes paths for multiple marketplaces", func() {
		correctPath1 := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugins", "plugin1")
		correctPath2 := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "every-marketplace", "plugins", "plugin2")

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

		registry := env.LoadPluginRegistry()

		for _, sp := range registry.GetPluginsAtScopes(claude.ValidScopes) {
			Expect(sp.PathExists()).To(BeFalse(), "Plugins should not exist at wrong paths")
		}

		plugin1, _ := registry.GetPluginAtScope("plugin1@claude-code-plugins", "user")
		plugin1.InstallPath = correctPath1
		registry.SetPlugin("plugin1@claude-code-plugins", plugin1)

		plugin2, _ := registry.GetPluginAtScope("plugin2@every-marketplace", "user")
		plugin2.InstallPath = correctPath2
		registry.SetPlugin("plugin2@every-marketplace", plugin2)

		err := claude.SavePlugins(env.ClaudeDir, registry)
		Expect(err).NotTo(HaveOccurred())

		registry = env.LoadPluginRegistry()

		plugin1, _ = registry.GetPluginAtScope("plugin1@claude-code-plugins", "user")
		Expect(plugin1.InstallPath).To(Equal(correctPath1))
		Expect(plugin1.PathExists()).To(BeTrue(), "Plugin1 should exist at corrected path")

		plugin2, _ = registry.GetPluginAtScope("plugin2@every-marketplace", "user")
		Expect(plugin2.InstallPath).To(Equal(correctPath2))
		Expect(plugin2.PathExists()).To(BeTrue(), "Plugin2 should exist at corrected path")
	})
})

var _ = Describe("UnifiedCleanupFixesAndRemoves", func() {
	var env *TestEnv

	BeforeEach(func() {
		env = SetupTestEnv()

		env.CreateMarketplace("claude-code-plugins", "anthropics/claude-code")

		env.CreatePlugin("fixable-plugin", "claude-code-plugins", "1.0.0", nil)
		env.CreatePlugin("valid-plugin", "claude-code-plugins", "1.0.0", nil)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("fixes and removes plugins in one pass", func() {
		correctPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugins", "fixable-plugin")
		wrongPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "fixable-plugin")
		validPath := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "claude-code-plugins", "plugins", "valid-plugin")

		env.CreatePluginRegistry(map[string]claude.PluginMetadata{
			"fixable-plugin@claude-code-plugins": {
				Version:     "1.0.0",
				InstallPath: wrongPath,
				IsLocal:     true,
			},
			"valid-plugin@claude-code-plugins": {
				Version:     "1.0.0",
				InstallPath: validPath,
				IsLocal:     true,
			},
			"missing-plugin@claude-code-plugins": {
				Version:     "1.0.0",
				InstallPath: "/non/existent/path",
				IsLocal:     true,
			},
		})

		Expect(env.PluginCount()).To(Equal(3))

		registry := env.LoadPluginRegistry()

		fixed := 0
		removed := 0

		for _, sp := range registry.GetPluginsAtScopes(claude.ValidScopes) {
			if !sp.PathExists() {
				var expectedPath string
				if strings.Contains(sp.InstallPath, "claude-code-plugins") {
					base := filepath.Dir(sp.InstallPath)
					pluginName := filepath.Base(sp.InstallPath)
					expectedPath = filepath.Join(base, "plugins", pluginName)
				}

				if expectedPath != "" {
					updatedPlugin := sp.PluginMetadata
					updatedPlugin.InstallPath = expectedPath
					if updatedPlugin.PathExists() {
						sp.InstallPath = expectedPath
						registry.SetPlugin(sp.Name, sp.PluginMetadata)
						fixed++
						continue
					}
				}

				registry.DisablePlugin(sp.Name)
				removed++
			}
		}

		err := claude.SavePlugins(env.ClaudeDir, registry)
		Expect(err).NotTo(HaveOccurred())

		Expect(fixed).To(Equal(1))
		Expect(removed).To(Equal(1))
		Expect(env.PluginCount()).To(Equal(2))

		registry = env.LoadPluginRegistry()
		fixedPlugin, _ := registry.GetPluginAtScope("fixable-plugin@claude-code-plugins", "user")
		Expect(fixedPlugin.InstallPath).To(Equal(correctPath))
		Expect(fixedPlugin.PathExists()).To(BeTrue(), "Fixed plugin should exist at corrected path")

		Expect(env.PluginExists("valid-plugin@claude-code-plugins")).To(BeTrue(), "valid-plugin should still exist")
		Expect(env.PluginExists("missing-plugin@claude-code-plugins")).To(BeFalse(), "missing-plugin should have been removed")
	})
})
