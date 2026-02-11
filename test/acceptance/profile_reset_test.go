// ABOUTME: Acceptance tests for profile reset command
// ABOUTME: Tests confirmation prompts, output messages, and error handling
package acceptance

import (
	"github.com/claudeup/claudeup/v5/internal/profile"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile reset", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
		env.CreateInstalledPlugins(map[string]interface{}{})
		env.CreateKnownMarketplaces(map[string]interface{}{})
	})

	Describe("confirmation prompt", func() {
		It("shows what will be removed for built-in profile", func() {
			result := env.RunWithInput("n\n", "profile", "reset", "hobson")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Reset profile: hobson"))
			Expect(result.Stdout).To(ContainSubstring("Will remove:"))
			Expect(result.Stdout).To(ContainSubstring("Marketplace:"))
		})

		It("cancels when user says no", func() {
			result := env.RunWithInput("n\n", "profile", "reset", "frontend")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Cancelled"))
			Expect(result.Stdout).NotTo(ContainSubstring("Profile reset complete"))
		})
	})

	Describe("with -y flag", func() {
		It("skips confirmation prompt", func() {
			result := env.Run("profile", "reset", "frontend", "-y")

			// Should proceed without prompting
			Expect(result.Stdout).NotTo(ContainSubstring("Proceed?"))
			// Note: actual removal may fail in test env without real claude CLI
			// but the command should attempt to proceed
		})
	})

	Describe("error handling", func() {
		It("returns error for non-existent profile", func() {
			result := env.Run("profile", "reset", "nonexistent-profile")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not found"))
		})

		It("handles profile with no components gracefully", func() {
			// Create a profile with no marketplaces, MCP servers, or plugins
			env.CreateProfile(&profile.Profile{
				Name:        "empty-profile",
				Description: "Profile with nothing to reset",
			})

			result := env.Run("profile", "reset", "empty-profile", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Nothing to remove"))
		})
	})

	Describe("with installed plugins from marketplace", func() {
		BeforeEach(func() {
			// Simulate installed plugins from wshobson-agents marketplace
			env.CreateInstalledPlugins(map[string]interface{}{
				"debugging-toolkit@wshobson-agents": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
				"code-review-ai@wshobson-agents": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})
		})

		It("shows plugins that will be removed", func() {
			result := env.RunWithInput("n\n", "profile", "reset", "hobson")

			Expect(result.Stdout).To(ContainSubstring("Will remove:"))
			Expect(result.Stdout).To(ContainSubstring("debugging-toolkit@wshobson-agents"))
			Expect(result.Stdout).To(ContainSubstring("code-review-ai@wshobson-agents"))
		})
	})

	Describe("reset vs restore distinction", func() {
		BeforeEach(func() {
			// Create a customized built-in profile
			env.CreateProfile(&profile.Profile{
				Name:        "frontend",
				Description: "My customized frontend",
			})
		})

		It("reset does not remove the profile file", func() {
			Expect(env.ProfileExists("frontend")).To(BeTrue())

			// Reset should NOT remove the profile file
			result := env.Run("profile", "reset", "frontend", "-y")

			// Profile file should still exist after reset
			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ProfileExists("frontend")).To(BeTrue())
		})

		It("restore removes the customization file", func() {
			Expect(env.ProfileExists("frontend")).To(BeTrue())

			// Restore SHOULD remove the profile file (customization)
			result := env.Run("profile", "restore", "frontend", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(env.ProfileExists("frontend")).To(BeFalse())
		})
	})
})
