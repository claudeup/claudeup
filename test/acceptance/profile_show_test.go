// ABOUTME: Acceptance tests for profile show command
// ABOUTME: Verifies display of multi-scope profiles with plugins, MCP servers, and extensions
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile show", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("with multi-scope profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "multi-scope",
				Description: "Multi-scope test profile",
				Marketplaces: []profile.Marketplace{
					{Source: "github", Repo: "test/marketplace"},
				},
				PerScope: &profile.PerScopeSettings{
					User: &profile.ScopeSettings{
						Plugins: []string{
							"plugin-a@marketplace",
							"plugin-b@marketplace",
						},
						MCPServers: []profile.MCPServer{
							{Name: "user-mcp", Command: "user-cmd", Scope: "user"},
						},
					},
					Project: &profile.ScopeSettings{
						Plugins: []string{
							"project-plugin@marketplace",
						},
					},
				},
				Extensions: &profile.ExtensionSettings{
					Agents:   []string{"test-runner/test-runner.md"},
					Commands: []string{"commit.md"},
					Skills:   []string{"golang"},
				},
			})
		})

		It("shows plugins from all scopes", func() {
			result := env.Run("profile", "show", "multi-scope")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
			Expect(result.Stdout).To(ContainSubstring("plugin-b@marketplace"))
			Expect(result.Stdout).To(ContainSubstring("project-plugin@marketplace"))
		})

		It("shows MCP servers from all scopes", func() {
			result := env.Run("profile", "show", "multi-scope")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("user-mcp"))
		})

		It("shows extensions", func() {
			result := env.Run("profile", "show", "multi-scope")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("test-runner/test-runner.md"))
			Expect(result.Stdout).To(ContainSubstring("commit.md"))
			Expect(result.Stdout).To(ContainSubstring("golang"))
		})

		It("shows scope labels for plugins", func() {
			result := env.Run("profile", "show", "multi-scope")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("user"))
			Expect(result.Stdout).To(ContainSubstring("project"))
		})
	})

	Context("with legacy profile format", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:    "legacy",
				Plugins: []string{"legacy-plugin@marketplace"},
				MCPServers: []profile.MCPServer{
					{Name: "legacy-mcp", Command: "legacy-cmd"},
				},
			})
		})

		It("shows plugins from legacy format", func() {
			result := env.Run("profile", "show", "legacy")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("legacy-plugin@marketplace"))
			Expect(result.Stdout).To(ContainSubstring("legacy-mcp"))
		})
	})
})
