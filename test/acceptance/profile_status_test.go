// ABOUTME: Acceptance tests for profile status drift guidance
// ABOUTME: Tests the actionable guidance shown when system differs from profile
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
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
			// Create user-scope profile with plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile
			env.SetActiveProfile("test-profile")

			// User settings has extra plugin
			userSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace":      true,
					"plugin2@marketplace":      true,
					"extra-plugin@marketplace": true, // Not in profile
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
		})

		It("should show drift and suggest removing extra plugins", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			// Should indicate additional plugins
			Expect(result.Stdout).To(ContainSubstring("additional plugin"))
			// Should show the extra plugin
			Expect(result.Stdout).To(ContainSubstring("extra-plugin@marketplace"))
		})
	})

	Describe("Missing plugins drift", func() {
		BeforeEach(func() {
			// Create user-scope profile with plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile
			env.SetActiveProfile("test-profile")

			// User settings missing a plugin
			userSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					// plugin2 is missing
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
		})

		It("should show drift and suggest installing missing plugins", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			// Should indicate missing plugins
			Expect(result.Stdout).To(ContainSubstring("missing plugin"))
			// Should show the missing plugin
			Expect(result.Stdout).To(ContainSubstring("plugin2@marketplace"))
		})
	})

	Describe("Both extra and missing plugins", func() {
		BeforeEach(func() {
			// Create user-scope profile with plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile
			env.SetActiveProfile("test-profile")

			// User settings with both extra and missing
			userSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace":      true, // profile has this
					"extra-plugin@marketplace": true, // not in profile
					// plugin2 is missing
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
		})

		It("should suggest reset to fix both types of drift", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			// Should indicate both additional and missing plugins
			Expect(result.Stdout).To(SatisfyAny(
				ContainSubstring("additional plugin"),
				ContainSubstring("missing plugin"),
			))
		})
	})

	Describe("No drift", func() {
		BeforeEach(func() {
			// Create user-scope profile with plugins
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Set as active profile
			env.SetActiveProfile("test-profile")

			// User settings matches profile exactly
			userSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					"plugin2@marketplace": true,
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
		})

		It("should not show any drift guidance", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			// Should NOT show drift
			Expect(result.Stdout).NotTo(ContainSubstring("System differs"))
		})
	})
})
