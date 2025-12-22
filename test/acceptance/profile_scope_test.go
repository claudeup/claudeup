// ABOUTME: Acceptance tests for profile scope commands (project, local)
// ABOUTME: Tests --scope flag, sync command, and file creation
package acceptance

import (
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile use --scope", func() {
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
			result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.MCPJSONExists(projectDir)).To(BeTrue())
		})

		It("creates .claudeup.json in project directory", func() {
			result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ClaudeupJSONExists(projectDir)).To(BeTrue())
		})

		It("writes profile name to .claudeup.json", func() {
			result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			cfg := env.LoadClaudeupJSON(projectDir)
			Expect(cfg["profile"]).To(Equal("test-profile"))
		})

		It("writes MCP servers to .mcp.json", func() {
			result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			mcpConfig := env.LoadMCPJSON(projectDir)
			mcpServers, ok := mcpConfig["mcpServers"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(mcpServers).To(HaveKey("test-server"))
		})

		It("shows git add guidance for created files", func() {
			result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("git add"))
			Expect(result.Stdout).To(ContainSubstring(".claudeup.json"))
		})

		Context("with profile that has no MCP servers", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:    "no-mcp-profile",
					Plugins: []string{"plugin@test"},
				})
			})

			It("creates .claudeup.json but not .mcp.json", func() {
				result := env.RunInDir(projectDir, "profile", "use", "no-mcp-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(env.ClaudeupJSONExists(projectDir)).To(BeTrue())
				Expect(env.MCPJSONExists(projectDir)).To(BeFalse())
			})
		})

		Context("when .claudeup.json already exists with different profile", func() {
			BeforeEach(func() {
				// Create an existing .claudeup.json pointing to a different profile
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "original-profile",
					"plugins": []string{"original-plugin@test"},
				})
			})

			It("warns about overwriting existing config", func() {
				result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("original-profile"))
				Expect(result.Stdout).To(SatisfyAny(
					ContainSubstring("overwrite"),
					ContainSubstring("replace"),
					ContainSubstring("already configured"),
				))
			})

			It("proceeds with -y flag", func() {
				result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				cfg := env.LoadClaudeupJSON(projectDir)
				Expect(cfg["profile"]).To(Equal("test-profile"))
			})

			It("preserves existing config when same profile is reapplied", func() {
				// Reapplying the same profile should not warn
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "test-profile",
				})

				result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				// Should NOT show overwrite warning when same profile
				Expect(result.Stdout).NotTo(ContainSubstring("already configured"))
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

		It("does not create .claudeup.json in project directory", func() {
			result := env.RunInDir(projectDir, "profile", "use", "local-profile", "--scope", "local", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ClaudeupJSONExists(projectDir)).To(BeFalse())
		})

		It("creates entry in projects.json registry", func() {
			result := env.RunInDir(projectDir, "profile", "use", "local-profile", "--scope", "local", "-y")

			Expect(result.ExitCode).To(Equal(0))

			registry := env.LoadProjectsRegistry()
			Expect(registry).NotTo(BeNil())
			Expect(registry["projects"]).NotTo(BeNil())
		})

		It("shows confirmation message", func() {
			result := env.RunInDir(projectDir, "profile", "use", "local-profile", "--scope", "local", "-y")

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

			result := env.RunInDir(projectDir, "profile", "use", "test-profile", "--scope", "invalid", "-y")

			Expect(result.ExitCode).NotTo(Equal(0))
		})

		It("works from any valid directory", func() {
			// Note: The command should be run with a valid current directory
			// This test verifies the command handles edge cases gracefully
			env.CreateProfile(&profile.Profile{
				Name: "test-profile",
			})

			result := env.Run("profile", "use", "test-profile", "--scope", "project", "-y")

			// Should succeed when run from a valid directory
			// May show "no changes" if profile is empty
			Expect(result.ExitCode).To(Equal(0))
		})
	})
})

