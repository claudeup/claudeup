// ABOUTME: Acceptance tests for plugin show command
// ABOUTME: Tests viewing plugin contents in tree format
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v3/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin browse --show", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with valid plugin", func() {
		var marketplacePath string
		var pluginPath string

		BeforeEach(func() {
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp")
			pluginPath = filepath.Join(marketplacePath, "plugins", "test-plugin")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(pluginPath, "agents"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(pluginPath, "agents", "test.md"), []byte("# Agent"), 0644)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": marketplacePath,
				},
			})

			env.CreateMarketplaceIndex(marketplacePath, "acme-marketplace", []map[string]string{
				{"name": "test-plugin", "description": "Test plugin", "version": "1.0.0"},
			})
		})

		It("displays tree when --show flag provided", func() {
			result := env.Run("plugin", "browse", "acme-marketplace", "--show", "test-plugin")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("test-plugin@acme-marketplace"))
			Expect(result.Stdout).To(MatchRegexp(`[├└]──`))
		})

		It("shows plugin version in output", func() {
			result := env.Run("plugin", "browse", "acme-marketplace", "--show", "test-plugin")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1.0.0"))
		})
	})
})

var _ = Describe("plugin show", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with invalid format", func() {
		It("shows error for missing @ separator", func() {
			result := env.Run("plugin", "show", "myplugin")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("<plugin>@<marketplace>"))
		})
	})

	Describe("with unknown marketplace", func() {
		It("shows helpful error", func() {
			result := env.Run("plugin", "show", "test-plugin@unknown-marketplace")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})

	Describe("with valid marketplace", func() {
		var marketplacePath string
		var pluginPath string

		BeforeEach(func() {
			// Create marketplace with plugin directory
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp")
			pluginPath = filepath.Join(marketplacePath, "plugins", "test-plugin")
			Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

			// Create plugin structure
			Expect(os.MkdirAll(filepath.Join(pluginPath, "agents"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(pluginPath, "skills", "foo"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(pluginPath, "agents", "test.md"), []byte("# Agent"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(pluginPath, "skills", "foo", "SKILL.md"), []byte("# Skill"), 0644)).To(Succeed())

			// Register marketplace
			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": marketplacePath,
				},
			})

			// Create marketplace index
			env.CreateMarketplaceIndex(marketplacePath, "acme-marketplace", []map[string]string{
				{"name": "test-plugin", "description": "Test plugin", "version": "1.0.0"},
			})
		})

		It("displays tree for plugin@marketplace notation", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("test-plugin@acme-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("1.0.0"))
		})

		It("shows tree with box-drawing characters", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`[├└]──`))
		})

		It("shows directory and file counts", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`\d+ director`))
			Expect(result.Stdout).To(MatchRegexp(`\d+ file`))
		})

		It("shows agents directory", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("agents/"))
		})

		It("shows skills directory", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("skills/"))
		})
	})

	Describe("with nonexistent plugin", func() {
		var marketplacePath string

		BeforeEach(func() {
			marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp")
			Expect(os.MkdirAll(filepath.Join(marketplacePath, "plugins"), 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"acme-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"source": "github",
						"repo":   "acme-corp/plugins",
					},
					"installLocation": marketplacePath,
				},
			})

			env.CreateMarketplaceIndex(marketplacePath, "acme-marketplace", []map[string]string{
				{"name": "other-plugin", "description": "Other", "version": "1.0.0"},
			})
		})

		It("shows error when plugin not in marketplace", func() {
			result := env.Run("plugin", "show", "nonexistent@acme-marketplace")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
			Expect(result.Stderr).To(ContainSubstring("plugin browse"))
		})
	})
})
