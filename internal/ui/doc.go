// ABOUTME: Package documentation for the ui package
// ABOUTME: Describes the purpose and usage patterns for terminal styling

// Package ui provides consistent terminal styling and output formatting
// for claudeup CLI commands using lipgloss.
//
// Usage:
//   - Use Print* functions for standalone messages: ui.PrintSuccess("Done!")
//   - Use inline helpers for composing output: fmt.Println(ui.Bold("Title:"), ui.Muted(detail))
//   - Respects NO_COLOR environment variable for accessibility
package ui
