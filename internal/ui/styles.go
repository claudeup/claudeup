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
	ColorAccent  = lipgloss.Color("#8b5cf6") // Purple
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
	initColorProfile()
}

func initColorProfile() {
	// Respect NO_COLOR standard (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}
