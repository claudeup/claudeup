// ABOUTME: Acceptance tests for profile rename command
// ABOUTME: Tests CLI behavior for renaming profiles
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile rename", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("with valid arguments", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "old-name",
				Description: "My profile",
				Plugins:     []string{"plugin-a@marketplace"},
			})
		})

		It("renames the profile file", func() {
			result := env.Run("profile", "rename", "old-name", "new-name")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Renamed"))
			Expect(env.ProfileExists("new-name")).To(BeTrue())
			Expect(env.ProfileExists("old-name")).To(BeFalse())
		})

		It("updates the name field in the JSON", func() {
			env.Run("profile", "rename", "old-name", "new-name")

			renamed := env.LoadProfile("new-name")
			Expect(renamed.Name).To(Equal("new-name"))
			Expect(renamed.Description).To(Equal("My profile"))
			Expect(renamed.Plugins).To(Equal([]string{"plugin-a@marketplace"}))
		})
	})

	Context("when old-name is the active profile", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{Name: "active-profile"})
			env.SetActiveProfile("active-profile")
		})

		It("updates the active profile config", func() {
			result := env.Run("profile", "rename", "active-profile", "renamed-profile")

			Expect(result.ExitCode).To(Equal(0))

			// Verify active profile was updated by checking profile current
			currentResult := env.Run("profile", "current")
			Expect(currentResult.Stdout).To(ContainSubstring("renamed-profile"))
		})
	})

	Context("when new-name already exists", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{Name: "source"})
			env.CreateProfile(&profile.Profile{Name: "target"})
		})

		It("returns an error without -y flag", func() {
			result := env.Run("profile", "rename", "source", "target")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("already exists"))
		})

		It("overwrites with -y flag", func() {
			result := env.Run("profile", "rename", "source", "target", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ProfileExists("target")).To(BeTrue())
			Expect(env.ProfileExists("source")).To(BeFalse())
		})
	})

	Context("when new-name is 'current'", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{Name: "my-profile"})
		})

		It("rejects the reserved name", func() {
			result := env.Run("profile", "rename", "my-profile", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

	Context("when old-name doesn't exist", func() {
		It("returns an error", func() {
			result := env.Run("profile", "rename", "nonexistent", "new-name")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})

	Context("when renaming a built-in profile", func() {
		It("returns an error", func() {
			result := env.Run("profile", "rename", "default", "my-default")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("built-in"))
		})
	})
})
