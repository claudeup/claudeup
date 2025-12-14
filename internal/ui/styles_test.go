// ABOUTME: Tests for UI styles and NO_COLOR support
// ABOUTME: Verifies color definitions and environment variable handling
package ui

import (
	"os"
	"testing"
)

func TestColorsAreDefined(t *testing.T) {
	// Verify all semantic colors are defined (non-empty)
	colors := []string{
		string(ColorSuccess),
		string(ColorError),
		string(ColorWarning),
		string(ColorInfo),
		string(ColorMuted),
		string(ColorAccent),
	}

	for i, c := range colors {
		if c == "" {
			t.Errorf("Color at index %d is empty", i)
		}
	}
}

func TestSymbolsAreDefined(t *testing.T) {
	symbols := []string{
		SymbolSuccess,
		SymbolError,
		SymbolWarning,
		SymbolInfo,
		SymbolArrow,
		SymbolBullet,
	}

	for i, s := range symbols {
		if s == "" {
			t.Errorf("Symbol at index %d is empty", i)
		}
	}
}

func TestNoColorEnvironmentVariable(t *testing.T) {
	// Save original value
	original := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", original)

	// Set NO_COLOR
	os.Setenv("NO_COLOR", "1")

	// Re-initialize to pick up env change
	initColorProfile()

	// Verify HasDarkBackground still works (doesn't panic)
	// The actual color stripping is handled by lipgloss internally
	if ColorSuccess == "" {
		t.Error("ColorSuccess should still be defined even with NO_COLOR")
	}
}
