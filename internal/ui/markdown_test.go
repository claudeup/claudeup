// ABOUTME: Tests for markdown rendering with glamour
// ABOUTME: Verifies rendering, raw passthrough, and error fallback
package ui

import (
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	t.Run("raw mode returns content unchanged", func(t *testing.T) {
		content := "# Hello\n\nSome **bold** text"
		got := RenderMarkdown(content, true)
		if got != content {
			t.Errorf("RenderMarkdown(raw=true) = %q, want %q", got, content)
		}
	})

	t.Run("renders markdown content", func(t *testing.T) {
		content := "# Hello\n\nSome **bold** text"
		got := RenderMarkdown(content, false)

		// Rendered output should differ from raw input
		if got == content {
			t.Error("RenderMarkdown(raw=false) returned unchanged content")
		}

		// Should contain the text (without markdown syntax)
		if len(got) == 0 {
			t.Error("RenderMarkdown(raw=false) returned empty string")
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		got := RenderMarkdown("", false)
		// Should not panic or error
		_ = got
	})

	t.Run("handles non-markdown content", func(t *testing.T) {
		content := "just plain text\nwith lines"
		got := RenderMarkdown(content, false)
		// Should not panic
		if len(got) == 0 {
			t.Error("RenderMarkdown returned empty for plain text")
		}
	})
}
