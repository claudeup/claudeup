// ABOUTME: Acceptance tests for scope list command
// ABOUTME: Tests CLI behavior for viewing plugins across different scopes
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/test/helpers"
)

var _ = Describe("claudeup scope list", func() {
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
				"claude-mem@thedotmack":                   true,
				"superpowers@superpowers-marketplace":     true,
				"episodic-memory@superpowers-marketplace": true,
			},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

		// Create project scope settings
		projectSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"gopls-lsp@claude-plugins-official":         true,
				"backend-development@claude-code-workflows": true,
				"tdd-workflows@claude-code-workflows":       true,
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

	Describe("scope list without --scope flag", func() {
		It("should show all three scopes", func() {
			result := env.RunInDir(projectDir, "scope", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Scope: User"))
			Expect(result.Stdout).To(ContainSubstring("Scope: Project"))
			Expect(result.Stdout).To(ContainSubstring("Scope: Local"))
		})

		It("should show user scope plugins", func() {
			result := env.RunInDir(projectDir, "scope", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claude-mem@thedotmack"))
			Expect(result.Stdout).To(ContainSubstring("superpowers@superpowers-marketplace"))
			Expect(result.Stdout).To(ContainSubstring("episodic-memory@superpowers-marketplace"))
		})

		It("should show project scope plugins", func() {
			result := env.RunInDir(projectDir, "scope", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("gopls-lsp@claude-plugins-official"))
			Expect(result.Stdout).To(ContainSubstring("backend-development@claude-code-workflows"))
			Expect(result.Stdout).To(ContainSubstring("tdd-workflows@claude-code-workflows"))
		})

		It("should show local scope plugins", func() {
			result := env.RunInDir(projectDir, "scope", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("systems-programming@claude-code-workflows"))
			Expect(result.Stdout).To(ContainSubstring("shell-scripting@claude-code-workflows"))
		})

		It("should show effective configuration count", func() {
			result := env.RunInDir(projectDir, "scope", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Effective Configuration: 8 unique plugins enabled"))
		})
	})

	Describe("scope list --scope user", func() {
		It("should only show user scope", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Scope: User"))
			Expect(result.Stdout).NotTo(ContainSubstring("Scope: Project"))
			Expect(result.Stdout).NotTo(ContainSubstring("Scope: Local"))
		})

		It("should show user scope plugins", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("claude-mem@thedotmack"))
			Expect(result.Stdout).To(ContainSubstring("superpowers@superpowers-marketplace"))
		})

		It("should not show effective configuration count", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("Effective Configuration"))
		})
	})

	Describe("scope list --scope project", func() {
		It("should only show project scope", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "project")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Scope: Project"))
			Expect(result.Stdout).NotTo(ContainSubstring("Scope: User"))
			Expect(result.Stdout).NotTo(ContainSubstring("Scope: Local"))
		})

		It("should show project scope plugins", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "project")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("gopls-lsp@claude-plugins-official"))
			Expect(result.Stdout).To(ContainSubstring("backend-development@claude-code-workflows"))
		})
	})

	Describe("scope list --scope local", func() {
		It("should only show local scope", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "local")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Scope: Local"))
			Expect(result.Stdout).NotTo(ContainSubstring("Scope: User"))
			Expect(result.Stdout).NotTo(ContainSubstring("Scope: Project"))
		})

		It("should show local scope plugins", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "local")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("systems-programming@claude-code-workflows"))
			Expect(result.Stdout).To(ContainSubstring("shell-scripting@claude-code-workflows"))
		})
	})

	Describe("error handling", func() {
		It("should reject invalid scope values", func() {
			result := env.RunInDir(projectDir, "scope", "list", "--scope", "invalid")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid scope"))
		})
	})

	Describe("when not in a project directory", func() {
		It("should show user scope and note about project/local", func() {
			// Run from temp directory without .claude folder
			result := env.Run("scope", "list")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Scope: User"))
			Expect(result.Stdout).To(ContainSubstring("Project scope: Not in a project directory"))
			Expect(result.Stdout).To(ContainSubstring("Local scope: Not configured for this directory"))
		})
	})
})
