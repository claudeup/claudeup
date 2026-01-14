// ABOUTME: Acceptance tests for profile create command
// ABOUTME: Tests CLI behavior for interactive wizard to create new profiles
package acceptance

import (
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile create", func() {
	// Note: Old tests that expected `create` to clone profiles have been removed.
	// The clone functionality now lives in `profile clone` command.
	// These tests now verify the interactive wizard behavior.

	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("wizard behavior", func() {
		It("starts wizard and fails gracefully in non-interactive mode", func() {
			result := env.Run("profile", "create", "new-profile")

			Expect(result.ExitCode).NotTo(Equal(0))
			// Wizard starts but fails due to lack of TTY
			Expect(result.Stderr).To(ContainSubstring("failed to select marketplaces"))
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
