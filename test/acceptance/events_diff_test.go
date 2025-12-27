// ABOUTME: Acceptance tests for 'claudeup events diff' command
// ABOUTME: Tests end-to-end diff display with real binary and event logs
package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup/internal/events"
	"github.com/claudeup/claudeup/test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("events diff", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Context("with no events", func() {
		It("shows informational message", func() {
			result := env.Run("events", "diff", "--file", "/some/file.json")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No events recorded yet"))
		})
	})

	Context("with event log", func() {
		var settingsPath string

		BeforeEach(func() {
			settingsPath = filepath.Join(env.ClaudeDir, "settings.json")

			// Create events directory
			eventsDir := filepath.Join(env.ClaudeupDir, "events")
			Expect(os.MkdirAll(eventsDir, 0755)).To(Succeed())

			// Create test events with content snapshots
			baseTime := time.Now().Add(-1 * time.Hour)

			beforeContent := `{"plugins":{"enabled":{}}}`
			afterContent := `{"plugins":{"enabled":{"test-plugin":true}},"description":"Added plugin"}`

			testEvents := []*events.FileOperation{
				{
					Timestamp:  baseTime,
					Operation:  "plugin install",
					File:       settingsPath,
					Scope:      "user",
					ChangeType: events.ChangeTypeUpdate,
					Before: &events.Snapshot{
						Hash:    "abc123",
						Size:    int64(len(beforeContent)),
						Content: beforeContent,
					},
					After: &events.Snapshot{
						Hash:    "def456",
						Size:    int64(len(afterContent)),
						Content: afterContent,
					},
				},
				{
					Timestamp:  baseTime.Add(-30 * time.Minute),
					Operation:  "settings update",
					File:       filepath.Join(env.ClaudeDir, "other.json"),
					Scope:      "user",
					ChangeType: events.ChangeTypeCreate,
					After: &events.Snapshot{
						Hash: "xyz789",
						Size: 100,
					},
				},
			}

			// Write events to JSONL file
			logPath := filepath.Join(eventsDir, "operations.log")
			file, err := os.Create(logPath)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			encoder := json.NewEncoder(file)
			for _, event := range testEvents {
				Expect(encoder.Encode(event)).To(Succeed())
			}
		})

		It("shows most recent change to specified file", func() {
			result := env.Run("events", "diff", "--file", settingsPath)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Most recent change to:"))
			Expect(result.Stdout).To(ContainSubstring(settingsPath))
		})

		It("displays event metadata", func() {
			result := env.Run("events", "diff", "--file", settingsPath)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Operation: plugin install"))
			Expect(result.Stdout).To(ContainSubstring("Scope: user"))
			Expect(result.Stdout).To(ContainSubstring("Change Type: update"))
			Expect(result.Stdout).To(ContainSubstring("Timestamp:"))
		})

		It("shows content diff when available", func() {
			result := env.Run("events", "diff", "--file", settingsPath)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Content diff:"))
			Expect(result.Stdout).To(ContainSubstring(events.SymbolAdded))
			Expect(result.Stdout).To(ContainSubstring("description"))
			Expect(result.Stdout).To(ContainSubstring(events.SymbolModified)) // ~ for plugins object
		})

		It("handles file with no events", func() {
			nonExistentFile := filepath.Join(env.ClaudeDir, "nonexistent.json")
			result := env.Run("events", "diff", "--file", nonExistentFile)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No events found for file:"))
		})

		It("handles file without content snapshots", func() {
			otherFile := filepath.Join(env.ClaudeDir, "other.json")
			result := env.Run("events", "diff", "--file", otherFile)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Most recent change to:"))
			Expect(result.Stdout).To(ContainSubstring("Content not available"))
		})

		It("requires --file flag", func() {
			result := env.Run("events", "diff")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("required flag(s)"))
			Expect(result.Stderr).To(ContainSubstring("file"))
		})

		It("handles relative paths", func() {
			// Test with a relative path - should be converted to absolute
			relPath := "settings.json"
			result := env.Run("events", "diff", "--file", relPath)

			// Since the events were recorded with absolute paths,
			// searching for a relative path won't find matches
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No events found for file:"))
		})

		It("shows truncated diff by default", func() {
			result := env.Run("events", "diff", "--file", settingsPath)

			Expect(result.ExitCode).To(Equal(0))
			// Nested objects should be truncated to {...}
			Expect(result.Stdout).To(ContainSubstring("{...}"))
		})

		It("shows deep diff with --full flag", func() {
			result := env.Run("events", "diff", "--file", settingsPath, "--full")

			Expect(result.ExitCode).To(Equal(0))
			// Should NOT show truncation
			Expect(result.Stdout).NotTo(ContainSubstring("{...}"))
			// Should show nested field changes
			Expect(result.Stdout).To(ContainSubstring("enabled:"))
			// Should show explicit labels
			Expect(result.Stdout).To(ContainSubstring("(added)"))
		})
	})

	Context("help and usage", func() {
		It("shows help text", func() {
			result := env.Run("events", "diff", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Display a human-readable diff"))
			Expect(result.Stdout).To(ContainSubstring("--file"))
		})

		It("shows examples in help", func() {
			result := env.Run("events", "diff", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Examples:"))
			Expect(result.Stdout).To(ContainSubstring("claudeup events diff"))
		})
	})
})
