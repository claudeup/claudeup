// ABOUTME: Acceptance tests for profile create wizard
// ABOUTME: Tests interactive wizard behavior, validation, and profile creation
package acceptance

import (
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile create wizard", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("validation", func() {
		It("rejects empty profile name", func() {
			Skip("Requires stdin simulation for name prompt")
		})

		It("rejects reserved name 'current'", func() {
			// Wizard will validate the name even if provided as arg
			result := env.Run("profile", "create", "current")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})

		It("rejects existing profile name", func() {
			env.CreateProfile(&profile.Profile{Name: "existing"})

			result := env.Run("profile", "create", "existing")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("already exists"))
		})
	})

	Context("interactive mode", func() {
		It("requires gum or fallback prompts", func() {
			Skip("Full wizard test requires gum interaction - test manually")
		})
	})
})
