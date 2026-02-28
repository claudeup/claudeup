// ABOUTME: Unit tests for doctor path-repair logic
// ABOUTME: Verifies getExpectedPath handles all known marketplace path patterns
package commands

import (
	"testing"
)

func TestGetExpectedPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "claude-code-plugins adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/claude-code-plugins/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/claude-code-plugins/plugins/my-plugin",
		},
		{
			name:     "claude-plugins-official adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/claude-plugins-official/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/claude-plugins-official/plugins/my-plugin",
		},
		{
			name:     "claude-code-templates adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/claude-code-templates/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/claude-code-templates/plugins/my-plugin",
		},
		{
			name:     "every-marketplace adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/every-marketplace/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/every-marketplace/plugins/my-plugin",
		},
		{
			name:     "awesome-claude-code-plugins adds plugins subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/awesome-claude-code-plugins/my-plugin",
			expected: "/home/user/.claude/plugins/marketplaces/awesome-claude-code-plugins/plugins/my-plugin",
		},
		{
			name:     "anthropic-agent-skills adds skills subdirectory",
			input:    "/home/user/.claude/plugins/marketplaces/anthropic-agent-skills/my-skill",
			expected: "/home/user/.claude/plugins/marketplaces/anthropic-agent-skills/skills/my-skill",
		},
		{
			name:     "unknown marketplace returns empty",
			input:    "/home/user/.claude/plugins/marketplaces/unknown-marketplace/my-plugin",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExpectedPath(tt.input)
			if got != tt.expected {
				t.Errorf("getExpectedPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
