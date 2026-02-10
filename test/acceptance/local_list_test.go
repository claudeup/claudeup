// ABOUTME: Acceptance tests for local list command
// ABOUTME: Tests output formatting, summary mode, and empty library guidance
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
			Expect(result.Stdout).To(ContainSubstring("No local items found"))
			Expect(result.Stdout).To(ContainSubstring("claudeup local install"))
		})

		It("does not show global message for specific category", func() {
			result := env.Run("local", "list", "rules")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("(empty)"))
			Expect(result.Stdout).NotTo(ContainSubstring("No local items found"))
		})
	})

	Context("with items in library", func() {
		BeforeEach(func() {
			// Create rule items in local storage
			rulesDir := filepath.Join(env.ClaudeupDir, "local", "rules")
			Expect(os.MkdirAll(rulesDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(rulesDir, "enabled-rule.md"), []byte("# Enabled"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(rulesDir, "disabled-rule.md"), []byte("# Disabled"), 0644)).To(Succeed())

			// Enable one rule via enabled.json
			enabledJSON := `{"rules":{"enabled-rule.md":true,"disabled-rule.md":false}}`
			Expect(os.WriteFile(filepath.Join(env.ClaudeupDir, "enabled.json"), []byte(enabledJSON), 0644)).To(Succeed())
		})

		It("does not show the empty library message", func() {
			result := env.Run("local", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("No local items found"))
		})

		It("shows checkmark for enabled items in specific category", func() {
			result := env.Run("local", "list", "rules")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`✓.*enabled-rule\.md`))
		})

		It("shows muted dot for disabled items in specific category", func() {
			result := env.Run("local", "list", "rules")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(MatchRegexp(`·.*disabled-rule\.md`))
		})

		It("does not use old * and x markers", func() {
			result := env.Run("local", "list", "rules")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(MatchRegexp(`\s+\*\s+\w`))
			Expect(result.Stdout).NotTo(MatchRegexp(`\s+x\s+\w`))
		})

		It("does not show the empty library message when filters exclude all items", func() {
			result := env.Run("local", "list", "--enabled")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("No local items found"))
		})

		Context("summary mode (default for all categories)", func() {
			It("shows summary counts instead of individual items", func() {
				result := env.Run("local", "list")

				Expect(result.ExitCode).To(Equal(0))
				// Should show count summary for rules category
				Expect(result.Stdout).To(MatchRegexp(`rules/:\s+2 items \(1 enabled\)`))
			})

			It("does not list individual items", func() {
				result := env.Run("local", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("enabled-rule.md"))
				Expect(result.Stdout).NotTo(ContainSubstring("disabled-rule.md"))
			})

			It("skips empty categories in summary", func() {
				result := env.Run("local", "list")

				Expect(result.ExitCode).To(Equal(0))
				// agents/ has no items, should not appear
				Expect(result.Stdout).NotTo(ContainSubstring("agents/"))
			})
		})

		Context("full listing", func() {
			It("shows individual items with --full flag", func() {
				result := env.Run("local", "list", "--full")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(MatchRegexp(`✓.*enabled-rule\.md`))
				Expect(result.Stdout).To(MatchRegexp(`·.*disabled-rule\.md`))
			})

			It("shows individual items for specific category", func() {
				result := env.Run("local", "list", "rules")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(MatchRegexp(`✓.*enabled-rule\.md`))
				Expect(result.Stdout).To(MatchRegexp(`·.*disabled-rule\.md`))
			})

			It("shows individual items with --enabled filter", func() {
				result := env.Run("local", "list", "--enabled")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(MatchRegexp(`✓.*enabled-rule\.md`))
				Expect(result.Stdout).NotTo(ContainSubstring("disabled-rule.md"))
			})

			It("shows individual items with --disabled filter", func() {
				result := env.Run("local", "list", "--disabled")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(MatchRegexp(`·.*disabled-rule\.md`))
				Expect(result.Stdout).NotTo(ContainSubstring("enabled-rule.md"))
			})
		})
	})
})
