package events

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAudit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Audit Suite")
}

var _ = Describe("Audit Report Generation", func() {
	var (
		testEvents []*FileOperation
		baseTime   time.Time
	)

	BeforeEach(func() {
		// Create test events with consistent timestamps
		baseTime = time.Date(2025, 12, 26, 14, 0, 0, 0, time.UTC)

		testEvents = []*FileOperation{
			{
				Timestamp:  baseTime,
				Operation:  "profile apply",
				File:       "/home/user/.claude/settings.json",
				Scope:      "user",
				ChangeType: ChangeTypeUpdate,
				Before:     &Snapshot{Hash: "abc123", Size: 1000},
				After:      &Snapshot{Hash: "def456", Size: 1133},
			},
			{
				Timestamp:  baseTime.Add(-1 * time.Hour),
				Operation:  "plugin install",
				File:       "/home/user/.claude/plugins/installed_plugins.json",
				Scope:      "user",
				ChangeType: ChangeTypeUpdate,
				Before:     &Snapshot{Hash: "old123", Size: 500},
				After:      &Snapshot{Hash: "new456", Size: 950},
			},
			{
				Timestamp:  baseTime.Add(-25 * time.Hour), // Previous day
				Operation:  "profile apply",
				File:       "/home/user/project/.claudeup/profile.json",
				Scope:      "project",
				ChangeType: ChangeTypeCreate,
				After:      &Snapshot{Hash: "proj123", Size: 200},
			},
			{
				Timestamp:  baseTime.Add(-25 * time.Hour),
				Operation:  "settings update",
				File:       "/home/user/.claude/settings.json",
				Scope:      "user",
				ChangeType: ChangeTypeUpdate,
				Before:     &Snapshot{Hash: "prev123", Size: 1000},
				After:      &Snapshot{Hash: "abc123", Size: 1000},
				Error:      "validation failed",
			},
		}
	})

	Describe("GenerateAuditReport", func() {
		It("creates a report with correct metadata", func() {
			opts := AuditOptions{
				Scope:     "user",
				Operation: "",
				Since:     baseTime.Add(-7 * 24 * time.Hour),
			}

			report := GenerateAuditReport(testEvents, opts)

			Expect(report).NotTo(BeNil())
			Expect(report.Scope).To(Equal("user"))
			Expect(report.Operation).To(Equal(""))
			Expect(report.Events).To(HaveLen(4))
			Expect(report.GeneratedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("includes all events in the report", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)

			Expect(report.Events).To(Equal(testEvents))
		})

		It("calculates summary statistics correctly", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)

			summary := report.Summary
			Expect(summary.TotalEvents).To(Equal(4))
			Expect(summary.FilesAffected).To(HaveLen(3))
			Expect(summary.FilesAffected).To(ContainElements(
				"/home/user/.claude/settings.json",
				"/home/user/.claude/plugins/installed_plugins.json",
				"/home/user/project/.claudeup/profile.json",
			))
			Expect(summary.Operations).To(HaveKeyWithValue("profile apply", 2))
			Expect(summary.Operations).To(HaveKeyWithValue("plugin install", 1))
			Expect(summary.Operations).To(HaveKeyWithValue("settings update", 1))
			Expect(summary.Scopes).To(HaveKeyWithValue("user", 3))
			Expect(summary.Scopes).To(HaveKeyWithValue("project", 1))
			Expect(summary.ChangeTypes).To(HaveKeyWithValue(ChangeTypeUpdate, 3))
			Expect(summary.ChangeTypes).To(HaveKeyWithValue(ChangeTypeCreate, 1))
			Expect(summary.Errors).To(Equal(1))
		})

		It("calculates total size changes correctly", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)

			// Event 1: +133 bytes
			// Event 2: +450 bytes
			// Event 3: no before, so not counted
			// Event 4: 0 bytes (same size)
			// Total: +583 bytes
			Expect(report.Summary.SizeChanges).To(Equal(int64(583)))
		})
	})

	Describe("calculateSummary", func() {
		It("handles empty event list", func() {
			summary := calculateSummary([]*FileOperation{})

			Expect(summary.TotalEvents).To(Equal(0))
			Expect(summary.FilesAffected).To(BeEmpty())
			Expect(summary.Operations).To(BeEmpty())
			Expect(summary.Scopes).To(BeEmpty())
			Expect(summary.Errors).To(Equal(0))
			Expect(summary.SizeChanges).To(Equal(int64(0)))
		})

		It("handles single event", func() {
			singleEvent := []*FileOperation{testEvents[0]}
			summary := calculateSummary(singleEvent)

			Expect(summary.TotalEvents).To(Equal(1))
			Expect(summary.FilesAffected).To(HaveLen(1))
			Expect(summary.Operations).To(HaveLen(1))
			Expect(summary.Scopes).To(HaveLen(1))
		})

		It("deduplicates files correctly", func() {
			// Add duplicate file reference
			duplicateEvents := append(testEvents, &FileOperation{
				Timestamp:  baseTime.Add(-2 * time.Hour),
				Operation:  "another operation",
				File:       "/home/user/.claude/settings.json", // Same file as event 0
				Scope:      "user",
				ChangeType: ChangeTypeUpdate,
			})

			summary := calculateSummary(duplicateEvents)

			Expect(summary.FilesAffected).To(HaveLen(3)) // Still only 3 unique files
		})

		It("sorts files alphabetically", func() {
			summary := calculateSummary(testEvents)

			// Files should be sorted
			Expect(summary.FilesAffected[0]).To(ContainSubstring(".claude/plugins"))
			Expect(summary.FilesAffected[1]).To(ContainSubstring(".claude/settings"))
			Expect(summary.FilesAffected[2]).To(ContainSubstring(".claudeup/profile"))
		})
	})

	Describe("formatPeriod", func() {
		It("returns 'All time' for zero time", func() {
			period := formatPeriod(time.Time{})
			Expect(period).To(Equal("All time"))
		})

		It("formats 24 hours correctly", func() {
			since := time.Now().Add(-12 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last 24 hours"))
		})

		It("formats single day correctly", func() {
			since := time.Now().Add(-25 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last day"))
		})

		It("formats multiple days correctly", func() {
			since := time.Now().Add(-5 * 24 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last 5 days"))
		})

		It("formats single week correctly", func() {
			since := time.Now().Add(-7 * 24 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last week"))
		})

		It("formats multiple weeks correctly", func() {
			since := time.Now().Add(-14 * 24 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last 2 weeks"))
		})

		It("formats single month correctly", func() {
			since := time.Now().Add(-30 * 24 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last month"))
		})

		It("formats multiple months correctly", func() {
			since := time.Now().Add(-90 * 24 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(Equal("Last 3 months"))
		})

		It("formats dates beyond a year with specific date", func() {
			since := time.Now().Add(-400 * 24 * time.Hour)
			period := formatPeriod(since)
			Expect(period).To(ContainSubstring("Since 20"))
		})
	})

	Describe("groupEventsByDate", func() {
		It("groups events by date correctly", func() {
			groups := groupEventsByDate(testEvents)

			// Should have 2 groups: today and yesterday
			Expect(groups).To(HaveLen(2))

			todayKey := baseTime.Format("2006-01-02")
			yesterdayKey := baseTime.Add(-25 * time.Hour).Format("2006-01-02")

			Expect(groups[todayKey]).To(HaveLen(2))
			Expect(groups[yesterdayKey]).To(HaveLen(2))
		})

		It("handles empty event list", func() {
			groups := groupEventsByDate([]*FileOperation{})
			Expect(groups).To(BeEmpty())
		})

		It("preserves event order within each date group", func() {
			groups := groupEventsByDate(testEvents)

			todayKey := baseTime.Format("2006-01-02")
			todayEvents := groups[todayKey]

			// Events should be in the same order they were added
			Expect(todayEvents[0].Timestamp).To(Equal(baseTime))
			Expect(todayEvents[1].Timestamp).To(Equal(baseTime.Add(-1 * time.Hour)))
		})
	})

	Describe("FormatAsText", func() {
		It("generates complete text report", func() {
			opts := AuditOptions{
				Scope: "user",
				Since: baseTime.Add(-7 * 24 * time.Hour),
			}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("Audit Report: claudeup Project"))
			Expect(text).To(ContainSubstring("==============================="))
			Expect(text).To(ContainSubstring("Generated:"))
			Expect(text).To(ContainSubstring("Scope: user"))
			Expect(text).To(ContainSubstring("Total Events: 4"))
		})

		It("includes summary section", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("Summary"))
			Expect(text).To(ContainSubstring("Files Modified: 3"))
			Expect(text).To(ContainSubstring("Operations:"))
			Expect(text).To(ContainSubstring("profile apply: 2 events"))
			Expect(text).To(ContainSubstring("plugin install: 1 events"))
			Expect(text).To(ContainSubstring("Scopes:"))
			Expect(text).To(ContainSubstring("user (3)"))
			Expect(text).To(ContainSubstring("project (1)"))
			Expect(text).To(ContainSubstring("Errors: 1"))
		})

		It("includes timeline section grouped by date", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("Timeline"))
			Expect(text).To(ContainSubstring("--------"))
			Expect(text).To(ContainSubstring("2025-12-26"))
			Expect(text).To(ContainSubstring("2025-12-25"))
		})

		It("formats individual events correctly", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			// Check event formatting
			Expect(text).To(ContainSubstring("PROFILE APPLY"))
			Expect(text).To(ContainSubstring("PLUGIN INSTALL"))
			Expect(text).To(ContainSubstring("File:"))
			Expect(text).To(ContainSubstring("Change:"))
			Expect(text).To(ContainSubstring("✓")) // Success icon
			Expect(text).To(ContainSubstring("✗")) // Error icon
		})

		It("shows size changes in events", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("+133 bytes"))
			Expect(text).To(ContainSubstring("+450 bytes"))
		})

		It("shows error information", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("Error: validation failed"))
		})

		It("handles empty events list", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport([]*FileOperation{}, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("No events to display"))
		})
	})

	Describe("FormatAsMarkdown", func() {
		It("generates complete markdown report", func() {
			opts := AuditOptions{
				Scope:     "user",
				Operation: "profile apply",
				Since:     baseTime.Add(-7 * 24 * time.Hour),
			}
			report := GenerateAuditReport(testEvents, opts)
			markdown := report.FormatAsMarkdown()

			Expect(markdown).To(ContainSubstring("# Audit Report: claudeup Project"))
			Expect(markdown).To(ContainSubstring("**Generated:**"))
			Expect(markdown).To(ContainSubstring("**Scope:** user"))
			Expect(markdown).To(ContainSubstring("**Operation:** profile apply"))
			Expect(markdown).To(ContainSubstring("**Total Events:** 4"))
		})

		It("includes summary section with markdown formatting", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			markdown := report.FormatAsMarkdown()

			Expect(markdown).To(ContainSubstring("## Summary"))
			Expect(markdown).To(ContainSubstring("- **Files Modified:** 3"))
			Expect(markdown).To(ContainSubstring("- **Operations:**"))
			Expect(markdown).To(ContainSubstring("- **Scopes:**"))
			Expect(markdown).To(ContainSubstring("- **Errors:** 1"))
		})

		It("includes timeline section with markdown headers", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			markdown := report.FormatAsMarkdown()

			Expect(markdown).To(ContainSubstring("## Timeline"))
			Expect(markdown).To(ContainSubstring("### 2025-12-26"))
			Expect(markdown).To(ContainSubstring("### 2025-12-25"))
		})

		It("formats individual events with markdown", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			markdown := report.FormatAsMarkdown()

			// Check markdown formatting
			Expect(markdown).To(ContainSubstring("#### "))
			Expect(markdown).To(ContainSubstring("- **File:** `"))
			Expect(markdown).To(ContainSubstring("- **Change:**"))
			// Note: Code blocks only appear when content is available in snapshots
		})

		It("includes file paths in code formatting", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			markdown := report.FormatAsMarkdown()

			Expect(markdown).To(ContainSubstring("`/home/user/.claude/settings.json`"))
			Expect(markdown).To(ContainSubstring("`/home/user/.claude/plugins/installed_plugins.json`"))
		})

		It("handles empty events list", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport([]*FileOperation{}, opts)
			markdown := report.FormatAsMarkdown()

			Expect(markdown).To(ContainSubstring("No events to display"))
		})
	})

	Describe("Event Timeline Formatting", func() {
		It("formats text timeline with timestamps", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			// Check for time format [HH:MM:SS]
			Expect(text).To(MatchRegexp(`\[14:00:00\]`))
			Expect(text).To(MatchRegexp(`\[13:00:00\]`))
		})

		It("sorts dates in descending order (most recent first)", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			// 2025-12-26 should appear before 2025-12-25
			idx2026 := strings.Index(text, "2025-12-26")
			idx2025 := strings.Index(text, "2025-12-25")
			Expect(idx2026).To(BeNumerically("<", idx2025))
		})

		It("shows operation scope in timeline", func() {
			opts := AuditOptions{}
			report := GenerateAuditReport(testEvents, opts)
			text := report.FormatAsText()

			Expect(text).To(ContainSubstring("(user)"))
			Expect(text).To(ContainSubstring("(project)"))
		})
	})
})
