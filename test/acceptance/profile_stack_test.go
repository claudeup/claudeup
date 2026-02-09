// ABOUTME: Acceptance tests for composable stack profile functionality
// ABOUTME: Tests stack apply, list, and show commands with end-to-end CLI execution
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v4/internal/profile"
	"github.com/claudeup/claudeup/v4/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Composable Stack Profiles", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("profile list", func() {
		Context("with a stack profile", func() {
			BeforeEach(func() {
				// Create leaf profiles
				env.CreateProfile(&profile.Profile{
					Name:    "go-tools",
					Plugins: []string{"go-linter@marketplace"},
				})
				env.CreateProfile(&profile.Profile{
					Name:    "testing-tools",
					Plugins: []string{"test-runner@marketplace"},
				})
				// Create stack profile
				env.CreateProfile(&profile.Profile{
					Name:        "go-dev",
					Description: "Go development stack",
					Includes:    []string{"go-tools", "testing-tools"},
				})
			})

			It("shows [stack] indicator for stack profiles", func() {
				result := env.Run("profile", "list")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("go-dev"))
				Expect(result.Stdout).To(ContainSubstring("[stack]"))
			})
		})
	})

	Describe("profile show", func() {
		Context("with a stack profile", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:    "base-tools",
					Plugins: []string{"plugin-a@marketplace"},
				})
				env.CreateProfile(&profile.Profile{
					Name:        "my-stack",
					Description: "Test stack",
					Includes:    []string{"base-tools"},
				})
			})

			It("displays include tree", func() {
				result := env.Run("profile", "show", "my-stack")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Includes:"))
				Expect(result.Stdout).To(ContainSubstring("base-tools"))
			})

			It("displays resolved summary", func() {
				result := env.Run("profile", "show", "my-stack")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("Resolved:"))
				Expect(result.Stdout).To(ContainSubstring("1 plugins"))
			})
		})
	})

	Describe("profile apply", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("stack-apply-test")
			Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())
		})

		Context("with a multi-scope stack", func() {
			BeforeEach(func() {
				// Create leaf profile with per-scope plugins
				env.CreateProfile(&profile.Profile{
					Name: "scoped-leaf",
					PerScope: &profile.PerScopeSettings{
						User: &profile.ScopeSettings{
							Plugins: []string{"user-tool@marketplace"},
						},
						Project: &profile.ScopeSettings{
							Plugins: []string{"project-tool@marketplace"},
						},
					},
				})
				// Create stack
				env.CreateProfile(&profile.Profile{
					Name:        "scoped-stack",
					Description: "Multi-scope stack",
					Includes:    []string{"scoped-leaf"},
				})
			})

			It("applies plugins to correct scopes", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "scoped-stack", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Verify user-scope settings
				userSettingsPath := filepath.Join(env.ClaudeDir, "settings.json")
				userData, err := os.ReadFile(userSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var userSettings map[string]interface{}
				Expect(json.Unmarshal(userData, &userSettings)).To(Succeed())
				enabledPlugins := userSettings["enabledPlugins"].(map[string]interface{})
				Expect(enabledPlugins).To(HaveKey("user-tool@marketplace"))
				Expect(enabledPlugins).NotTo(HaveKey("project-tool@marketplace"))

				// Verify project-scope settings
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				projectData, err := os.ReadFile(projectSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var projectSettings map[string]interface{}
				Expect(json.Unmarshal(projectData, &projectSettings)).To(Succeed())
				projectPlugins := projectSettings["enabledPlugins"].(map[string]interface{})
				Expect(projectPlugins).To(HaveKey("project-tool@marketplace"))
			})
		})

		Context("with --scope flag", func() {
			BeforeEach(func() {
				env.CreateProfile(&profile.Profile{
					Name:    "leaf",
					Plugins: []string{"leaf-plugin@marketplace"},
				})
				env.CreateProfile(&profile.Profile{
					Name:     "stack-with-scope",
					Includes: []string{"leaf"},
				})
			})

			It("rejects explicit scope for stack profiles", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "stack-with-scope", "--scope", "user", "-y")

				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.Stderr).To(ContainSubstring("scope"))
			})
		})

		Context("with merged leaf profiles", func() {
			BeforeEach(func() {
				// Two leaf profiles with different plugins
				env.CreateProfile(&profile.Profile{
					Name:    "tools-a",
					Plugins: []string{"plugin-a@marketplace"},
				})
				env.CreateProfile(&profile.Profile{
					Name:    "tools-b",
					Plugins: []string{"plugin-b@marketplace"},
				})
				// Stack composing both
				env.CreateProfile(&profile.Profile{
					Name:     "combined",
					Includes: []string{"tools-a", "tools-b"},
				})
			})

			It("merges plugins from all included profiles", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "combined", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Verify both plugins end up in user settings (flat plugins -> user scope)
				userSettingsPath := filepath.Join(env.ClaudeDir, "settings.json")
				userData, err := os.ReadFile(userSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var userSettings map[string]interface{}
				Expect(json.Unmarshal(userData, &userSettings)).To(Succeed())
				enabledPlugins := userSettings["enabledPlugins"].(map[string]interface{})
				Expect(enabledPlugins).To(HaveKey("plugin-a@marketplace"))
				Expect(enabledPlugins).To(HaveKey("plugin-b@marketplace"))
			})
		})
	})
})
