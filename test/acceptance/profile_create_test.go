// ABOUTME: Acceptance tests for profile create command
// ABOUTME: Tests CLI behavior for creating profiles by copying existing ones
package acceptance

import (
	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile create", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	// Wizard stub tests - these test the current stub implementation
	// Once wizard is fully implemented, these will need to be rewritten

	Context("wizard stub behavior", func() {
		It("shows wizard stub message", func() {
			result := env.Run("profile", "create", "new-profile")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("wizard implementation in progress"))
		})
	})


	Context("when target profile already exists", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{Name: "existing"})
		})

		It("returns an error suggesting profile save", func() {
			result := env.Run("profile", "create", "existing")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("already exists"))
			Expect(result.Stderr).To(ContainSubstring("profile save"))
		})
	})

	Context("wizard mode", func() {
		It("shows help text", func() {
			result := env.Run("profile", "create", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Interactive wizard"))
		})
	})
})
