// ABOUTME: Acceptance tests for scope-aware drift detection in status command
// ABOUTME: Tests user-scope profile drift detection (local scope uses registry)
package acceptance

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v3/internal/profile"
	"github.com/claudeup/claudeup/v3/test/helpers"
)

var _ = Describe("Status drift detection scope awareness", func() {
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

	Describe("User-scoped profile drift detection", func() {
		BeforeEach(func() {
			// Create user-level profile config using test env helper
			env.SetActiveProfile("user-profile")

			// Create profile definition using test env helper
			env.CreateProfile(&profile.Profile{
				Name:         "user-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})
		})

		Context("when user scope differs from profile", func() {
			BeforeEach(func() {
				// User scope: empty
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
			})

			It("should show drift for user scope", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("System differs"))
				Expect(result.Stdout).To(ContainSubstring("user scope"))
			})

			It("should show user scope profile", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("user-profile (user scope)"))
			})
		})

		Context("when user scope matches profile", func() {
			BeforeEach(func() {
				// User scope: matches profile
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"plugin1@marketplace": true,
						"plugin2@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)
			})

			It("should not show drift", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).NotTo(ContainSubstring("System differs"))
			})
		})
	})

	Describe("Local-scoped profile drift detection", func() {
		BeforeEach(func() {
			// Create and apply local-scope profile
			env.CreateProfile(&profile.Profile{
				Name:         "local-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Apply at local scope
			result := env.RunInDir(projectDir, "profile", "apply", "local-profile", "--scope", "local", "-y")
			Expect(result.ExitCode).To(Equal(0))
		})

		Context("when local scope differs from profile", func() {
			BeforeEach(func() {
				// Local scope settings: empty (differs from profile)
				localSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.local.json"), localSettings)
			})

			It("should show drift for local scope", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("System differs"))
				Expect(result.Stdout).To(ContainSubstring("local scope"))
			})
		})
	})

	Describe("No active profile", func() {
		It("should not show drift when no profile is active", func() {
			result := env.RunInDir(projectDir, "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Active Profile: none"))
			Expect(result.Stdout).NotTo(ContainSubstring("System differs"))
		})
	})
})
