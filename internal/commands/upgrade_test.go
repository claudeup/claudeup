// ABOUTME: Unit tests for upgrade command argument parsing
// ABOUTME: Tests target detection (marketplace vs plugin) and scope resolution
package commands

import (
	"os"
	"path/filepath"
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

var _ = Describe("availableScopes", func() {
	It("returns all scopes when allFlag is true", func() {
		scopes := availableScopes(true)
		Expect(scopes).To(Equal(claude.ValidScopes))
	})

	It("always includes user scope when allFlag is false", func() {
		scopes := availableScopes(false)
		Expect(scopes).To(ContainElement("user"))
	})

	Context("in a project directory", func() {
		var tempDir string
		var claudeHome string
		var origClaudeDir string
		var origWd string

		BeforeEach(func() {
			var err error
			origWd, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			origClaudeDir = claudeDir

			// Create a temp dir with a .claude subdirectory to simulate a project
			tempDir, _ = os.MkdirTemp("", "scope-test-*")
			err = os.MkdirAll(filepath.Join(tempDir, ".claude"), 0755)
			Expect(err).NotTo(HaveOccurred())

			// Point claudeDir elsewhere so IsProjectContext detects a distinct project
			claudeHome, _ = os.MkdirTemp("", "claude-home-*")
			claudeDir = claudeHome

			// Change to the project directory
			err = os.Chdir(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Chdir(origWd)
			claudeDir = origClaudeDir
			os.RemoveAll(tempDir)
			os.RemoveAll(claudeHome)
		})

		It("includes project and local scopes when allFlag is false", func() {
			scopes := availableScopes(false)
			Expect(scopes).To(ContainElement("user"))
			Expect(scopes).To(ContainElement("project"))
			Expect(scopes).To(ContainElement("local"))
		})

		It("returns all valid scopes when allFlag is true regardless of context", func() {
			scopes := availableScopes(true)
			Expect(scopes).To(Equal(claude.ValidScopes))
		})
	})

	Context("outside a project directory", func() {
		var tempDir string
		var claudeHome string
		var origClaudeDir string
		var origWd string

		BeforeEach(func() {
			var err error
			origWd, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			origClaudeDir = claudeDir

			// Create a temp dir WITHOUT .claude to simulate non-project context
			tempDir, _ = os.MkdirTemp("", "scope-test-*")
			claudeHome, _ = os.MkdirTemp("", "claude-home-*")
			claudeDir = claudeHome

			err = os.Chdir(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Chdir(origWd)
			claudeDir = origClaudeDir
			os.RemoveAll(tempDir)
			os.RemoveAll(claudeHome)
		})

		It("returns only user scope when allFlag is false", func() {
			scopes := availableScopes(false)
			Expect(scopes).To(Equal([]string{"user"}))
		})
	})
})
