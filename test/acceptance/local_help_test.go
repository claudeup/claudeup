// ABOUTME: Acceptance tests for local command help text
// ABOUTME: Verifies help output clarifies import vs install vs import-all
package acceptance

import (
	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("local help", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	It("includes a section explaining how to add items", func() {
		result := env.Run("local", "--help")

		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("Adding items"))
	})

	It("explains that install copies from external paths", func() {
		result := env.Run("local", "--help")

		Expect(result.ExitCode).To(Equal(0))
		// The description should explain install copies from external sources
		Expect(result.Stdout).To(MatchRegexp(`(?i)install.*cop(y|ies).*external`))
	})

	It("explains that import moves from active directories", func() {
		result := env.Run("local", "--help")

		Expect(result.ExitCode).To(Equal(0))
		// The description should explain import moves from active dirs
		Expect(result.Stdout).To(MatchRegexp(`(?i)import\s.*mov(e|es).*active`))
	})
})
