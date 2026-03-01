// ABOUTME: Acceptance tests for the extras prompt during profile apply
// ABOUTME: Tests interactive add/replace behavior when live config has items not in profile
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/internal/claude"
	"github.com/claudeup/claudeup/v5/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile apply extras prompt", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
		env.CreateClaudeSettings()
		env.CreateInstalledPlugins(map[string]any{})
		env.CreateKnownMarketplaces(map[string]any{})
	})

	// createMultiScopeProfile writes a multi-scope profile with the given user-scope plugins.
	createMultiScopeProfile := func(name string, userPlugins []string) {
		profileData := map[string]any{
			"name": name,
			"perScope": map[string]any{
				"user": map[string]any{
					"plugins": userPlugins,
				},
			},
		}
		data, err := json.MarshalIndent(profileData, "", "  ")
		Expect(err).NotTo(HaveOccurred())

		profilePath := filepath.Join(env.ClaudeupDir, "profiles", name+".json")
		Expect(os.MkdirAll(filepath.Dir(profilePath), 0755)).To(Succeed())
		Expect(os.WriteFile(profilePath, data, 0644)).To(Succeed())
	}

	// setLivePlugins writes enabled plugins to the user-scope settings.json.
	setLivePlugins := func(plugins map[string]bool) {
		settingsPath := filepath.Join(env.ClaudeDir, "settings.json")
		data, err := os.ReadFile(settingsPath)
		Expect(err).NotTo(HaveOccurred())

		var settings map[string]any
		Expect(json.Unmarshal(data, &settings)).To(Succeed())

		settings["enabledPlugins"] = plugins

		out, err := json.MarshalIndent(settings, "", "  ")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(settingsPath, out, 0644)).To(Succeed())
	}

	// loadUserSettings reads the user-scope settings.json after apply.
	loadUserSettings := func() *claude.Settings {
		settings, err := claude.LoadSettings(env.ClaudeDir)
		Expect(err).NotTo(HaveOccurred())
		return settings
	}

	Describe("when live config has extras not in the profile", func() {
		BeforeEach(func() {
			// Profile only contains plugin-a
			createMultiScopeProfile("test-extras", []string{"plugin-a@market"})

			// Live config has plugin-a plus an extra plugin-b
			setLivePlugins(map[string]bool{
				"plugin-a@market": true,
				"plugin-b@market": true,
			})
		})

		It("choosing A preserves extras", func() {
			result := env.RunWithInput("a\n", "profile", "apply", "test-extras")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("not in this profile"))
			Expect(result.Stdout).To(ContainSubstring("plugin-b@market"))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))

			settings := loadUserSettings()
			Expect(settings.EnabledPlugins).To(HaveKey("plugin-a@market"))
			Expect(settings.EnabledPlugins).To(HaveKey("plugin-b@market"))
		})

		It("choosing R removes extras", func() {
			result := env.RunWithInput("r\n", "profile", "apply", "test-extras")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("not in this profile"))
			Expect(result.Stdout).To(ContainSubstring("plugin-b@market"))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))

			settings := loadUserSettings()
			Expect(settings.EnabledPlugins).To(HaveKey("plugin-a@market"))
			Expect(settings.EnabledPlugins).NotTo(HaveKey("plugin-b@market"))
		})

		It("--replace bypasses prompt and removes extras", func() {
			result := env.Run("profile", "apply", "test-extras", "-y", "--replace")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("not in this profile"))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))

			settings := loadUserSettings()
			Expect(settings.EnabledPlugins).To(HaveKey("plugin-a@market"))
			Expect(settings.EnabledPlugins).NotTo(HaveKey("plugin-b@market"))
		})

		It("-y bypasses prompt and defaults to additive", func() {
			result := env.Run("profile", "apply", "test-extras", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("not in this profile"))
			Expect(result.Stdout).To(ContainSubstring("Profile applied"))

			settings := loadUserSettings()
			Expect(settings.EnabledPlugins).To(HaveKey("plugin-a@market"))
			Expect(settings.EnabledPlugins).To(HaveKey("plugin-b@market"))
		})
	})

	Describe("when live config matches profile exactly", func() {
		BeforeEach(func() {
			// Profile and live config both have only plugin-a
			createMultiScopeProfile("test-no-extras", []string{"plugin-a@market"})

			setLivePlugins(map[string]bool{
				"plugin-a@market": true,
			})
		})

		It("skips the extras prompt", func() {
			result := env.Run("profile", "apply", "test-no-extras", "-y")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).NotTo(ContainSubstring("not in this profile"))
		})
	})
})
