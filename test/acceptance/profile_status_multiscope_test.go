// ABOUTME: Acceptance tests for profile status with multi-scope and overlapping plugins
// ABOUTME: Tests plugin count accuracy and scope precedence display
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
)

var _ = Describe("Profile status multi-scope scenarios", func() {
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

	Describe("User scope profile status", func() {
		BeforeEach(func() {
			// Create user-scope profile with plugins
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

			// User settings matches profile
			userSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					"plugin2@marketplace": true,
				},
			}
			helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
		})

		It("should show user scope profile as active", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("user-profile"))
			Expect(result.Stdout).To(ContainSubstring("user scope"))
		})

		It("should show correct plugin count", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			// Should show 2 plugins from profile
			Expect(result.Stdout).To(ContainSubstring("2"))
		})
	})

	Describe("Local scope profile status", func() {
		BeforeEach(func() {
			// Create profile
			env.CreateProfile(&profile.Profile{
				Name:         "local-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
				},
			})

			// Apply at local scope
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")
			Expect(result.ExitCode).To(Equal(0))
		})

		It("should show local scope profile as active", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("local-profile"))
			Expect(result.Stdout).To(ContainSubstring("local scope"))
		})
	})

	Describe("No active profile", func() {
		It("should error when no profile is active", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(1))
			Expect(result.Stderr).To(ContainSubstring("no active profile set"))
		})
	})

	Describe("Local scope overrides user scope", func() {
		BeforeEach(func() {
			// Create both profiles
			env.CreateProfile(&profile.Profile{
				Name:    "user-profile",
				Plugins: []string{"user-plugin@marketplace"},
			})
			env.CreateProfile(&profile.Profile{
				Name:    "local-profile",
				Plugins: []string{"local-plugin@marketplace"},
			})

			// Set user-level profile
			env.SetActiveProfile("user-profile")

			// Apply local profile (should take precedence)
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")
			Expect(result.ExitCode).To(Equal(0))
		})

		It("should show local profile as active", func() {
			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("local-profile"))
			Expect(result.Stdout).To(ContainSubstring("local scope"))
		})
	})
})
