// ABOUTME: Events audit command implementation for comprehensive audit trails
// ABOUTME: Generates timeline reports with summary statistics in text or markdown format
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup/v3/internal/config"
	"github.com/claudeup/claudeup/v3/internal/events"
	"github.com/claudeup/claudeup/v3/internal/ui"
	"github.com/spf13/cobra"
)

var (
	auditScope     string
	auditUser      bool
	auditProject   bool
	auditLocal     bool
	auditOperation string
	auditSince     string
	auditFormat    string
)

var eventsAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Generate comprehensive audit trail of file operations",
	Long: `Generate a comprehensive audit report showing timeline of file operations,
summary statistics, and change history.

By default, shows all events from the last 7 days in text format.

Examples:
  claudeup events audit                           # Last 7 days, all scopes
  claudeup events audit --user                    # User scope only
  claudeup events audit --since 30d               # Last 30 days
  claudeup events audit --operation "profile apply"
  claudeup events audit --format markdown > report.md
  claudeup events audit --project --since 2025-01-01`,
	Args: cobra.NoArgs,
	RunE: runEventsAudit,
}

func init() {
	eventsCmd.AddCommand(eventsAuditCmd)

	eventsAuditCmd.Flags().StringVar(&auditScope, "scope", "", "Filter by scope (user/project/local)")
	eventsAuditCmd.Flags().BoolVar(&auditUser, "user", false, "Filter to user scope")
	eventsAuditCmd.Flags().BoolVar(&auditProject, "project", false, "Filter to project scope")
	eventsAuditCmd.Flags().BoolVar(&auditLocal, "local", false, "Filter to local scope")
	eventsAuditCmd.Flags().StringVar(&auditOperation, "operation", "", "Filter by operation name")
	eventsAuditCmd.Flags().StringVar(&auditSince, "since", "7d", "Show events since duration (e.g., 24h, 7d, 30d) or date (YYYY-MM-DD)")
	eventsAuditCmd.Flags().StringVar(&auditFormat, "format", "text", "Output format: text or markdown")
}

func runEventsAudit(cmd *cobra.Command, args []string) error {
	// Resolve scope from --scope or boolean aliases
	resolvedScope, err := resolveScopeFlags(auditScope, auditUser, auditProject, auditLocal)
	if err != nil {
		return err
	}
	auditScope = resolvedScope

	// Validate format
	if auditFormat != "text" && auditFormat != "markdown" {
		return fmt.Errorf("invalid format: %s (must be 'text' or 'markdown')", auditFormat)
	}

	// Get event log path
	eventsDir := filepath.Join(config.MustClaudeupHome(), "events")
	logPath := filepath.Join(eventsDir, "operations.log")

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		ui.PrintInfo("No events recorded yet.")
		ui.PrintInfo("File operations will be tracked automatically.")
		return nil
	}

	// Create writer to query events
	writer, err := events.NewJSONLWriter(logPath)
	if err != nil {
		return fmt.Errorf("failed to open events log: %w", err)
	}

	// Parse since time
	var sinceTime time.Time
	if auditSince != "" {
		// Try parsing as date first (YYYY-MM-DD)
		if parsedDate, err := time.Parse("2006-01-02", auditSince); err == nil {
			sinceTime = parsedDate
		} else {
			// Try parsing as duration (7d, 24h, etc.)
			duration, err := parseDuration(auditSince)
			if err != nil {
				return fmt.Errorf("invalid --since value: %w", err)
			}
			sinceTime = time.Now().Add(-duration)
		}
	}

	// Build filters
	filters := events.EventFilters{
		Scope:     auditScope,
		Operation: auditOperation,
		Since:     sinceTime,
		Limit:     0, // No limit for audit - get all matching events
	}

	// Query events
	eventList, err := writer.Query(filters)
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}

	// Handle empty results
	if len(eventList) == 0 {
		ui.PrintInfo("No events found matching the filters.")
		return nil
	}

	// Generate audit report
	opts := events.AuditOptions{
		Scope:     auditScope,
		Operation: auditOperation,
		Since:     sinceTime,
	}
	report := events.GenerateAuditReport(eventList, opts)

	// Format and output report
	var output string
	switch auditFormat {
	case "markdown":
		output = report.FormatAsMarkdown()
	default: // "text"
		output = report.FormatAsText()
	}

	fmt.Print(output)

	return nil
}
