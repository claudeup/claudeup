// ABOUTME: Tests for Uninstall function
// ABOUTME: Verifies item removal from extension storage with config and symlink cleanup
package ext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstall(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Setup: create and enable a rule
	rulesDir := filepath.Join(claudeupHome, "ext", "rules")
	os.MkdirAll(rulesDir, 0755)
	os.WriteFile(filepath.Join(rulesDir, "my-rule.md"), []byte("# Rule"), 0644)
	if _, _, err := manager.Enable("rules", []string{"my-rule.md"}); err != nil {
		t.Fatalf("Setup failed: Enable() error = %v", err)
	}

	// Verify setup
	symlinkPath := filepath.Join(claudeDir, "rules", "my-rule.md")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Fatal("Setup failed: symlink not created")
	}

	// Uninstall
	removed, notFound, err := manager.Uninstall("rules", []string{"my-rule.md"})
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(removed) != 1 || removed[0] != "my-rule.md" {
		t.Errorf("Uninstall() removed = %v, want [my-rule.md]", removed)
	}
	if len(notFound) != 0 {
		t.Errorf("Uninstall() notFound = %v, want []", notFound)
	}

	// File should be gone from extension storage
	if _, err := os.Stat(filepath.Join(rulesDir, "my-rule.md")); !os.IsNotExist(err) {
		t.Error("File was not removed from extension storage")
	}

	// Symlink should be gone
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("Symlink was not removed")
	}

	// Config should not have the entry
	config, _ := manager.LoadConfig()
	if _, exists := config["rules"]["my-rule.md"]; exists {
		t.Error("Config still has entry for removed item")
	}
}

func TestUninstallWildcard(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	rulesDir := filepath.Join(claudeupHome, "ext", "rules")
	os.MkdirAll(rulesDir, 0755)
	os.WriteFile(filepath.Join(rulesDir, "gsd-one.md"), []byte("# One"), 0644)
	os.WriteFile(filepath.Join(rulesDir, "gsd-two.md"), []byte("# Two"), 0644)
	os.WriteFile(filepath.Join(rulesDir, "keep.md"), []byte("# Keep"), 0644)
	if _, _, err := manager.Enable("rules", []string{"gsd-one.md", "gsd-two.md", "keep.md"}); err != nil {
		t.Fatalf("Setup failed: Enable() error = %v", err)
	}

	removed, _, err := manager.Uninstall("rules", []string{"gsd-*"})
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("Uninstall() removed %d items, want 2", len(removed))
	}

	// keep.md should still exist
	if _, err := os.Stat(filepath.Join(rulesDir, "keep.md")); os.IsNotExist(err) {
		t.Error("keep.md was incorrectly removed")
	}
}

func TestUninstallNotFound(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	removed, notFound, err := manager.Uninstall("rules", []string{"nonexistent.md"})
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("Uninstall() removed = %v, want []", removed)
	}
	if len(notFound) != 1 || notFound[0] != "nonexistent.md" {
		t.Errorf("Uninstall() notFound = %v, want [nonexistent.md]", notFound)
	}
}

func TestUninstallDisabledItem(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create item without enabling it
	rulesDir := filepath.Join(claudeupHome, "ext", "rules")
	os.MkdirAll(rulesDir, 0755)
	os.WriteFile(filepath.Join(rulesDir, "disabled-rule.md"), []byte("# Disabled"), 0644)

	removed, _, err := manager.Uninstall("rules", []string{"disabled-rule.md"})
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(removed) != 1 {
		t.Errorf("Uninstall() removed = %v, want [disabled-rule.md]", removed)
	}

	if _, err := os.Stat(filepath.Join(rulesDir, "disabled-rule.md")); !os.IsNotExist(err) {
		t.Error("File was not removed from extension storage")
	}
}

func TestUninstallInvalidCategory(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	_, _, err := manager.Uninstall("invalid", []string{"foo"})
	if err == nil {
		t.Error("Uninstall() expected error for invalid category")
	}
}
