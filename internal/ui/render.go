// ABOUTME: Complex rendering functions for headers, sections, and detail views
// ABOUTME: Provides consistent formatting for structured CLI output
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

const (
	// HeaderMinWidth is the minimum width for header boxes
	HeaderMinWidth = 42
	// HeaderMaxWidth is the maximum width for header boxes (for readability)
	HeaderMaxWidth = 80
)

var (
	headerBaseStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 2).
			Align(lipgloss.Center)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorInfo)

	labelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	valueStyle = lipgloss.NewStyle()
)

// RenderHeader returns a styled header box with the given title
// Width adapts to terminal size, capped between HeaderMinWidth and HeaderMaxWidth
func RenderHeader(title string) string {
	width := getHeaderWidth()
	return headerBaseStyle.Width(width).Render(title)
}

// getHeaderWidth returns appropriate header width based on terminal size
func getHeaderWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		return HeaderMinWidth
	}
	// Clamp between min and max
	if width < HeaderMinWidth {
		return HeaderMinWidth
	}
	if width > HeaderMaxWidth {
		return HeaderMaxWidth
	}
	return width
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
