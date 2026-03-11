// ABOUTME: Unit tests for upgrade command argument parsing
// ABOUTME: Tests target detection (marketplace vs plugin) and scope resolution
package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup/v5/internal/claude"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade Commands Suite")
}

var _ = Describe("parseUpgradeTargets", func() {
	It("detects plugins by @ symbol", func() {
		marketplaces, plugins := parseUpgradeTargets([]string{"hookify@claude-code-plugins"})
		Expect(marketplaces).To(BeEmpty())
		Expect(plugins).To(ConsistOf("hookify@claude-code-plugins"))
	})

	It("detects marketplaces by absence of @", func() {
		marketplaces, plugins := parseUpgradeTargets([]string{"superpowers-marketplace"})
		Expect(marketplaces).To(ConsistOf("superpowers-marketplace"))
		Expect(plugins).To(BeEmpty())
	})

	It("handles mixed targets", func() {
		marketplaces, plugins := parseUpgradeTargets([]string{
			"superpowers-marketplace",
			"hookify@plugins",
			"other-marketplace",
		})
		Expect(marketplaces).To(ConsistOf("superpowers-marketplace", "other-marketplace"))
		Expect(plugins).To(ConsistOf("hookify@plugins"))
	})

	It("returns empty slices for no args", func() {
		marketplaces, plugins := parseUpgradeTargets([]string{})
		Expect(marketplaces).To(BeEmpty())
		Expect(plugins).To(BeEmpty())
	})
})

var _ = Describe("findUnmatchedTargets", func() {
	var (
		marketplaceUpdates []MarketplaceUpdate
		pluginUpdates      []PluginUpdate
	)

	BeforeEach(func() {
		marketplaceUpdates = []MarketplaceUpdate{
			{Name: "superpowers-marketplace", HasUpdate: true},
			{Name: "claude-code-plugins", HasUpdate: false},
		}
		pluginUpdates = []PluginUpdate{
			{Name: "hookify@claude-code-plugins", HasUpdate: true},
			{Name: "tdd@superpowers-marketplace", HasUpdate: false},
		}
	})

	It("returns empty for matching marketplace targets", func() {
		unmatched := findUnmatchedTargets(
			[]string{"superpowers-marketplace"},
			nil,
			marketplaceUpdates,
			pluginUpdates,
		)
		Expect(unmatched).To(BeEmpty())
	})

	It("returns empty for matching plugin targets", func() {
		unmatched := findUnmatchedTargets(
			nil,
			[]string{"hookify@claude-code-plugins"},
			marketplaceUpdates,
			pluginUpdates,
		)
		Expect(unmatched).To(BeEmpty())
	})

	It("returns unmatched marketplace targets", func() {
		unmatched := findUnmatchedTargets(
			[]string{"nonexistent-marketplace", "superpowers-marktplace"},
			nil,
			marketplaceUpdates,
			pluginUpdates,
		)
		Expect(unmatched).To(ConsistOf("nonexistent-marketplace", "superpowers-marktplace"))
	})

	It("returns unmatched plugin targets", func() {
		unmatched := findUnmatchedTargets(
			nil,
			[]string{"unknown@marketplace", "typo@plugins"},
			marketplaceUpdates,
			pluginUpdates,
		)
		Expect(unmatched).To(ConsistOf("unknown@marketplace", "typo@plugins"))
	})

	It("handles mixed matched and unmatched targets", func() {
		unmatched := findUnmatchedTargets(
			[]string{"superpowers-marketplace", "nonexistent"},
			[]string{"hookify@claude-code-plugins", "missing@plugins"},
			marketplaceUpdates,
			pluginUpdates,
		)
		Expect(unmatched).To(ConsistOf("nonexistent", "missing@plugins"))
	})

	It("returns empty when no targets specified", func() {
		unmatched := findUnmatchedTargets(nil, nil, marketplaceUpdates, pluginUpdates)
		Expect(unmatched).To(BeEmpty())
	})

	It("matches targets with HasUpdate false (up-to-date items)", func() {
		unmatched := findUnmatchedTargets(
			[]string{"claude-code-plugins"},
			[]string{"tdd@superpowers-marketplace"},
			marketplaceUpdates,
			pluginUpdates,
		)
		Expect(unmatched).To(BeEmpty())
	})
})

