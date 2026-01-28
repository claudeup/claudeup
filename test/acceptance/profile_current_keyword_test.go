// ABOUTME: Acceptance tests for 'current' keyword handling in profile commands
// ABOUTME: Tests that 'current' is reserved and shows/refers to the active profile
package acceptance

import (
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("'current' keyword handling", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("profile show current", func() {
		Context("when an active profile is set at user scope", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:        "my-active-profile",
					Description: "This is my active profile",
					Plugins:     []string{"plugin-a@marketplace", "plugin-b@marketplace"},
					Marketplaces: []profile.Marketplace{
						{Source: "github", Repo: "test/marketplace"},
					},
				})
				env.SetActiveProfile("my-active-profile")
			})

			It("shows the active profile contents", func() {
				result := env.Run("profile", "show", "current")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("my-active-profile"))
				Expect(result.Stdout).To(ContainSubstring("This is my active profile"))
				Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("test/marketplace"))
			})
		})

		Context("when project scope profile takes precedence over user scope", func() {
			BeforeEach(func() {
				// Create user-scope profile
				env.CreateProfile(&profile.Profile{
					Name:        "user-profile",
					Description: "User scope profile",
					Plugins:     []string{"user-plugin@marketplace"},
				})
				env.SetActiveProfile("user-profile")

				// Create project-scope profile (should take precedence)
				env.CreateProfile(&profile.Profile{
					Name:        "project-profile",
					Description: "Project scope profile",
					Plugins:     []string{"project-plugin@marketplace"},
				})
				env.CreateClaudeupJSON(env.TempDir, map[string]interface{}{
					"version": "1",
					"profile": "project-profile",
				})
			})

			It("shows the project-scope profile, not user-scope", func() {
				// Run from env.TempDir where .claudeup.json was created
				result := env.RunInDir(env.TempDir, "profile", "show", "current")

				Expect(result.ExitCode).To(Equal(0))
				// Should show project-scope profile
				Expect(result.Stdout).To(ContainSubstring("project-profile"))
				Expect(result.Stdout).To(ContainSubstring("Project scope profile"))
				Expect(result.Stdout).To(ContainSubstring("project-plugin@marketplace"))
				// Should NOT show user-scope profile
				Expect(result.Stdout).NotTo(ContainSubstring("user-profile"))
				Expect(result.Stdout).NotTo(ContainSubstring("user-plugin@marketplace"))
			})
		})

		Context("when no active profile is set", func() {
			It("returns an error", func() {
				result := env.Run("profile", "show", "current")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("no active profile"))
			})
		})
	})

	Describe("profile save current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "save", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

	Describe("profile create current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "create", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

	Describe("profile apply current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "apply", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})
})
