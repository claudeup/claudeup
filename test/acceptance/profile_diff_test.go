// ABOUTME: Acceptance tests for profile diff command drift guidance
// ABOUTME: Ensures profile diff provides actionable commands for fixing drift
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
)

var _ = Describe("Profile diff drift guidance", func() {
	var (
		env        *helpers.TestEnv
		binaryPath string
		projectDir string
	)

	BeforeEach(func() {
		binaryPath = helpers.BuildBinary()
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()

		projectDir = env.ProjectDir("test-project")
		err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
		Expect(err).NotTo(HaveOccurred())

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

	Describe("Extra plugins drift", func() {
		BeforeEach(func() {
			// Create project profile with 2 plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile at project scope
			projectConfig := map[string]interface{}{
				"version":       "1",
				"profile":       "test-profile",
				"profileSource": "custom",
				"marketplaces":  []interface{}{},
				"plugins": []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

			// Project settings has extra plugin
			projectSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace":      true,
					"plugin2@marketplace":      true,
					"extra-plugin@marketplace": true, // Not in profile
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
		})

		It("should show drift and suggest removing extra plugins", func() {
			result := env.RunInDir(projectDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			// Should show the extra plugin
			Expect(result.Stdout).To(ContainSubstring("additional plugin"))
			Expect(result.Stdout).To(ContainSubstring("extra-plugin@marketplace"))
			// Should suggest how to fix it
			Expect(result.Stdout).To(ContainSubstring("Remove extra plugin"))
			Expect(result.Stdout).To(ContainSubstring("profile apply test-profile --scope project --reset"))
			Expect(result.Stdout).To(ContainSubstring("profile clean --scope project"))
		})
	})

	Describe("Missing plugins drift", func() {
		BeforeEach(func() {
			// Create project profile with 2 plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile at project scope
			projectConfig := map[string]interface{}{
				"version":       "1",
				"profile":       "test-profile",
				"profileSource": "custom",
				"marketplaces":  []interface{}{},
				"plugins": []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

			// Project settings missing a plugin
			projectSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					// plugin2 is missing
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
		})

		It("should show drift and suggest installing missing plugins", func() {
			result := env.RunInDir(projectDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			// Should show the missing plugin
			Expect(result.Stdout).To(ContainSubstring("missing plugin"))
			Expect(result.Stdout).To(ContainSubstring("plugin2@marketplace"))
			// Should suggest how to fix it (profile sync for project scope)
			Expect(result.Stdout).To(ContainSubstring("Install missing plugin"))
			Expect(result.Stdout).To(ContainSubstring("profile sync"))
		})
	})

	Describe("Both extra and missing plugins", func() {
		BeforeEach(func() {
			// Create project profile with 2 plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile at project scope
			projectConfig := map[string]interface{}{
				"version":       "1",
				"profile":       "test-profile",
				"profileSource": "custom",
				"marketplaces":  []interface{}{},
				"plugins": []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

			// Project settings has extra plugin AND missing plugin
			projectSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace":      true,
					"extra-plugin@marketplace": true, // Not in profile
					// plugin2 is missing
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
		})

		It("should suggest reset to fix both types of drift", func() {
			result := env.RunInDir(projectDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			// Should show both types of drift
			Expect(result.Stdout).To(ContainSubstring("additional plugin"))
			Expect(result.Stdout).To(ContainSubstring("missing plugin"))
			// Should suggest reset (handles both)
			Expect(result.Stdout).To(ContainSubstring("Reset to profile"))
			Expect(result.Stdout).To(ContainSubstring("removes extra, installs missing"))
			Expect(result.Stdout).To(ContainSubstring("profile apply test-profile --scope project --reset"))
		})
	})

	Describe("User scope missing plugins", func() {
		BeforeEach(func() {
			// Create user-level profile
			env.CreateProfile(&profile.Profile{
				Name:         "user-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile at user scope
			env.SetActiveProfile("user-profile")

			// User settings missing a plugin
			userSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					// plugin2 is missing
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
		})

		It("should suggest profile apply (not sync) for user scope", func() {
			result := env.RunInDir(projectDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			// Should show the missing plugin
			Expect(result.Stdout).To(ContainSubstring("missing plugin"))
			// Should suggest profile apply (not sync, which is project-only)
			Expect(result.Stdout).To(ContainSubstring("Install missing plugin"))
			Expect(result.Stdout).To(ContainSubstring("profile apply user-profile --scope user"))
			Expect(result.Stdout).NotTo(ContainSubstring("profile sync"))
		})
	})

	Describe("No drift", func() {
		BeforeEach(func() {
			// Create project profile
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile at project scope
			projectConfig := map[string]interface{}{
				"version":       "1",
				"profile":       "test-profile",
				"profileSource": "custom",
				"marketplaces":  []interface{}{},
				"plugins": []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

			// Project settings matches profile exactly
			projectSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					"plugin2@marketplace": true,
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
		})

		It("should not show any drift guidance", func() {
			result := env.RunInDir(projectDir, "profile", "diff")

			Expect(result.ExitCode).To(Equal(0))
			// Should show match
			Expect(result.Stdout).To(ContainSubstring("Matches saved profile"))
			// Should NOT show any guidance commands
			Expect(result.Stdout).NotTo(ContainSubstring("Remove extra"))
			Expect(result.Stdout).NotTo(ContainSubstring("Install missing"))
			Expect(result.Stdout).NotTo(ContainSubstring("Reset to profile"))
		})
	})
})
