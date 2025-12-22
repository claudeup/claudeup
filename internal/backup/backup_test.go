// ABOUTME: Tests for scope backup and restore functionality
// ABOUTME: Covers saving, loading, and managing scope backups
package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBackupDir(t *testing.T) {
	tempDir := t.TempDir()

	backupDir, err := EnsureBackupDir(tempDir)
	if err != nil {
		t.Fatalf("EnsureBackupDir failed: %v", err)
	}

	expected := filepath.Join(tempDir, ".claudeup", "backups")
	if backupDir != expected {
		t.Errorf("got %s, want %s", backupDir, expected)
	}

	info, err := os.Stat(backupDir)
	if err != nil {
		t.Fatalf("backup dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("backup path is not a directory")
	}
}
