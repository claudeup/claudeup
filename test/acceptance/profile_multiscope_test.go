// ABOUTME: Acceptance tests for multi-scope profile functionality
// ABOUTME: Tests save/apply commands with per-scope settings capture and restoration
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v2/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multi-Scope Profiles", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
	})

	Describe("profile save", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("multiscope-test")
		})

		Context("when settings exist at multiple scopes", func() {
			BeforeEach(func() {
				// Set up user-scope settings
				env.CreateSettings(map[string]bool{
					"user-plugin-a@marketplace": true,
					"user-plugin-b@marketplace": true,
				})

				// Set up project-scope settings
				claudeDir := filepath.Join(projectDir, ".claude")
				Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
				projectSettings := `{"enabledPlugins":{"project-plugin-x@marketplace":true}}`
				Expect(os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(projectSettings), 0644)).To(Succeed())

				// Set up local-scope settings
				localSettings := `{"enabledPlugins":{"local-plugin-z@marketplace":true}}`
				Expect(os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(localSettings), 0644)).To(Succeed())
			})

			It("captures all scopes in a single profile", func() {
				result := env.RunInDir(projectDir, "profile", "save", "multi-scope-test", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Load the saved profile
				profilePath := filepath.Join(env.ProfilesDir, "multi-scope-test.json")
				data, err := os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				var profile map[string]interface{}
				Expect(json.Unmarshal(data, &profile)).To(Succeed())

				// Verify perScope structure exists
				perScope, ok := profile["perScope"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "expected perScope field in profile")

				// Verify user scope
				userScope, ok := perScope["user"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "expected user scope in perScope")
				userPlugins := userScope["plugins"].([]interface{})
				Expect(userPlugins).To(HaveLen(2))

				// Verify project scope
				projectScope, ok := perScope["project"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "expected project scope in perScope")
				projectPlugins := projectScope["plugins"].([]interface{})
				Expect(projectPlugins).To(HaveLen(1))
				Expect(projectPlugins[0]).To(Equal("project-plugin-x@marketplace"))

				// Verify local scope
				localScope, ok := perScope["local"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "expected local scope in perScope")
				localPlugins := localScope["plugins"].([]interface{})
				Expect(localPlugins).To(HaveLen(1))
				Expect(localPlugins[0]).To(Equal("local-plugin-z@marketplace"))
			})

			It("saves profile to user directory (not project)", func() {
				result := env.RunInDir(projectDir, "profile", "save", "multi-scope-test", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Profile should be in user profiles directory
				userProfilePath := filepath.Join(env.ProfilesDir, "multi-scope-test.json")
				Expect(userProfilePath).To(BeAnExistingFile())

				// Profile should NOT be in project directory
				projectProfilePath := filepath.Join(projectDir, ".claudeup", "profiles", "multi-scope-test.json")
				Expect(projectProfilePath).NotTo(BeAnExistingFile())
			})
		})

		Context("when only user-scope settings exist", func() {
			BeforeEach(func() {
				env.CreateSettings(map[string]bool{
					"user-only-plugin@marketplace": true,
				})
			})

			It("creates profile with only user scope populated", func() {
				result := env.RunInDir(projectDir, "profile", "save", "user-only-test", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Load and verify profile
				profilePath := filepath.Join(env.ProfilesDir, "user-only-test.json")
				data, err := os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				var profile map[string]interface{}
				Expect(json.Unmarshal(data, &profile)).To(Succeed())

				perScope, ok := profile["perScope"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				// User scope should exist
				_, hasUser := perScope["user"]
				Expect(hasUser).To(BeTrue())

				// Project and local should be nil/absent
				_, hasProject := perScope["project"]
				_, hasLocal := perScope["local"]
				Expect(hasProject).To(BeFalse(), "project scope should not exist")
				Expect(hasLocal).To(BeFalse(), "local scope should not exist")
			})
		})
	})

	Describe("profile apply", func() {
		var projectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("apply-test")
			Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())
		})

		Context("with a multi-scope profile", func() {
			BeforeEach(func() {
				// Create a multi-scope profile directly
				profile := map[string]interface{}{
					"name":        "multi-test",
					"description": "Multi-scope test profile",
					"perScope": map[string]interface{}{
						"user": map[string]interface{}{
							"plugins": []string{"user-plugin@marketplace"},
						},
						"project": map[string]interface{}{
							"plugins": []string{"project-plugin@marketplace"},
						},
						"local": map[string]interface{}{
							"plugins": []string{"local-plugin@marketplace"},
						},
					},
				}
				data, _ := json.MarshalIndent(profile, "", "  ")
				Expect(os.WriteFile(filepath.Join(env.ProfilesDir, "multi-test.json"), data, 0644)).To(Succeed())
			})

			It("applies settings to all scopes", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "multi-test", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Verify user-scope settings
				userSettingsPath := filepath.Join(env.ClaudeDir, "settings.json")
				userData, err := os.ReadFile(userSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var userSettings map[string]interface{}
				Expect(json.Unmarshal(userData, &userSettings)).To(Succeed())
				enabledPlugins := userSettings["enabledPlugins"].(map[string]interface{})
				Expect(enabledPlugins["user-plugin@marketplace"]).To(BeTrue())
				// User scope should NOT have project/local plugins
				Expect(enabledPlugins).NotTo(HaveKey("project-plugin@marketplace"))
				Expect(enabledPlugins).NotTo(HaveKey("local-plugin@marketplace"))

				// Verify project-scope settings
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				projectData, err := os.ReadFile(projectSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var projectSettings map[string]interface{}
				Expect(json.Unmarshal(projectData, &projectSettings)).To(Succeed())
				projectPlugins := projectSettings["enabledPlugins"].(map[string]interface{})
				Expect(projectPlugins["project-plugin@marketplace"]).To(BeTrue())

				// Verify local-scope settings
				localSettingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
				localData, err := os.ReadFile(localSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var localSettings map[string]interface{}
				Expect(json.Unmarshal(localData, &localSettings)).To(Succeed())
				localPlugins := localSettings["enabledPlugins"].(map[string]interface{})
				Expect(localPlugins["local-plugin@marketplace"]).To(BeTrue())
			})
		})

		Context("with a legacy (flat) profile", func() {
			BeforeEach(func() {
				// Create a legacy profile (no perScope)
				profile := map[string]interface{}{
					"name":        "legacy-test",
					"description": "Legacy test profile",
					"plugins":     []string{"legacy-plugin@marketplace"},
				}
				data, _ := json.MarshalIndent(profile, "", "  ")
				Expect(os.WriteFile(filepath.Join(env.ProfilesDir, "legacy-test.json"), data, 0644)).To(Succeed())
			})

			It("applies settings to user scope only (backward compatibility)", func() {
				result := env.RunInDir(projectDir, "profile", "apply", "legacy-test", "-y")

				Expect(result.ExitCode).To(Equal(0))

				// Verify user-scope settings
				userSettingsPath := filepath.Join(env.ClaudeDir, "settings.json")
				userData, err := os.ReadFile(userSettingsPath)
				Expect(err).NotTo(HaveOccurred())
				var userSettings map[string]interface{}
				Expect(json.Unmarshal(userData, &userSettings)).To(Succeed())
				enabledPlugins := userSettings["enabledPlugins"].(map[string]interface{})
				Expect(enabledPlugins["legacy-plugin@marketplace"]).To(BeTrue())

				// Project settings should NOT be created or modified
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				if _, err := os.Stat(projectSettingsPath); err == nil {
					projectData, _ := os.ReadFile(projectSettingsPath)
					var projectSettings map[string]interface{}
					json.Unmarshal(projectData, &projectSettings)
					if enabledPlugins, ok := projectSettings["enabledPlugins"].(map[string]interface{}); ok {
						Expect(enabledPlugins).NotTo(HaveKey("legacy-plugin@marketplace"))
					}
				}
			})
		})
	})

	Describe("round-trip: save then apply", func() {
		var projectDir string
		var newProjectDir string

		BeforeEach(func() {
			projectDir = env.ProjectDir("source-project")
			newProjectDir = env.ProjectDir("target-project")

			// Create .claude directories
			Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(newProjectDir, ".claude"), 0755)).To(Succeed())
		})

		It("preserves scope information through save and apply cycle", func() {
			// Set up source project with multi-scope settings
			env.CreateSettings(map[string]bool{
				"user-plugin@marketplace": true,
			})
			projectSettings := `{"enabledPlugins":{"project-plugin@marketplace":true}}`
			Expect(os.WriteFile(filepath.Join(projectDir, ".claude", "settings.json"), []byte(projectSettings), 0644)).To(Succeed())
			localSettings := `{"enabledPlugins":{"local-plugin@marketplace":true}}`
			Expect(os.WriteFile(filepath.Join(projectDir, ".claude", "settings.local.json"), []byte(localSettings), 0644)).To(Succeed())

			// Save the profile
			saveResult := env.RunInDir(projectDir, "profile", "save", "round-trip-test", "-y")
			Expect(saveResult.ExitCode).To(Equal(0))

			// Clear all settings to ensure apply actually writes them
			env.CreateSettings(map[string]bool{})

			// Apply to new project
			applyResult := env.RunInDir(newProjectDir, "profile", "apply", "round-trip-test", "-y")
			Expect(applyResult.ExitCode).To(Equal(0))

			// Verify user scope was restored
			userSettingsPath := filepath.Join(env.ClaudeDir, "settings.json")
			userData, _ := os.ReadFile(userSettingsPath)
			var userSettings map[string]interface{}
			json.Unmarshal(userData, &userSettings)
			enabledPlugins := userSettings["enabledPlugins"].(map[string]interface{})
			Expect(enabledPlugins["user-plugin@marketplace"]).To(BeTrue())

			// Verify project scope was restored to NEW project
			projectSettingsPath := filepath.Join(newProjectDir, ".claude", "settings.json")
			projectData, _ := os.ReadFile(projectSettingsPath)
			var projectSettingsRestored map[string]interface{}
			json.Unmarshal(projectData, &projectSettingsRestored)
			projectPlugins := projectSettingsRestored["enabledPlugins"].(map[string]interface{})
			Expect(projectPlugins["project-plugin@marketplace"]).To(BeTrue())

			// Verify local scope was restored to NEW project
			localSettingsPath := filepath.Join(newProjectDir, ".claude", "settings.local.json")
			localData, _ := os.ReadFile(localSettingsPath)
			var localSettingsRestored map[string]interface{}
			json.Unmarshal(localData, &localSettingsRestored)
			localPlugins := localSettingsRestored["enabledPlugins"].(map[string]interface{})
			Expect(localPlugins["local-plugin@marketplace"]).To(BeTrue())
		})
	})
})
