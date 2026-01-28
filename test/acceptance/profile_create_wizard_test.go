// ABOUTME: Acceptance tests for profile create wizard
// ABOUTME: Tests interactive wizard behavior, validation, and profile creation
package acceptance

import (
	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
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

	Context("auto-apply behavior", func() {
		// These tests document expected behavior that requires manual verification
		// because the wizard requires interactive TTY input

		It("applies at user scope to avoid overwriting project config", func() {
			Skip(`Manual verification required:
1. cd to a directory with existing .claudeup.json
2. Run: claudeup profile create test-profile
3. Select marketplaces/plugins in wizard
4. Answer 'Y' to 'Apply this profile now?'
5. Verify: Profile should apply at user scope
6. Verify: Original .claudeup.json should be preserved
7. Verify: Message should appear noting existing project config`)
		})

		It("informs user about existing project config after applying", func() {
			Skip(`Manual verification required:
When applying after create in a directory with .claudeup.json,
should show message like:
  ℹ Applied profile "my-profile" at user scope.
    Note: This directory has an existing project config ("original-profile").
    Use 'claudeup profile apply my-profile --scope project' to apply at project level.`)
		})
	})
})
