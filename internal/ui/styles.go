// ABOUTME: Defines color palette, symbols, and NO_COLOR initialization
// ABOUTME: Centralizes all UI styling constants for consistent appearance
package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Semantic color definitions
var (
	ColorSuccess = lipgloss.Color("#22c55e") // Green
	ColorError   = lipgloss.Color("#ef4444") // Red
	ColorWarning = lipgloss.Color("#eab308") // Yellow
	ColorInfo    = lipgloss.Color("#06b6d4") // Cyan
	ColorMuted   = lipgloss.Color("#6b7280") // Gray
	ColorAccent  = lipgloss.Color("#096becff") // Blue
	ColorFlags  = lipgloss.Color("#ffffff") // White
)

// Symbol definitions
var (
	SymbolSuccess = "✓"
	SymbolError   = "✗"
	SymbolWarning = "⚠"
	SymbolInfo    = "ℹ"
	SymbolArrow   = "→"
	SymbolBullet  = "•"
)

func init() {
	InitColorProfile()
}

// InitColorProfile configures lipgloss color output based on environment.
// Disables ANSI colors when NO_COLOR is set (https://no-color.org/) or TERM=dumb.
// Called automatically at init time. Tests can call this after setting either
// variable to force plain-text output regardless of terminal capabilities.
func InitColorProfile() {
	// Respect NO_COLOR standard (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}
