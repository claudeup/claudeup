// ABOUTME: Acceptance tests for 'claudeup events audit' command
// ABOUTME: Tests end-to-end audit report generation with real binary
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

var _ = Describe("events audit", func() {
	var env *helpers.TestEnv

	BeforeEach(func() {
		env = helpers.NewTestEnv(binaryPath)
	})

	Context("with no events", func() {
		It("shows informational message", func() {
			result := env.Run("events", "audit")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("No events recorded yet"))
		})
	})

	Context("with event log", func() {
		BeforeEach(func() {
			// Create events directory
			eventsDir := filepath.Join(env.ClaudeupDir, "events")
			Expect(os.MkdirAll(eventsDir, 0755)).To(Succeed())

			// Create test events
			baseTime := time.Now().Add(-24 * time.Hour)
			testEvents := []*events.FileOperation{
				{
					Timestamp:  baseTime,
					Operation:  "profile apply",
					File:       filepath.Join(env.ClaudeDir, "settings.json"),
					Scope:      "user",
					ChangeType: events.ChangeTypeUpdate,
					Before:     &events.Snapshot{Hash: "abc123", Size: 1000},
					After:      &events.Snapshot{Hash: "def456", Size: 1133},
				},
				{
					Timestamp:  baseTime.Add(-1 * time.Hour),
					Operation:  "plugin install",
					File:       filepath.Join(env.ClaudeDir, "plugins", "installed_plugins.json"),
					Scope:      "user",
					ChangeType: events.ChangeTypeUpdate,
					Before:     &events.Snapshot{Hash: "old123", Size: 500},
					After:      &events.Snapshot{Hash: "new456", Size: 950},
				},
				{
					Timestamp:  baseTime.Add(-2 * time.Hour),
					Operation:  "settings update",
					File:       filepath.Join(env.ClaudeDir, "settings.json"),
					Scope:      "project",
					ChangeType: events.ChangeTypeCreate,
					After:      &events.Snapshot{Hash: "proj123", Size: 200},
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

		It("generates audit report with default options", func() {
			result := env.Run("events", "audit")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Audit Report: claudeup Project"))
			Expect(result.Stdout).To(ContainSubstring("Generated:"))
			Expect(result.Stdout).To(ContainSubstring("Total Events: 3"))
		})

		It("shows summary section", func() {
			result := env.Run("events", "audit")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Summary"))
			Expect(result.Stdout).To(ContainSubstring("Files Modified:"))
			Expect(result.Stdout).To(ContainSubstring("Operations:"))
			Expect(result.Stdout).To(ContainSubstring("profile apply:"))
			Expect(result.Stdout).To(ContainSubstring("plugin install:"))
			Expect(result.Stdout).To(ContainSubstring("Scopes:"))
		})

		It("shows timeline section", func() {
			result := env.Run("events", "audit")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Timeline"))
			Expect(result.Stdout).To(ContainSubstring("PROFILE APPLY"))
			Expect(result.Stdout).To(ContainSubstring("PLUGIN INSTALL"))
			Expect(result.Stdout).To(ContainSubstring("SETTINGS UPDATE"))
		})

		It("filters by scope", func() {
			result := env.Run("events", "audit", "--scope", "user")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Scope: user"))
			Expect(result.Stdout).To(ContainSubstring("Total Events: 2"))
			Expect(result.Stdout).To(ContainSubstring("PROFILE APPLY"))
			Expect(result.Stdout).To(ContainSubstring("PLUGIN INSTALL"))
		})

		It("filters by operation", func() {
			result := env.Run("events", "audit", "--operation", "profile apply")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Operation: profile apply"))
			Expect(result.Stdout).To(ContainSubstring("Total Events: 1"))
			Expect(result.Stdout).To(ContainSubstring("PROFILE APPLY"))
			Expect(result.Stdout).NotTo(ContainSubstring("PLUGIN INSTALL"))
		})

		It("filters by time range with duration", func() {
			result := env.Run("events", "audit", "--since", "48h")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Period: Last 2 days"))
			Expect(result.Stdout).To(ContainSubstring("Total Events: 3"))
		})

		It("filters by time range with date", func() {
			yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
			result := env.Run("events", "audit", "--since", yesterday)

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Total Events: 3"))
		})

		It("generates markdown format", func() {
			result := env.Run("events", "audit", "--format", "markdown")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("# Audit Report: claudeup Project"))
			Expect(result.Stdout).To(ContainSubstring("**Generated:**"))
			Expect(result.Stdout).To(ContainSubstring("**Total Events:**"))
			Expect(result.Stdout).To(ContainSubstring("## Summary"))
			Expect(result.Stdout).To(ContainSubstring("## Timeline"))
			Expect(result.Stdout).To(ContainSubstring("###")) // Date headers
			Expect(result.Stdout).To(ContainSubstring("####")) // Event headers
		})

		It("combines multiple filters", func() {
			result := env.Run("events", "audit",
				"--scope", "user",
				"--operation", "profile apply",
				"--format", "markdown")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("**Scope:** user"))
			Expect(result.Stdout).To(ContainSubstring("**Operation:** profile apply"))
			Expect(result.Stdout).To(ContainSubstring("**Total Events:** 1"))
		})

		It("rejects invalid format", func() {
			result := env.Run("events", "audit", "--format", "json")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid format"))
			Expect(result.Stderr).To(ContainSubstring("must be 'text' or 'markdown'"))
		})

		It("handles invalid since duration", func() {
			result := env.Run("events", "audit", "--since", "invalid")

			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.Stderr).To(ContainSubstring("invalid --since value"))
		})
	})

	Context("help and usage", func() {
		It("shows help text", func() {
			result := env.Run("events", "audit", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Generate a comprehensive audit report"))
			Expect(result.Stdout).To(ContainSubstring("--scope"))
			Expect(result.Stdout).To(ContainSubstring("--since"))
			Expect(result.Stdout).To(ContainSubstring("--format"))
			Expect(result.Stdout).To(ContainSubstring("--operation"))
		})

		It("shows examples in help", func() {
			result := env.Run("events", "audit", "--help")

			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Examples:"))
			Expect(result.Stdout).To(ContainSubstring("claudeup events audit"))
			Expect(result.Stdout).To(ContainSubstring("--scope user"))
			Expect(result.Stdout).To(ContainSubstring("--format markdown"))
		})
	})
})
