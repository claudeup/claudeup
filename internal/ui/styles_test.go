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

	// Verify colors are still defined even with NO_COLOR
	if ColorSuccess == "" {
		t.Error("ColorSuccess should still be defined even with NO_COLOR")
	}
}

func TestNoColorStripsANSICodes(t *testing.T) {
	// Save original value
	original := os.Getenv("NO_COLOR")
	defer func() {
		if original == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", original)
		}
		initColorProfile() // Restore original color profile
	}()

	// Set NO_COLOR
	os.Setenv("NO_COLOR", "1")
	initColorProfile()

	// Render styled text - should NOT contain ANSI escape codes
	result := Success("test message")

	// ANSI escape codes start with ESC (0x1b or \033)
	if containsANSI(result) {
		t.Errorf("With NO_COLOR=1, Success() should not contain ANSI codes, got: %q", result)
	}

	// But should still contain the actual text
	if result != "test message" {
		t.Errorf("With NO_COLOR=1, Success() should return plain text, got: %q", result)
	}
}

func TestTermDumbStripsANSICodes(t *testing.T) {
	// Save original values
	originalNoColor := os.Getenv("NO_COLOR")
	originalTerm := os.Getenv("TERM")
	defer func() {
		if originalNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalNoColor)
		}
		if originalTerm == "" {
			os.Unsetenv("TERM")
		} else {
			os.Setenv("TERM", originalTerm)
		}
		initColorProfile()
	}()

	// Clear NO_COLOR, set TERM=dumb
	os.Unsetenv("NO_COLOR")
	os.Setenv("TERM", "dumb")
	initColorProfile()

	result := Error("error text")

	if containsANSI(result) {
		t.Errorf("With TERM=dumb, Error() should not contain ANSI codes, got: %q", result)
	}
}

// containsANSI checks if a string contains ANSI escape sequences
func containsANSI(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b { // ESC character
			return true
		}
	}
	return false
}
