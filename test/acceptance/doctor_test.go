// ABOUTME: Acceptance tests for doctor command
// ABOUTME: Tests diagnostic output including missing plugin detection and recommendations
package acceptance

import (
	"github.com/claudeup/claudeup/v5/test/helpers"
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
			// Create empty plugin registry (no plugins installed)
			env.CreateInstalledPlugins(map[string]interface{}{})
			// Enable a plugin in settings that is NOT installed
			env.CreateSettings(map[string]bool{
				"missing-plugin@test-marketplace": true,
			})
		})

		It("reports missing plugin with scope and install recommendation", func() {
			result := env.Run("doctor")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("1 plugin enabled but not installed"))
			Expect(result.Stdout).To(ContainSubstring("missing-plugin@test-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("(user)"))
			Expect(result.Stdout).To(ContainSubstring("claude plugin install --scope <scope> <plugin-name>"))
			Expect(result.Stdout).To(ContainSubstring("claudeup profile clean --<scope> <plugin-name>"))
		})
	})
})
