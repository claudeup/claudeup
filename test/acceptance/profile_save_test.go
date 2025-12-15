// ABOUTME: Acceptance tests for profile save command
// ABOUTME: Tests CLI behavior for saving profiles from current state
package acceptance

import (
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile save", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("with a new profile name", func() {
		It("creates the profile from current state", func() {
			result := env.Run("profile", "save", "my-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
			Expect(env.ProfileExists("my-profile")).To(BeTrue())
		})

		It("marks the saved profile as active", func() {
			result := env.Run("profile", "save", "my-profile")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(Equal("my-profile"))
		})
	})

	Context("with an existing profile name", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "existing",
				Description: "Existing profile",
			})
		})

		It("prompts for confirmation and cancels on 'n'", func() {
			result := env.RunWithInput("n\n", "profile", "save", "existing")

			Expect(result.Stdout).To(ContainSubstring("Overwrite?"))
			Expect(result.Stdout).To(ContainSubstring("Cancelled"))
		})

		It("overwrites when user confirms with 'y'", func() {
			result := env.RunWithInput("y\n", "profile", "save", "existing")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
		})

		It("overwrites without prompting when -y flag is used", func() {
			result := env.Run("profile", "save", "existing", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Saved profile"))
		})

		It("marks the overwritten profile as active", func() {
			result := env.RunWithInput("y\n", "profile", "save", "existing")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.GetActiveProfile()).To(Equal("existing"))
		})
	})

	Context("without a profile name", func() {
		Context("when an active profile is set", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{Name: "active-one"})
				env.SetActiveProfile("active-one")
			})

			It("saves to the active profile", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Saving to active profile"))
				Expect(result.Stdout).To(ContainSubstring("active-one"))
			})

			It("keeps the profile as active", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).To(Equal(0))
				Expect(env.GetActiveProfile()).To(Equal("active-one"))
			})
		})

		Context("when no active profile is set", func() {
			It("returns an error", func() {
				result := env.Run("profile", "save")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("no profile name"))
			})
		})
	})
})
