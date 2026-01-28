// ABOUTME: Acceptance tests for setup command
// ABOUTME: Tests --claude-dir flag, profile application, and existing installation handling
package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("setup", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		// Skip setup tests if claude CLI is not installed
		// These tests require the real claude CLI to be present
		if _, err := exec.LookPath("claude"); err != nil {
			Skip("claude CLI not installed, skipping setup tests")
		}

		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("existing installation handling", func() {
		It("offers save and continue options when existing installation detected", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Run setup, answer 'a' to abort so we can check the prompt
			result := env.RunWithInput("a\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Existing Claude Code installation detected"))
			Expect(result.Stdout).To(ContainSubstring("[s] Save current setup as a profile"))
			Expect(result.Stdout).To(ContainSubstring("[c] Continue without saving"))
			Expect(result.Stdout).To(ContainSubstring("[a] Abort"))
		})

		It("prevents overwriting embedded profiles when saving existing installation", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Try to save with "default" (embedded profile), then "custom-name" (valid)
			result := env.RunWithInput("s\ndefault\ncustom-name\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Profile name [my-setup]:"))
			Expect(result.Stdout).To(ContainSubstring("Cannot overwrite built-in profile"))
			// Should prompt again after rejection
			Expect(result.Stdout).To(ContainSubstring("Saved as 'custom-name'"))
		})

		It("defaults profile name to 'my-setup'", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Choose save option, press enter to accept default
			result := env.RunWithInput("s\n\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Profile name [my-setup]:"))
			Expect(result.Stdout).To(ContainSubstring("Saved as 'my-setup'"))
		})

		It("ignores --profile flag for existing installations", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Run setup with a non-existent profile but existing installation
			// The --profile flag should be ignored for existing installations
			result := env.RunWithInput("c\n", "setup", "--profile", "nonexistent")

			// Should succeed because --profile is ignored for existing installations
			Expect(result.ExitCode).To(Equal(0))
			// Should show existing installation prompt (not profile error)
			Expect(result.Stdout).To(ContainSubstring("Existing Claude Code installation detected"))
		})

		It("validates profile for fresh installations", func() {
			// Don't create any existing installation content
			// Just ensure the claude directory exists (setup creates it)

			// Try to setup with non-existent profile on fresh install
			result := env.Run("setup", "--profile", "nonexistent")

			// Should fail because the profile doesn't exist
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("profile \"nonexistent\" does not exist"))
			Expect(result.Stderr).To(ContainSubstring("claudeup profile list"))
		})

		It("installs plugins when saving profile for existing installation", func() {
			// Create an existing installation with enabled plugins but not installed
			env.CreateClaudeSettingsWithPlugins(map[string]bool{
				"test-plugin@test-marketplace": true,
			})

			// Create the marketplace and plugin so installation can succeed
			env.CreateMarketplace("test-marketplace", "github.com/test/marketplace")
			env.CreateMarketplacePlugin("test-marketplace", "test-plugin", "1.0.0")

			// Run setup with -y to auto-confirm, accept defaults
			result := env.Run("setup", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Installing plugins"))
			Expect(result.Stdout).To(ContainSubstring("1 plugins installed"))
		})
	})

	Describe("--claude-dir flag", func() {
		It("operates on the specified directory instead of default", func() {
			// Create a custom claude directory with a plugin
			customClaudeDir := filepath.Join(env.TempDir, "custom-claude")
			Expect(os.MkdirAll(filepath.Join(customClaudeDir, "plugins"), 0755)).To(Succeed())

			// Create installed_plugins.json in custom dir with a plugin
			pluginsFile := filepath.Join(customClaudeDir, "plugins", "installed_plugins.json")
			Expect(os.WriteFile(pluginsFile, []byte(`{
				"version": 2,
				"plugins": {
					"test-plugin@test-marketplace": [{"scope": "user", "version": "1.0"}]
				}
			}`), 0644)).To(Succeed())

			// Create settings.json with plugin enabled
			settingsFile := filepath.Join(customClaudeDir, "settings.json")
			Expect(os.WriteFile(settingsFile, []byte(`{
				"enabledPlugins": {
					"test-plugin@test-marketplace": true
				}
			}`), 0644)).To(Succeed())

			// Create empty known_marketplaces.json
			marketplacesFile := filepath.Join(customClaudeDir, "plugins", "known_marketplaces.json")
			Expect(os.WriteFile(marketplacesFile, []byte(`{}`), 0644)).To(Succeed())

			// Create .claude.json settings file for custom dir
			claudeJSONFile := filepath.Join(customClaudeDir, ".claude.json")
			Expect(os.WriteFile(claudeJSONFile, []byte(`{"mcpServers": {}}`), 0644)).To(Succeed())

			// Run setup with --claude-dir pointing to custom directory
			// Use 'c' to continue without saving
			result := env.RunWithInput("c\n", "setup", "--claude-dir", customClaudeDir)

			// The setup should detect the existing installation in customClaudeDir
			// (which has 1 plugin) and offer to save it as a profile
			Expect(result.Stdout).To(ContainSubstring("Existing Claude Code installation detected"))
			Expect(result.Stdout).To(ContainSubstring("1 plugins"))
		})

		It("prompts when --claude-dir does not exist", func() {
			nonexistentDir := filepath.Join(env.TempDir, "does-not-exist")

			// Run setup with nonexistent directory, answer 'a' to abort
			result := env.RunWithInput("a\n", "setup", "--claude-dir", nonexistentDir)

			Expect(result.Stdout).To(ContainSubstring("does not exist"))
			Expect(result.Stdout).To(ContainSubstring("[c] Create it and continue"))
			Expect(result.Stdout).To(ContainSubstring("[a] Abort"))
		})

		It("creates directory when user chooses 'c' for nonexistent --claude-dir", func() {
			nonexistentDir := filepath.Join(env.TempDir, "new-claude-dir")

			// Verify directory doesn't exist
			_, err := os.Stat(nonexistentDir)
			Expect(os.IsNotExist(err)).To(BeTrue())

			// Run setup, choose 'c' to create, then 'y' to proceed with setup
			result := env.RunWithInput("c\ny\n", "setup", "--claude-dir", nonexistentDir)

			Expect(result.Stdout).To(ContainSubstring("does not exist"))
			Expect(result.ExitCode).To(Equal(0))

			// Directory should now exist
			_, err = os.Stat(nonexistentDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("aborts when user chooses 'a' for nonexistent --claude-dir", func() {
			nonexistentDir := filepath.Join(env.TempDir, "abort-test-dir")

			// Run setup, choose 'a' to abort
			result := env.RunWithInput("a\n", "setup", "--claude-dir", nonexistentDir)

			Expect(result.Stdout).To(ContainSubstring("does not exist"))
			Expect(result.Stdout).To(ContainSubstring("Setup aborted"))

			// Directory should NOT exist
			_, err := os.Stat(nonexistentDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("does not modify the default claude directory when --claude-dir is specified", func() {
			// Create plugins in the DEFAULT directory (env.ClaudeDir = ~/.claude in test)
			env.CreateInstalledPlugins(map[string]interface{}{
				"default-plugin@default-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Create a separate custom directory with different content
			customClaudeDir := filepath.Join(env.TempDir, "isolated-claude")
			Expect(os.MkdirAll(filepath.Join(customClaudeDir, "plugins"), 0755)).To(Succeed())

			// Create empty registries in custom dir (fresh installation)
			pluginsFile := filepath.Join(customClaudeDir, "plugins", "installed_plugins.json")
			Expect(os.WriteFile(pluginsFile, []byte(`{"version": 2, "plugins": {}}`), 0644)).To(Succeed())

			marketplacesFile := filepath.Join(customClaudeDir, "plugins", "known_marketplaces.json")
			Expect(os.WriteFile(marketplacesFile, []byte(`{}`), 0644)).To(Succeed())

			settingsFile := filepath.Join(customClaudeDir, ".claude.json")
			Expect(os.WriteFile(settingsFile, []byte(`{"mcpServers": {}}`), 0644)).To(Succeed())

			// Run setup targeting the custom directory
			result := env.RunWithInput("y\n", "setup", "--claude-dir", customClaudeDir, "--profile", "default")

			Expect(result.ExitCode).To(Equal(0))

			// Verify the DEFAULT directory was NOT touched
			// The plugin we created there should still exist
			defaultPluginsData, err := os.ReadFile(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(defaultPluginsData)).To(ContainSubstring("default-plugin@default-marketplace"))
		})
	})
})
