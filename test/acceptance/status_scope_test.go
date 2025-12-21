// ABOUTME: Acceptance tests for status command with --scope flag
// ABOUTME: Tests CLI behavior for scope-specific status checks

package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
)

func TestStatusScope(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Status Scope Acceptance Suite")
}

var _ = Describe("claudeup status --scope", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
		projectDir string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)

		// Create a project directory for testing
		projectDir = env.ProjectDir("test-project")

		// Create Claude directory structure
		err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
		Expect(err).NotTo(HaveOccurred())

		// Create user scope settings
		userSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"claude-mem@thedotmack":            true,
				"superpowers@superpowers-marketplace": true,
			},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

		// Create project scope settings
		projectSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"gopls-lsp@claude-plugins-official":       true,
				"backend-development@claude-code-workflows": true,
			},
		}
		helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)

		// Create local scope settings
		localSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"systems-programming@claude-code-workflows": true,
				"shell-scripting@claude-code-workflows":     true,
			},
		}
		helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.local.json"), localSettings)

		// Create installed_plugins.json with all plugins
		installedPlugins := map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{
				"claude-mem@thedotmack": []interface{}{
					map[string]interface{}{
						"scope":       "user",
						"version":     "7.4.5",
						"installedAt": "2025-01-01T00:00:00Z",
					},
				},
				"superpowers@superpowers-marketplace": []interface{}{
					map[string]interface{}{
						"scope":       "user",
						"version":     "4.0.0",
						"installedAt": "2025-01-01T00:00:00Z",
					},
				},
				"gopls-lsp@claude-plugins-official": []interface{}{
					map[string]interface{}{
						"scope":       "project",
						"version":     "1.0.0",
						"installedAt": "2025-01-01T00:00:00Z",
					},
				},
				"backend-development@claude-code-workflows": []interface{}{
					map[string]interface{}{
						"scope":       "project",
						"version":     "1.2.3",
						"installedAt": "2025-01-01T00:00:00Z",
					},
				},
				"systems-programming@claude-code-workflows": []interface{}{
					map[string]interface{}{
						"scope":       "local",
						"version":     "1.2.0",
						"installedAt": "2025-01-01T00:00:00Z",
					},
				},
				"shell-scripting@claude-code-workflows": []interface{}{
					map[string]interface{}{
						"scope":       "local",
						"version":     "1.2.1",
						"installedAt": "2025-01-01T00:00:00Z",
					},
				},
			},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), installedPlugins)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("status without --scope flag", func() {
		It("should show all plugins from all scopes", func() {
			result := env.RunInDir(projectDir, "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claude-mem@thedotmack"))
			Expect(result.Stdout).To(ContainSubstring("superpowers@superpowers-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("gopls-lsp@claude-plugins-official"))
			Expect(result.Stdout).To(ContainSubstring("backend-development@claude-code-workflows"))
			Expect(result.Stdout).To(ContainSubstring("systems-programming@claude-code-workflows"))
			Expect(result.Stdout).To(ContainSubstring("shell-scripting@claude-code-workflows"))
		})

		It("should show total plugin count across all scopes", func() {
			result := env.RunInDir(projectDir, "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins (6)"))
		})
	})

	Describe("status --scope user", func() {
		It("should only show user scope plugins", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claude-mem@thedotmack"))
			Expect(result.Stdout).To(ContainSubstring("superpowers@superpowers-marketplace"))
			Expect(result.Stdout).NotTo(ContainSubstring("gopls-lsp@claude-plugins-official"))
			Expect(result.Stdout).NotTo(ContainSubstring("backend-development@claude-code-workflows"))
			Expect(result.Stdout).NotTo(ContainSubstring("systems-programming@claude-code-workflows"))
			Expect(result.Stdout).NotTo(ContainSubstring("shell-scripting@claude-code-workflows"))
		})

		It("should show user scope plugin count", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins (2)"))
		})
	})

	Describe("status --scope project", func() {
		It("should only show project scope plugins", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "project")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("gopls-lsp@claude-plugins-official"))
			Expect(result.Stdout).To(ContainSubstring("backend-development@claude-code-workflows"))
			Expect(result.Stdout).NotTo(ContainSubstring("claude-mem@thedotmack"))
			Expect(result.Stdout).NotTo(ContainSubstring("superpowers@superpowers-marketplace"))
			Expect(result.Stdout).NotTo(ContainSubstring("systems-programming@claude-code-workflows"))
			Expect(result.Stdout).NotTo(ContainSubstring("shell-scripting@claude-code-workflows"))
		})

		It("should show project scope plugin count", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "project")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins (2)"))
		})
	})

	Describe("status --scope local", func() {
		It("should only show local scope plugins", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "local")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("systems-programming@claude-code-workflows"))
			Expect(result.Stdout).To(ContainSubstring("shell-scripting@claude-code-workflows"))
			Expect(result.Stdout).NotTo(ContainSubstring("claude-mem@thedotmack"))
			Expect(result.Stdout).NotTo(ContainSubstring("superpowers@superpowers-marketplace"))
			Expect(result.Stdout).NotTo(ContainSubstring("gopls-lsp@claude-plugins-official"))
			Expect(result.Stdout).NotTo(ContainSubstring("backend-development@claude-code-workflows"))
		})

		It("should show local scope plugin count", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "local")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Plugins (2)"))
		})
	})

	Describe("error handling", func() {
		It("should reject invalid scope values", func() {
			result := env.RunInDir(projectDir, "status", "--scope", "invalid")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})
	})
})
