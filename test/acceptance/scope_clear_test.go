// ABOUTME: Acceptance tests for scope clear command
// ABOUTME: Tests CLI behavior for clearing settings at different scopes
package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
)

func TestScopeClear(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scope Clear Acceptance Suite")
}

var _ = Describe("claudeup scope clear", func() {
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
				"claude-mem@thedotmack":               true,
				"superpowers@superpowers-marketplace": true,
			},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

		// Create project scope settings
		projectSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"gopls-lsp@claude-plugins-official":         true,
				"backend-development@claude-code-workflows": true,
			},
		}
		helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)

		// Create local scope settings
		localSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"systems-programming@claude-code-workflows": true,
			},
		}
		helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.local.json"), localSettings)

		// Create minimal installed_plugins.json
		installedPlugins := map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), installedPlugins)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("scope clear user --force", func() {
		It("should clear user scope settings without prompting", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "user", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Cleared user scope settings"))
		})

		It("should show what is being cleared", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "user", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("2 enabled plugins"))
		})

		It("should reset user settings to empty", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "user", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Verify settings are empty
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "user")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No plugins enabled"))
		})

		It("should not affect project or local scopes", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "user", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Verify project scope still has plugins
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "project")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("gopls-lsp@claude-plugins-official"))

			// Verify local scope still has plugins
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "local")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("systems-programming@claude-code-workflows"))
		})
	})

	Describe("scope clear project --force", func() {
		It("should clear project scope settings", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "project", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Cleared project scope settings"))
		})

		It("should show team impact warning", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "project", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Team Impact Warning"))
		})

		It("should remove project settings file", func() {
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			Expect(settingsPath).To(BeARegularFile())

			result := env.RunInDir(projectDir, "scope", "clear", "project", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Verify file is removed
			_, err := os.Stat(settingsPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should not affect user or local scopes", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "project", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Verify user scope still has plugins
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "user")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claude-mem@thedotmack"))

			// Verify local scope still has plugins
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "local")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("systems-programming@claude-code-workflows"))
		})
	})

	Describe("scope clear local --force", func() {
		It("should clear local scope settings", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "local", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Cleared local scope settings"))
		})

		It("should show local-only message", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "local", "--force")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("This only affects this machine"))
		})

		It("should remove local settings file", func() {
			settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			Expect(settingsPath).To(BeARegularFile())

			result := env.RunInDir(projectDir, "scope", "clear", "local", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Verify file is removed
			_, err := os.Stat(settingsPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should not affect user or project scopes", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "local", "--force")
			Expect(result.ExitCode).To(Equal(0))

			// Verify user scope still has plugins
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "user")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claude-mem@thedotmack"))

			// Verify project scope still has plugins
			result = env.RunInDir(projectDir, "scope", "list", "--scope", "project")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("gopls-lsp@claude-plugins-official"))
		})
	})

	Describe("error handling", func() {
		It("should reject invalid scope", func() {
			result := env.RunInDir(projectDir, "scope", "clear", "invalid", "--force")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})

		It("should error when scope doesn't exist", func() {
			// Remove project settings first
			os.Remove(filepath.Join(projectDir, ".claude", "settings.json"))

			result := env.RunInDir(projectDir, "scope", "clear", "project", "--force")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("not configured"))
		})
	})

	Describe("help and usage", func() {
		It("should show help text", func() {
			result := env.Run("scope", "clear", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Remove settings at the specified scope"))
			Expect(result.Stdout).To(ContainSubstring("--force"))
		})

		It("should require scope argument", func() {
			result := env.Run("scope", "clear")

			Expect(result.ExitCode).NotTo(Equal(0))
		})
	})
})
