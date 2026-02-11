// ABOUTME: Acceptance tests for plugin show file viewing
// ABOUTME: Tests viewing individual file contents with markdown rendering
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin show file", func() {
	var env *helpers.TestEnv
	var marketplacePath string
	var pluginPath string

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)

		// Create marketplace with plugin containing various files
		marketplacePath = filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "acme-corp")
		pluginPath = filepath.Join(marketplacePath, "plugins", "test-plugin")
		Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

		// Create plugin file structure
		Expect(os.MkdirAll(filepath.Join(pluginPath, "agents"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(pluginPath, "skills", "awesome-skill"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(pluginPath, "hooks"), 0755)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(pluginPath, "agents", "network-engineer.md"),
			[]byte("# Network Engineer\n\nThis agent handles networking tasks."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginPath, "skills", "awesome-skill", "SKILL.md"),
			[]byte("# Awesome Skill\n\nDoes awesome things."), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginPath, "hooks", "pre-check.sh"),
			[]byte("#!/bin/bash\necho 'checking'"), 0644)).To(Succeed())

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

	Describe("viewing a markdown file", func() {
		It("renders markdown content", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace", "agents/network-engineer.md")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Network Engineer"))
			Expect(result.Stdout).To(ContainSubstring("networking tasks"))
		})
	})

	Describe("extension inference", func() {
		It("resolves agents/test without .md extension", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace", "agents/network-engineer")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Network Engineer"))
		})
	})

	Describe("skill directory resolution", func() {
		It("resolves skill directory to SKILL.md", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace", "skills/awesome-skill")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Awesome Skill"))
		})
	})

	Describe("--raw flag", func() {
		It("outputs raw markdown without rendering", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace", "agents/network-engineer", "--raw")

			Expect(result.ExitCode).To(Equal(0))
			// Raw output should contain the markdown syntax characters
			Expect(result.Stdout).To(ContainSubstring("# Network Engineer"))
		})
	})

	Describe("non-markdown file", func() {
		It("displays raw content for non-markdown files", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace", "hooks/pre-check.sh")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("#!/bin/bash"))
			Expect(result.Stdout).To(ContainSubstring("echo 'checking'"))
		})
	})

	Describe("nonexistent file", func() {
		It("shows error for missing file", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace", "agents/nonexistent")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})

	Describe("tree view compatibility", func() {
		It("still shows tree when no file argument", func() {
			result := env.Run("plugin", "show", "test-plugin@acme-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("test-plugin@acme-marketplace"))
			Expect(result.Stdout).To(MatchRegexp(`[├└]──`))
		})
	})
})
