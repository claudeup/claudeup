// ABOUTME: Tests for centralized path resolution functions
// ABOUTME: Verifies CLAUDEUP_HOME and CLAUDE_CONFIG_DIR environment variables are respected

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMustClaudeupHome(t *testing.T) {
	t.Run("uses CLAUDEUP_HOME when set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "/custom/path")
		got := MustClaudeupHome()
		if got != "/custom/path" {
			t.Errorf("got %q, want /custom/path", got)
		}
	})

	t.Run("trims whitespace from CLAUDEUP_HOME", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "  /custom/path  ")
		got := MustClaudeupHome()
		if got != "/custom/path" {
			t.Errorf("got %q, want /custom/path", got)
		}
	})

	t.Run("falls back to ~/.claudeup when not set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "")
		got := MustClaudeupHome()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".claudeup")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("panics on whitespace-only CLAUDEUP_HOME", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "   ")
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for whitespace-only CLAUDEUP_HOME")
			}
		}()
		MustClaudeupHome()
	})

	t.Run("panics on relative path CLAUDEUP_HOME", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "relative/path")
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for relative path CLAUDEUP_HOME")
			}
		}()
		MustClaudeupHome()
	})
}

func TestMustClaudeDir(t *testing.T) {
	t.Run("uses CLAUDE_CONFIG_DIR when set", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "/custom/claude/path")
		got := MustClaudeDir()
		if got != "/custom/claude/path" {
			t.Errorf("got %q, want /custom/claude/path", got)
		}
	})

	t.Run("falls back to ~/.claude when not set", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")
		got := MustClaudeDir()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".claude")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestClaudeDirDisplay(t *testing.T) {
	t.Run("shows CLAUDE_CONFIG_DIR when set", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "/custom/claude/path")
		got := ClaudeDirDisplay()
		if got != "/custom/claude/path" {
			t.Errorf("got %q, want /custom/claude/path", got)
		}
	})

	t.Run("shows ~/.claude when not set", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")
		got := ClaudeDirDisplay()
		if got != "~/.claude" {
			t.Errorf("got %q, want ~/.claude", got)
		}
	})
}

func TestClaudeupHomeDisplay(t *testing.T) {
	t.Run("shows CLAUDEUP_HOME when set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "/custom/claudeup")
		got := ClaudeupHomeDisplay()
		if got != "/custom/claudeup" {
			t.Errorf("got %q, want /custom/claudeup", got)
		}
	})

	t.Run("shows ~/.claudeup when not set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "")
		got := ClaudeupHomeDisplay()
		if got != "~/.claudeup" {
			t.Errorf("got %q, want ~/.claudeup", got)
		}
	})
}
