// ABOUTME: Tests for interactive prompt UI functions
// ABOUTME: Tests non-interactive paths (--yes flag, empty inputs)
package ui

import (
	"testing"

	"github.com/claudeup/claudeup/internal/config"
)

func TestSelectFromList_WithYesFlag(t *testing.T) {
	// Save and restore original flag value
	originalFlag := config.YesFlag
	defer func() { config.YesFlag = originalFlag }()

	config.YesFlag = true
	items := []string{"item1", "item2", "item3"}

	selected, err := SelectFromList("Select items:", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(selected) != len(items) {
		t.Errorf("expected %d items, got %d", len(items), len(selected))
	}

	for i, item := range items {
		if selected[i] != item {
			t.Errorf("expected item %d to be %q, got %q", i, item, selected[i])
		}
	}
}

func TestSelectFromList_EmptyItems(t *testing.T) {
	// Save and restore original flag value
	originalFlag := config.YesFlag
	defer func() { config.YesFlag = originalFlag }()

	config.YesFlag = false
	items := []string{}

	selected, err := SelectFromList("Select items:", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(selected) != 0 {
		t.Errorf("expected 0 items, got %d", len(selected))
	}
}

func TestSelectFromList_YesFlagWithEmptyItems(t *testing.T) {
	// Save and restore original flag value
	originalFlag := config.YesFlag
	defer func() { config.YesFlag = originalFlag }()

	config.YesFlag = true
	items := []string{}

	selected, err := SelectFromList("Select items:", items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(selected) != 0 {
		t.Errorf("expected 0 items, got %d", len(selected))
	}
}

func TestErrUserCancelled(t *testing.T) {
	// Verify the error is defined and has expected message
	if ErrUserCancelled == nil {
		t.Fatal("ErrUserCancelled should not be nil")
	}

	expected := "cancelled by user"
	if ErrUserCancelled.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, ErrUserCancelled.Error())
	}
}

func TestConfirmYesNo_WithYesFlag(t *testing.T) {
	// Save and restore original flag value
	originalFlag := config.YesFlag
	defer func() { config.YesFlag = originalFlag }()

	config.YesFlag = true

	confirmed, err := ConfirmYesNo("Proceed?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !confirmed {
		t.Error("expected confirmed to be true when YesFlag is set")
	}
}
