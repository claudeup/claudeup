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

		It("shows extensions under scope section", func() {
			result := env.Run("profile", "show", "multi-scope")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Extensions:"))
			Expect(result.Stdout).To(ContainSubstring("test-runner/test-runner.md"))
			Expect(result.Stdout).To(ContainSubstring("commit.md"))
			Expect(result.Stdout).To(ContainSubstring("golang"))
		})

		It("shows scope headers instead of inline scope tags", func() {
			result := env.Run("profile", "show", "multi-scope")

			Expect(result.ExitCode).To(Equal(0))
			// Scope-grouped format uses scope headers
			Expect(result.Stdout).To(ContainSubstring("User scope"))
			Expect(result.Stdout).To(ContainSubstring("Project scope"))
			// Should NOT have inline [scope] tags
			Expect(result.Stdout).NotTo(ContainSubstring("[user]"))
			Expect(result.Stdout).NotTo(ContainSubstring("[project]"))
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

	Context("with legacy profile having all section types", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:    "legacy-full",
				Plugins: []string{"full-plugin@marketplace"},
				MCPServers: []profile.MCPServer{
					{Name: "full-mcp", Command: "full-cmd"},
				},
				Extensions: &profile.ExtensionSettings{
					Agents: []string{"full-agent.md"},
					Rules:  []string{"full-rule.md"},
				},
			})
		})

		It("separates sections with blank lines", func() {
			result := env.Run("profile", "show", "legacy-full")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins:"))
			Expect(result.Stdout).To(ContainSubstring("full-plugin@marketplace"))
			Expect(result.Stdout).To(ContainSubstring("MCP Servers:"))
			Expect(result.Stdout).To(ContainSubstring("full-mcp"))
			Expect(result.Stdout).To(ContainSubstring("Extensions:"))
			Expect(result.Stdout).To(ContainSubstring("full-agent.md"))
			Expect(result.Stdout).To(ContainSubstring("full-rule.md"))
		})
	})

	Context("with MCP server secrets", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name: "mcp-secrets",
				MCPServers: []profile.MCPServer{
					{
						Name:    "secret-server",
						Command: "npx",
						Secrets: map[string]profile.SecretRef{
							"API_KEY": {
								Description: "API key for service",
								Sources:     []profile.SecretSource{{Type: "env", Key: "API_KEY"}},
							},
							"DB_PASS": {
								Description: "Database password",
								Sources:     []profile.SecretSource{{Type: "env", Key: "DB_PASS"}},
							},
						},
					},
				},
			})
		})

		It("renders requires lines for secrets in sorted order", func() {
			result := env.Run("profile", "show", "mcp-secrets")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("secret-server (npx)"))
			Expect(result.Stdout).To(ContainSubstring("requires: API_KEY"))
			Expect(result.Stdout).To(ContainSubstring("requires: DB_PASS"))
		})
	})
})
