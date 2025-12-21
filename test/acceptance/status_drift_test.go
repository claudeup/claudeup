// ABOUTME: Acceptance tests for scope-aware drift detection in status command
// ABOUTME: Ensures project profiles only check project scope, preventing false drift warnings
package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/internal/profile"
	"github.com/claudeup/claudeup/test/helpers"
)

func TestStatusDrift(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Status Drift Detection Suite")
}

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

	Describe("Project-scoped profile drift detection", func() {
		BeforeEach(func() {
			// Create project profile config (.claudeup.json)
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

			// Create corresponding profile definition using test env helper
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})
		})

		Context("when user scope is empty but project scope matches", func() {
			BeforeEach(func() {
				// User scope: no plugins
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project scope: matches profile
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"plugin1@marketplace": true,
						"plugin2@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should NOT show drift for user scope", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				// Should NOT complain about user scope
				Expect(result.Stdout).NotTo(ContainSubstring("user scope"))
				Expect(result.Stdout).NotTo(ContainSubstring("System differs"))
			})

			It("should show project scope profile", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("test-profile (project scope)"))
			})
		})

		Context("when project scope differs from profile", func() {
			BeforeEach(func() {
				// User scope: has plugins (but shouldn't be checked)
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"different-plugin@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project scope: missing plugins
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should show drift for project scope only", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				Expect(result.Stdout).To(ContainSubstring("System differs"))
				Expect(result.Stdout).To(ContainSubstring("project scope"))
				// Should NOT mention user scope
				Expect(result.Stdout).NotTo(ContainSubstring("user scope"))
			})
		})

		Context("when local scope has overrides", func() {
			BeforeEach(func() {
				// Project scope: matches profile
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"plugin1@marketplace": true,
						"plugin2@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)

				// Local scope: has additional plugins (personal overrides)
				localSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"local-plugin@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.local.json"), localSettings)
			})

			It("should NOT show drift for local scope", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				// Local scope is for personal overrides, should not be checked for drift
				Expect(result.Stdout).NotTo(ContainSubstring("local scope"))
				Expect(result.Stdout).NotTo(ContainSubstring("System differs"))
			})
		})

		Context("when user scope has different marketplaces", func() {
			BeforeEach(func() {
				// Project scope: matches profile
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"plugin1@marketplace": true,
						"plugin2@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)

				// User scope: has different marketplaces installed
				// (Marketplaces are always user-scope)
				// Create known_marketplaces.json with extra marketplaces
				knownMarketplaces := map[string]interface{}{
					"extra-marketplace": map[string]interface{}{
						"source": map[string]interface{}{
							"source": "github",
							"repo":   "user/extra-marketplace",
						},
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "plugins", "known_marketplaces.json"), knownMarketplaces)
			})

			It("should NOT show drift for marketplaces", func() {
				result := env.RunInDir(projectDir, "status")

				Expect(result.ExitCode).To(Equal(0))
				// Marketplaces are user-scope only, should not be checked for project profiles
				Expect(result.Stdout).NotTo(ContainSubstring("marketplaces not in profile"))
				Expect(result.Stdout).NotTo(ContainSubstring("marketplaces missing"))
				Expect(result.Stdout).NotTo(ContainSubstring("System differs"))
			})
		})
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
	})

	Describe("Drift message clarity", func() {
		BeforeEach(func() {
			// Set up project profile with drift
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

			// Create profile definition using test env helper
			env.CreateProfile(&profile.Profile{
				Name:         "test-profile",
				Marketplaces: []profile.Marketplace{},
				Plugins: []string{
					"plugin1@marketplace",
					"plugin2@marketplace",
				},
			})

			// Project scope: missing plugins
			projectSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
		})

		It("should use 'missing' instead of 'removed'", func() {
			result := env.RunInDir(projectDir, "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("missing"))
			Expect(result.Stdout).NotTo(ContainSubstring("removed"))
		})

		It("should say 'System differs from profile'", func() {
			result := env.RunInDir(projectDir, "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("System differs from profile"))
			Expect(result.Stdout).NotTo(ContainSubstring("has unsaved changes"))
		})

		It("should show both sync options", func() {
			result := env.RunInDir(projectDir, "status")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("To sync:"))
			Expect(result.Stdout).To(ContainSubstring("Update profile to match system"))
			Expect(result.Stdout).To(ContainSubstring("Install missing to match profile"))
		})
	})
})