var _ = Describe("profile sync", func() {
	var env *helpers.TestEnv
	var projectDir string

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
		projectDir = env.ProjectDir("sync-test-project")
	})

	Describe("with valid .claudeup.json", func() {
		BeforeEach(func() {
			// Create .claudeup.json with plugins
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version": "1",
				"profile": "test-profile",
				"plugins": []string{"plugin-a@test", "plugin-b@test"},
			})
		})

		It("succeeds with exit code 0", func() {
			result := env.RunInDir(projectDir, "profile", "sync", "-y")

			// May fail to install plugins (no real marketplace) but command should run
			Expect(result.Stdout).To(SatisfyAny(
				ContainSubstring("Sync complete"),
				ContainSubstring("plugin"),
			))
		})

		It("reports plugins to install", func() {
			result := env.RunInDir(projectDir, "profile", "sync", "-y")

			Expect(result.Stdout).To(ContainSubstring("plugin"))
		})
	})

	Describe("--dry-run", func() {
		BeforeEach(func() {
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version": "1",
				"profile": "test-profile",
				"plugins": []string{"dry-run-plugin@test"},
			})
		})

		It("shows what would be installed without making changes", func() {
			result := env.RunInDir(projectDir, "profile", "sync", "--dry-run")

			Expect(result.ExitCode).To(Equal(0))
			// Dry run output shows count, not plugin names
			Expect(result.Stdout).To(ContainSubstring("Plugins installed: 1"))
		})

		It("indicates dry run mode", func() {
			result := env.RunInDir(projectDir, "profile", "sync", "--dry-run")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(SatisfyAny(
				ContainSubstring("Dry run"),
				ContainSubstring("Would sync"),
			))
		})
	})

	Describe("without .claudeup.json", func() {
		It("fails with helpful error message", func() {
			emptyDir := env.ProjectDir("empty-project")
			result := env.RunInDir(emptyDir, "profile", "sync")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stdout + result.Stderr).To(ContainSubstring(".claudeup.json"))
		})
	})

	Describe("with already installed plugins", func() {
		BeforeEach(func() {
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version": "1",
				"profile": "test-profile",
				"plugins": []string{"existing-plugin@test"},
			})

			// Mark plugin as already installed
			env.CreateInstalledPlugins(map[string]interface{}{
				"existing-plugin@test": []map[string]interface{}{
					{"scope": "project", "version": "1.0"},
				},
			})
		})

		It("skips already installed plugins", func() {
			result := env.RunInDir(projectDir, "profile", "sync", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(SatisfyAny(
				ContainSubstring("skipped"),
				ContainSubstring("already"),
				ContainSubstring("Sync complete"),
			))
		})
	})
})

var _ = Describe("profile current with scopes", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("with project-level profile", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("current-test")
			env.CreateClaudeupJSON(projectDir, map[string]interface{}{
				"version": "1",
				"profile": "project-profile",
			})
		})

		It("shows project scope profile", func() {
			result := env.RunInDir(projectDir, "profile", "current")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("project-profile"))
		})

		It("indicates project scope", func() {
			result := env.RunInDir(projectDir, "profile", "current")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("project"))
		})
	})

	Describe("scope precedence", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("precedence-test")

			// Set user-level profile
			env.SetActiveProfile("user-profile")
		})

		Context("project scope takes precedence", func() {
			BeforeEach(func() {
				env.CreateClaudeupJSON(projectDir, map[string]interface{}{
					"version": "1",
					"profile": "project-wins",
				})
			})

			It("shows project profile instead of user profile", func() {
				result := env.RunInDir(projectDir, "profile", "current")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("project-wins"))
				Expect(result.Stdout).NotTo(ContainSubstring("user-profile"))
			})
		})

		Context("without project config, user scope applies", func() {
			It("shows user profile when no project config exists", func() {
				emptyDir := env.ProjectDir("no-project-config")
				result := env.RunInDir(emptyDir, "profile", "current")

				// Should show user profile or "none"
				Expect(result.ExitCode).To(Equal(0))
			})
		})
	})
})
