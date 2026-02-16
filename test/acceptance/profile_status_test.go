// ABOUTME: Acceptance tests for profile status untracked scope hints
// ABOUTME: Verifies hints appear for scopes with settings but no tracked profile
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

	Describe("untracked scope hints", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("status-hint-test")
			env.CreateProfile(&profile.Profile{
				Name:        "my-profile",
				Description: "Test profile",
				Plugins:     []string{"some-plugin@marketplace"},
			})
			env.SetActiveProfile("my-profile")
		})

		Context("with untracked project-scope settings", func() {
			BeforeEach(func() {
				env.CreateProjectScopeSettings(projectDir, map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
				})
			})

			It("shows warning about untracked project scope", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("project"))
				Expect(result.Stdout).To(ContainSubstring("2 plugins"))
				Expect(result.Stdout).To(ContainSubstring("no profile tracked"))
			})

			It("shows suggested save command with --apply flag", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("profile save <name> --project --apply"))
			})
		})

		Context("with no untracked settings", func() {
			It("does not show untracked hints", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("no profile tracked"))
			})
		})
	})
})
