// ABOUTME: Acceptance tests for upgrade command
// ABOUTME: Tests marketplace and plugin update functionality
package acceptance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("upgrade", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no marketplaces or plugins", func() {
		It("shows up to date message", func() {
			result := env.Run("upgrade")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("up to date"))
		})
	})

	Describe("with positional arguments", func() {
		It("warns about unknown marketplaces", func() {
			result := env.Run("upgrade", "nonexistent-marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Unknown target"))
		})

		It("warns about unknown plugins", func() {
			result := env.Run("upgrade", "unknown@marketplace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Unknown target"))
		})
	})

	Describe("--all flag", func() {
		var multiScopePlugins map[string]interface{}

		BeforeEach(func() {
			multiScopePlugins = map[string]interface{}{
				"user-plugin@marketplace": []interface{}{
					map[string]interface{}{
						"scope": "user", "version": "1.0.0",
						"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
						"installPath": "/nonexistent/path", "gitCommitSha": "abc1234",
					},
				},
				"project-plugin@marketplace": []interface{}{
					map[string]interface{}{
						"scope": "project", "version": "1.0.0",
						"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
						"installPath": "/nonexistent/path", "gitCommitSha": "def5678",
					},
				},
				"multi-plugin@marketplace": []interface{}{
					map[string]interface{}{
						"scope": "user", "version": "1.0.0",
						"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
						"installPath": "/nonexistent/path", "gitCommitSha": "aaa1111",
					},
					map[string]interface{}{
						"scope": "project", "version": "1.0.0",
						"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
						"installPath": "/nonexistent/path", "gitCommitSha": "bbb2222",
					},
				},
			}
		})

		Context("from a non-project directory", func() {
			It("checks only user-scope plugins without --all", func() {
				env.CreateInstalledPlugins(multiScopePlugins)

				result := env.Run("upgrade")

				Expect(result.ExitCode).To(Equal(0))
				// 2 user-scope plugins: user-plugin and multi-plugin(user)
				Expect(result.Stdout).To(ContainSubstring("Plugins (2)"))
			})

			It("checks all scopes with --all", func() {
				env.CreateInstalledPlugins(multiScopePlugins)

				result := env.Run("upgrade", "--all")

				Expect(result.ExitCode).To(Equal(0))
				// 4 total: user-plugin(user) + project-plugin(project) + multi-plugin(user) + multi-plugin(project)
				Expect(result.Stdout).To(ContainSubstring("Plugins (4)"))
			})
		})

		Context("from a project directory", func() {
			var projectDir string

			BeforeEach(func() {
				projectDir = filepath.Join(env.TempDir, "myproject")
				Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())
			})

			It("checks user, project, and local scopes without --all", func() {
				// Set projectPath on project-scope plugins to match this project dir
				projectPlugins := map[string]interface{}{
					"user-plugin@marketplace": []interface{}{
						map[string]interface{}{
							"scope": "user", "version": "1.0.0",
							"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
							"installPath": "/nonexistent/path", "gitCommitSha": "abc1234",
						},
					},
					"project-plugin@marketplace": []interface{}{
						map[string]interface{}{
							"scope": "project", "version": "1.0.0", "projectPath": projectDir,
							"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
							"installPath": "/nonexistent/path", "gitCommitSha": "def5678",
						},
					},
					"multi-plugin@marketplace": []interface{}{
						map[string]interface{}{
							"scope": "user", "version": "1.0.0",
							"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
							"installPath": "/nonexistent/path", "gitCommitSha": "aaa1111",
						},
						map[string]interface{}{
							"scope": "project", "version": "1.0.0", "projectPath": projectDir,
							"installedAt": "2025-01-01T00:00:00Z", "lastUpdated": "2025-01-01T00:00:00Z",
							"installPath": "/nonexistent/path", "gitCommitSha": "bbb2222",
						},
					},
				}
				env.CreateInstalledPlugins(projectPlugins)

				result := env.RunInDir(projectDir, "upgrade")

				Expect(result.ExitCode).To(Equal(0))
				// 4 total: project context includes user + project plugins matching this project
				Expect(result.Stdout).To(ContainSubstring("Plugins (4)"))
			})
		})
	})

	Describe("plugin upgrade after marketplace pull", func() {
		It("updates plugins when their marketplace has new commits", func() {
			env = helpers.NewTestEnv(binaryPath)

			// Create a bare git repo to act as the remote
			bareRepo := filepath.Join(env.TempDir, "bare-repo.git")
			Expect(exec.Command("git", "init", "--bare", bareRepo).Run()).To(Succeed())

			// Clone the bare repo as the marketplace directory
			marketplacesDir := filepath.Join(env.ClaudeDir, "plugins", "marketplaces")
			Expect(os.MkdirAll(marketplacesDir, 0755)).To(Succeed())
			marketplaceDir := filepath.Join(marketplacesDir, "test-marketplace")
			Expect(exec.Command("git", "clone", bareRepo, marketplaceDir).Run()).To(Succeed())

			// Configure git identity in the clone
			Expect(exec.Command("git", "-C", marketplaceDir, "config", "user.email", "test@example.com").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "config", "user.name", "Test").Run()).To(Succeed())

			// Create initial plugin content and commit
			pluginDir := filepath.Join(marketplaceDir, "plugins", "test-plugin")
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{"name":"test-plugin","version":"1.0.0"}`), 0644)).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "add", ".").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "-c", "commit.gpgsign=false", "commit", "-m", "initial").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "push", "origin", "HEAD").Run()).To(Succeed())

			// Record the initial commit SHA
			initialSHABytes, err := exec.Command("git", "-C", marketplaceDir, "rev-parse", "HEAD").Output()
			Expect(err).NotTo(HaveOccurred())
			initialSHA := strings.TrimSpace(string(initialSHABytes))

			// Create a cached copy of the plugin
			cacheDir := filepath.Join(env.ClaudeDir, "plugins", "cache", "test-plugin")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "plugin.json"), []byte(`{"name":"test-plugin","version":"1.0.0"}`), 0644)).To(Succeed())

			// Register the marketplace and plugin
			env.CreateKnownMarketplaces(map[string]interface{}{
				"test-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"repo": bareRepo,
					},
					"installLocation": marketplaceDir,
				},
			})
			env.CreateInstalledPlugins(map[string]interface{}{
				"test-plugin@test-marketplace": []interface{}{
					map[string]interface{}{
						"scope":        "user",
						"version":      "1.0.0",
						"installedAt":  "2025-01-01T00:00:00Z",
						"lastUpdated":  "2025-01-01T00:00:00Z",
						"installPath":  cacheDir,
						"gitCommitSha": initialSHA,
					},
				},
			})

			// Push a new commit to the remote (simulating marketplace author publishing an update)
			// We do this by cloning again, making a change, and pushing
			tempClone := filepath.Join(env.TempDir, "temp-clone")
			Expect(exec.Command("git", "clone", bareRepo, tempClone).Run()).To(Succeed())
			Expect(exec.Command("git", "-C", tempClone, "config", "user.email", "test@example.com").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", tempClone, "config", "user.name", "Test").Run()).To(Succeed())
			pluginInClone := filepath.Join(tempClone, "plugins", "test-plugin")
			Expect(os.WriteFile(filepath.Join(pluginInClone, "plugin.json"), []byte(`{"name":"test-plugin","version":"2.0.0"}`), 0644)).To(Succeed())
			Expect(exec.Command("git", "-C", tempClone, "add", ".").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", tempClone, "-c", "commit.gpgsign=false", "commit", "-m", "bump version").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", tempClone, "push", "origin", "HEAD").Run()).To(Succeed())

			// Run upgrade
			result := env.Run("upgrade")

			Expect(result.ExitCode).To(Equal(0), "stdout: %s\nstderr: %s", result.Stdout, result.Stderr)
			// The plugin should have been detected as outdated and updated
			Expect(result.Stdout).To(ContainSubstring("Update available"), "plugin should be detected as needing update")
			Expect(result.Stdout).To(ContainSubstring("Updated"), "plugin should have been updated")

			// Verify the plugin's gitCommitSha was updated in installed_plugins.json
			pluginsData, err := os.ReadFile(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"))
			Expect(err).NotTo(HaveOccurred())
			var registry map[string]interface{}
			Expect(json.Unmarshal(pluginsData, &registry)).To(Succeed())
			plugins := registry["plugins"].(map[string]interface{})
			instances := plugins["test-plugin@test-marketplace"].([]interface{})
			instance := instances[0].(map[string]interface{})
			Expect(instance["gitCommitSha"]).NotTo(Equal(initialSHA), "plugin SHA should have been updated to new commit")

			// Verify the cached plugin was updated with new content
			cachedManifest, err := os.ReadFile(filepath.Join(cacheDir, "plugin.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(cachedManifest)).To(ContainSubstring("2.0.0"), "cached plugin should have updated content")
		})
	})

	Describe("help output", func() {
		It("shows usage information", func() {
			result := env.Run("upgrade", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Update installed marketplaces and plugins"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})

		It("shows examples", func() {
			result := env.Run("upgrade", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claudeup upgrade"))
		})
	})
})
