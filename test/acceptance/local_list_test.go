// ABOUTME: Acceptance tests for local list command
// ABOUTME: Tests output formatting and empty library guidance
package acceptance

import (
	"os"
	"path/filepath"

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

		It("does not show global message for specific category", func() {
			result := env.Run("local", "list", "rules")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("(empty)"))
			Expect(result.Stdout).NotTo(ContainSubstring("No items in library"))
		})
	})

	Context("with items in library", func() {
		BeforeEach(func() {
			// Create a rule item in the local storage
			rulesDir := filepath.Join(env.ClaudeupDir, "local", "rules")
			Expect(os.MkdirAll(rulesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(rulesDir, "test-rule.md"), []byte("# Test"), 0644)).To(Succeed())
		})

		It("does not show the empty library message", func() {
			result := env.Run("local", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("No items in library"))
			Expect(result.Stdout).To(ContainSubstring("test-rule.md"))
		})

		It("does not show the empty library message when filters exclude all items", func() {
			// All items are disabled by default; --enabled should show no items
			// but should NOT show the "No items in library" message
			result := env.Run("local", "list", "--enabled")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("No items in library"))
		})
	})
})
