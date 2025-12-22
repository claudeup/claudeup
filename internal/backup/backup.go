// ABOUTME: Backup and restore functionality for scope settings
// ABOUTME: Manages ~/.claudeup/backups/ for scope recovery
package backup

import (
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
