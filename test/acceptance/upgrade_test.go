// ABOUTME: Acceptance tests for upgrade command
// ABOUTME: Tests marketplace and plugin update functionality
package acceptance

import (
	"os"
	"path/filepath"

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
