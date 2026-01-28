// ABOUTME: Integration tests for Claude CLI format compatibility
// ABOUTME: Smoke tests against real ~/.claude directory to catch format changes
package claude_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/claudeup/claudeup/v3/internal/claude"
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
			if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
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

		It("can parse settings.json from user's Claude dir", func() {
			// Skip if Claude not installed
			if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
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
