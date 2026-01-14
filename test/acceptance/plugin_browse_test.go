// ABOUTME: Acceptance tests for plugin browse command
// ABOUTME: Tests browsing available plugins from marketplaces
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin browse", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no arguments", func() {
		It("shows error requiring marketplace argument", func() {
			result := env.Run("plugin", "browse")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("accepts 1 arg"))
		})
	})

	Describe("with unknown marketplace", func() {
		It("shows error with helpful message", func() {
			result := env.Run("plugin", "browse", "unknown-marketplace")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
			Expect(result.Stderr).To(ContainSubstring("claude marketplace add"))
		})
	})

	Describe("with installed marketplace", func() {
		var marketplacePath string

		BeforeEach(func() {
			// Create marketplace directory
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			// Register marketplace in known_marketplaces.json
			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			// Create marketplace index
			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "acme-marketplace", []map[string]string{
				{"name": "plugin-a", "description": "First plugin", "version": "1.0.0"},
				{"name": "plugin-b", "description": "Second plugin", "version": "2.0.0"},
			})
		})

		It("lists available plugins by marketplace name", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("acme-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("plugin-a"))
			Expect(result.Stdout).To(ContainSubstring("plugin-b"))
		})

		It("lists available plugins by repo", func() {
			result := env.Run("plugin", "browse", "acme-corp/plugins")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("plugin-a"))
		})

		It("shows plugin descriptions", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("First plugin"))
			Expect(result.Stdout).To(ContainSubstring("Second plugin"))
		})

		It("shows plugin versions", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1.0.0"))
			Expect(result.Stdout).To(ContainSubstring("2.0.0"))
		})

		It("shows plugin count", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("2"))
		})
	})

	Describe("with installed plugins", func() {
		var marketplacePath string
		var pluginPath string

		BeforeEach(func() {
			// Create marketplace
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "acme-marketplace", []map[string]string{
				{"name": "plugin-a", "description": "First plugin", "version": "1.0.0"},
				{"name": "plugin-b", "description": "Second plugin", "version": "2.0.0"},
			})

			// Install one plugin
			pluginPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "plugin-a")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			env.CreateInstalledPlugins(map[string]interface{}{
				"plugin-a@acme-marketplace": []interface{}{
					map[string]interface{}{
						"version":     "1.0.0",
						"installPath": pluginPath,
						"scope":       "user",
					},
				},
			})
		})

		It("shows installed status for installed plugins", func() {
			result := env.Run("plugin", "browse", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("installed"))
		})
	})

	Describe("table format", func() {
		var marketplacePath string

		BeforeEach(func() {
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "acme-marketplace", []map[string]string{
				{"name": "plugin-a", "description": "First plugin", "version": "1.0.0"},
			})
		})

		It("shows table headers", func() {
			result := env.Run("plugin", "browse", "--format", "table", "acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("PLUGIN"))
			Expect(result.Stdout).To(ContainSubstring("DESCRIPTION"))
			Expect(result.Stdout).To(ContainSubstring("VERSION"))
		})
	})

	Describe("empty marketplace", func() {
		var marketplacePath string

		BeforeEach(func() {
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "empty-corp", "plugins")
			Expect(os.MkdirAll(marketplacePath, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"empty-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "empty-corp/plugins",
					},
					"installLocation": filepath.Dir(marketplacePath),
				},
			})

			env.CreateMarketplaceIndex(filepath.Dir(marketplacePath), "empty-marketplace", []map[string]string{})
		})

		It("shows no plugins message", func() {
			result := env.Run("plugin", "browse", "empty-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No plugins"))
		})
	})
})
