// ABOUTME: Integration tests for scoped profile workflows
// ABOUTME: Tests applying profiles at different scopes and detecting drift

package profile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/profile"
)

func TestScopedProfiles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scoped Profile Integration Suite")
}

var _ = Describe("Scoped Profile Workflows", func() {
	var (
		tempDir     string
		claudeDir   string
		projectDir  string
		profileMgr  *profile.Manager
		claudeAPI   *claude.API
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "scoped-profile-test-*")
		Expect(err).NotTo(HaveOccurred())

		claudeDir = filepath.Join(tempDir, ".claude")
		projectDir = filepath.Join(tempDir, "project")

		err = os.MkdirAll(filepath.Join(claudeDir, "plugins"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(projectDir, ".claudeup", "profiles"), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
		Expect(err).NotTo(HaveOccurred())

		// Create user scope settings
		userSettings := map[string]interface{}{
			"enabledPlugins": map[string]bool{
				"claude-mem@thedotmack":            true,
				"superpowers@superpowers-marketplace": true,
			},
		}
		userSettingsPath := filepath.Join(claudeDir, "settings.json")
		data, err := json.MarshalIndent(userSettings, "", "  ")
		Expect(err).NotTo(HaveOccurred())
		err = os.WriteFile(userSettingsPath, data, 0644)
		Expect(err).NotTo(HaveOccurred())

		claudeAPI = claude.NewAPI(claudeDir)
		profileMgr = profile.NewManager(projectDir, claudeAPI)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("Applying profile at project scope", func() {
		var backendProfile *profile.Profile

		BeforeEach(func() {
			backendProfile = &profile.Profile{
				Name: "backend-stack",
				Plugins: []string{
					"gopls-lsp@claude-plugins-official",
					"backend-development@claude-code-workflows",
					"tdd-workflows@claude-code-workflows",
				},
			}

			profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "backend-stack.json")
			data, err := json.MarshalIndent(backendProfile, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(profilePath, data, 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should apply profile to project scope settings", func() {
			err := profileMgr.ApplyProfile("backend-stack", claude.ScopeProject)
			Expect(err).NotTo(HaveOccurred())

			// Verify project settings were created
			projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			Expect(projectSettingsPath).To(BeARegularFile())

			// Read and verify settings
			data, err := os.ReadFile(projectSettingsPath)
			Expect(err).NotTo(HaveOccurred())

			var settings map[string]interface{}
			err = json.Unmarshal(data, &settings)
			Expect(err).NotTo(HaveOccurred())

			enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(enabledPlugins).To(HaveLen(3))
			Expect(enabledPlugins["gopls-lsp@claude-plugins-official"]).To(BeTrue())
			Expect(enabledPlugins["backend-development@claude-code-workflows"]).To(BeTrue())
			Expect(enabledPlugins["tdd-workflows@claude-code-workflows"]).To(BeTrue())
		})

		Context("when checking for drift", func() {
			BeforeEach(func() {
				err := profileMgr.ApplyProfile("backend-stack", claude.ScopeProject)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should detect no drift when settings match profile", func() {
				drift, err := profileMgr.DetectDrift("backend-stack", claude.ScopeProject)
				Expect(err).NotTo(HaveOccurred())
				Expect(drift).To(BeEmpty())
			})

			It("should detect drift when extra plugins are enabled", func() {
				// Manually add a plugin not in the profile
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				data, err := os.ReadFile(projectSettingsPath)
				Expect(err).NotTo(HaveOccurred())

				var settings map[string]interface{}
				err = json.Unmarshal(data, &settings)
				Expect(err).NotTo(HaveOccurred())

				enabledPlugins := settings["enabledPlugins"].(map[string]interface{})
				enabledPlugins["extra-plugin@marketplace"] = true

				data, err = json.MarshalIndent(settings, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(projectSettingsPath, data, 0644)
				Expect(err).NotTo(HaveOccurred())

				// Check for drift
				drift, err := profileMgr.DetectDrift("backend-stack", claude.ScopeProject)
				Expect(err).NotTo(HaveOccurred())
				Expect(drift).NotTo(BeEmpty())
				Expect(drift.ExtraPlugins).To(ContainElement("extra-plugin@marketplace"))
			})

			It("should detect drift when profile plugins are missing", func() {
				// Remove a plugin from settings
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				data, err := os.ReadFile(projectSettingsPath)
				Expect(err).NotTo(HaveOccurred())

				var settings map[string]interface{}
				err = json.Unmarshal(data, &settings)
				Expect(err).NotTo(HaveOccurred())

				enabledPlugins := settings["enabledPlugins"].(map[string]interface{})
				delete(enabledPlugins, "tdd-workflows@claude-code-workflows")

				data, err = json.MarshalIndent(settings, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(projectSettingsPath, data, 0644)
				Expect(err).NotTo(HaveOccurred())

				// Check for drift
				drift, err := profileMgr.DetectDrift("backend-stack", claude.ScopeProject)
				Expect(err).NotTo(HaveOccurred())
				Expect(drift).NotTo(BeEmpty())
				Expect(drift.MissingPlugins).To(ContainElement("tdd-workflows@claude-code-workflows"))
			})
		})

		Context("when saving changes", func() {
			It("should save profile with current enabled plugins", func() {
				// Apply profile
				err := profileMgr.ApplyProfile("backend-stack", claude.ScopeProject)
				Expect(err).NotTo(HaveOccurred())

				// Add an extra plugin
				projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				data, err := os.ReadFile(projectSettingsPath)
				Expect(err).NotTo(HaveOccurred())

				var settings map[string]interface{}
				err = json.Unmarshal(data, &settings)
				Expect(err).NotTo(HaveOccurred())

				enabledPlugins := settings["enabledPlugins"].(map[string]interface{})
				enabledPlugins["debugging-toolkit@claude-code-workflows"] = true

				data, err = json.MarshalIndent(settings, "", "  ")
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(projectSettingsPath, data, 0644)
				Expect(err).NotTo(HaveOccurred())

				// Save changes to profile
				err = profileMgr.SaveChanges("backend-stack", claude.ScopeProject)
				Expect(err).NotTo(HaveOccurred())

				// Verify profile was updated
				profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "backend-stack.json")
				data, err = os.ReadFile(profilePath)
				Expect(err).NotTo(HaveOccurred())

				var updatedProfile profile.Profile
				err = json.Unmarshal(data, &updatedProfile)
				Expect(err).NotTo(HaveOccurred())

				Expect(updatedProfile.Plugins).To(HaveLen(4))
				Expect(updatedProfile.Plugins).To(ContainElement("debugging-toolkit@claude-code-workflows"))
			})
		})
	})

	Describe("Applying profile at local scope", func() {
		var devProfile *profile.Profile

		BeforeEach(func() {
			devProfile = &profile.Profile{
				Name: "dev-tools",
				Plugins: []string{
					"systems-programming@claude-code-workflows",
					"shell-scripting@claude-code-workflows",
				},
			}

			profilePath := filepath.Join(projectDir, ".claudeup", "profiles", "dev-tools.json")
			data, err := json.MarshalIndent(devProfile, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(profilePath, data, 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should apply profile to local scope settings", func() {
			err := profileMgr.ApplyProfile("dev-tools", claude.ScopeLocal)
			Expect(err).NotTo(HaveOccurred())

			// Verify local settings were created
			localSettingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			Expect(localSettingsPath).To(BeARegularFile())

			// Read and verify settings
			data, err := os.ReadFile(localSettingsPath)
			Expect(err).NotTo(HaveOccurred())

			var settings map[string]interface{}
			err = json.Unmarshal(data, &settings)
			Expect(err).NotTo(HaveOccurred())

			enabledPlugins, ok := settings["enabledPlugins"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(enabledPlugins).To(HaveLen(2))
			Expect(enabledPlugins["systems-programming@claude-code-workflows"]).To(BeTrue())
			Expect(enabledPlugins["shell-scripting@claude-code-workflows"]).To(BeTrue())
		})

		It("should not affect project or user scope settings", func() {
			// Apply profile at local scope
			err := profileMgr.ApplyProfile("dev-tools", claude.ScopeLocal)
			Expect(err).NotTo(HaveOccurred())

			// Verify user settings unchanged
			userSettingsPath := filepath.Join(claudeDir, "settings.json")
			data, err := os.ReadFile(userSettingsPath)
			Expect(err).NotTo(HaveOccurred())

			var userSettings map[string]interface{}
			err = json.Unmarshal(data, &userSettings)
			Expect(err).NotTo(HaveOccurred())

			enabledPlugins := userSettings["enabledPlugins"].(map[string]interface{})
			Expect(enabledPlugins).To(HaveLen(2))
			Expect(enabledPlugins["claude-mem@thedotmack"]).To(BeTrue())
			Expect(enabledPlugins["superpowers@superpowers-marketplace"]).To(BeTrue())

			// Verify project settings don't exist (we only applied local)
			projectSettingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			_, err = os.Stat(projectSettingsPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
