// ABOUTME: Events command implementation for viewing file operation history
// ABOUTME: Displays tracked file changes with filtering and formatting options
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/claudeup/claudeup/v2/internal/config"
	"github.com/claudeup/claudeup/v2/internal/events"
	"github.com/claudeup/claudeup/v2/internal/ui"
	"github.com/spf13/cobra"
)

var (
	eventsFile      string
	eventsOperation string
	eventsScope     string
	eventsSince     string
	eventsLimit     int
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "View file operation history",
	Long: `Display tracked file operations with optional filtering.

Examples:
  claudeup events                          # Show recent events
  claudeup events --limit 20               # Show last 20 events
  claudeup events --file ~/.claude/settings.json
  claudeup events --operation "profile apply"
  claudeup events --scope user
  claudeup events --since 24h`,
	Args: cobra.NoArgs,
	RunE: runEvents,
}

func init() {
	rootCmd.AddCommand(eventsCmd)

	eventsCmd.Flags().StringVar(&eventsFile, "file", "", "Filter by file path")
	eventsCmd.Flags().StringVar(&eventsOperation, "operation", "", "Filter by operation name")
	eventsCmd.Flags().StringVar(&eventsScope, "scope", "", "Filter by scope (user/project/local)")
	eventsCmd.Flags().StringVar(&eventsSince, "since", "", "Show events since duration (e.g., 24h, 7d)")
	eventsCmd.Flags().IntVar(&eventsLimit, "limit", 20, "Maximum number of events to show")
}

func runEvents(cmd *cobra.Command, args []string) error {
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

	// Parse since duration if provided
	var sinceTime time.Time
	if eventsSince != "" {
		duration, err := parseDuration(eventsSince)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		sinceTime = time.Now().Add(-duration)
	}

	// Build filters
	filters := events.EventFilters{
		File:      eventsFile,
		Operation: eventsOperation,
		Scope:     eventsScope,
		Since:     sinceTime,
		Limit:     eventsLimit,
	}

	// Query events
	eventList, err := writer.Query(filters)
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}

	// Display events
	if len(eventList) == 0 {
		ui.PrintInfo("No events found matching the filters.")
		return nil
	}

	ui.PrintSuccess(fmt.Sprintf("Found %d event(s):", len(eventList)))
	fmt.Println()

	for _, event := range eventList {
		displayEvent(event)
		fmt.Println()
	}

	return nil
}

func displayEvent(event *events.FileOperation) {
	// Format timestamp
	timeStr := event.Timestamp.Format("2006-01-02 15:04:05")

	// Determine status icon
	statusIcon := "✓"
	if event.Error != "" {
		statusIcon = "✗"
	}

	// Print header
	ui.PrintInfo(fmt.Sprintf("%s  %s  %s (%s scope)",
		statusIcon,
		timeStr,
		strings.ToUpper(event.Operation),
		event.Scope,
	))

	// Print file path
	fmt.Printf("  File: %s\n", event.File)

	// Print change type
	changeIcon := "→"
	switch event.ChangeType {
	case events.ChangeTypeCreate:
		changeIcon = events.SymbolAdded
	case events.ChangeTypeUpdate:
		changeIcon = events.SymbolModified
	case events.ChangeTypeDelete:
		changeIcon = events.SymbolRemoved
	}
	fmt.Printf("  Change: %s %s\n", changeIcon, event.ChangeType)

	// Print before/after info if available
	if event.Before != nil && event.After != nil {
		sizeDiff := event.After.Size - event.Before.Size
		sizeDiffStr := fmt.Sprintf("%+d bytes", sizeDiff)
		if sizeDiff == 0 {
			sizeDiffStr = "no size change"
		}
		fmt.Printf("  Size: %s\n", sizeDiffStr)
	}

	// Print error if present
	if event.Error != "" {
		ui.PrintError(fmt.Sprintf("  Error: %s", event.Error))
	}
}

// parseDuration parses duration strings like "24h", "7d", "30m"
func parseDuration(s string) (time.Duration, error) {
	// Handle days specially
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, err
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}

	// Use standard time.ParseDuration for hours, minutes, seconds
	return time.ParseDuration(s)
}
