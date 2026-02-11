// ABOUTME: Acceptance tests for 'current' keyword handling in profile commands
// ABOUTME: Tests that 'current' is reserved and shows/refers to the active profile
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
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

		Context("when local scope profile takes precedence over user scope", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("local-precedence-test")

				// Create user-scope profile
				env.CreateProfile(&profile.Profile{
					Name:        "user-profile",
					Description: "User scope profile",
					Plugins:     []string{"user-plugin@marketplace"},
				})
				env.SetActiveProfile("user-profile")

				// Create and apply local-scope profile (should take precedence)
				env.CreateProfile(&profile.Profile{
					Name:        "local-profile",
					Description: "Local scope profile",
					Plugins:     []string{"local-plugin@marketplace"},
				})
				result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")
				Expect(result.ExitCode).To(Equal(0))
			})

			It("shows the local-scope profile, not user-scope", func() {
				result := env.RunInDir(projectDir, "profile", "show", "current")

				Expect(result.ExitCode).To(Equal(0))
				// Should show local-scope profile
				Expect(result.Stdout).To(ContainSubstring("local-profile"))
				Expect(result.Stdout).To(ContainSubstring("Local scope profile"))
				Expect(result.Stdout).To(ContainSubstring("local-plugin@marketplace"))
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
