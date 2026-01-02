// ABOUTME: Unit tests for profile bootstrap functionality.
// ABOUTME: Tests first-run detection, config writing, and sentinel management.
package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFirstRun(t *testing.T) {
	t.Run("empty directory is first run", func(t *testing.T) {
		stateDir := t.TempDir()
		if !IsFirstRun(stateDir) {
			t.Error("expected first run for empty directory")
		}
	})

	t.Run("directory with sentinel is not first run", func(t *testing.T) {
		stateDir := t.TempDir()
		sentinel := filepath.Join(stateDir, ".bootstrapped")
		if err := os.WriteFile(sentinel, []byte("2026-01-01"), 0644); err != nil {
			t.Fatal(err)
		}
		if IsFirstRun(stateDir) {
			t.Error("expected not first run when sentinel exists")
		}
	})

	t.Run("directory with other files but no sentinel is first run", func(t *testing.T) {
		stateDir := t.TempDir()
		otherFile := filepath.Join(stateDir, "settings.json")
		if err := os.WriteFile(otherFile, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if !IsFirstRun(stateDir) {
			t.Error("expected first run when no sentinel")
		}
	})
}
