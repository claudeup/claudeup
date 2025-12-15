// ABOUTME: Acceptance tests for setup command
// ABOUTME: Tests --claude-dir flag, profile application, and existing installation handling
package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/claudeup/claudeup/test/helpers"
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
			// Create an existing installation with content
			env.CreateInstalledPlugins(map[string]interface{}{
				"backup-test-plugin@test-marketplace": []map[string]interface{}{
					{"scope": "user", "version": "1.0"},
				},
			})

			// Run setup, choose 'b' to backup, then 'y' to proceed
			result := env.RunWithInput("b\ny\n", "setup")

			Expect(result.Stdout).To(ContainSubstring("Created backup"))

			// Check that a backup directory was created
			// Backup should be at ~/.claude.backup or ~/.claude.backup.1, etc.
			entries, err := filepath.Glob(filepath.Join(env.TempDir, ".claude.backup*"))
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).NotTo(BeEmpty(), "Expected backup directory to be created")

			// Verify backup contains the original plugin data
			for _, entry := range entries {
				info, err := os.Stat(entry)
				if err == nil && info.IsDir() {
					backupPlugins := filepath.Join(entry, "plugins", "installed_plugins.json")
					data, err := os.ReadFile(backupPlugins)
					if err == nil {
						Expect(string(data)).To(ContainSubstring("backup-test-plugin"))
						break
					}
				}
			}
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

			// Create empty known_marketplaces.json
			marketplacesFile := filepath.Join(customClaudeDir, "plugins", "known_marketplaces.json")
			Expect(os.WriteFile(marketplacesFile, []byte(`{}`), 0644)).To(Succeed())

			// Create .claude.json settings file for custom dir
			settingsFile := filepath.Join(customClaudeDir, ".claude.json")
			Expect(os.WriteFile(settingsFile, []byte(`{"mcpServers": {}}`), 0644)).To(Succeed())

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
