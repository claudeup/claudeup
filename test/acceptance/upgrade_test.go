// ABOUTME: Acceptance tests for upgrade command
// ABOUTME: Tests marketplace and plugin update functionality
package acceptance

import (
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("upgrade", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no marketplaces or plugins", func() {
		It("shows up to date message", func() {
			result := env.Run("upgrade")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("up to date"))
		})
	})

	Describe("with positional arguments", func() {
		It("warns about unknown marketplaces", func() {
			result := env.Run("upgrade", "nonexistent-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Unknown target"))
		})

		It("warns about unknown plugins", func() {
			result := env.Run("upgrade", "unknown@marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Unknown target"))
		})
	})

	Describe("help output", func() {
		It("shows usage information", func() {
			result := env.Run("upgrade", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Update installed marketplaces and plugins"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})

		It("shows examples", func() {
			result := env.Run("upgrade", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claudeup upgrade"))
		})
	})
})
