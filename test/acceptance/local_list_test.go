// ABOUTME: Acceptance tests for local list command
// ABOUTME: Tests output formatting and empty library guidance
package acceptance

import (
	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("local list", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Context("with empty library", func() {
		It("shows a helpful message", func() {
			result := env.Run("local", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No items in library"))
			Expect(result.Stdout).To(ContainSubstring("claudeup local install"))
		})
	})
})
