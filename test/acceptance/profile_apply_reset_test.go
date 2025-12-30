// ABOUTME: Acceptance tests for profile apply --reset flag
// ABOUTME: Tests CLI behavior for clearing scope before applying profile
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
)

var _ = Describe("claudeup profile apply --reset", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)

		// Create user settings with existing plugins
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"existing-plugin@test": true,
			},
		})

		// Create minimal installed_plugins.json
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{},
		})

		// Create a test profile
		profilesDir := filepath.Join(env.TempDir, ".claudeup", "profiles")
		err := os.MkdirAll(profilesDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		helpers.WriteJSON(filepath.Join(profilesDir, "test-profile.json"), map[string]interface{}{
			"name":        "test-profile",
			"description": "Test profile",
			"plugins":     []string{},
		})
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("with --reset flag", func() {
		It("should clear scope before applying profile", func() {
			result := env.Run("profile", "apply", "test-profile", "--reset", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// Scope is cleared, then profile is applied (may show "no changes" if profile is empty)
			Expect(result.Stdout).To(ContainSubstring("Cleared user scope"))

			// Verify existing plugin is gone
			result = env.Run("scope", "list", "--scope", "user")
			Expect(result.Stdout).NotTo(ContainSubstring("existing-plugin@test"))
		})

		It("should show cleared message", func() {
			result := env.Run("profile", "apply", "test-profile", "--reset", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Cleared user scope"))
		})
	})

	Describe("help text", func() {
		It("should document --reset flag", func() {
			result := env.Run("profile", "apply", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--reset"))
			Expect(result.Stdout).To(ContainSubstring("Clear target scope"))
		})
	})
})
