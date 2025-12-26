// ABOUTME: Audit report generation for file operation history
// ABOUTME: Provides timeline views, summary statistics, and export formats for event analysis
package events

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// maxTimelineDiffLines limits diff output in timeline for readability
	maxTimelineDiffLines = 5
)

// AuditReport represents a comprehensive audit trail of file operations.
type AuditReport struct {
	GeneratedAt time.Time
	Scope       string // user/project/local or empty for all
	Operation   string // filter by operation or empty for all
	Period      string // human-readable time range description
	Events      []*FileOperation
	Summary     AuditSummary
}

// AuditSummary provides aggregate statistics about events in the report.
type AuditSummary struct {
	TotalEvents   int
	FilesAffected []string            // unique file paths
	Operations    map[string]int      // operation name -> count
	Scopes        map[string]int      // scope -> count
	ChangeTypes   map[string]int      // create/update/delete -> count
	Errors        int                 // count of failed operations
	SizeChanges   int64               // total bytes added/removed
}

// AuditOptions configures how the audit report is generated.
type AuditOptions struct {
	Scope     string
	Operation string
	Since     time.Time
}

// GenerateAuditReport creates an audit report from a list of events.
// Events should already be filtered and sorted (newest first from writer.Query).
func GenerateAuditReport(events []*FileOperation, opts AuditOptions) *AuditReport {
	report := &AuditReport{
		GeneratedAt: time.Now(),
		Scope:       opts.Scope,
		Operation:   opts.Operation,
		Events:      events,
		Summary:     calculateSummary(events),
	}

	// Generate human-readable period description
	report.Period = formatPeriod(opts.Since)

	return report
}

// calculateSummary computes aggregate statistics from events.
func calculateSummary(events []*FileOperation) AuditSummary {
	summary := AuditSummary{
		TotalEvents: len(events),
		Operations:  make(map[string]int),
		Scopes:      make(map[string]int),
		ChangeTypes: make(map[string]int),
	}

	filesMap := make(map[string]bool)

	for _, event := range events {
		// Track unique files
		filesMap[event.File] = true

		// Count operations
		summary.Operations[event.Operation]++

		// Count scopes
		summary.Scopes[event.Scope]++

		// Count change types
		summary.ChangeTypes[event.ChangeType]++

		// Count errors
		if event.Error != "" {
			summary.Errors++
		}

		// Calculate size changes
		if event.Before != nil && event.After != nil {
			summary.SizeChanges += event.After.Size - event.Before.Size
		}
	}

	// Convert files map to sorted slice
	summary.FilesAffected = make([]string, 0, len(filesMap))
	for file := range filesMap {
		summary.FilesAffected = append(summary.FilesAffected, file)
	}
	sort.Strings(summary.FilesAffected)

	return summary
}

// formatPeriod converts a Since time into a human-readable period description.
func formatPeriod(since time.Time) string {
	if since.IsZero() {
		return "All time"
	}

	duration := time.Since(since)
	days := int(duration.Hours() / 24)

	switch {
	case days == 0:
		return "Last 24 hours"
	case days == 1:
		return "Last day"
	case days < 7:
		return fmt.Sprintf("Last %d days", days)
	case days < 30:
		weeks := days / 7
		if weeks == 1 {
			return "Last week"
		}
		return fmt.Sprintf("Last %d weeks", weeks)
	case days < 365:
		months := days / 30
		if months == 1 {
			return "Last month"
		}
		return fmt.Sprintf("Last %d months", months)
	default:
		return fmt.Sprintf("Since %s", since.Format("2006-01-02"))
	}
}

