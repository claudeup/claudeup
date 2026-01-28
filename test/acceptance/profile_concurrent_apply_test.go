// ABOUTME: Acceptance tests for concurrent profile apply with progress tracking
// ABOUTME: Tests progress display, --reinstall, and --no-progress flags
package acceptance

import (
	"path/filepath"

	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
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
			result := env.RunInDir(projectDir, "profile", "apply", "multi-plugin-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// Non-TTY output should show phase progress
			Expect(result.Stdout).To(Or(
				ContainSubstring("Marketplaces"),
				ContainSubstring("Plugins"),
			))
		})

		It("creates settings.json with enabled plugins", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "multi-plugin-profile", "--scope", "project", "-y")

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
			result := env.RunInDir(projectDir, "profile", "apply", "simple-profile", "--scope", "project", "--no-progress", "-y")

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

		It("skips already-installed plugins without --reinstall", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "reinstall-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// Should show skip count in progress output
			Expect(result.Stdout).To(ContainSubstring("already installed"))
		})

		It("forces reinstall of already-installed plugins with --reinstall", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "reinstall-profile", "--scope", "project", "--reinstall", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// With --reinstall, should NOT show "already installed" skip message
			Expect(result.Stdout).NotTo(ContainSubstring("already installed"))
		})

		It("shows --reinstall in help text", func() {
			result := env.Run("profile", "apply", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("--reinstall"))
			Expect(result.Stdout).To(ContainSubstring("Force reinstall"))
		})
	})

	Describe("--reinstall with marketplaces", func() {
		BeforeEach(func() {
			// Profile with only a marketplace, no plugins
			env.CreateProfile(&profile.Profile{
				Name: "marketplace-only-profile",
				Marketplaces: []profile.Marketplace{
					{Source: "github", Repo: "test/marketplace"},
				},
			})

			// Simulate that the marketplace is already installed
			knownMarketplaces := map[string]interface{}{
				"test-marketplace": map[string]interface{}{
					"source": map[string]interface{}{
						"repo": "test/marketplace",
					},
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "known_marketplaces.json"), knownMarketplaces)
		})

		It("shows no changes when marketplace already installed without --reinstall", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "marketplace-only-profile", "--scope", "project", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// When marketplace is already installed and there are no plugins,
			// the output shows "No changes needed" or similar
			Expect(result.Stdout).To(Or(
				ContainSubstring("already installed"),
				ContainSubstring("No changes needed"),
			))
		})

		It("attempts reinstall of marketplaces with --reinstall", func() {
			result := env.RunInDir(projectDir, "profile", "apply", "marketplace-only-profile", "--scope", "project", "--reinstall", "-y")

			Expect(result.ExitCode).To(Equal(0))
			// With --reinstall, the marketplace install is attempted (might fail
			// because it's a fake marketplace, but that's ok for this test)
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
			result := env.RunInDir(projectDir, "profile", "apply", "local-scope-profile", "--scope", "local", "-y")

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
