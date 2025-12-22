// ABOUTME: Backup and restore functionality for scope settings
// ABOUTME: Manages ~/.claudeup/backups/ for scope recovery
package backup

import (
	"crypto/sha256"
	"encoding/hex"
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

// copyFile copies src to dst, creating dst if needed
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
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

	if err := copyFile(settingsPath, backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}

// SaveLocalScopeBackup saves a backup with project-specific naming
func SaveLocalScopeBackup(homeDir, projectDir, settingsPath string) (string, error) {
	backupDir, err := EnsureBackupDir(homeDir)
	if err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create a short hash of the project path for unique naming
	hash := sha256.Sum256([]byte(projectDir))
	// Use first 8 hex characters of SHA256 (~16.7M combinations, sufficient for project disambiguation)
	shortHash := hex.EncodeToString(hash[:])[:8]

	backupFileName := fmt.Sprintf("local-scope-%s.json", shortHash)
	backupPath := filepath.Join(backupDir, backupFileName)

	if err := copyFile(settingsPath, backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}
