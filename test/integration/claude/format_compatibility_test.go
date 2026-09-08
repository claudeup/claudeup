// ABOUTME: Integration tests for Claude CLI format compatibility
// ABOUTME: Smoke tests against real ~/.claude directory to catch format changes
package claude_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v5/internal/claude"
)

var _ = Describe("Claude CLI Format Compatibility", func() {
	var claudeDir string

	BeforeEach(func() {
		homeDir, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		claudeDir = filepath.Join(homeDir, ".claude")
	})

	Context("Smoke tests against real Claude installation", func() {
		It("can parse installed_plugins.json from user's Claude dir", func() {
			// Skip if Claude not installed
			if _, err := os.Stat(claudeDir); errors.Is(err, fs.ErrNotExist) {
				Skip("Claude CLI not installed on this system")
			}

			// Attempt to load real plugins file
			registry, err := claude.LoadPlugins(claudeDir)
			Expect(err).NotTo(HaveOccurred(),
				"Failed to parse real installed_plugins.json - Claude CLI format may have changed")

			// Validate we got reasonable data
			Expect(registry).NotTo(BeNil())
			Expect(registry.Version).To(BeNumerically(">=", 1),
				"Plugin registry version should be at least 1")
		})

		It("classifies every real plugin source as either a resolvable path or external", func() {
			marketplacesDir := filepath.Join(claudeDir, "plugins", "marketplaces")
			entries, err := os.ReadDir(marketplacesDir)
			if errors.Is(err, fs.ErrNotExist) {
				Skip("No marketplaces installed on this system")
			}
			Expect(err).NotTo(HaveOccurred())

			// Count directories carrying a plugin index. A marketplace directory
			// without one is a half-finished add or a user's backup copy, neither
			// of which says anything about the Claude CLI format.
			indexed := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				indexPath := filepath.Join(marketplacesDir, entry.Name(), ".claude-plugin", "marketplace.json")
				if _, err := os.Stat(indexPath); err == nil {
					indexed++
				}
			}
			if indexed == 0 {
				Skip("No indexed marketplaces installed on this system")
			}

			checked := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				marketplacePath := filepath.Join(marketplacesDir, entry.Name())
				index, err := claude.LoadMarketplaceIndex(marketplacePath)
				if errors.Is(err, fs.ErrNotExist) {
					// Directory carries no plugin index.
					continue
				}
				Expect(err).NotTo(HaveOccurred(),
					"Failed to parse marketplace index for %q - Claude CLI format may have changed",
					entry.Name())
				for _, plugin := range index.Plugins {
					if plugin.Source == nil {
						continue
					}
					checked++
					if !plugin.Source.IsRelativePath() {
						// External source; Claude Code fetches it. Nothing to resolve locally.
						continue
					}
					// A source classified as relative must carry a path. An empty
					// one means a source type is being read as a path, and the join
					// below would silently resolve to the marketplace root.
					Expect(plugin.Source.RelativePath).NotTo(BeEmpty(),
						"plugin %q in marketplace %q classified as a relative path but carries no path",
						plugin.Name, entry.Name())

					// The path must exist inside the marketplace. A path that does
					// not exist means upgrade would fail on a path built from a
					// source type it is reading as a path.
					resolved := filepath.Join(marketplacePath, plugin.Source.RelativePath)
					_, statErr := os.Stat(resolved)
					Expect(statErr).NotTo(HaveOccurred(),
						"plugin %q in marketplace %q classified as a relative path to %q, which does not exist",
						plugin.Name, entry.Name(), resolved)
				}
			}
			Expect(checked).To(BeNumerically(">", 0),
				"Found %d indexed marketplaces but no plugin sources to check - Claude CLI layout may have changed",
				indexed)
		})

		It("can parse settings.json from user's Claude dir", func() {
			// Skip if Claude not installed
			if _, err := os.Stat(claudeDir); errors.Is(err, fs.ErrNotExist) {
				Skip("Claude CLI not installed on this system")
			}

			// Attempt to load real settings file
			settings, err := claude.LoadSettings(claudeDir)
			Expect(err).NotTo(HaveOccurred(),
				"Failed to parse real settings.json - Claude CLI format may have changed")

			// Validate we got reasonable data
			Expect(settings).NotTo(BeNil())
			Expect(settings.EnabledPlugins).NotTo(BeNil(),
				"Settings should have EnabledPlugins map")
		})
	})

	Context("Error handling for missing Claude installation", func() {
		It("returns clear error when Claude directory doesn't exist", func() {
			nonExistentDir := "/tmp/claude-does-not-exist-12345"

			_, err := claude.LoadPlugins(nonExistentDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Claude CLI not found"))
		})
	})

	Context("Error handling for file path changes", func() {
		It("returns empty registry when plugins file missing (fresh install)", func() {
			// Create temp Claude dir without plugins file
			// This simulates a fresh Claude install
			tempDir := GinkgoT().TempDir()

			registry, err := claude.LoadPlugins(tempDir)
			Expect(err).NotTo(HaveOccurred(),
				"LoadPlugins should handle fresh install gracefully")

			// Should return empty V2 registry
			Expect(registry).NotTo(BeNil())
			Expect(registry.Version).To(Equal(2))
			Expect(registry.Plugins).NotTo(BeNil())
			Expect(registry.Plugins).To(BeEmpty())
		})

		It("returns PathNotFoundError when settings file missing but Claude dir exists", func() {
			// Create temp Claude dir without settings file
			tempDir := GinkgoT().TempDir()

			_, err := claude.LoadSettings(tempDir)
			Expect(err).To(HaveOccurred())

			// Should be PathNotFoundError
			pathErr, ok := err.(*claude.PathNotFoundError)
			Expect(ok).To(BeTrue(), "Should return PathNotFoundError when file missing")
			Expect(pathErr.Component).To(Equal("settings"))
		})
	})
})
