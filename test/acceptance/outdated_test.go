// ABOUTME: Acceptance tests for outdated command
// ABOUTME: Tests display of available updates for CLI and plugins
package acceptance

import (
	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("outdated", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no marketplaces or plugins", func() {
		It("shows CLI section", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("CLI"))
		})

		It("shows Marketplaces section", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Marketplaces"))
		})

		It("shows Plugins section", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins"))
		})

		It("shows suggested commands footer", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claudeup update"))
			Expect(result.Stdout).To(ContainSubstring("claudeup upgrade"))
		})
	})

	Describe("help output", func() {
		It("shows usage information", func() {
			result := env.Run("outdated", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Check for available updates"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})
	})
})
