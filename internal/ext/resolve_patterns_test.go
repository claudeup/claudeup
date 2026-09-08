// ABOUTME: Tests for side-effect-free pattern resolution
// ABOUTME: Verifies Resolve mirrors Enable matching without touching disk
package ext

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveReportsMatchesAndMissing(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	extDir := filepath.Join(claudeupHome, "ext")
	if err := os.MkdirAll(filepath.Join(extDir, "rules"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "rules", "present.md"), []byte("# rule"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(extDir, "commands", "gsd"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "commands", "gsd", "plan.md"), []byte("# cmd"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Extension inference ("present" -> "present.md") and missing patterns.
	matched, notFound, err := manager.Resolve("rules", []string{"present", "missing.md"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(matched, []string{"present.md"}) {
		t.Errorf("Resolve() matched = %v, want [present.md]", matched)
	}
	if !reflect.DeepEqual(notFound, []string{"missing.md"}) {
		t.Errorf("Resolve() notFound = %v, want [missing.md]", notFound)
	}

	// Directory expansion, same as Enable.
	matched, notFound, err = manager.Resolve("commands", []string{"gsd"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(matched, []string{"gsd/plan.md"}) {
		t.Errorf("Resolve() matched = %v, want [gsd/plan.md]", matched)
	}
	if len(notFound) != 0 {
		t.Errorf("Resolve() notFound = %v, want []", notFound)
	}

	// Nothing was enabled: no symlinks in the active dir and no enabled.json.
	if _, err := os.Lstat(filepath.Join(claudeDir, "rules", "present.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Resolve() created a symlink in claudeDir; Lstat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(claudeupHome, "enabled.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Resolve() wrote enabled.json; Stat error = %v", err)
	}
}

func TestResolveEmptyStorage(t *testing.T) {
	manager := NewManager(t.TempDir(), t.TempDir())

	matched, notFound, err := manager.Resolve("agents", []string{"reviewer", "*"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("Resolve() matched = %v, want []", matched)
	}
	if !reflect.DeepEqual(notFound, []string{"reviewer", "*"}) {
		t.Errorf("Resolve() notFound = %v, want [reviewer *]", notFound)
	}
}

func TestResolveInvalidCategory(t *testing.T) {
	manager := NewManager(t.TempDir(), t.TempDir())

	if _, _, err := manager.Resolve("bogus", []string{"x"}); err == nil {
		t.Error("Resolve() with invalid category expected error, got nil")
	}
}
