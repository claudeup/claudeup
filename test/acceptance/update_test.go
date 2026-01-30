// ABOUTME: Acceptance tests for update command (self-update)
// ABOUTME: Tests CLI binary self-update behavior
package acceptance

import (
	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("update", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("help output", func() {
		It("shows usage information", func() {
			result := env.Run("update", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Update the claudeup CLI"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})
	})
})
