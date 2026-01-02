// ABOUTME: Acceptance tests for sandbox --copy-auth flag
// ABOUTME: Tests flag validation and help output (Docker-independent tests)
package acceptance

import (
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sandbox --copy-auth", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("flag validation", func() {
		Context("when --copy-auth is used without --profile", func() {
			It("returns an error", func() {
				result := env.Run("sandbox", "--copy-auth")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("--copy-auth requires --profile"))
			})

			It("explains why profile is required", func() {
				result := env.Run("sandbox", "--copy-auth")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("ephemeral mode has no persistent state"))
			})
		})
	})

	Describe("help output", func() {
		It("shows --copy-auth flag in help", func() {
			result := env.Run("sandbox", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--copy-auth"))
		})

		It("describes --copy-auth purpose", func() {
			result := env.Run("sandbox", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Copy authentication from ~/.claude.json"))
		})
	})
})