var _ = Describe("findMarketplacePath", func() {
	var marketplaces claude.MarketplaceRegistry

	BeforeEach(func() {
		marketplaces = claude.MarketplaceRegistry{
			"superpowers": claude.MarketplaceMetadata{
				InstallLocation: "/home/.claude/plugins/marketplaces/superpowers",
			},
			"community": claude.MarketplaceMetadata{
				InstallLocation: "/home/.claude/plugins/marketplaces/community",
			},
		}
	})

	It("resolves marketplace from plugin name suffix", func() {
		path := findMarketplacePath("hookify@superpowers", "/home/.claude/plugins/cache/hookify", marketplaces)
		Expect(path).To(Equal("/home/.claude/plugins/marketplaces/superpowers"))
	})

	It("falls back to install path matching", func() {
		path := findMarketplacePath("hookify", "/home/.claude/plugins/marketplaces/community/plugins/hookify", marketplaces)
		Expect(path).To(Equal("/home/.claude/plugins/marketplaces/community"))
	})

	It("returns empty when no match found", func() {
		path := findMarketplacePath("hookify", "/some/other/path", marketplaces)
		Expect(path).To(BeEmpty())
	})

	It("does not false-match overlapping marketplace path prefixes", func() {
		overlapping := claude.MarketplaceRegistry{
			"community": claude.MarketplaceMetadata{
				InstallLocation: "/home/.claude/plugins/marketplaces/community",
			},
			"community-extra": claude.MarketplaceMetadata{
				InstallLocation: "/home/.claude/plugins/marketplaces/community-extra",
			},
		}
		path := findMarketplacePath("hookify", "/home/.claude/plugins/marketplaces/community-extra/plugins/hookify", overlapping)
		Expect(path).To(Equal("/home/.claude/plugins/marketplaces/community-extra"))
	})

	It("prefers name-based lookup over path matching", func() {
		// Plugin name says superpowers, but path contains community
		path := findMarketplacePath("hookify@superpowers", "/home/.claude/plugins/marketplaces/community/plugins/hookify", marketplaces)
		Expect(path).To(Equal("/home/.claude/plugins/marketplaces/superpowers"))
	})
})

var _ = Describe("updatePluginViaCLI", func() {
	var (
		tempDir     string
		origPath    string
		argsLogFile string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "cli-test-*")
		Expect(err).NotTo(HaveOccurred())

		argsLogFile = filepath.Join(tempDir, "args.log")

		// Create a fake "claude" binary that logs its arguments
		fakeClaude := filepath.Join(tempDir, "claude")
		script := "#!/bin/sh\necho \"$@\" > " + argsLogFile + "\n"
		err = os.WriteFile(fakeClaude, []byte(script), 0755)
		Expect(err).NotTo(HaveOccurred())

		origPath = os.Getenv("PATH")
		os.Setenv("PATH", tempDir+":"+origPath)
	})

	AfterEach(func() {
		os.Setenv("PATH", origPath)
		os.RemoveAll(tempDir)
	})

	It("passes --scope flag to claude plugin update", func() {
		err := updatePluginViaCLI("hookify@superpowers", "project")
		Expect(err).NotTo(HaveOccurred())

		logged, err := os.ReadFile(argsLogFile)
		Expect(err).NotTo(HaveOccurred())
		args := strings.TrimSpace(string(logged))
		Expect(args).To(Equal("plugin update --scope project hookify@superpowers"))
	})

	It("passes user scope by default convention", func() {
		err := updatePluginViaCLI("tdd@marketplace", "user")
		Expect(err).NotTo(HaveOccurred())

		logged, err := os.ReadFile(argsLogFile)
		Expect(err).NotTo(HaveOccurred())
		args := strings.TrimSpace(string(logged))
		Expect(args).To(Equal("plugin update --scope user tdd@marketplace"))
	})

	It("passes local scope correctly", func() {
		err := updatePluginViaCLI("debug@tools", "local")
		Expect(err).NotTo(HaveOccurred())

		logged, err := os.ReadFile(argsLogFile)
		Expect(err).NotTo(HaveOccurred())
		args := strings.TrimSpace(string(logged))
		Expect(args).To(Equal("plugin update --scope local debug@tools"))
	})

	It("includes command output in error on failure", func() {
		// Replace fake claude with one that fails
		fakeClaude := filepath.Join(tempDir, "claude")
		script := "#!/bin/sh\necho 'plugin not found: bogus' >&2\nexit 1\n"
		err := os.WriteFile(fakeClaude, []byte(script), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = updatePluginViaCLI("bogus@marketplace", "user")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("plugin not found: bogus"))
	})

	It("includes original error when command output is empty", func() {
		// Replace fake claude with one that fails silently
		fakeClaude := filepath.Join(tempDir, "claude")
		script := "#!/bin/sh\nexit 1\n"
		err := os.WriteFile(fakeClaude, []byte(script), 0755)
		Expect(err).NotTo(HaveOccurred())

		err = updatePluginViaCLI("bogus@marketplace", "user")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exit status"))
	})
})

