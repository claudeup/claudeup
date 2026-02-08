// ABOUTME: Shared markdown rendering using glamour
// ABOUTME: Provides terminal-friendly markdown output with auto-styling
package ui

import (
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/term"
)

// RenderMarkdown renders markdown content for terminal display.
// When raw is true, returns content unchanged (for piping).
// Falls back to raw content on rendering errors.
func RenderMarkdown(content string, raw bool) string {
	if raw {
		return content
	}

	width := terminalWidth()

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}

	return rendered
}

func terminalWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		return 80
	}
	return width
}
