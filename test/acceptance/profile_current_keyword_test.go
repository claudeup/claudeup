// ABOUTME: Acceptance tests for 'current' keyword handling in profile commands
// ABOUTME: Tests that 'current' is reserved and shows/refers to the active profile
package acceptance

import (
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
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
		Context("when an active profile is set", func() {
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
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name: "source-profile",
			})
		})

		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "create", "current", "--from", "source-profile")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

	Describe("profile use current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "use", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})
})
