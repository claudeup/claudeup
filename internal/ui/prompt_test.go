// ABOUTME: Tests for interactive prompt UI functions
// ABOUTME: Tests non-interactive paths (--yes flag, empty inputs)
package ui

import (
	"testing"
)

func TestConfirmYesNo_WithYesFlag(t *testing.T) {
	withYesFlag(t, true, func() {
		confirmed, err := ConfirmYesNo("Proceed?")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !confirmed {
			t.Error("expected confirmed to be true when YesFlag is set")
		}
	})
}

func TestValidateTypedConfirmation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		want     bool
	}{
		{"yes", "yes", true},
		{"YES", "yes", true},
		{"Yes", "yes", true},
		{"y", "yes", false},
		{"no", "yes", false},
		{"", "yes", false},
		{"  yes  ", "yes", true}, // handles whitespace
	}

	for _, tt := range tests {
		got := ValidateTypedConfirmation(tt.input, tt.expected)
		if got != tt.want {
			t.Errorf("ValidateTypedConfirmation(%q, %q) = %v, want %v",
				tt.input, tt.expected, got, tt.want)
		}
	}
}
