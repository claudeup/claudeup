// ABOUTME: Tests for profile comparison and diff functionality
// ABOUTME: Validates detection of changes between saved and current profiles
package profile

import (
	"testing"

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
			Expect(diff.Summary()).To(Equal("1 plugin added"))
		})

		It("formats multiple plugins added", func() {
			diff := &ProfileDiff{
				PluginsAdded: []string{"plugin1", "plugin2"},
			}
			Expect(diff.Summary()).To(Equal("2 plugins added"))
		})

		It("formats single plugin removed", func() {
			diff := &ProfileDiff{
				PluginsRemoved: []string{"plugin1"},
			}
			Expect(diff.Summary()).To(Equal("1 plugin removed"))
		})

		It("formats multiple plugins removed", func() {
			diff := &ProfileDiff{
				PluginsRemoved: []string{"plugin1", "plugin2"},
			}
			Expect(diff.Summary()).To(Equal("2 plugins removed"))
		})

		It("formats single marketplace added", func() {
			diff := &ProfileDiff{
				MarketplacesAdded: []Marketplace{{Source: "github", Repo: "org/repo"}},
			}
			Expect(diff.Summary()).To(Equal("1 marketplace added"))
		})

		It("formats multiple marketplaces added", func() {
			diff := &ProfileDiff{
				MarketplacesAdded: []Marketplace{
					{Source: "github", Repo: "org/repo1"},
					{Source: "github", Repo: "org/repo2"},
				},
			}
			Expect(diff.Summary()).To(Equal("2 marketplaces added"))
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
			Expect(diff.Summary()).To(Equal("2 plugins added, 1 marketplace removed, 1 MCP server modified"))
		})

		It("formats all change types", func() {
			diff := &ProfileDiff{
				PluginsAdded:         []string{"plugin1"},
				PluginsRemoved:       []string{"plugin2"},
				MarketplacesAdded:    []Marketplace{{Source: "github", Repo: "org/repo1"}},
				MarketplacesRemoved:  []Marketplace{{Source: "github", Repo: "org/repo2"}},
				MCPServersAdded:      []MCPServer{{Name: "server1", Command: "cmd1"}},
				MCPServersRemoved:    []MCPServer{{Name: "server2", Command: "cmd2"}},
				MCPServersModified:   []MCPServer{{Name: "server3", Command: "cmd3"}},
			}
			summary := diff.Summary()
			Expect(summary).To(ContainSubstring("1 plugin added"))
			Expect(summary).To(ContainSubstring("1 plugin removed"))
			Expect(summary).To(ContainSubstring("1 marketplace added"))
			Expect(summary).To(ContainSubstring("1 marketplace removed"))
			Expect(summary).To(ContainSubstring("1 MCP server added"))
			Expect(summary).To(ContainSubstring("1 MCP server removed"))
			Expect(summary).To(ContainSubstring("1 MCP server modified"))
		})
	})
})
