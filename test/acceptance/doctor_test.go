// ABOUTME: Acceptance tests for doctor command
// ABOUTME: Tests diagnostic output including scope-aware recommendations
package acceptance

import (
	"github.com/claudeup/claudeup/v4/internal/profile"
	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("doctor", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("missing plugin recommendations", func() {
		BeforeEach(func() {
			// Create empty plugin registry first (this also creates settings)
			env.CreateInstalledPlugins(map[string]interface{}{})
			// Then enable a plugin in settings that is NOT installed
			env.CreateSettings(map[string]bool{
				"missing-plugin@test-marketplace": true,
			})

			// Create a profile that lists this plugin
			env.CreateProfile(&profile.Profile{
				Name:    "my-profile",
				Plugins: []string{"missing-plugin@test-marketplace"},
			})
		})

		Context("with user-scope active profile", func() {
			BeforeEach(func() {
				env.SetActiveProfile("my-profile")
			})

			It("includes profile name and --scope user in recommendation", func() {
				result := env.Run("doctor")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("claudeup profile apply my-profile --scope user"))
			})
		})

		Context("with local-scope active profile", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("test-project")
				env.RegisterProject(projectDir, "my-profile")
			})

			It("includes profile name and --scope local in recommendation", func() {
				result := env.RunInDir(projectDir, "doctor")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("claudeup profile apply my-profile --scope local"))
			})
		})
	})
})
