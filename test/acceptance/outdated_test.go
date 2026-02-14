// ABOUTME: Acceptance tests for outdated command
// ABOUTME: Tests display of available updates for CLI and plugins
package acceptance

import (
	"os"
	"path/filepath"

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
