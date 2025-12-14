// ABOUTME: Tests for complex UI rendering functions
// ABOUTME: Verifies headers, sections, and detail views render correctly
package ui

import (
	"strings"
	"testing"
)

func TestRenderHeader(t *testing.T) {
	result := RenderHeader("Test Title")

	if result == "" {
		t.Error("RenderHeader should return non-empty string")
	}
	if !strings.Contains(result, "Test Title") {
		t.Errorf("RenderHeader should contain title, got: %s", result)
	}
}

func TestRenderSectionWithCount(t *testing.T) {
	result := RenderSection("Plugins", 5)

	if !strings.Contains(result, "Plugins") {
		t.Errorf("RenderSection should contain title, got: %s", result)
	}
	if !strings.Contains(result, "5") {
		t.Errorf("RenderSection should contain count, got: %s", result)
	}
}

func TestRenderSectionWithoutCount(t *testing.T) {
	result := RenderSection("Settings", -1)

	if !strings.Contains(result, "Settings") {
		t.Errorf("RenderSection should contain title, got: %s", result)
	}
	// Should not have parentheses when count is -1
	if strings.Contains(result, "(") {
		t.Errorf("RenderSection should not have count when -1, got: %s", result)
	}
}

func TestRenderDetail(t *testing.T) {
	result := RenderDetail("Version", "1.0.0")

	if !strings.Contains(result, "Version") {
		t.Errorf("RenderDetail should contain label, got: %s", result)
	}
	if !strings.Contains(result, "1.0.0") {
		t.Errorf("RenderDetail should contain value, got: %s", result)
	}
}

func TestIndent(t *testing.T) {
	result := Indent("hello", 2)

	if !strings.HasPrefix(result, "    ") {
		t.Errorf("Indent(2) should add 4 spaces, got: %q", result)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("Indent should preserve content, got: %s", result)
	}
}

func TestIndentZeroLevel(t *testing.T) {
	result := Indent("hello", 0)

	if result != "hello" {
		t.Errorf("Indent(0) should return unchanged string, got: %q", result)
	}
}

func TestRenderHeaderLongTitle(t *testing.T) {
	// Test with a title longer than the fixed width of 42
	longTitle := "This is a very long title that exceeds the header width"
	result := RenderHeader(longTitle)

	if result == "" {
		t.Error("RenderHeader should handle long titles")
	}
	// Long titles should still be present (lipgloss handles truncation/wrapping)
	if !strings.Contains(result, "This is a very long title") {
		t.Errorf("RenderHeader should contain the title text, got: %s", result)
	}
}

func TestRenderHeaderEmptyTitle(t *testing.T) {
	result := RenderHeader("")

	// Should still render the header box even with empty title
	if result == "" {
		t.Error("RenderHeader should render even with empty title")
	}
}

func TestRenderSectionZeroCount(t *testing.T) {
	result := RenderSection("Empty", 0)

	if !strings.Contains(result, "Empty") {
		t.Errorf("RenderSection should contain title, got: %s", result)
	}
	if !strings.Contains(result, "(0)") {
		t.Errorf("RenderSection should show zero count, got: %s", result)
	}
}

func TestRenderDetailEmptyValue(t *testing.T) {
	result := RenderDetail("Label", "")

	if !strings.Contains(result, "Label") {
		t.Errorf("RenderDetail should contain label even with empty value, got: %s", result)
	}
}

func TestGetHeaderWidthReturnsWithinBounds(t *testing.T) {
	width := getHeaderWidth()

	if width < HeaderMinWidth {
		t.Errorf("getHeaderWidth should return at least %d, got: %d", HeaderMinWidth, width)
	}
	if width > HeaderMaxWidth {
		t.Errorf("getHeaderWidth should return at most %d, got: %d", HeaderMaxWidth, width)
	}
}
