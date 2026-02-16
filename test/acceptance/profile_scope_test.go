// ABOUTME: Acceptance tests for profile scope commands (project, local)
// ABOUTME: Tests --scope flag and file creation
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile apply --scope", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("--scope project", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("test-project")

			// Create a profile with MCP servers and plugins
			env.CreateProfile(&profile.Profile{
				Name:        "test-profile",
				Description: "Test profile for scope testing",
				Marketplaces: []profile.Marketplace{
					{Source: "github", Repo: "test/marketplace"},
				},
				Plugins: []string{"test-plugin@test-marketplace"},
				MCPServers: []profile.MCPServer{
					{
						Name:    "test-server",
						Command: "node",
						Args:    []string{"server.js"},
					},
				},
			})
		})

		It("creates .mcp.json in project directory", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.MCPJSONExists(projectDir)).To(BeTrue())
		})

		It("does not save profile to project-local .claudeup/profiles/", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ProjectProfileExists(projectDir, "test-profile")).To(BeFalse())
		})

		It("writes MCP servers to .mcp.json", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			mcpConfig := env.LoadMCPJSON(projectDir)
			mcpServers, ok := mcpConfig["mcpServers"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(mcpServers).To(HaveKey("test-server"))
		})

		It("shows git add guidance for created files", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("git add"))
			Expect(result.Stdout).To(ContainSubstring(".claudeup/"))
		})

		Context("team member apply workflow", func() {
			It("allows apply using project profile", func() {
				// Step 1: Team lead applies profile at project scope
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")
				Expect(result.ExitCode).To(Equal(0))

				// Step 2: Team member runs apply with explicit profile name
				applyResult := env.RunInDir(projectDir, "profile", "apply", "test-profile", "-y")
				Expect(applyResult.ExitCode).To(Equal(0))
			})
		})

		Context("with profile that has no MCP servers", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:    "no-mcp-profile",
					Plugins: []string{"plugin@test"},
				})
			})

			It("does not create .mcp.json when profile has no MCP servers", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "no-mcp-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(env.MCPJSONExists(projectDir)).To(BeFalse())
			})
		})
	})

	Describe("--scope local", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("local-test-project")

			env.CreateProfile(&profile.Profile{
				Name:        "local-profile",
				Description: "Test profile for local scope",
				Plugins:     []string{"local-plugin@test"},
			})
		})

		It("creates entry in projects.json registry", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")

			Expect(result.ExitCode).To(Equal(0))

			registry := env.LoadProjectsRegistry()
			Expect(registry).NotTo(BeNil())
			Expect(registry["projects"]).NotTo(BeNil())
		})

		It("shows confirmation message", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))
		})
	})

	Describe("scope validation", func() {
		It("rejects invalid scope value", func() {
			projectDir := env.ProjectDir("invalid-scope")
			env.CreateProfile(&profile.Profile{
				Name: "test-profile",
			})

			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "invalid", "-y")

			Expect(result.ExitCode).NotTo(Equal(0))
		})

		It("works from any valid directory", func() {
			// Note: The command should be run with a valid current directory
			// This test verifies the command handles edge cases gracefully
			env.CreateProfile(&profile.Profile{
				Name: "test-profile",
			})

			result := env.Run("profile", "apply", "test-profile", "--scope", "project", "-y")

			// Should succeed when run from a valid directory
			// May show "no changes" if profile is empty
			Expect(result.ExitCode).To(Equal(0))
		})
	})
})

var _ = Describe("profile current with scopes", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("with local-level profile", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("current-test")

			// Create profile and apply at local scope
			env.CreateProfile(&profile.Profile{
				Name:        "local-profile",
				Description: "Test local profile",
			})

			// Apply at local scope to register in projects.json
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")
			Expect(result.ExitCode).To(Equal(0))
		})

		It("shows local scope profile", func() {
			result := env.RunInDir(projectDir, "profile", "current")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("local-profile"))
		})

		It("indicates local scope", func() {
			result := env.RunInDir(projectDir, "profile", "current")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("local"))
		})
	})

	Describe("scope precedence", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("precedence-test")

			// Set user-level profile
			env.SetActiveProfile("user-profile")
		})

		Context("local scope takes precedence over user", func() {
			BeforeEach(func() {
				// Create and apply local profile
				env.CreateProfile(&profile.Profile{
					Name: "local-wins",
				})
				result := env.RunInDir(projectDir, "profile", "apply", "local-wins", "--scope", "local", "-y")
				Expect(result.ExitCode).To(Equal(0))
			})

			It("shows local profile instead of user profile", func() {
				result := env.RunInDir(projectDir, "profile", "current")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("local-wins"))
			})
		})

		Context("without local config, user scope applies", func() {
			It("shows user profile when no local config exists", func() {
				emptyDir := env.ProjectDir("no-local-config")
				result := env.RunInDir(emptyDir, "profile", "current")

				// Should show user profile or "none"
				Expect(result.ExitCode).To(Equal(0))
			})
		})
	})
})
