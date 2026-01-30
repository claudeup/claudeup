// ABOUTME: Tests for plugin scope analysis functionality
// ABOUTME: Validates multi-scope plugin tracking and precedence logic
package claude_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup/v4/internal/claude"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPluginAnalysis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Analysis Suite")
}

var _ = Describe("AnalyzePluginScopes", func() {
	var (
		tempDir    string
		claudeDir  string
		projectDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "claudeup-plugin-analysis-test-*")
		Expect(err).NotTo(HaveOccurred())

		claudeDir = filepath.Join(tempDir, ".claude")
		projectDir = filepath.Join(tempDir, "project")

		// Create directory structure
		Expect(os.MkdirAll(filepath.Join(claudeDir, "plugins"), 0755)).To(Succeed())
		Expect(os.MkdirAll(projectDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("with no plugins", func() {
		It("returns empty analysis", func() {
			// Create empty registry
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: make(map[string][]claude.PluginMetadata),
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Create empty settings
			settings := &claude.Settings{
				EnabledPlugins: make(map[string]bool),
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis.Installed).To(BeEmpty())
		})
	})

	Context("with plugin installed and enabled at user scope only", func() {
		It("shows plugin enabled at user scope", func() {
			// Install plugin at user scope
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"test-plugin@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "test-plugin"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at user scope
			settings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"test-plugin@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis.Installed).To(HaveLen(1))

			info := analysis.Installed["test-plugin@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.Name).To(Equal("test-plugin@marketplace"))
			Expect(info.EnabledAt).To(Equal([]string{"user"}))
			Expect(info.InstalledAt).To(HaveLen(1))
			Expect(info.InstalledAt[0].Scope).To(Equal("user"))
			Expect(info.ActiveSource).To(Equal("user"))
		})
	})

	Context("with plugin enabled at multiple scopes", func() {
		It("shows all scopes where enabled", func() {
			// Install plugin at user and project scopes
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"multi-scope@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "multi-scope"),
						},
						{
							Scope:       "project",
							Version:     "1.0.0",
							InstalledAt: "2024-01-02",
							InstallPath: filepath.Join(projectDir, ".claude", "plugins", "multi-scope"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at user scope
			userSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"multi-scope@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

			// Enable at project scope
			projectSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"multi-scope@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis.Installed).To(HaveLen(1))

			info := analysis.Installed["multi-scope@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(ConsistOf("user", "project"))
			Expect(info.InstalledAt).To(HaveLen(2))
			// Project scope has higher precedence
			Expect(info.ActiveSource).To(Equal("project"))
		})
	})

	Context("with plugin enabled at local scope", func() {
		It("uses local installation with highest precedence", func() {
			// Install at all three scopes
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"local-plugin@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "local-plugin"),
						},
						{
							Scope:       "project",
							Version:     "1.0.0",
							InstalledAt: "2024-01-02",
							InstallPath: filepath.Join(projectDir, ".claude", "plugins", "local-plugin"),
						},
						{
							Scope:       "local",
							Version:     "1.0.1",
							InstalledAt: "2024-01-03",
							InstallPath: filepath.Join(projectDir, ".claude-local", "plugins", "local-plugin"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at local scope
			localSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"local-plugin@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, localSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis.Installed).To(HaveLen(1))

			info := analysis.Installed["local-plugin@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(Equal([]string{"local"}))
			Expect(info.InstalledAt).To(HaveLen(3))
			// Local scope has highest precedence
			Expect(info.ActiveSource).To(Equal("local"))
		})
	})

	Context("with plugin installed but not enabled", func() {
		It("shows plugin with no enabled scopes", func() {
			// Install plugin
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"disabled-plugin@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "disabled-plugin"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Don't enable it in settings
			settings := &claude.Settings{
				EnabledPlugins: make(map[string]bool),
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis.Installed).To(HaveLen(1))

			info := analysis.Installed["disabled-plugin@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(BeEmpty())
			Expect(info.InstalledAt).To(HaveLen(1))
			Expect(info.ActiveSource).To(Equal(""))
		})
	})

	Context("when Claude directory doesn't exist", func() {
		It("returns an error", func() {
			nonExistentDir := filepath.Join(tempDir, "nonexistent")
			_, err := claude.AnalyzePluginScopesWithOrphans(nonExistentDir, projectDir)
			Expect(err).To(HaveOccurred())
		})
	})

	// Fallback path tests: these exercise the logic when installation and enablement
	// are at different scopes (not an exact match)
	Context("fallback: installed at higher precedence than enabled", func() {
		It("uses project installation when only enabled at user", func() {
			// Install at project scope only
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"cross-scope@marketplace": {
						{
							Scope:       "project",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(projectDir, ".claude", "plugins", "cross-scope"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at user scope only (lower precedence than installation)
			userSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"cross-scope@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			info := analysis.Installed["cross-scope@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(Equal([]string{"user"}))
			Expect(info.InstalledAt).To(HaveLen(1))
			// Project installation should be used (higher precedence)
			Expect(info.ActiveSource).To(Equal("project"))
		})

		It("uses local installation when only enabled at user", func() {
			// Install at local scope only
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"local-cross@marketplace": {
						{
							Scope:       "local",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(projectDir, ".claude-local", "plugins", "local-cross"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at user scope only
			userSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"local-cross@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			info := analysis.Installed["local-cross@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(Equal([]string{"user"}))
			// Local installation should be used (highest precedence)
			Expect(info.ActiveSource).To(Equal("local"))
		})
	})

	Context("fallback: installed at lower precedence than enabled", func() {
		It("uses user installation when only enabled at local", func() {
			// Install at user scope only
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"user-only@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "user-only"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at local scope only (higher precedence than installation)
			localSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"user-only@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, localSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			info := analysis.Installed["user-only@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(Equal([]string{"local"}))
			// User installation should be used (only available option)
			Expect(info.ActiveSource).To(Equal("user"))
		})

		It("uses user installation when only enabled at project", func() {
			// Install at user scope only
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"user-proj@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "user-proj"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at project scope only
			projectSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"user-proj@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			info := analysis.Installed["user-proj@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(Equal([]string{"project"}))
			// User installation should be used (only available option)
			Expect(info.ActiveSource).To(Equal("user"))
		})
	})

	Context("fallback: multiple installations with cross-scope enablement", func() {
		It("uses highest precedence installation regardless of where enabled", func() {
			// Install at user and local scopes
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"multi-install@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "multi-install"),
						},
						{
							Scope:       "local",
							Version:     "1.0.1",
							InstalledAt: "2024-01-02",
							InstallPath: filepath.Join(projectDir, ".claude-local", "plugins", "multi-install"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at project scope only (between user and local in precedence)
			projectSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"multi-install@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

			analysis, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			info := analysis.Installed["multi-install@marketplace"]
			Expect(info).NotTo(BeNil())
			Expect(info.EnabledAt).To(Equal([]string{"project"}))
			Expect(info.InstalledAt).To(HaveLen(2))
			// Local installation should be used (highest precedence available)
			Expect(info.ActiveSource).To(Equal("local"))
		})
	})

	Context("with plugins enabled but not installed", func() {
		It("returns enabled-but-not-installed plugins separately", func() {
			// Install one plugin
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"installed-plugin@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: filepath.Join(claudeDir, "plugins", "cache", "installed-plugin"),
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable both installed and non-installed plugins
			settings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"installed-plugin@marketplace":      true,
					"orphan-plugin-1@other-marketplace": true,
					"orphan-plugin-2@other-marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			result, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			// Installed plugins should be in the main analysis
			Expect(result.Installed).To(HaveLen(1))
			Expect(result.Installed["installed-plugin@marketplace"]).NotTo(BeNil())

			// Non-installed but enabled plugins should be in EnabledNotInstalled
			Expect(result.EnabledNotInstalled).To(HaveLen(2))
			Expect(result.EnabledNotInstalled).To(ContainElements(
				"orphan-plugin-1@other-marketplace",
				"orphan-plugin-2@other-marketplace",
			))
		})

		It("deduplicates plugins enabled at multiple scopes", func() {
			// No plugins installed
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: make(map[string][]claude.PluginMetadata),
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable same plugin at user scope
			userSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"orphan@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

			// Enable same plugin at project scope
			projectSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"orphan@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

			result, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			// Should only appear once despite being enabled at multiple scopes
			Expect(result.EnabledNotInstalled).To(HaveLen(1))
			Expect(result.EnabledNotInstalled).To(ContainElement("orphan@marketplace"))
		})

		It("excludes explicitly disabled plugins from orphan list", func() {
			// No plugins installed
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: make(map[string][]claude.PluginMetadata),
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable one, disable another (false value in enabledPlugins)
			settings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"enabled-orphan@marketplace":  true,
					"disabled-orphan@marketplace": false,
				},
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			result, err := claude.AnalyzePluginScopesWithOrphans(claudeDir, projectDir)
			Expect(err).NotTo(HaveOccurred())

			// Only the enabled one should appear
			Expect(result.EnabledNotInstalled).To(HaveLen(1))
			Expect(result.EnabledNotInstalled).To(ContainElement("enabled-orphan@marketplace"))
		})
	})
})
