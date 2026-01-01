// ABOUTME: Tests for centralized path resolution functions
// ABOUTME: Verifies CLAUDEUP_HOME environment variable is respected

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

	t.Run("falls back to ~/.claudeup when not set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "")
		got := MustClaudeupHome()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".claudeup")
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
