// ABOUTME: Backup and restore functionality for scope settings
// ABOUTME: Manages ~/.claudeup/backups/ for scope recovery
package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureBackupDir creates the backup directory if it doesn't exist
// Returns the path to the backup directory
func EnsureBackupDir(homeDir string) (string, error) {
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	return backupDir, nil
}

// SaveScopeBackup copies the settings file to the backup directory
// Returns the path to the backup file
func SaveScopeBackup(homeDir, scope, settingsPath string) (string, error) {
	backupDir, err := EnsureBackupDir(homeDir)
	if err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupFileName := fmt.Sprintf("%s-scope.json", scope)
	backupPath := filepath.Join(backupDir, backupFileName)

	// Copy the file
	src, err := os.Open(settingsPath)
	if err != nil {
		return "", fmt.Errorf("failed to open settings file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy settings: %w", err)
	}

	return backupPath, nil
}
