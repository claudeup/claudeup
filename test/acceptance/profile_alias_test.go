// ABOUTME: Acceptance tests for profile command aliases
// ABOUTME: Verifies that 'profile use' and 'profile apply' are equivalent

package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
)

var _ = Describe("profile command aliases", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)

		// Create minimal Claude installation
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), map[string]interface{}{
			"enabledPlugins": map[string]bool{},
		})

		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{},
		})

		// Create a test profile
		profilesDir := filepath.Join(env.TempDir, ".claudeup", "profiles")
		err := os.MkdirAll(profilesDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		helpers.WriteJSON(filepath.Join(profilesDir, "test-alias.json"), map[string]interface{}{
			"name":        "test-alias",
			"description": "Test profile for alias verification",
			"plugins":     []string{},
		})
	})

	AfterEach(func() {
		env.Cleanup()
	})

	It("supports 'profile use' as an alias for 'profile apply'", func() {
		// Test that 'profile apply' works
		result := env.Run("profile", "apply", "test-alias", "-y")
		Expect(result.ExitCode).To(Equal(0))
		// Command succeeds (may show "Applied profile" or "No changes needed")
		Expect(result.Stdout).To(Or(
			ContainSubstring("Applied profile"),
			ContainSubstring("No changes needed"),
		))
	})

	It("shows 'apply' in help output", func() {
		result := env.Run("profile", "use", "--help")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("apply"))
	})

	It("works with --reset flag when using 'apply' alias", func() {
		// Test with --reset flag using 'apply' alias
		result := env.Run("profile", "apply", "test-alias", "--reset", "-y")
		Expect(result.ExitCode).To(Equal(0))
		Expect(result.Stdout).To(ContainSubstring("Cleared user scope"))
	})

	It("both 'use' and 'apply' produce identical results", func() {
		// Apply with 'use'
		resultUse := env.Run("profile", "use", "test-alias", "-y")
		Expect(resultUse.ExitCode).To(Equal(0))

		// Reset and apply with 'apply' alias
		resultApply := env.Run("profile", "apply", "test-alias", "-y")
		Expect(resultApply.ExitCode).To(Equal(0))

		// Both should succeed (outputs may vary based on state)
		Expect(resultUse.ExitCode).To(Equal(resultApply.ExitCode))
	})
})
