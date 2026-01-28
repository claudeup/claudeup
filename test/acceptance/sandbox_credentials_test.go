// ABOUTME: Acceptance tests for sandbox credential mounting.
// ABOUTME: Tests --creds and --no-creds CLI behavior with real binary.
package acceptance

import (
	"github.com/claudeup/claudeup/v3/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sandbox credentials", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("--creds flag", func() {
		It("shows credentials in help output", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--creds"))
		})

		It("describes --creds purpose", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Credentials to mount"))
		})
	})

	Describe("--no-creds flag", func() {
		It("shows no-creds in help output", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--no-creds"))
		})

		It("describes --no-creds purpose", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Credentials to exclude"))
		})
	})

	Describe("--sync flag", func() {
		It("shows sync in help output", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--sync"))
		})

		It("describes --sync purpose", func() {
			result := env.Run("sandbox", "--help")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Re-apply profile settings"))
		})
	})
})
