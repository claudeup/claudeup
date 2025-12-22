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

func TestSaveUserScopeBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock settings file
	claudeDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{"test@example":true}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Save backup
	backupPath, err := SaveScopeBackup(tempDir, "user", settingsPath)
	if err != nil {
		t.Fatalf("SaveScopeBackup failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created: %v", err)
	}

	// Verify content matches
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != `{"enabledPlugins":{"test@example":true}}` {
		t.Errorf("backup content mismatch: %s", content)
	}
}
