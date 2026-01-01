// ABOUTME: Events diff command implementation for showing detailed file changes
// ABOUTME: between before/after snapshots from the event tracking system.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/config"
	"github.com/claudeup/claudeup/internal/events"
	"github.com/claudeup/claudeup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	diffFile string
	diffFull bool
)

var eventsDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show detailed changes for a file operation",
	Long: `Display a human-readable diff of what changed in a file operation.

By default, shows the most recent change to the specified file with nested
objects truncated for readability. Use --full to see complete nested structures.

Examples:
  claudeup events diff --file ~/.claude/settings.json
  claudeup events diff --file ~/.claude/plugins/installed_plugins.json --full`,
	Args: cobra.NoArgs,
	RunE: runEventsDiff,
}

func init() {
	eventsCmd.AddCommand(eventsDiffCmd)

	eventsDiffCmd.Flags().StringVar(&diffFile, "file", "", "File path to show diff for (required)")
	eventsDiffCmd.Flags().BoolVar(&diffFull, "full", false, "Show complete nested objects without truncation")
	eventsDiffCmd.MarkFlagRequired("file")
}

func runEventsDiff(cmd *cobra.Command, args []string) error {
	// Expand and validate file path
	if !filepath.IsAbs(diffFile) {
		absPath, err := filepath.Abs(diffFile)
		if err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}
		diffFile = absPath
	}
	diffFile = filepath.Clean(diffFile)

	// Get event log path
	eventsDir := filepath.Join(config.MustClaudeupHome(), "events")
	logPath := filepath.Join(eventsDir, "operations.log")

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		ui.PrintInfo("No events recorded yet.")
		return nil
	}

	// Create writer to query events
	writer, err := events.NewJSONLWriter(logPath)
	if err != nil {
		return fmt.Errorf("failed to open events log: %w", err)
	}

	// Query for most recent event for this file
	filters := events.EventFilters{
		File:  diffFile,
		Limit: 1,
	}

	eventList, err := writer.Query(filters)
	if err != nil {
		return fmt.Errorf("failed to query events: %w", err)
	}

	if len(eventList) == 0 {
		ui.PrintInfo(fmt.Sprintf("No events found for file: %s", diffFile))
		return nil
	}

	event := eventList[0]

	// Display event header
	ui.PrintSuccess(fmt.Sprintf("Most recent change to: %s", diffFile))
	fmt.Printf("  Operation: %s\n", event.Operation)
	fmt.Printf("  Timestamp: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Scope: %s\n", event.Scope)
	fmt.Printf("  Change Type: %s\n", event.ChangeType)
	fmt.Println()

	// Generate and display diff
	diffResult := events.DiffSnapshots(event.Before, event.After, diffFull)

	if !diffResult.HasChanges {
		ui.PrintInfo("No changes detected in this operation.")
		return nil
	}

	// Display diff summary
	if diffResult.ContentAvailable {
		ui.PrintSuccess("Content diff:")
		fmt.Println(diffResult.Summary)
	} else {
		ui.PrintWarning("Content not available - showing hash-only diff:")
		fmt.Println(diffResult.Summary)

		// Try to read current file content for comparison
		if _, err := os.Stat(diffFile); err == nil {
			ui.PrintInfo("\nNote: File still exists on disk. Content may have changed since this event.")
		}
	}

	// Show error if operation failed
	if event.Error != "" {
		fmt.Println()
		ui.PrintError(fmt.Sprintf("Operation error: %s", event.Error))
	}

	return nil
}
