// ABOUTME: Complex rendering functions for headers, sections, and detail views
// ABOUTME: Provides consistent formatting for structured CLI output
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// HeaderWidth is the fixed width for header boxes
	HeaderWidth = 42
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 2).
			Width(HeaderWidth).
			Align(lipgloss.Center)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorInfo)

	labelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	valueStyle = lipgloss.NewStyle()
)

// RenderHeader returns a styled header box with the given title
func RenderHeader(title string) string {
	return headerStyle.Render(title)
}

// RenderSection returns a styled section header with optional count
// Pass -1 for count to omit the count display
func RenderSection(title string, count int) string {
	if count >= 0 {
		return sectionStyle.Render(fmt.Sprintf("%s (%d)", title, count))
	}
	return sectionStyle.Render(title)
}

// RenderDetail returns a label: value pair with consistent formatting
func RenderDetail(label, value string) string {
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}

// Indent returns the string with the specified indentation level (2 spaces per level)
func Indent(s string, level int) string {
	prefix := strings.Repeat("  ", level)
	return prefix + s
}
