// ABOUTME: Tests for scope backup and restore functionality
// ABOUTME: Covers saving, loading, and managing scope backups
package backup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestSaveLocalScopeBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock local settings file
	projectDir := filepath.Join(tempDir, "my-project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{"local@test":true}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Save backup with project path
	backupPath, err := SaveLocalScopeBackup(tempDir, projectDir, settingsPath)
	if err != nil {
		t.Fatalf("SaveLocalScopeBackup failed: %v", err)
	}

	// Verify backup exists and has project hash in name
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created: %v", err)
	}

	// Should contain "local-scope-" prefix
	filename := filepath.Base(backupPath)
	if !strings.HasPrefix(filename, "local-scope-") {
		t.Errorf("unexpected filename: %s", filename)
	}

	// Verify content matches
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != `{"enabledPlugins":{"local@test":true}}` {
		t.Errorf("backup content mismatch: %s", content)
	}
}

func TestRestoreScopeBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create backup directory and file
	backupDir := filepath.Join(tempDir, ".claudeup", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(backupDir, "user-scope.json")
	if err := os.WriteFile(backupPath, []byte(`{"enabledPlugins":{"restored@test":true}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create target settings file (will be overwritten)
	claudeDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Restore backup
	err := RestoreScopeBackup(tempDir, "user", settingsPath)
	if err != nil {
		t.Fatalf("RestoreScopeBackup failed: %v", err)
	}

	// Verify content was restored
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != `{"enabledPlugins":{"restored@test":true}}` {
		t.Errorf("restore content mismatch: %s", content)
	}
}

func TestRestoreScopeBackupNotFound(t *testing.T) {
	tempDir := t.TempDir()

	err := RestoreScopeBackup(tempDir, "user", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing backup")
	}
	if !errors.Is(err, ErrNoBackup) {
		t.Errorf("expected ErrNoBackup, got: %v", err)
	}
}
