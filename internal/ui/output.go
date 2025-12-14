// ABOUTME: Print helper functions for consistent CLI output
// ABOUTME: Provides success, error, warning, info, and muted output styles
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	errorStyle   = lipgloss.NewStyle().Foreground(ColorError)
	warningStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	infoStyle    = lipgloss.NewStyle().Foreground(ColorInfo)
	mutedStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

// PrintSuccess prints a success message with checkmark symbol
func PrintSuccess(msg string) {
	fmt.Println(successStyle.Render(SymbolSuccess + " " + msg))
}

// PrintError prints an error message with X symbol
func PrintError(msg string) {
	fmt.Println(errorStyle.Render(SymbolError + " " + msg))
}

// PrintWarning prints a warning message with warning symbol
func PrintWarning(msg string) {
	fmt.Println(warningStyle.Render(SymbolWarning + " " + msg))
}

// PrintInfo prints an info message with info symbol
func PrintInfo(msg string) {
	fmt.Println(infoStyle.Render(SymbolInfo + " " + msg))
}

// PrintMuted prints a muted/secondary message
func PrintMuted(msg string) {
	fmt.Println(mutedStyle.Render(msg))
}

// Muted returns a string styled as muted (for inline use)
func Muted(s string) string {
	return mutedStyle.Render(s)
}

// Bold returns a string styled as bold (for inline use)
func Bold(s string) string {
	return boldStyle.Render(s)
}

// Success returns a string styled as success (for inline use)
func Success(s string) string {
	return successStyle.Render(s)
}

// Error returns a string styled as error (for inline use)
func Error(s string) string {
	return errorStyle.Render(s)
}

// Warning returns a string styled as warning (for inline use)
func Warning(s string) string {
	return warningStyle.Render(s)
}

// Info returns a string styled as info (for inline use)
func Info(s string) string {
	return infoStyle.Render(s)
}
