// ABOUTME: Acceptance tests for plugin enable/disable commands
// ABOUTME: Tests the claudeup plugin enable and disable workflow
package acceptance

import (
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin enable/disable", func() {
	var env *helpers.TestEnv
	var pluginPath string
	const pluginName = "test-plugin@test-marketplace"

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)

		// Create a valid plugin directory
		pluginPath = filepath.Join(env.ClaudeDir, "plugins", "cache", "test-plugin")
		Expect(os.MkdirAll(pluginPath, 0755)).To(Succeed())

		// Register the plugin
		env.CreateInstalledPlugins(map[string]interface{}{
			pluginName: []interface{}{
				map[string]interface{}{
					"version":      "1.0.0",
					"installPath":  pluginPath,
					"gitCommitSha": "abc123",
					"isLocal":      false,
					"installedAt":  "2024-01-15T10:30:00Z",
					"scope":        "user",
				},
			},
		})
	})

	Describe("plugin disable", func() {
		BeforeEach(func() {
			// Start with plugin enabled
			env.CreateSettings(map[string]bool{
				pluginName: true,
			})
		})

		It("disables an enabled plugin", func() {
			Expect(env.IsPluginEnabled(pluginName)).To(BeTrue(), "plugin should start enabled")

			result := env.Run("plugin", "disable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Disabled"))
			Expect(env.IsPluginEnabled(pluginName)).To(BeFalse(), "plugin should be disabled after command")
		})

		It("shows success message with plugin name", func() {
			result := env.Run("plugin", "disable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring(pluginName))
		})

		It("shows re-enable hint", func() {
			result := env.Run("plugin", "disable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("plugin enable"))
		})

		It("is idempotent for already disabled plugin", func() {
			// Disable once
			env.Run("plugin", "disable", pluginName)

			// Disable again
			result := env.Run("plugin", "disable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("already disabled"))
		})

		It("fails for non-existent plugin", func() {
			result := env.Run("plugin", "disable", "nonexistent-plugin")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not installed"))
		})
	})

	Describe("plugin enable", func() {
		BeforeEach(func() {
			// Start with plugin disabled
			env.CreateSettings(map[string]bool{
				pluginName: false,
			})
		})

		It("enables a disabled plugin", func() {
			Expect(env.IsPluginEnabled(pluginName)).To(BeFalse(), "plugin should start disabled")

			result := env.Run("plugin", "enable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Enabled"))
			Expect(env.IsPluginEnabled(pluginName)).To(BeTrue(), "plugin should be enabled after command")
		})

		It("shows success message with plugin name", func() {
			result := env.Run("plugin", "enable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring(pluginName))
		})

		It("shows disable hint", func() {
			result := env.Run("plugin", "enable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("plugin disable"))
		})

		It("is idempotent for already enabled plugin", func() {
			// Enable once
			env.Run("plugin", "enable", pluginName)

			// Enable again
			result := env.Run("plugin", "enable", pluginName)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("already enabled"))
		})

		It("fails for non-existent plugin", func() {
			result := env.Run("plugin", "enable", "nonexistent-plugin")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not installed"))
		})
	})

	Describe("enable/disable round trip", func() {
		BeforeEach(func() {
			env.CreateSettings(map[string]bool{
				pluginName: true,
			})
		})

		It("can disable then re-enable a plugin", func() {
			// Verify initial state
			Expect(env.IsPluginEnabled(pluginName)).To(BeTrue())

			// Disable
			result := env.Run("plugin", "disable", pluginName)
			Expect(result.ExitCode).To(Equal(0))
			Expect(env.IsPluginEnabled(pluginName)).To(BeFalse())

			// Re-enable
			result = env.Run("plugin", "enable", pluginName)
			Expect(result.ExitCode).To(Equal(0))
			Expect(env.IsPluginEnabled(pluginName)).To(BeTrue())
		})
	})

	Describe("plugin list reflects enable/disable state", func() {
		BeforeEach(func() {
			env.CreateSettings(map[string]bool{
				pluginName: true,
			})
		})

		It("shows enabled status after enable", func() {
			result := env.Run("plugin", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("enabled"))
		})

		It("shows disabled status after disable", func() {
			env.Run("plugin", "disable", pluginName)

			result := env.Run("plugin", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("disabled"))
		})
	})

	Describe("help output", func() {
		It("shows plugin disable help", func() {
			result := env.Run("plugin", "disable", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Disable a plugin"))
		})

		It("shows plugin enable help", func() {
			result := env.Run("plugin", "enable", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Enable"))
		})
	})
})
