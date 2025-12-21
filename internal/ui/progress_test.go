// ABOUTME: Tests for progress tracker used during profile apply
// ABOUTME: Validates phase tracking, item recording, and state management
package ui

import (
	"strings"
	"testing"
)

func TestNewProgressTracker(t *testing.T) {
	tracker := NewProgressTracker(TrackerConfig{
		Phases: []string{"Marketplaces", "Plugins", "MCP Servers"},
		Window: 5,
	})

	if tracker == nil {
		t.Fatal("expected tracker to be non-nil")
	}

	if len(tracker.phases) != 3 {
		t.Errorf("expected 3 phases, got %d", len(tracker.phases))
	}

	if tracker.window != 5 {
		t.Errorf("expected window of 5, got %d", tracker.window)
	}
}

func TestPhaseSetTotals(t *testing.T) {
	tracker := NewProgressTracker(TrackerConfig{
		Phases: []string{"Plugins"},
		Window: 5,
	})

	tracker.SetPhaseTotals("Plugins", 10, 15)

	phase := tracker.phases[0]
	if phase.Total != 10 {
		t.Errorf("expected total 10, got %d", phase.Total)
	}
	if phase.Skipped != 5 {
		t.Errorf("expected skipped 5, got %d", phase.Skipped)
	}
}

func TestPhaseUpdate(t *testing.T) {
	tracker := NewProgressTracker(TrackerConfig{
		Phases: []string{"Plugins"},
		Window: 5,
	})
	tracker.SetPhaseTotals("Plugins", 3, 3)

	// Record success
	tracker.RecordResult("Plugins", ItemResult{
		Name:    "plugin-a",
		Success: true,
	})

	phase := tracker.phases[0]
	if phase.Completed != 1 {
		t.Errorf("expected completed 1, got %d", phase.Completed)
	}
	if len(phase.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(phase.Items))
	}
	if phase.Items[0].Name != "plugin-a" {
		t.Errorf("expected plugin-a, got %s", phase.Items[0].Name)
	}
}

func TestSlidingWindow(t *testing.T) {
	tracker := NewProgressTracker(TrackerConfig{
		Phases: []string{"Plugins"},
		Window: 3,
	})
	tracker.SetPhaseTotals("Plugins", 5, 5)

	// Add 5 items, window should only keep last 3
	for i := 0; i < 5; i++ {
		tracker.RecordResult("Plugins", ItemResult{
			Name:    string(rune('a' + i)),
			Success: true,
		})
	}

	phase := tracker.phases[0]
	if len(phase.Items) != 3 {
		t.Errorf("expected window of 3 items, got %d", len(phase.Items))
	}

	// Should have c, d, e (last 3)
	expected := []string{"c", "d", "e"}
	for i, item := range phase.Items {
		if item.Name != expected[i] {
			t.Errorf("expected %s at position %d, got %s", expected[i], i, item.Name)
		}
	}
}

func TestPhaseCompletion(t *testing.T) {
	tracker := NewProgressTracker(TrackerConfig{
		Phases: []string{"Plugins"},
		Window: 5,
	})
	tracker.SetPhaseTotals("Plugins", 2, 2)

	if tracker.phases[0].Done {
		t.Error("phase should not be done initially")
	}

	tracker.RecordResult("Plugins", ItemResult{Name: "a", Success: true})
	if tracker.phases[0].Done {
		t.Error("phase should not be done after 1/2")
	}

	tracker.RecordResult("Plugins", ItemResult{Name: "b", Success: true})
	if !tracker.phases[0].Done {
		t.Error("phase should be done after 2/2")
	}
}

func TestErrorRecording(t *testing.T) {
	tracker := NewProgressTracker(TrackerConfig{
		Phases: []string{"Plugins"},
		Window: 5,
	})
	tracker.SetPhaseTotals("Plugins", 1, 1)

	tracker.RecordResult("Plugins", ItemResult{
		Name:    "broken-plugin",
		Success: false,
		Error:   "not found in marketplace",
	})

	phase := tracker.phases[0]
	if phase.Completed != 1 {
		t.Errorf("failed items should still count as completed, got %d", phase.Completed)
	}
	if phase.Items[0].Success {
		t.Error("item should be marked as failed")
	}
	if phase.Items[0].Error != "not found in marketplace" {
		t.Errorf("error message not preserved: %s", phase.Items[0].Error)
	}
}

func TestRenderProgressBar(t *testing.T) {
	// Test bar rendering at various completion levels
	tests := []struct {
		completed int
		total     int
		wantFull  int // number of filled segments expected
	}{
		{0, 10, 0},
		{5, 10, 10},  // 50% = 10 filled out of 20
		{10, 10, 20}, // 100% = 20 filled
	}

	for _, tt := range tests {
		bar := renderProgressBar(tt.completed, tt.total, 20)
		// Count filled characters (━)
		filled := 0
		for _, r := range bar {
			if r == '━' {
				filled++
			}
		}
		if filled != tt.wantFull {
			t.Errorf("renderProgressBar(%d, %d): got %d filled, want %d",
				tt.completed, tt.total, filled, tt.wantFull)
		}
	}
}

func TestRenderPhaseLineComplete(t *testing.T) {
	phase := &Phase{
		Name:      "Plugins",
		Total:     10,
		Completed: 10,
		Skipped:   5,
		Done:      true,
	}

	line := renderPhaseLine(phase, 20)

	// Should contain phase name, checkmark for done, and skipped count
	if !containsSubstring(line, "Plugins") {
		t.Error("phase line should contain phase name")
	}
	if !containsSubstring(line, "10/10") {
		t.Errorf("phase line should show completion count, got: %s", line)
	}
}

func TestRenderPhaseLineInProgress(t *testing.T) {
	phase := &Phase{
		Name:      "Marketplaces",
		Total:     5,
		Completed: 2,
		Skipped:   0,
		Done:      false,
	}

	line := renderPhaseLine(phase, 20)

	if !containsSubstring(line, "Marketplaces") {
		t.Error("phase line should contain phase name")
	}
	if !containsSubstring(line, "2/5") {
		t.Errorf("phase line should show progress count, got: %s", line)
	}
}

func TestRenderItemSuccess(t *testing.T) {
	item := ItemResult{Name: "my-plugin", Success: true}
	line := renderItemLine(item)

	if !containsSubstring(line, "my-plugin") {
		t.Error("item line should contain item name")
	}
	if !containsSubstring(line, SymbolSuccess) {
		t.Errorf("success item should have checkmark, got: %s", line)
	}
}

func TestRenderItemFailure(t *testing.T) {
	item := ItemResult{Name: "broken", Success: false, Error: "failed"}
	line := renderItemLine(item)

	if !containsSubstring(line, "broken") {
		t.Error("item line should contain item name")
	}
	if !containsSubstring(line, SymbolError) {
		t.Errorf("failed item should have error symbol, got: %s", line)
	}
}

// containsSubstring checks if s contains substr, ignoring ANSI codes
func containsSubstring(s, substr string) bool {
	// Simple check - in real output ANSI codes might be present
	// but the text content should still be findable
	return len(s) > 0 && len(substr) > 0 &&
		(strings.Contains(s, substr) || strings.Contains(stripANSI(s), substr))
}

func stripANSI(s string) string {
	// Simple ANSI stripper for testing
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}
