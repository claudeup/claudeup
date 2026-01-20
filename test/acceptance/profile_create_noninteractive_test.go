// ABOUTME: Acceptance tests for non-interactive profile create
// ABOUTME: Tests flags mode, file mode, and validation errors
package acceptance

import (
	"github.com/claudeup/claudeup/v2/internal/profile"
	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile create non-interactive", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Context("flags mode", func() {
		It("creates profile with all flags", func() {
			result := env.Run("profile", "create", "test-profile",
				"--description", "Test description",
				"--marketplace", "anthropics/claude-code",
				"--plugin", "plugin-dev@claude-code-plugins",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)
			Expect(result.Stdout).To(ContainSubstring("created successfully"))
			Expect(env.ProfileExists("test-profile")).To(BeTrue())

			// Verify profile contents
			p := env.LoadProfile("test-profile")
			Expect(p.Description).To(Equal("Test description"))
			Expect(p.Marketplaces).To(HaveLen(1))
			Expect(p.Marketplaces[0].Repo).To(Equal("anthropics/claude-code"))
			Expect(p.Plugins).To(HaveLen(1))
			Expect(p.Plugins[0]).To(Equal("plugin-dev@claude-code-plugins"))
		})

		It("creates profile with multiple marketplaces", func() {
			result := env.Run("profile", "create", "multi-market",
				"--description", "Multi marketplace",
				"--marketplace", "anthropics/claude-code",
				"--marketplace", "obra/superpowers-marketplace",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)

			p := env.LoadProfile("multi-market")
			Expect(p.Marketplaces).To(HaveLen(2))
		})

		It("creates profile with multiple plugins", func() {
			result := env.Run("profile", "create", "multi-plugin",
				"--description", "Multi plugin profile",
				"--marketplace", "anthropics/claude-code",
				"--plugin", "plugin-a@marketplace-ref",
				"--plugin", "plugin-b@marketplace-ref",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)

			p := env.LoadProfile("multi-plugin")
			Expect(p.Plugins).To(HaveLen(2))
			Expect(p.Plugins).To(ContainElements("plugin-a@marketplace-ref", "plugin-b@marketplace-ref"))
		})

		It("creates profile with description and marketplace only (no plugins)", func() {
			result := env.Run("profile", "create", "no-plugins",
				"--description", "Profile without plugins",
				"--marketplace", "anthropics/claude-code",
			)
			Expect(result.ExitCode).To(Equal(0), "stderr: %s", result.Stderr)

			p := env.LoadProfile("no-plugins")
			Expect(p.Description).To(Equal("Profile without plugins"))
			Expect(p.Marketplaces).To(HaveLen(1))
			Expect(p.Plugins).To(BeEmpty())
		})

		It("fails without description in flags mode", func() {
			result := env.Run("profile", "create", "no-desc",
				"--marketplace", "owner/repo",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("description is required"))
		})

		It("fails without marketplaces in flags mode", func() {
			result := env.Run("profile", "create", "no-market",
				"--description", "Test",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("at least one marketplace is required"))
		})

		It("fails with invalid marketplace format", func() {
			result := env.Run("profile", "create", "bad-market",
				"--description", "Test",
				"--marketplace", "invalid",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid marketplace format"))
		})

		It("fails with invalid plugin format", func() {
			result := env.Run("profile", "create", "bad-plugin",
				"--description", "Test",
				"--marketplace", "owner/repo",
				"--plugin", "no-at-sign",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid plugin format"))
		})

		It("fails when profile already exists", func() {
			// Create existing profile first
			env.CreateProfile(&profile.Profile{Name: "existing-profile"})

			result := env.Run("profile", "create", "existing-profile",
				"--description", "Test",
				"--marketplace", "owner/repo",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("already exists"))
		})

		It("fails with reserved name 'current'", func() {
			result := env.Run("profile", "create", "current",
				"--description", "Test",
				"--marketplace", "owner/repo",
			)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("reserved"))
		})
	})
})