var _ = Describe("resolvePluginSource", func() {
	var marketplaceDir string

	BeforeEach(func() {
		var err error
		marketplaceDir, err = os.MkdirTemp("", "marketplace-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(marketplaceDir)
	})

	It("finds plugin in plugins/ subdirectory", func() {
		pluginDir := filepath.Join(marketplaceDir, "plugins", "hookify")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		sourcePath, version, err := resolvePluginSource(marketplaceDir, "hookify")
		Expect(err).NotTo(HaveOccurred())
		Expect(sourcePath).To(Equal(pluginDir))
		Expect(version).To(BeEmpty())
	})

	It("finds plugin in skills/ subdirectory", func() {
		skillDir := filepath.Join(marketplaceDir, "skills", "tdd")
		Expect(os.MkdirAll(skillDir, 0755)).To(Succeed())

		sourcePath, version, err := resolvePluginSource(marketplaceDir, "tdd")
		Expect(err).NotTo(HaveOccurred())
		Expect(sourcePath).To(Equal(skillDir))
		Expect(version).To(BeEmpty())
	})

	Context("with marketplace index", func() {
		writeIndex := func(content string) {
			indexDir := filepath.Join(marketplaceDir, ".claude-plugin")
			Expect(os.MkdirAll(indexDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(indexDir, "marketplace.json"), []byte(content), 0644)).To(Succeed())
		}

		It("resolves relative path from index", func() {
			// Create the target directory
			targetDir := filepath.Join(marketplaceDir, "plugins", "hookify")
			Expect(os.MkdirAll(targetDir, 0755)).To(Succeed())

			writeIndex(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "hookify", "version": "1.2.0", "source": "./plugins/hookify"}
				]
			}`)

			// Source should come from directory scan, not index (directory takes priority)
			sourcePath, _, err := resolvePluginSource(marketplaceDir, "hookify")
			Expect(err).NotTo(HaveOccurred())
			Expect(sourcePath).To(Equal(targetDir))
		})

		It("resolves relative path from index when not in standard dirs", func() {
			// Plugin is at a custom path, not under plugins/ or skills/
			targetDir := filepath.Join(marketplaceDir, "custom", "hookify")
			Expect(os.MkdirAll(targetDir, 0755)).To(Succeed())

			writeIndex(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "hookify", "version": "2.0.0", "source": "./custom/hookify"}
				]
			}`)

			sourcePath, version, err := resolvePluginSource(marketplaceDir, "hookify")
			Expect(err).NotTo(HaveOccurred())
			Expect(sourcePath).To(Equal(targetDir))
			Expect(version).To(Equal("2.0.0"))
		})

		It("returns empty sourcePath for external URL source", func() {
			writeIndex(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "remote-plugin", "version": "3.0.0", "source": {"source": "git", "url": "https://github.com/org/repo"}}
				]
			}`)

			sourcePath, version, err := resolvePluginSource(marketplaceDir, "remote-plugin")
			Expect(err).NotTo(HaveOccurred())
			Expect(sourcePath).To(BeEmpty())
			Expect(version).To(Equal("3.0.0"))
		})

		It("returns error when plugin not in index", func() {
			writeIndex(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "other-plugin", "version": "1.0.0", "source": "./plugins/other"}
				]
			}`)

			_, _, err := resolvePluginSource(marketplaceDir, "hookify")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found in marketplace index"))
		})

		It("rejects path traversal in source field", func() {
			writeIndex(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "evil", "version": "1.0.0", "source": "../../etc/passwd"}
				]
			}`)

			_, _, err := resolvePluginSource(marketplaceDir, "evil")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resolves outside marketplace directory"))
		})

		It("returns error when relative path does not exist", func() {
			writeIndex(`{
				"name": "test-marketplace",
				"plugins": [
					{"name": "hookify", "version": "1.0.0", "source": "./nonexistent/hookify"}
				]
			}`)

			_, _, err := resolvePluginSource(marketplaceDir, "hookify")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	It("returns error when no index and not in standard dirs", func() {
		_, _, err := resolvePluginSource(marketplaceDir, "hookify")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cannot read index"))
	})
})

var _ = Describe("availableScopes", func() {
	It("returns all scopes when allFlag is true", func() {
		scopes := availableScopes(true, "")
		Expect(scopes).To(Equal(claude.ValidScopes))
	})

	It("always includes user scope when allFlag is false", func() {
		scopes := availableScopes(false, "")
		Expect(scopes).To(ContainElement("user"))
	})

	Context("in a project directory", func() {
		var tempDir string
		var claudeHome string
		var origClaudeDir string

		BeforeEach(func() {
			var err error
			origClaudeDir = claudeDir

			// Create a temp dir with a .claude subdirectory to simulate a project
			tempDir, err = os.MkdirTemp("", "scope-test-*")
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(filepath.Join(tempDir, ".claude"), 0755)
			Expect(err).NotTo(HaveOccurred())

			// Point claudeDir elsewhere so IsProjectContext detects a distinct project
			claudeHome, err = os.MkdirTemp("", "claude-home-*")
			Expect(err).NotTo(HaveOccurred())
			claudeDir = claudeHome
		})

		AfterEach(func() {
			claudeDir = origClaudeDir
			os.RemoveAll(tempDir)
			os.RemoveAll(claudeHome)
		})

		It("includes project and local scopes when allFlag is false", func() {
			scopes := availableScopes(false, tempDir)
			Expect(scopes).To(ContainElement("user"))
			Expect(scopes).To(ContainElement("project"))
			Expect(scopes).To(ContainElement("local"))
		})

		It("returns all valid scopes when allFlag is true regardless of context", func() {
			scopes := availableScopes(true, tempDir)
			Expect(scopes).To(Equal(claude.ValidScopes))
		})
	})

	Context("outside a project directory", func() {
		var tempDir string
		var claudeHome string
		var origClaudeDir string

		BeforeEach(func() {
			var err error
			origClaudeDir = claudeDir

			// Create a temp dir WITHOUT .claude to simulate non-project context
			tempDir, err = os.MkdirTemp("", "scope-test-*")
			Expect(err).NotTo(HaveOccurred())
			claudeHome, err = os.MkdirTemp("", "claude-home-*")
			Expect(err).NotTo(HaveOccurred())
			claudeDir = claudeHome
		})

		AfterEach(func() {
			claudeDir = origClaudeDir
			os.RemoveAll(tempDir)
			os.RemoveAll(claudeHome)
		})

		It("returns only user scope when allFlag is false", func() {
			scopes := availableScopes(false, tempDir)
			Expect(scopes).To(Equal([]string{"user"}))
		})
	})
})
