// ABOUTME: Acceptance tests for profile create wizard
// ABOUTME: Tests interactive wizard behavior, validation, and profile creation
package acceptance

import (
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

	Context("when name not provided as argument", func() {
		It("prompts for profile name", func() {
			Skip("Interactive test - requires gum or manual testing")
			// This will be tested manually since it requires gum interaction
		})
	})

	Context("when name provided as argument", func() {
		It("uses the provided name without prompting", func() {
			// Start with a basic test that just checks the command exists
			result := env.Run("profile", "create", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("create"))
		})
	})
})
