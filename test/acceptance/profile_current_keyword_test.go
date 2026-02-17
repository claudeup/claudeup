// ABOUTME: Acceptance tests for 'current' keyword handling in profile commands
// ABOUTME: Tests that 'current' is reserved and 'show current' delegates to live status view
package acceptance

import (
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("'current' keyword handling", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("profile show current", func() {
		Context("with plugins at user scope", func() {
			BeforeEach(func() {
				env.CreateSettings(map[string]bool{
					"plugin-a@marketplace": true,
				})
			})

			It("shows live effective configuration (same as status)", func() {
				result := env.Run("profile", "show", "current")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Effective configuration"))
				Expect(result.Stdout).To(ContainSubstring("plugin-a@marketplace"))
			})
		})

		Context("with plugins at multiple scopes", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = env.ProjectDir("multi-scope-test")

				// User-scope plugins
				env.CreateSettings(map[string]bool{
					"user-plugin@marketplace": true,
				})

				// Local-scope plugins (via settings.local.json in project dir)
				env.CreateLocalScopeSettings(projectDir, map[string]bool{
					"local-plugin@marketplace": true,
				})
			})

			It("shows plugins from all scopes", func() {
				result := env.RunInDir(projectDir, "profile", "show", "current")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Effective configuration"))
				// Both scopes' plugins should appear
				Expect(result.Stdout).To(ContainSubstring("user-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("local-plugin@marketplace"))
			})
		})

		Context("with no plugins at any scope", func() {
			It("succeeds with empty message instead of erroring", func() {
				result := env.Run("profile", "show", "current")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("No plugins"))
			})
		})
	})

	Describe("profile apply current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "apply", "current", "--user")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

	Describe("profile save current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "save", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

	Describe("profile create current", func() {
		It("rejects 'current' as a reserved name", func() {
			result := env.Run("profile", "create", "current")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})

})
