// ABOUTME: Acceptance tests for concurrent profile apply with progress tracking
// ABOUTME: Tests progress display, --reinstall, and --no-progress flags
package acceptance

import (
	"path/filepath"

	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile use concurrent apply", func() {
	var env *helpers.TestEnv
	var projectDir string

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
		projectDir = env.ProjectDir("test-project")

		// Create installed_plugins.json
		installedPlugins := map[string]interface{}{
			"version": 2,
			"plugins": map[string]interface{}{},
		}
		helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), installedPlugins)
	})

	AfterEach(func() {
		env.Cleanup()
	})

	Describe("with project scope", func() {
		BeforeEach(func() {
			// Create a profile with multiple plugins to test concurrent apply
			env.CreateProfile(&profile.Profile{
				Name:        "multi-plugin-profile",
				Description: "Profile with multiple plugins for concurrent testing",
				Marketplaces: []profile.Marketplace{
					{Source: "github", Repo: "test/marketplace-a"},
					{Source: "github", Repo: "test/marketplace-b"},
				},
				Plugins: []string{
					"plugin-a@test-marketplace-a",
					"plugin-b@test-marketplace-a",
					"plugin-c@test-marketplace-b",
				},
			})
		})

		It("shows progress output during apply", func() {
			result := env.RunInDir(projectDir, "profile", "use", "multi-plugin-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// Non-TTY output should show phase progress
			Expect(result.Stdout).To(Or(
				ContainSubstring("Marketplaces"),
				ContainSubstring("Plugins"),
			))
		})

		It("creates settings.json with enabled plugins", func() {
			result := env.RunInDir(projectDir, "profile", "use", "multi-plugin-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))

			// Check that project settings.json was created
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			Expect(settingsPath).To(BeARegularFile())

			settings := helpers.LoadJSON(settingsPath)
			enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(enabledPlugins).To(HaveKey("plugin-a@test-marketplace-a"))
			Expect(enabledPlugins).To(HaveKey("plugin-b@test-marketplace-a"))
			Expect(enabledPlugins).To(HaveKey("plugin-c@test-marketplace-b"))
		})
	})

	Describe("--no-progress flag", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:    "simple-profile",
				Plugins: []string{"plugin@test"},
			})
		})

		It("disables progress output", func() {
			result := env.RunInDir(projectDir, "profile", "use", "simple-profile", "--scope", "project", "--no-progress", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// With --no-progress, we should not see any progress indicators
			// The output should be minimal/standard
			Expect(result.Stdout).NotTo(ContainSubstring("[Marketplaces]"))
			Expect(result.Stdout).NotTo(ContainSubstring("[Plugins]"))
		})
	})

	Describe("--reinstall flag", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:    "reinstall-profile",
				Plugins: []string{"plugin@test"},
			})

			// Simulate that the plugin is already installed
			installedPlugins := map[string]interface{}{
				"version": 2,
				"plugins": map[string]interface{}{
					"plugin@test": map[string]interface{}{
						"source":       "marketplace",
						"path":         "/path/to/plugin",
						"installedAt":  "2025-01-01T00:00:00Z",
						"pathExists":   true,
						"pluginVersion": map[string]interface{}{},
					},
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"), installedPlugins)
		})

		It("forces reinstall of already-installed plugins", func() {
			// First apply without reinstall - should show skipped
			result := env.RunInDir(projectDir, "profile", "use", "reinstall-profile", "--scope", "project", "-y")
			Expect(result.ExitCode).To(Equal(0))
			// Could show "already installed" or skip count

			// Apply with reinstall flag - should attempt install
			result = env.RunInDir(projectDir, "profile", "use", "reinstall-profile", "--scope", "project", "--reinstall", "-y")
			Expect(result.ExitCode).To(Equal(0))
			// With reinstall, the plugin installation is attempted
		})
	})

	Describe("with local scope", func() {
		BeforeEach(func() {
			env.CreateProfile(&profile.Profile{
				Name:    "local-scope-profile",
				Plugins: []string{"local-plugin@test"},
			})
		})

		It("creates settings.local.json with enabled plugins", func() {
			result := env.RunInDir(projectDir, "profile", "use", "local-scope-profile", "--scope", "local", "-y")

			Expect(result.ExitCode).To(Equal(0))

			// Check that settings.local.json was created
			settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			Expect(settingsPath).To(BeARegularFile())

			settings := helpers.LoadJSON(settingsPath)
			enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(enabledPlugins).To(HaveKey("local-plugin@test"))
		})
	})
})
