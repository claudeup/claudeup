// ABOUTME: Regression tests ensuring profile operations preserve non-plugin settings
// ABOUTME: Prevents critical bug where applying profiles wiped mcpServers and other fields
package profile_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v4/internal/claude"
)

var _ = Describe("Profile operations preserve non-plugin settings", func() {
	var (
		tempDir    string
		claudeDir  string
		projectDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "claudeup-test-*")
		Expect(err).NotTo(HaveOccurred())

		claudeDir = filepath.Join(tempDir, ".claude")
		projectDir = filepath.Join(tempDir, "project")

		// Create directories
		Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("User scope settings preservation", func() {
		It("should preserve mcpServers when applying profile", func() {
			// Create settings with both plugins and mcpServers
			settingsPath := filepath.Join(claudeDir, "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"old-plugin@marketplace": true,
				},
				"mcpServers": map[string]interface{}{
					"filesystem": map[string]interface{}{
						"command": "npx",
						"args":    []string{"-y", "@modelcontextprotocol/server-filesystem"},
					},
				},
				"customField": "should-be-preserved",
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Load settings and verify initial state
			settings, err := claude.LoadSettings(claudeDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(settings.EnabledPlugins).To(HaveKey("old-plugin@marketplace"))

			// Update plugins (simulating profile apply)
			settings.EnabledPlugins = map[string]bool{
				"new-plugin@marketplace": true,
			}

			// Save settings
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			// Verify mcpServers and custom fields are preserved
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			// Verify plugins changed
			plugins, ok := savedData["enabledPlugins"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(plugins).To(HaveKey("new-plugin@marketplace"))
			Expect(plugins).NotTo(HaveKey("old-plugin@marketplace"))

			// Verify mcpServers preserved
			mcpServers, ok := savedData["mcpServers"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(mcpServers).To(HaveKey("filesystem"))

			// Verify custom field preserved
			Expect(savedData).To(HaveKeyWithValue("customField", "should-be-preserved"))
		})

		It("should preserve all fields when clearing plugins", func() {
			// Create settings with multiple fields
			settingsPath := filepath.Join(claudeDir, "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
					"plugin2@marketplace": true,
				},
				"mcpServers": map[string]interface{}{
					"test-server": map[string]interface{}{
						"command": "test",
					},
				},
				"customProviders": []interface{}{
					map[string]interface{}{
						"name": "custom-provider",
					},
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Load and clear plugins
			settings, err := claude.LoadSettings(claudeDir)
			Expect(err).NotTo(HaveOccurred())
			settings.EnabledPlugins = make(map[string]bool) // Clear plugins
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			// Verify all other fields preserved
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			// Verify plugins cleared
			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(BeEmpty())

			// Verify other fields preserved
			Expect(savedData).To(HaveKey("mcpServers"))
			Expect(savedData).To(HaveKey("customProviders"))
		})
	})

	Describe("Project scope settings preservation", func() {
		It("should preserve non-plugin fields when applying profile", func() {
			// Create project settings
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"old-plugin@marketplace": true,
				},
				"projectCustomField": "preserve-me",
				"mcpServers": map[string]interface{}{
					"project-server": map[string]interface{}{
						"command": "test",
					},
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Load and update plugins
			settings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			settings.EnabledPlugins = map[string]bool{
				"new-plugin@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, settings)).To(Succeed())

			// Verify preservation
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			Expect(savedData).To(HaveKeyWithValue("projectCustomField", "preserve-me"))
			Expect(savedData).To(HaveKey("mcpServers"))
		})

		It("should preserve env configuration when applying profile", func() {
			// Create project settings with env configuration
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"old-plugin@marketplace": true,
				},
				"env": map[string]interface{}{
					"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "64000",
					"MAX_THINKING_TOKENS":           "31999",
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Load and update plugins (simulating profile apply)
			settings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			settings.EnabledPlugins = map[string]bool{
				"new-plugin@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, settings)).To(Succeed())

			// Verify env configuration is preserved
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			Expect(savedData).To(HaveKey("env"))
			envData := savedData["env"].(map[string]interface{})
			Expect(envData).To(HaveKeyWithValue("CLAUDE_CODE_MAX_OUTPUT_TOKENS", "64000"))
			Expect(envData).To(HaveKeyWithValue("MAX_THINKING_TOKENS", "31999"))

			// Verify plugins were updated
			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(1))
			Expect(plugins).To(HaveKey("new-plugin@marketplace"))
		})

		It("should only enable new profile plugins when switching to profile with fewer plugins", func() {
			// Setup: Project has 5 plugins enabled
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
					"plugin-c@marketplace": true,
					"plugin-d@marketplace": true,
					"plugin-e@marketplace": true,
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Switch to profile with only 2 plugins
			settings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			// Simulate profile apply - replace with new profile's plugins
			settings.EnabledPlugins = map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-c@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, settings)).To(Succeed())

			// Verify only the 2 new profile plugins are enabled
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(2), "should have exactly 2 plugins from new profile")
			Expect(plugins).To(HaveKey("plugin-a@marketplace"))
			Expect(plugins).To(HaveKey("plugin-c@marketplace"))
			Expect(plugins).NotTo(HaveKey("plugin-b@marketplace"), "old plugin should be disabled")
			Expect(plugins).NotTo(HaveKey("plugin-d@marketplace"), "old plugin should be disabled")
			Expect(plugins).NotTo(HaveKey("plugin-e@marketplace"), "old plugin should be disabled")
		})

		It("should add new plugins when switching to profile with more plugins", func() {
			// Setup: Project has 2 plugins enabled
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Switch to profile with 5 plugins (additions)
			settings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			settings.EnabledPlugins = map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
				"plugin-c@marketplace": true,
				"plugin-d@marketplace": true,
				"plugin-e@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, settings)).To(Succeed())

			// Verify all 5 plugins are enabled
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(5), "should have all 5 plugins from new profile")
			Expect(plugins).To(HaveKey("plugin-a@marketplace"))
			Expect(plugins).To(HaveKey("plugin-b@marketplace"))
			Expect(plugins).To(HaveKey("plugin-c@marketplace"))
			Expect(plugins).To(HaveKey("plugin-d@marketplace"))
			Expect(plugins).To(HaveKey("plugin-e@marketplace"))
		})

		It("should maintain same plugins when switching to equivalent profile", func() {
			// Setup: Project has 3 plugins enabled
			settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
					"plugin-c@marketplace": true,
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Switch to profile with same 3 plugins (clone/equivalent)
			settings, err := claude.LoadSettingsForScope("project", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			settings.EnabledPlugins = map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
				"plugin-c@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, settings)).To(Succeed())

			// Verify same 3 plugins remain
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(3), "should have same 3 plugins")
			Expect(plugins).To(HaveKey("plugin-a@marketplace"))
			Expect(plugins).To(HaveKey("plugin-b@marketplace"))
			Expect(plugins).To(HaveKey("plugin-c@marketplace"))
		})
	})

	Describe("Local scope settings preservation", func() {
		It("should preserve non-plugin fields when applying profile", func() {
			// Create local settings
			settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"old-plugin@marketplace": true,
				},
				"localCustomField": "keep-me",
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Load and update plugins
			settings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			settings.EnabledPlugins = map[string]bool{
				"new-plugin@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, settings)).To(Succeed())

			// Verify preservation
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			Expect(savedData).To(HaveKeyWithValue("localCustomField", "keep-me"))
		})

		It("should only enable new profile plugins when switching to profile with fewer plugins", func() {
			// Setup: Local has 5 plugins enabled
			settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
					"plugin-c@marketplace": true,
					"plugin-d@marketplace": true,
					"plugin-e@marketplace": true,
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Switch to profile with only 2 plugins
			settings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			settings.EnabledPlugins = map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-c@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, settings)).To(Succeed())

			// Verify only the 2 new profile plugins are enabled
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(2), "should have exactly 2 plugins from new profile")
			Expect(plugins).To(HaveKey("plugin-a@marketplace"))
			Expect(plugins).To(HaveKey("plugin-c@marketplace"))
			Expect(plugins).NotTo(HaveKey("plugin-b@marketplace"), "old plugin should be disabled")
			Expect(plugins).NotTo(HaveKey("plugin-d@marketplace"), "old plugin should be disabled")
			Expect(plugins).NotTo(HaveKey("plugin-e@marketplace"), "old plugin should be disabled")
		})

		It("should add new plugins when switching to profile with more plugins", func() {
			// Setup: Local has 2 plugins enabled
			settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Switch to profile with 5 plugins (additions)
			settings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			settings.EnabledPlugins = map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
				"plugin-c@marketplace": true,
				"plugin-d@marketplace": true,
				"plugin-e@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, settings)).To(Succeed())

			// Verify all 5 plugins are enabled
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(5), "should have all 5 plugins from new profile")
			Expect(plugins).To(HaveKey("plugin-a@marketplace"))
			Expect(plugins).To(HaveKey("plugin-b@marketplace"))
			Expect(plugins).To(HaveKey("plugin-c@marketplace"))
			Expect(plugins).To(HaveKey("plugin-d@marketplace"))
			Expect(plugins).To(HaveKey("plugin-e@marketplace"))
		})

		It("should maintain same plugins when switching to equivalent profile", func() {
			// Setup: Local has 3 plugins enabled
			settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin-a@marketplace": true,
					"plugin-b@marketplace": true,
					"plugin-c@marketplace": true,
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Switch to profile with same 3 plugins (clone/equivalent)
			settings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			settings.EnabledPlugins = map[string]bool{
				"plugin-a@marketplace": true,
				"plugin-b@marketplace": true,
				"plugin-c@marketplace": true,
			}
			Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, settings)).To(Succeed())

			// Verify same 3 plugins remain
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			plugins := savedData["enabledPlugins"].(map[string]interface{})
			Expect(plugins).To(HaveLen(3), "should have same 3 plugins")
			Expect(plugins).To(HaveKey("plugin-a@marketplace"))
			Expect(plugins).To(HaveKey("plugin-b@marketplace"))
			Expect(plugins).To(HaveKey("plugin-c@marketplace"))
		})
	})

	Describe("Profile apply operation", func() {
		It("should preserve mcpServers when applying profile to user scope", func() {
			// Create user settings with mcpServers
			settingsPath := filepath.Join(claudeDir, "settings.json")
			settingsData := map[string]interface{}{
				"enabledPlugins": map[string]bool{
					"plugin1@marketplace": true,
				},
				"mcpServers": map[string]interface{}{
					"important-server": map[string]interface{}{
						"command": "npx",
						"args":    []string{"-y", "test-server"},
					},
				},
			}
			data, err := json.MarshalIndent(settingsData, "", "  ")
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(settingsPath, data, 0644)).To(Succeed())

			// Simulate what profile apply does: load, update plugins, save
			userSettings, err := claude.LoadSettings(claudeDir)
			Expect(err).NotTo(HaveOccurred())

			// Apply profile plugins (replace)
			userSettings.EnabledPlugins = map[string]bool{
				"profile-plugin@marketplace": true,
			}

			Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

			// Verify mcpServers survived
			var savedData map[string]interface{}
			savedBytes, err := os.ReadFile(settingsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(savedBytes, &savedData)).To(Succeed())

			mcpServers, ok := savedData["mcpServers"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "mcpServers should be preserved")
			Expect(mcpServers).To(HaveKey("important-server"))
		})
	})
})
