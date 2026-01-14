// ABOUTME: Acceptance tests for profile clone command
// ABOUTME: Tests CLI behavior for cloning profiles from existing ones
package acceptance

import (
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile clone", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("with --from flag", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:        "source",
				Description: "Source profile",
				Plugins:     []string{"plugin-a", "plugin-b"},
			})
		})

		It("creates a copy of the source profile", func() {
			result := env.Run("profile", "clone", "new-profile", "--from", "source")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Created profile"))
			Expect(result.Stdout).To(ContainSubstring("based on"))

			created := env.LoadProfile("new-profile")
			Expect(created.Name).To(Equal("new-profile"))
			Expect(created.Plugins).To(Equal([]string{"plugin-a", "plugin-b"}))
		})

		It("errors when source profile doesn't exist", func() {
			result := env.Run("profile", "clone", "new-profile", "--from", "nonexistent")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})
	})

	Context("with -y flag", func() {
		Context("when active profile is set", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:    "active-source",
					Plugins: []string{"active-plugin"},
				})
				env.SetActiveProfile("active-source")
			})

			It("uses the active profile as source", func() {
				result := env.Run("profile", "clone", "new-from-active", "-y")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Using active profile"))

				created := env.LoadProfile("new-from-active")
				Expect(created.Plugins).To(Equal([]string{"active-plugin"}))
			})
		})

		Context("when no active profile is set", func() {
			It("returns an error", func() {
				result := env.Run("profile", "clone", "new-profile", "-y")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("no active profile"))
			})
		})
	})

	Context("when target profile already exists", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{Name: "existing"})
			env.CreateProfile(&profile.Profile{Name: "source"})
		})

		It("returns an error suggesting profile save", func() {
			result := env.Run("profile", "clone", "existing", "--from", "source")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("already exists"))
			Expect(result.Stderr).To(ContainSubstring("profile save"))
		})
	})
})
