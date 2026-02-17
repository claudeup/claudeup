// ABOUTME: Acceptance tests for profile status live effective configuration
// ABOUTME: Verifies status shows live settings across all scopes with tracking annotations
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile status", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("live effective configuration", func() {
		Context("with user-scope plugins only", func() {
			BeforeEach(func() {
				env.CreateSettings(map[string]bool{
					"plugin-a@marketplace":        true,
					"plugin-b@marketplace":        true,
					"disabled-plugin@marketplace": false,
				})
			})

			It("shows user-scope plugins", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("plugin-b@marketplace"))
			})

			It("shows disabled plugins", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Disabled"))
				Expect(result.Stdout).To(ContainSubstring("disabled-plugin@marketplace"))
			})

			Context("with tracked user profile", func() {
				BeforeEach(func() {
					env.CreateProfile(&profile.Profile{
						Name:    "my-profile",
						Plugins: []string{"plugin-a@marketplace"},
					})
					env.SetActiveProfile("my-profile")
				})

				It("shows profile name annotation", func() {
					result := env.Run("profile", "status")

					Expect(result.ExitCode).To(Equal(0))
					Expect(result.Stdout).To(ContainSubstring("profile: my-profile"))
				})
			})

			Context("without tracked profile", func() {
				It("shows untracked annotation", func() {
					result := env.Run("profile", "status")

					Expect(result.ExitCode).To(Equal(0))
					Expect(result.Stdout).To(ContainSubstring("untracked"))
				})
			})
		})

		Context("with multi-scope plugins", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("multi-scope-test")

				// User scope
				env.CreateSettings(map[string]bool{
					"user-plugin@marketplace": true,
				})
				env.SetActiveProfile("user-prof")
				env.CreateProfile(&profile.Profile{
					Name:    "user-prof",
					Plugins: []string{"user-plugin@marketplace"},
				})

				// Project scope
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"proj-plugin@marketplace": true,
				})
			})

			It("shows plugins from both scopes", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("User scope"))
				Expect(result.Stdout).To(ContainSubstring("user-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("Project scope"))
				Expect(result.Stdout).To(ContainSubstring("proj-plugin@marketplace"))
			})

			It("shows untracked for project scope without profile", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(MatchRegexp(`Project scope.*untracked`))
			})
		})

		Context("with no plugins at any scope", func() {
			It("shows empty configuration message", func() {
				result := env.Run("profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("No plugins"))
			})
		})
	})
})
