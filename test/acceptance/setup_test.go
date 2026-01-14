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
		It("offers backup option when existing installation detected", func() {
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
			Expect(result.Stdout).To(ContainSubstring("[b] Create backup"))
			Expect(result.Stdout).To(ContainSubstring("[c] Continue anyway"))
			Expect(result.Stdout).To(ContainSubstring("[a] Abort"))
		})

		It("creates backup when user chooses 'b'", func() {
			// Clean up any pre-existing backup directories to ensure test isolation
			existingBackups, _ := filepath.Glob(filepath.Join(env.TempDir, ".claude.backup*"))
			for _, backup := range existingBackups {
				os.RemoveAll(backup)
			}

			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"backup-test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Run setup, choose 'b' to backup, then 'y' to proceed
			result := env.RunWithInput("b\ny\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Created backup"))

			// Check that exactly one backup directory was created
			entries, err := filepath.Glob(filepath.Join(env.TempDir, ".claude.backup*"))
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1), "Expected exactly one backup directory to be created")

			// Verify backup contains the original plugin data
			backupDir := entries[0]
			backupPlugins := filepath.Join(backupDir, "plugins", "installed_plugins.json")
			data, err := os.ReadFile(backupPlugins)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("backup-test-plugin"))
		})

		It("prevents overwriting embedded profiles when saving existing installation", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Try to save with "default" (embedded profile), then "my-setup" (valid), then abort
			result := env.RunWithInput("s\ndefault\nmy-setup\na\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Profile name [saved]:"))
			Expect(result.Stdout).To(ContainSubstring("Cannot overwrite built-in profile"))
			Expect(result.Stdout).To(ContainSubstring("Profile name [saved]:"))
		})

		It("defaults profile name to 'saved' not 'current'", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Choose save option, press enter to accept default, then abort
			result := env.RunWithInput("s\n\na\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Profile name [saved]:"))
		})

		It("validates profile exists before prompting to save existing installation", func() {
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Try to setup with non-existent profile
			result := env.Run("setup", "--profile", "nonexistent")

			// Should fail immediately without prompting
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("profile \"nonexistent\" does not exist"))
			Expect(result.Stderr).To(ContainSubstring("claudeup profile list"))
			// Should NOT have prompted for saving
			Expect(result.Stdout).NotTo(ContainSubstring("Save current setup"))
			Expect(result.Stdout).NotTo(ContainSubstring("Profile name"))
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
			// Use 'c' to continue without saving, 'y' to proceed
			result := env.RunWithInput("c\ny\n", "setup", "--claude-dir", customClaudeDir)

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
