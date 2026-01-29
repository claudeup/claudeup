// ABOUTME: Acceptance tests for profile status with multi-scope and overlapping plugins
// ABOUTME: Tests plugin count accuracy and CLAUDE_CONFIG_DIR path display
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

	Describe("Overlapping plugins across scopes", func() {
		Context("when same plugin exists in both user and project scope of profile", func() {
			BeforeEach(func() {
				// Create multi-scope profile with OVERLAPPING plugin (same in both scopes)
				env.CreateProfile(&profile.Profile{
					Name:         "overlap-profile",
					Marketplaces: []profile.Marketplace{},
					PerScope: &profile.PerScopeSettings{
						User: &profile.ScopeSettings{
							Plugins: []string{
								"shared-plugin@marketplace",   // In both scopes
								"user-only-plugin@marketplace",
							},
						},
						Project: &profile.ScopeSettings{
							Plugins: []string{
								"shared-plugin@marketplace",      // In both scopes
								"project-only-plugin@marketplace",
							},
						},
					},
				})

				// Set as active profile at project scope
				projectConfig := map[string]interface{}{
					"version":       "1",
					"profile":       "overlap-profile",
					"profileSource": "custom",
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// User settings has the user-scope plugins
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"shared-plugin@marketplace":    true,
						"user-only-plugin@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project settings has the project-scope plugins
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"shared-plugin@marketplace":      true,
						"project-only-plugin@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should correctly count unique plugins in effective configuration", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// When all plugins match exactly (no drift), should show simple match message
				// Profile has: shared-plugin (dedupe), user-only-plugin, project-only-plugin = 3 unique
				// Current has exactly those 3 plugins (2 in user scope + 2 in project scope, with 1 shared)
				// Since there's no diff, it shows "Matches saved profile"
				Expect(result.Stdout).To(ContainSubstring("Effective configuration"))
				Expect(result.Stdout).To(ContainSubstring("Matches saved profile"))
			})

			It("should show active scope matches profile", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// Project scope should show match with 2 plugins
				Expect(result.Stdout).To(ContainSubstring("project scope"))
				Expect(result.Stdout).To(ContainSubstring("Matches saved profile (2 plugins)"))
			})

			It("should show user scope matches profile", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// User scope should show match with 2 plugins
				Expect(result.Stdout).To(ContainSubstring("user scope"))
				Expect(result.Stdout).To(ContainSubstring("Matches profile (2 plugins)"))
			})
		})

		Context("when overlapping plugin is missing from one scope", func() {
			BeforeEach(func() {
				// Create multi-scope profile with OVERLAPPING plugin
				env.CreateProfile(&profile.Profile{
					Name:         "overlap-profile",
					Marketplaces: []profile.Marketplace{},
					PerScope: &profile.PerScopeSettings{
						User: &profile.ScopeSettings{
							Plugins: []string{
								"shared-plugin@marketplace",
								"user-only-plugin@marketplace",
							},
						},
						Project: &profile.ScopeSettings{
							Plugins: []string{
								"shared-plugin@marketplace",
								"project-only-plugin@marketplace",
							},
						},
					},
				})

				// Set as active profile at project scope
				projectConfig := map[string]interface{}{
					"version":       "1",
					"profile":       "overlap-profile",
					"profileSource": "custom",
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// User settings - MISSING shared-plugin
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"user-only-plugin@marketplace": true,
						// shared-plugin is missing
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project settings has all plugins
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"shared-plugin@marketplace":      true,
						"project-only-plugin@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should show missing plugin in user scope", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// User scope should show 1 missing
				Expect(result.Stdout).To(ContainSubstring("1 missing"))
				Expect(result.Stdout).To(ContainSubstring("shared-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(missing)"))
			})
		})
	})

	Describe("Effective configuration calculation", func() {
		Context("with plugins at multiple scopes plus extras", func() {
			BeforeEach(func() {
				// Create profile with 2 user plugins and 1 project plugin
				env.CreateProfile(&profile.Profile{
					Name:         "calc-test-profile",
					Marketplaces: []profile.Marketplace{},
					PerScope: &profile.PerScopeSettings{
						User: &profile.ScopeSettings{
							Plugins: []string{
								"profile-user-1@marketplace",
								"profile-user-2@marketplace",
							},
						},
						Project: &profile.ScopeSettings{
							Plugins: []string{
								"profile-project-1@marketplace",
							},
						},
					},
				})

				// Set as active profile at project scope
				projectConfig := map[string]interface{}{
					"version":       "1",
					"profile":       "calc-test-profile",
					"profileSource": "custom",
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// User settings has profile plugins + 1 extra
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"profile-user-1@marketplace": true,
						"profile-user-2@marketplace": true,
						"extra-user@marketplace":     true, // Not in profile
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project settings has profile plugin + 1 extra
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"profile-project-1@marketplace": true,
						"extra-project@marketplace":     true, // Not in profile
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should correctly calculate effective plugin totals", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// Profile has 3 unique plugins (2 user + 1 project)
				// Current has 5 unique plugins (3 user + 2 project)
				// So: 5 total = 3 from profile + 2 not in profile
				Expect(result.Stdout).To(ContainSubstring("5 plugins total"))
				Expect(result.Stdout).To(ContainSubstring("3 from profile"))
				Expect(result.Stdout).To(ContainSubstring("2 not in profile"))
			})
		})

		Context("with missing plugins", func() {
			BeforeEach(func() {
				// Create profile with 3 plugins
				env.CreateProfile(&profile.Profile{
					Name:         "missing-test-profile",
					Marketplaces: []profile.Marketplace{},
					PerScope: &profile.PerScopeSettings{
						User: &profile.ScopeSettings{
							Plugins: []string{
								"profile-user-1@marketplace",
								"profile-user-2@marketplace",
							},
						},
						Project: &profile.ScopeSettings{
							Plugins: []string{
								"profile-project-1@marketplace",
							},
						},
					},
				})

				// Set as active profile at project scope
				projectConfig := map[string]interface{}{
					"version":       "1",
					"profile":       "missing-test-profile",
					"profileSource": "custom",
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// User settings missing 1 plugin
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"profile-user-1@marketplace": true,
						// profile-user-2 is missing
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project settings has all
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"profile-project-1@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should show missing count in effective configuration", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// Profile has 3, current has 2, 1 missing
				// So: 2 total = 3 from profile - 1 missing
				Expect(result.Stdout).To(ContainSubstring("2 plugins total"))
				Expect(result.Stdout).To(ContainSubstring("3 from profile"))
				Expect(result.Stdout).To(ContainSubstring("1 missing"))
			})
		})
	})

	Describe("CLAUDE_CONFIG_DIR path display", func() {
		It("should show the actual CLAUDE_CONFIG_DIR path in user scope display", func() {
			// Create a simple profile
			env.CreateProfile(&profile.Profile{
				Name:         "path-test-profile",
				Marketplaces: []profile.Marketplace{},
				PerScope: &profile.PerScopeSettings{
					Project: &profile.ScopeSettings{
						Plugins: []string{"test-plugin@marketplace"},
					},
				},
			})

			// Set as active profile at project scope
			projectConfig := map[string]interface{}{
				"version":       "1",
				"profile":       "path-test-profile",
				"profileSource": "custom",
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

			// Project settings
			projectSettings := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"test-plugin@marketplace": true,
				},
			}
			helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)

			result := env.RunInDir(projectDir, "profile", "status")

			Expect(result.ExitCode).To(Equal(0))

			// Should show the actual ClaudeDir path (from test env), NOT hardcoded ~/.claude
			// The test env sets CLAUDE_CONFIG_DIR to a temp directory
			Expect(result.Stdout).To(ContainSubstring(env.ClaudeDir))
			Expect(result.Stdout).NotTo(ContainSubstring("~/.claude/settings.json"))
		})
	})

	Describe("Non-active scope missing plugins display", func() {
		Context("when user scope has missing plugins (profile active at project)", func() {
			BeforeEach(func() {
				// Create multi-scope profile
				env.CreateProfile(&profile.Profile{
					Name:         "missing-scope-test",
					Marketplaces: []profile.Marketplace{},
					PerScope: &profile.PerScopeSettings{
						User: &profile.ScopeSettings{
							Plugins: []string{
								"user-plugin-1@marketplace",
								"user-plugin-2@marketplace",
								"user-plugin-3@marketplace",
							},
						},
						Project: &profile.ScopeSettings{
							Plugins: []string{
								"project-plugin@marketplace",
							},
						},
					},
				})

				// Set as active profile at project scope
				projectConfig := map[string]interface{}{
					"version":       "1",
					"profile":       "missing-scope-test",
					"profileSource": "custom",
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claudeup.json"), projectConfig)

				// User settings - only has 1 of 3 profile plugins, plus 1 extra
				userSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"user-plugin-1@marketplace": true,
						// user-plugin-2 and user-plugin-3 missing
						"extra-plugin@marketplace": true, // not in profile
					},
				}
				helpers.WriteJSON(filepath.Join(env.ClaudeDir, "settings.json"), userSettings)

				// Project settings matches
				projectSettings := map[string]interface{}{
					"enabledPlugins": map[string]bool{
						"project-plugin@marketplace": true,
					},
				}
				helpers.WriteJSON(filepath.Join(projectDir, ".claude", "settings.json"), projectSettings)
			})

			It("should show both extra and missing plugins for non-active scope", func() {
				result := env.RunInDir(projectDir, "profile", "status")

				Expect(result.ExitCode).To(Equal(0))

				// User scope should show mix: 1 from profile, 1 extra, 2 missing
				Expect(result.Stdout).To(ContainSubstring("user scope"))
				Expect(result.Stdout).To(ContainSubstring("1 from profile"))
				Expect(result.Stdout).To(ContainSubstring("1 extra"))
				Expect(result.Stdout).To(ContainSubstring("2 missing"))

				// Should list the extra and missing plugins
				Expect(result.Stdout).To(ContainSubstring("extra-plugin@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(extra)"))
				Expect(result.Stdout).To(ContainSubstring("user-plugin-2@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("user-plugin-3@marketplace"))
				Expect(result.Stdout).To(ContainSubstring("(missing)"))
			})
		})
	})
})
