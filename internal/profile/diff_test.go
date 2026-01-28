// ABOUTME: Tests for profile comparison and diff functionality
// ABOUTME: Validates detection of changes between saved and current profiles
package profile

import (
	"os"
	"testing"

	"github.com/claudeup/claudeup/v2/internal/claude"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDiff(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Profile Diff Suite")
}

var _ = Describe("ProfileDiff", func() {
	Describe("HasChanges", func() {
		It("returns false for empty diff", func() {
			diff := &ProfileDiff{}
			Expect(diff.HasChanges()).To(BeFalse())
		})

		It("returns true when plugins added", func() {
			diff := &ProfileDiff{
				PluginsAdded: []string{"plugin1"},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})

		It("returns true when plugins removed", func() {
			diff := &ProfileDiff{
				PluginsRemoved: []string{"plugin1"},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})

		It("returns true when marketplaces added", func() {
			diff := &ProfileDiff{
				MarketplacesAdded: []Marketplace{{Source: "github", Repo: "test/repo"}},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})

		It("returns true when marketplaces removed", func() {
			diff := &ProfileDiff{
				MarketplacesRemoved: []Marketplace{{Source: "github", Repo: "test/repo"}},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})

		It("returns true when MCP servers added", func() {
			diff := &ProfileDiff{
				MCPServersAdded: []MCPServer{{Name: "server1", Command: "cmd"}},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})

		It("returns true when MCP servers removed", func() {
			diff := &ProfileDiff{
				MCPServersRemoved: []MCPServer{{Name: "server1", Command: "cmd"}},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})

		It("returns true when MCP servers modified", func() {
			diff := &ProfileDiff{
				MCPServersModified: []MCPServer{{Name: "server1", Command: "new-cmd"}},
			}
			Expect(diff.HasChanges()).To(BeTrue())
		})
	})

	Describe("HasSignificantChanges", func() {
		It("returns false for empty diff", func() {
			diff := &ProfileDiff{}
			Expect(diff.HasSignificantChanges()).To(BeFalse())
		})

		It("returns true when plugins added", func() {
			diff := &ProfileDiff{
				PluginsAdded: []string{"plugin1"},
			}
			Expect(diff.HasSignificantChanges()).To(BeTrue())
		})

		It("returns true when plugins removed", func() {
			diff := &ProfileDiff{
				PluginsRemoved: []string{"plugin1"},
			}
			Expect(diff.HasSignificantChanges()).To(BeTrue())
		})

		It("returns false when only marketplaces added", func() {
			diff := &ProfileDiff{
				MarketplacesAdded: []Marketplace{{Source: "github", Repo: "test/repo"}},
			}
			Expect(diff.HasSignificantChanges()).To(BeFalse())
		})

		It("returns false when only marketplaces removed", func() {
			diff := &ProfileDiff{
				MarketplacesRemoved: []Marketplace{{Source: "github", Repo: "test/repo"}},
			}
			Expect(diff.HasSignificantChanges()).To(BeFalse())
		})

		It("returns true when MCP servers added", func() {
			diff := &ProfileDiff{
				MCPServersAdded: []MCPServer{{Name: "server1", Command: "cmd"}},
			}
			Expect(diff.HasSignificantChanges()).To(BeTrue())
		})

		It("returns true when MCP servers removed", func() {
			diff := &ProfileDiff{
				MCPServersRemoved: []MCPServer{{Name: "server1", Command: "cmd"}},
			}
			Expect(diff.HasSignificantChanges()).To(BeTrue())
		})

		It("returns true when MCP servers modified", func() {
			diff := &ProfileDiff{
				MCPServersModified: []MCPServer{{Name: "server1", Command: "new-cmd"}},
			}
			Expect(diff.HasSignificantChanges()).To(BeTrue())
		})

		It("returns true when plugins changed along with marketplaces", func() {
			diff := &ProfileDiff{
				PluginsAdded:      []string{"plugin1"},
				MarketplacesAdded: []Marketplace{{Source: "github", Repo: "test/repo"}},
			}
			Expect(diff.HasSignificantChanges()).To(BeTrue())
		})
	})

	Describe("compare", func() {
		It("detects no changes when profiles are identical", func() {
			saved := &Profile{
				Plugins:      []string{"plugin1", "plugin2"},
				Marketplaces: []Marketplace{{Source: "github", Repo: "test/repo"}},
				MCPServers:   []MCPServer{{Name: "server1", Command: "cmd", Args: []string{"arg1"}}},
			}
			current := &Profile{
				Plugins:      []string{"plugin1", "plugin2"},
				Marketplaces: []Marketplace{{Source: "github", Repo: "test/repo"}},
				MCPServers:   []MCPServer{{Name: "server1", Command: "cmd", Args: []string{"arg1"}}},
			}

			diff := compare(saved, current)
			Expect(diff.HasChanges()).To(BeFalse())
		})

		It("detects plugins added", func() {
			saved := &Profile{
				Plugins: []string{"plugin1"},
			}
			current := &Profile{
				Plugins: []string{"plugin1", "plugin2", "plugin3"},
			}

			diff := compare(saved, current)
			Expect(diff.PluginsAdded).To(ConsistOf("plugin2", "plugin3"))
			Expect(diff.PluginsRemoved).To(BeEmpty())
		})

		It("detects plugins removed", func() {
			saved := &Profile{
				Plugins: []string{"plugin1", "plugin2", "plugin3"},
			}
			current := &Profile{
				Plugins: []string{"plugin1"},
			}

			diff := compare(saved, current)
			Expect(diff.PluginsRemoved).To(ConsistOf("plugin2", "plugin3"))
			Expect(diff.PluginsAdded).To(BeEmpty())
		})

		It("skips plugin comparison when skipPluginDiff is true", func() {
			saved := &Profile{
				Plugins:        []string{"plugin1"},
				SkipPluginDiff: true,
			}
			current := &Profile{
				Plugins: []string{"plugin1", "plugin2", "plugin3"},
			}

			diff := compare(saved, current)
			Expect(diff.PluginsAdded).To(BeEmpty())
			Expect(diff.PluginsRemoved).To(BeEmpty())
		})

		It("detects marketplaces added", func() {
			saved := &Profile{
				Marketplaces: []Marketplace{{Source: "github", Repo: "org/repo1"}},
			}
			current := &Profile{
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "org/repo1"},
					{Source: "github", Repo: "org/repo2"},
				},
			}

			diff := compare(saved, current)
			Expect(diff.MarketplacesAdded).To(HaveLen(1))
			Expect(diff.MarketplacesAdded[0].Repo).To(Equal("org/repo2"))
		})

		It("detects marketplaces removed", func() {
			saved := &Profile{
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "org/repo1"},
					{Source: "github", Repo: "org/repo2"},
				},
			}
			current := &Profile{
				Marketplaces: []Marketplace{{Source: "github", Repo: "org/repo1"}},
			}

			diff := compare(saved, current)
			Expect(diff.MarketplacesRemoved).To(HaveLen(1))
			Expect(diff.MarketplacesRemoved[0].Repo).To(Equal("org/repo2"))
		})

		It("detects MCP servers added", func() {
			saved := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "cmd1"}},
			}
			current := &Profile{
				MCPServers: []MCPServer{
					{Name: "server1", Command: "cmd1"},
					{Name: "server2", Command: "cmd2"},
				},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersAdded).To(HaveLen(1))
			Expect(diff.MCPServersAdded[0].Name).To(Equal("server2"))
		})

		It("detects MCP servers removed", func() {
			saved := &Profile{
				MCPServers: []MCPServer{
					{Name: "server1", Command: "cmd1"},
					{Name: "server2", Command: "cmd2"},
				},
			}
			current := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "cmd1"}},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersRemoved).To(HaveLen(1))
			Expect(diff.MCPServersRemoved[0].Name).To(Equal("server2"))
		})

		It("detects MCP servers modified when command changes", func() {
			saved := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "old-cmd", Args: []string{"arg1"}}},
			}
			current := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "new-cmd", Args: []string{"arg1"}}},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersModified).To(HaveLen(1))
			Expect(diff.MCPServersModified[0].Name).To(Equal("server1"))
			Expect(diff.MCPServersModified[0].Command).To(Equal("new-cmd"))
		})

		It("detects MCP servers modified when args change", func() {
			saved := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "cmd", Args: []string{"arg1"}}},
			}
			current := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "cmd", Args: []string{"arg2"}}},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersModified).To(HaveLen(1))
			Expect(diff.MCPServersModified[0].Name).To(Equal("server1"))
		})

		It("detects MCP servers modified when scope changes", func() {
			saved := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "cmd", Scope: "user"}},
			}
			current := &Profile{
				MCPServers: []MCPServer{{Name: "server1", Command: "cmd", Scope: "workspace"}},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersModified).To(HaveLen(1))
			Expect(diff.MCPServersModified[0].Name).To(Equal("server1"))
		})

		It("detects MCP servers modified when secrets change", func() {
			saved := &Profile{
				MCPServers: []MCPServer{{
					Name:    "server1",
					Command: "cmd",
					Secrets: map[string]SecretRef{
						"API_KEY": {
							Description: "API Key",
							Sources:     []SecretSource{{Type: "env", Key: "API_KEY"}},
						},
					},
				}},
			}
			current := &Profile{
				MCPServers: []MCPServer{{
					Name:    "server1",
					Command: "cmd",
					Secrets: map[string]SecretRef{
						"API_KEY": {
							Description: "API Key Updated",
							Sources:     []SecretSource{{Type: "env", Key: "API_KEY"}},
						},
					},
				}},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersModified).To(HaveLen(1))
			Expect(diff.MCPServersModified[0].Name).To(Equal("server1"))
		})

		It("does not detect modification when MCP servers are identical including scope and secrets", func() {
			saved := &Profile{
				MCPServers: []MCPServer{{
					Name:    "server1",
					Command: "cmd",
					Args:    []string{"arg1"},
					Scope:   "user",
					Secrets: map[string]SecretRef{
						"API_KEY": {
							Description: "API Key",
							Sources:     []SecretSource{{Type: "env", Key: "API_KEY"}},
						},
					},
				}},
			}
			current := &Profile{
				MCPServers: []MCPServer{{
					Name:    "server1",
					Command: "cmd",
					Args:    []string{"arg1"},
					Scope:   "user",
					Secrets: map[string]SecretRef{
						"API_KEY": {
							Description: "API Key",
							Sources:     []SecretSource{{Type: "env", Key: "API_KEY"}},
						},
					},
				}},
			}

			diff := compare(saved, current)
			Expect(diff.MCPServersModified).To(BeEmpty())
		})
	})

	Describe("Summary", func() {
		It("returns empty string for empty diff", func() {
			diff := &ProfileDiff{}
			Expect(diff.Summary()).To(BeEmpty())
		})

		It("formats single plugin added", func() {
			diff := &ProfileDiff{
				PluginsAdded: []string{"plugin1"},
			}
			Expect(diff.Summary()).To(Equal("1 plugin not in profile"))
		})

		It("formats multiple plugins added", func() {
			diff := &ProfileDiff{
				PluginsAdded: []string{"plugin1", "plugin2"},
			}
			Expect(diff.Summary()).To(Equal("2 plugins not in profile"))
		})

		It("formats single plugin removed", func() {
			diff := &ProfileDiff{
				PluginsRemoved: []string{"plugin1"},
			}
			Expect(diff.Summary()).To(Equal("1 plugin missing"))
		})

		It("formats multiple plugins removed", func() {
			diff := &ProfileDiff{
				PluginsRemoved: []string{"plugin1", "plugin2"},
			}
			Expect(diff.Summary()).To(Equal("2 plugins missing"))
		})

		It("formats single marketplace added", func() {
			diff := &ProfileDiff{
				MarketplacesAdded: []Marketplace{{Source: "github", Repo: "org/repo"}},
			}
			Expect(diff.Summary()).To(Equal("1 marketplace not in profile"))
		})

		It("formats multiple marketplaces added", func() {
			diff := &ProfileDiff{
				MarketplacesAdded: []Marketplace{
					{Source: "github", Repo: "org/repo1"},
					{Source: "github", Repo: "org/repo2"},
				},
			}
			Expect(diff.Summary()).To(Equal("2 marketplaces not in profile"))
		})

		It("formats single MCP server modified", func() {
			diff := &ProfileDiff{
				MCPServersModified: []MCPServer{{Name: "server1", Command: "cmd"}},
			}
			Expect(diff.Summary()).To(Equal("1 MCP server modified"))
		})

		It("formats multiple MCP servers modified", func() {
			diff := &ProfileDiff{
				MCPServersModified: []MCPServer{
					{Name: "server1", Command: "cmd1"},
					{Name: "server2", Command: "cmd2"},
				},
			}
			Expect(diff.Summary()).To(Equal("2 MCP servers modified"))
		})

		It("formats combined changes", func() {
			diff := &ProfileDiff{
				PluginsAdded:        []string{"plugin1", "plugin2"},
				MarketplacesRemoved: []Marketplace{{Source: "github", Repo: "org/repo"}},
				MCPServersModified:  []MCPServer{{Name: "server1", Command: "cmd"}},
			}
			Expect(diff.Summary()).To(Equal("2 plugins not in profile, 1 marketplace missing, 1 MCP server modified"))
		})

		It("formats all change types", func() {
			diff := &ProfileDiff{
				PluginsAdded:        []string{"plugin1"},
				PluginsRemoved:      []string{"plugin2"},
				MarketplacesAdded:   []Marketplace{{Source: "github", Repo: "org/repo1"}},
				MarketplacesRemoved: []Marketplace{{Source: "github", Repo: "org/repo2"}},
				MCPServersAdded:     []MCPServer{{Name: "server1", Command: "cmd1"}},
				MCPServersRemoved:   []MCPServer{{Name: "server2", Command: "cmd2"}},
				MCPServersModified:  []MCPServer{{Name: "server3", Command: "cmd3"}},
			}
			summary := diff.Summary()
			Expect(summary).To(ContainSubstring("1 plugin not in profile"))
			Expect(summary).To(ContainSubstring("1 plugin missing"))
			Expect(summary).To(ContainSubstring("1 marketplace not in profile"))
			Expect(summary).To(ContainSubstring("1 marketplace missing"))
			Expect(summary).To(ContainSubstring("1 MCP server not in profile"))
			Expect(summary).To(ContainSubstring("1 MCP server missing"))
			Expect(summary).To(ContainSubstring("1 MCP server modified"))
		})
	})

	Describe("CompareWithScope", func() {
		var (
			tempDir    string
			claudeDir  string
			projectDir string
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			claudeDir = tempDir + "/.claude"
			projectDir = tempDir + "/project"

			// Create directory structure
			Expect(os.MkdirAll(claudeDir+"/plugins", 0755)).To(Succeed())
			Expect(os.MkdirAll(projectDir+"/.claude", 0755)).To(Succeed())
		})

		It("compares profile with user scope", func() {
			// Create plugin registry
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"test-plugin@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: claudeDir + "/plugins/cache/test-plugin",
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at user scope
			settings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"test-plugin@marketplace":  true,
					"extra-plugin@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			// Saved profile with only one plugin
			saved := &Profile{
				Name:    "test",
				Plugins: []string{"test-plugin@marketplace"},
			}

			diff, err := CompareWithScope(saved, claudeDir, "", projectDir, "user")
			Expect(err).NotTo(HaveOccurred())
			Expect(diff.PluginsAdded).To(ConsistOf("extra-plugin@marketplace"))
			Expect(diff.PluginsRemoved).To(BeEmpty())
		})

		It("compares profile with project scope", func() {
			// Create plugin registry
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"project-plugin@marketplace": {
						{
							Scope:       "project",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: projectDir + "/.claude/plugins/project-plugin",
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at project scope
			projectSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"project-plugin@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

			// Multi-scope profile with different plugin at project scope
			saved := &Profile{
				Name: "test",
				PerScope: &PerScopeSettings{
					Project: &ScopeSettings{
						Plugins: []string{"other-plugin@marketplace"},
					},
				},
			}

			diff, err := CompareWithScope(saved, claudeDir, "", projectDir, "project")
			Expect(err).NotTo(HaveOccurred())
			Expect(diff.PluginsAdded).To(ConsistOf("project-plugin@marketplace"))
			Expect(diff.PluginsRemoved).To(ConsistOf("other-plugin@marketplace"))
		})

		It("compares profile with local scope", func() {
			// Create plugin registry
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"local-plugin@marketplace": {
						{
							Scope:       "local",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: projectDir + "/.claude-local/plugins/local-plugin",
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

			// Saved profile empty
			saved := &Profile{
				Name:    "test",
				Plugins: []string{},
			}

			diff, err := CompareWithScope(saved, claudeDir, "", projectDir, "local")
			Expect(err).NotTo(HaveOccurred())
			Expect(diff.PluginsAdded).To(ConsistOf("local-plugin@marketplace"))
			Expect(diff.PluginsRemoved).To(BeEmpty())
		})

		It("returns error for invalid scope", func() {
			saved := &Profile{
				Name: "test",
			}

			_, err := CompareWithScope(saved, claudeDir, "", projectDir, "invalid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid scope"))
		})
	})

	Describe("IsProfileModifiedAtScope", func() {
		var (
			tempDir     string
			claudeDir   string
			projectDir  string
			profilesDir string
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			claudeDir = tempDir + "/.claude"
			projectDir = tempDir + "/project"
			profilesDir = tempDir + "/.claudeup/profiles"

			// Create directory structure
			Expect(os.MkdirAll(claudeDir+"/plugins", 0755)).To(Succeed())
			Expect(os.MkdirAll(projectDir+"/.claude", 0755)).To(Succeed())
			Expect(os.MkdirAll(profilesDir, 0755)).To(Succeed())
		})

		It("returns true when profile is modified at user scope", func() {
			// Create plugin registry
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"test-plugin@marketplace": {
						{
							Scope:       "user",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: claudeDir + "/plugins/cache/test-plugin",
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at user scope
			settings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"test-plugin@marketplace":  true,
					"extra-plugin@marketplace": true,
				},
			}
			Expect(claude.SaveSettings(claudeDir, settings)).To(Succeed())

			// Save profile with only one plugin
			profile := &Profile{
				Name:    "test",
				Plugins: []string{"test-plugin@marketplace"},
			}
			Expect(Save(profilesDir, profile)).To(Succeed())

			// Check for modifications
			modified, err := IsProfileModifiedAtScope("test", profilesDir, claudeDir, "", projectDir, "user")
			Expect(err).NotTo(HaveOccurred())
			Expect(modified).To(BeTrue())
		})

		It("returns false when profile is not modified at project scope", func() {
			// Create plugin registry
			registry := &claude.PluginRegistry{
				Version: 2,
				Plugins: map[string][]claude.PluginMetadata{
					"project-plugin@marketplace": {
						{
							Scope:       "project",
							Version:     "1.0.0",
							InstalledAt: "2024-01-01",
							InstallPath: projectDir + "/.claude/plugins/project-plugin",
						},
					},
				},
			}
			Expect(claude.SavePlugins(claudeDir, registry)).To(Succeed())

			// Enable at project scope
			projectSettings := &claude.Settings{
				EnabledPlugins: map[string]bool{
					"project-plugin@marketplace": true,
				},
			}
			Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

			// Save multi-scope profile matching current state at project scope
			profile := &Profile{
				Name: "test",
				PerScope: &PerScopeSettings{
					Project: &ScopeSettings{
						Plugins: []string{"project-plugin@marketplace"},
					},
				},
			}
			Expect(Save(profilesDir, profile)).To(Succeed())

			// Check for modifications
			modified, err := IsProfileModifiedAtScope("test", profilesDir, claudeDir, "", projectDir, "project")
			Expect(err).NotTo(HaveOccurred())
			Expect(modified).To(BeFalse())
		})

		It("returns false for empty profile name", func() {
			modified, err := IsProfileModifiedAtScope("", profilesDir, claudeDir, "", projectDir, "user")
			Expect(err).NotTo(HaveOccurred())
			Expect(modified).To(BeFalse())
		})

		It("returns error for nonexistent profile", func() {
			modified, err := IsProfileModifiedAtScope("nonexistent", profilesDir, claudeDir, "", projectDir, "user")
			Expect(err).To(HaveOccurred())
			Expect(modified).To(BeFalse())
		})

		It("returns error for invalid scope", func() {
			// Save a valid profile
			profile := &Profile{
				Name:    "test",
				Plugins: []string{},
			}
			Expect(Save(profilesDir, profile)).To(Succeed())

			modified, err := IsProfileModifiedAtScope("test", profilesDir, claudeDir, "", projectDir, "invalid")
			Expect(err).To(HaveOccurred())
			Expect(modified).To(BeFalse())
		})
	})

	Describe("SnapshotCombined and IsProfileModifiedCombined", func() {
		var (
			tempDir     string
			claudeDir   string
			projectDir  string
			profilesDir string
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			claudeDir = tempDir + "/.claude"
			projectDir = tempDir + "/project"
			profilesDir = tempDir + "/.claudeup/profiles"

			// Create directory structure
			Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())
			Expect(os.MkdirAll(projectDir+"/.claude", 0755)).To(Succeed())
			Expect(os.MkdirAll(profilesDir, 0755)).To(Succeed())
		})

		Describe("SnapshotCombined", func() {
			It("combines plugins from user and project scopes", func() {
				// User scope: plugin-a enabled
				userSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-a@marketplace": true,
						"plugin-b@marketplace": false, // Disabled at user scope
					},
				}
				Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

				// Project scope: plugin-b enabled (overrides user), plugin-c added
				projectSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-b@marketplace": true, // Override: now enabled
						"plugin-c@marketplace": true, // New plugin
					},
				}
				Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

				// Snapshot combined should include: plugin-a (user), plugin-b (project override), plugin-c (project)
				snapshot, err := SnapshotCombined("test", claudeDir, "", projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(snapshot.Plugins).To(ConsistOf("plugin-a@marketplace", "plugin-b@marketplace", "plugin-c@marketplace"))
			})

			It("local scope overrides project and user", func() {
				// User scope
				userSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-a@marketplace": true,
					},
				}
				Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

				// Project scope
				projectSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-b@marketplace": true,
					},
				}
				Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

				// Local scope: disable plugin-a, enable plugin-b, add plugin-c
				localSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-a@marketplace": false, // Override user
						"plugin-c@marketplace": true,  // New
					},
				}
				Expect(claude.SaveSettingsForScope("local", claudeDir, projectDir, localSettings)).To(Succeed())

				// Combined should be: plugin-b (project), plugin-c (local)
				// plugin-a is disabled by local scope
				snapshot, err := SnapshotCombined("test", claudeDir, "", projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(snapshot.Plugins).To(ConsistOf("plugin-b@marketplace", "plugin-c@marketplace"))
				Expect(snapshot.Plugins).NotTo(ContainElement("plugin-a@marketplace"))
			})
		})

		Describe("IsProfileModifiedCombined", func() {
			It("detects modifications across all scopes", func() {
				// Save a profile with plugin-a and plugin-b
				profile := &Profile{
					Name:    "test",
					Plugins: []string{"plugin-a@marketplace", "plugin-b@marketplace"},
				}
				Expect(Save(profilesDir, profile)).To(Succeed())

				// User scope: only plugin-a
				userSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-a@marketplace": true,
					},
				}
				Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

				// Project scope: adds plugin-c (not in profile)
				projectSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-c@marketplace": true,
					},
				}
				Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

				// Combined state has plugin-a + plugin-c, but profile expects plugin-a + plugin-b
				// Should detect modification
				modified, err := IsProfileModifiedCombined("test", profilesDir, claudeDir, "", projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(modified).To(BeTrue())
			})

			It("returns false when combined state matches profile", func() {
				// Save a profile
				profile := &Profile{
					Name:    "test",
					Plugins: []string{"plugin-a@marketplace", "plugin-b@marketplace"},
				}
				Expect(Save(profilesDir, profile)).To(Succeed())

				// User scope: plugin-a
				userSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-a@marketplace": true,
					},
				}
				Expect(claude.SaveSettings(claudeDir, userSettings)).To(Succeed())

				// Project scope: plugin-b
				projectSettings := &claude.Settings{
					EnabledPlugins: map[string]bool{
						"plugin-b@marketplace": true,
					},
				}
				Expect(claude.SaveSettingsForScope("project", claudeDir, projectDir, projectSettings)).To(Succeed())

				// Combined = plugin-a + plugin-b, matches profile
				modified, err := IsProfileModifiedCombined("test", profilesDir, claudeDir, "", projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(modified).To(BeFalse())
			})
		})
	})
})