// FormatAsText renders the audit report as human-readable text.
func (r *AuditReport) FormatAsText() string {
	var b strings.Builder

	// Header
	b.WriteString("Audit Report: claudeup Project\n")
	b.WriteString("===============================\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", r.GeneratedAt.Format("2006-01-02 15:04:05")))
	if r.Scope != "" {
		b.WriteString(fmt.Sprintf("Scope: %s\n", r.Scope))
	}
	if r.Operation != "" {
		b.WriteString(fmt.Sprintf("Operation: %s\n", r.Operation))
	}
	b.WriteString(fmt.Sprintf("Period: %s\n", r.Period))
	b.WriteString(fmt.Sprintf("Total Events: %d\n", r.Summary.TotalEvents))
	b.WriteString("\n")

	// Summary section
	b.WriteString("Summary\n")
	b.WriteString("-------\n")
	b.WriteString(fmt.Sprintf("Files Modified: %d\n", len(r.Summary.FilesAffected)))

	if len(r.Summary.Operations) > 0 {
		b.WriteString("Operations:\n")
		// Sort operations by count (descending)
		type opCount struct {
			name  string
			count int
		}
		ops := make([]opCount, 0, len(r.Summary.Operations))
		for name, count := range r.Summary.Operations {
			ops = append(ops, opCount{name, count})
		}
		sort.Slice(ops, func(i, j int) bool {
			return ops[i].count > ops[j].count
		})
		for _, op := range ops {
			b.WriteString(fmt.Sprintf("  - %s: %d events\n", op.name, op.count))
		}
	}

	if len(r.Summary.Scopes) > 0 {
		scopeParts := make([]string, 0, len(r.Summary.Scopes))
		for scope, count := range r.Summary.Scopes {
			scopeParts = append(scopeParts, fmt.Sprintf("%s (%d)", scope, count))
		}
		sort.Strings(scopeParts)
		b.WriteString(fmt.Sprintf("Scopes: %s\n", strings.Join(scopeParts, ", ")))
	}

	if r.Summary.Errors > 0 {
		b.WriteString(fmt.Sprintf("Errors: %d\n", r.Summary.Errors))
	}

	if r.Summary.SizeChanges != 0 {
		b.WriteString(fmt.Sprintf("Total Size Change: %+d bytes\n", r.Summary.SizeChanges))
	}

	b.WriteString("\n")

	// Timeline section
	b.WriteString("Timeline\n")
	b.WriteString("--------\n")
	b.WriteString("\n")

	if len(r.Events) == 0 {
		b.WriteString("No events to display.\n")
		return b.String()
	}

	// Group events by date
	eventsByDate := groupEventsByDate(r.Events)

	// Sort dates descending (most recent first)
	dates := make([]string, 0, len(eventsByDate))
	for date := range eventsByDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	for _, date := range dates {
		b.WriteString(fmt.Sprintf("%s\n", date))
		b.WriteString("----------\n")

		dayEvents := eventsByDate[date]
		for _, event := range dayEvents {
			b.WriteString(formatEventForTimeline(event))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// FormatAsMarkdown renders the audit report as GitHub-flavored markdown.
func (r *AuditReport) FormatAsMarkdown() string {
	var b strings.Builder

	// Header
	b.WriteString("# Audit Report: claudeup Project\n\n")
	b.WriteString(fmt.Sprintf("**Generated:** %s\n", r.GeneratedAt.Format("2006-01-02 15:04:05")))
	if r.Scope != "" {
		b.WriteString(fmt.Sprintf("**Scope:** %s\n", r.Scope))
	}
	if r.Operation != "" {
		b.WriteString(fmt.Sprintf("**Operation:** %s\n", r.Operation))
	}
	b.WriteString(fmt.Sprintf("**Period:** %s\n", r.Period))
	b.WriteString(fmt.Sprintf("**Total Events:** %d\n\n", r.Summary.TotalEvents))

	// Summary section
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- **Files Modified:** %d\n", len(r.Summary.FilesAffected)))

	if len(r.Summary.Operations) > 0 {
		b.WriteString("- **Operations:**\n")
		// Sort operations by count (descending)
		type opCount struct {
			name  string
			count int
		}
		ops := make([]opCount, 0, len(r.Summary.Operations))
		for name, count := range r.Summary.Operations {
			ops = append(ops, opCount{name, count})
		}
		sort.Slice(ops, func(i, j int) bool {
			return ops[i].count > ops[j].count
		})
		for _, op := range ops {
			b.WriteString(fmt.Sprintf("  - %s: %d events\n", op.name, op.count))
		}
	}

	if len(r.Summary.Scopes) > 0 {
		scopeParts := make([]string, 0, len(r.Summary.Scopes))
		for scope, count := range r.Summary.Scopes {
			scopeParts = append(scopeParts, fmt.Sprintf("%s (%d)", scope, count))
		}
		sort.Strings(scopeParts)
		b.WriteString(fmt.Sprintf("- **Scopes:** %s\n", strings.Join(scopeParts, ", ")))
	}

	if r.Summary.Errors > 0 {
		b.WriteString(fmt.Sprintf("- **Errors:** %d\n", r.Summary.Errors))
	}

	if r.Summary.SizeChanges != 0 {
		b.WriteString(fmt.Sprintf("- **Total Size Change:** %+d bytes\n", r.Summary.SizeChanges))
	}

	b.WriteString("\n")

	// Timeline section
	b.WriteString("## Timeline\n\n")

	if len(r.Events) == 0 {
		b.WriteString("No events to display.\n")
		return b.String()
	}

	// Group events by date
	eventsByDate := groupEventsByDate(r.Events)

	// Sort dates descending (most recent first)
	dates := make([]string, 0, len(eventsByDate))
	for date := range eventsByDate {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	for _, date := range dates {
		b.WriteString(fmt.Sprintf("### %s\n\n", date))

		dayEvents := eventsByDate[date]
		for _, event := range dayEvents {
			b.WriteString(formatEventForMarkdown(event))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// groupEventsByDate groups events by their date (YYYY-MM-DD).
func groupEventsByDate(events []*FileOperation) map[string][]*FileOperation {
	groups := make(map[string][]*FileOperation)

	for _, event := range events {
		dateKey := event.Timestamp.Format("2006-01-02")
		groups[dateKey] = append(groups[dateKey], event)
	}

	return groups
}

// formatEventForTimeline formats a single event for text timeline display.
func formatEventForTimeline(event *FileOperation) string {
	var b strings.Builder

	// Time and operation header
	timeStr := event.Timestamp.Format("15:04:05")
	statusIcon := "✓"
	if event.Error != "" {
		statusIcon = "✗"
	}
	b.WriteString(fmt.Sprintf("[%s] %s %s (%s)\n", timeStr, statusIcon, strings.ToUpper(event.Operation), event.Scope))

	// File path
	b.WriteString(fmt.Sprintf("  File: %s\n", event.File))

	// Change info
	changeInfo := event.ChangeType
	if event.Before != nil && event.After != nil {
		sizeDiff := event.After.Size - event.Before.Size
		if sizeDiff != 0 {
			changeInfo = fmt.Sprintf("%s (%+d bytes)", event.ChangeType, sizeDiff)
		}
	}
	b.WriteString(fmt.Sprintf("  Change: %s\n", changeInfo))

	// Show brief diff if content available
	if event.Before != nil || event.After != nil {
		diff := DiffSnapshots(event.Before, event.After)
		if diff.HasChanges && diff.ContentAvailable {
			// Show first line of diff summary only
			lines := strings.Split(diff.Summary, "\n")
			if len(lines) > 0 && lines[0] != "" {
				b.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(lines[0])))
			}
		}
	}

	// Error if present
	if event.Error != "" {
		b.WriteString(fmt.Sprintf("  Error: %s\n", event.Error))
	}

	return b.String()
}

// formatEventForMarkdown formats a single event for markdown timeline display.
func formatEventForMarkdown(event *FileOperation) string {
	var b strings.Builder

	// Time and operation header
	timeStr := event.Timestamp.Format("15:04:05")
	statusIcon := "✓"
	if event.Error != "" {
		statusIcon = "✗"
	}
	b.WriteString(fmt.Sprintf("#### %s %s %s (%s scope)\n\n", timeStr, statusIcon, strings.ToUpper(event.Operation), event.Scope))

	// File path
	b.WriteString(fmt.Sprintf("- **File:** `%s`\n", event.File))

	// Change info
	changeInfo := event.ChangeType
	if event.Before != nil && event.After != nil {
		sizeDiff := event.After.Size - event.Before.Size
		if sizeDiff != 0 {
			changeInfo = fmt.Sprintf("%s (%+d bytes)", event.ChangeType, sizeDiff)
		}
	}
	b.WriteString(fmt.Sprintf("- **Change:** %s\n", changeInfo))

	// Show brief diff if content available
	if event.Before != nil || event.After != nil {
		diff := DiffSnapshots(event.Before, event.After)
		if diff.HasChanges && diff.ContentAvailable {
			// Show first few lines of diff in code block
			lines := strings.Split(diff.Summary, "\n")
			if len(lines) > 0 && lines[0] != "" {
				b.WriteString("- **Details:**\n  ```\n")
				// Limit to first few lines for brevity in timeline
				if len(lines) > maxTimelineDiffLines {
					lines = lines[:maxTimelineDiffLines]
				}
				for _, line := range lines {
					if line != "" {
						b.WriteString(fmt.Sprintf("  %s\n", line))
					}
				}
				b.WriteString("  ```\n")
			}
		}
	}

	// Error if present
	if event.Error != "" {
		b.WriteString(fmt.Sprintf("- **Error:** %s\n", event.Error))
	}

	return b.String()
}
