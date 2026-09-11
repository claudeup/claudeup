// ABOUTME: Acceptance tests for outdated command
// ABOUTME: Tests display of available updates for CLI and plugins
package acceptance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("outdated", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Describe("with no marketplaces or plugins", func() {
		It("shows CLI section", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("CLI"))
		})

		It("shows Marketplaces section", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Marketplaces"))
		})

		It("shows Plugins section", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins"))
		})

		It("shows suggested commands footer", func() {
			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claudeup update"))
			Expect(result.Stdout).To(ContainSubstring("claudeup upgrade"))
		})
	})

	Describe("externally sourced plugins", func() {
		It("does not claim to know whether an external plugin is up to date", func() {
			marketplaceDir := filepath.Join(env.ClaudeDir, "plugins", "marketplaces", "test-marketplace")
			Expect(os.MkdirAll(filepath.Join(marketplaceDir, ".claude-plugin"), 0755)).To(Succeed())
			// The marketplace declares 0.0.1 while Claude Code recorded 9.9.9
			// from the plugin's own plugin.json. Neither that nor the marketplace
			// commit says whether the plugin is behind.
			Expect(os.WriteFile(filepath.Join(marketplaceDir, ".claude-plugin", "marketplace.json"), []byte(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "ext-plugin", "version": "0.0.1", "source": {"source": "url", "url": "https://example.com/ext-plugin.git"}}
				]
			}`), 0644)).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "init").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "config", "user.email", "test@example.com").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "config", "user.name", "Test").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "add", ".").Run()).To(Succeed())
			Expect(exec.Command("git", "-C", marketplaceDir, "-c", "commit.gpgsign=false", "commit", "-m", "initial").Run()).To(Succeed())
			headBytes, err := exec.Command("git", "-C", marketplaceDir, "rev-parse", "HEAD").Output()
			Expect(err).NotTo(HaveOccurred())

			cacheDir := filepath.Join(env.ClaudeDir, "plugins", "cache", "test-marketplace", "ext-plugin", "9.9.9")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())

			env.CreateKnownMarketplaces(map[string]interface{}{
				"test-marketplace": map[string]interface{}{
					"source":          map[string]interface{}{"repo": "example/test-marketplace"},
					"installLocation": marketplaceDir,
				},
			})
			env.CreateInstalledPlugins(map[string]interface{}{
				"ext-plugin@test-marketplace": []interface{}{
					map[string]interface{}{
						"scope":        "user",
						"version":      "9.9.9",
						"installedAt":  "2025-01-01T00:00:00Z",
						"lastUpdated":  "2025-01-01T00:00:00Z",
						"installPath":  cacheDir,
						"gitCommitSha": strings.TrimSpace(string(headBytes)),
					},
				},
			})

			result := env.Run("outdated")

			Expect(result.ExitCode).To(Equal(0), "stdout: %s\nstderr: %s", result.Stdout, result.Stderr)
			Expect(result.Stdout).To(ContainSubstring("ext-plugin@test-marketplace (user)"))
			Expect(result.Stdout).To(ContainSubstring("external source"),
				"a read-only check cannot know whether an external plugin is behind")
			Expect(result.Stdout).NotTo(ContainSubstring("All plugins up to date"),
				"that is a claim this command cannot make for an external plugin")
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

				result := env.Run("outdated")

				Expect(result.ExitCode).To(Equal(0))
				// 2 user-scope plugins: user-plugin and multi-plugin(user)
				Expect(result.Stdout).To(ContainSubstring("Plugins (2)"))
			})

			It("checks all scopes with --all", func() {
				env.CreateInstalledPlugins(multiScopePlugins)

				result := env.Run("outdated", "--all")

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

				result := env.RunInDir(projectDir, "outdated")

				Expect(result.ExitCode).To(Equal(0))
				// 4 total: project context includes user + project plugins matching this project
				Expect(result.Stdout).To(ContainSubstring("Plugins (4)"))
			})
		})
	})

	Describe("help output", func() {
		It("shows usage information", func() {
			result := env.Run("outdated", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Check for available updates"))
			Expect(result.Stdout).To(ContainSubstring("Usage:"))
		})
	})
})
