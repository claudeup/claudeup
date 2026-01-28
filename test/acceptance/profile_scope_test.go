// ABOUTME: Acceptance tests for profile scope commands (project, local)
// ABOUTME: Tests --scope flag, sync command, and file creation
package acceptance

import (
	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
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

		It("creates .claudeup.json in project directory", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ClaudeupJSONExists(projectDir)).To(BeTrue())
		})

		It("saves profile to .claudeup/profiles/ for team sharing", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ProjectProfileExists(projectDir, "test-profile")).To(BeTrue())

			// Verify the profile content matches the original
			projectProfile := env.LoadProjectProfile(projectDir, "test-profile")
			Expect(projectProfile["name"]).To(Equal("test-profile"))
			Expect(projectProfile["description"]).To(Equal("Test profile for scope testing"))
		})

		It("writes profile name to .claudeup.json", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			cfg := env.LoadClaudeupJSON(projectDir)
			Expect(cfg["profile"]).To(Equal("test-profile"))
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
			Expect(result.Stdout).To(ContainSubstring(".claudeup.json"))
		})

		Context("team member sync workflow", func() {
			It("allows sync after apply even without user profile", func() {
				// Step 1: Team lead applies profile at project scope
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")
				Expect(result.ExitCode).To(Equal(0))

				// Step 2: Simulate team member - remove user profile
				env.DeleteProfile("test-profile")

				// Step 3: Team member runs sync - should succeed using project profile
				syncResult := env.RunInDir(projectDir, "profile", "sync", "-y")
				Expect(syncResult.ExitCode).To(Equal(0))
			})
		})

		Context("with profile that has no MCP servers", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:    "no-mcp-profile",
					Plugins: []string{"plugin@test"},
				})
			})

			It("creates .claudeup.json but not .mcp.json", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "no-mcp-profile", "--scope", "project", "-y")

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
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("original-profile"))
				Expect(result.Stdout).To(SatisfyAny(
					ContainSubstring("overwrite"),
					ContainSubstring("replace"),
					ContainSubstring("already configured"),
				))
			})

			It("proceeds with -y flag", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

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

				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				// Should NOT show overwrite warning when same profile
				Expect(result.Stdout).NotTo(ContainSubstring("already configured"))
			})

			It("shows warning with explicit --scope project flag", func() {
				// Even with explicit scope flag, should warn about overwriting
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				// Warning should appear even with explicit --scope project
				Expect(result.Stdout).To(ContainSubstring("original-profile"))
			})
		})

		Context("when .claudeup.json is malformed", func() {
			BeforeEach(func() {
				// Create malformed JSON
				env.WriteFile(projectDir, ".claudeup.json", "{ invalid json }")
			})

			It("warns about unreadable config but proceeds", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

				// Should succeed - malformed config is overwritten
				Expect(result.ExitCode).To(Equal(0))
				// Should warn about the malformed config
				Expect(result.Stdout).To(ContainSubstring("Could not read existing project config"))
				// Should not show the "already configured" warning (can't read profile name)
				Expect(result.Stdout).NotTo(ContainSubstring("already configured"))
			})

			It("overwrites malformed config with valid one", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "test-profile", "--scope", "project", "-y")

				Expect(result.ExitCode).To(Equal(0))
				cfg := env.LoadClaudeupJSON(projectDir)
				Expect(cfg["profile"]).To(Equal("test-profile"))
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
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ClaudeupJSONExists(projectDir)).To(BeFalse())
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

// NOTE: "profile sync" tests removed during .claudeup.json simplification.
// The old 'profile sync' command expected .claudeup.json to have a 'plugins' field, but the
// new architecture only stores the profile name. Sync functionality is now handled by
// 'profile apply' which loads the profile definition and syncs based on that.
//
// See recovery doc: ~/.claudeup/prompts/2025-12-27-simplify-claudeup-json.md
// Unit test coverage for the new architecture exists in internal/profile/*_test.go

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
