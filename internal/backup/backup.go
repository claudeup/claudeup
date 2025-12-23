// ABOUTME: Backup and restore functionality for scope settings
// ABOUTME: Manages ~/.claudeup/backups/ for scope recovery
package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// EnsureBackupDir creates the backup directory if it doesn't exist
// Returns the path to the backup directory
func EnsureBackupDir(homeDir string) (string, error) {
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")
	// Use 0700 for user-only access (backups may contain sensitive settings)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", err
	}
	return backupDir, nil
}

// copyFile copies src to dst, preserving file permissions
func copyFile(src, dst string) error {
	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

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

	// Preserve original file permissions
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
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

// ErrNoBackup is returned when no backup exists for a scope
var ErrNoBackup = errors.New("no backup found")

// RestoreScopeBackup copies the backup file back to the settings location
func RestoreScopeBackup(homeDir, scope, settingsPath string) error {
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")
	backupFileName := fmt.Sprintf("%s-scope.json", scope)
	backupPath := filepath.Join(backupDir, backupFileName)

	// Check backup exists
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNoBackup
		}
		return err
	}

	// Use copyFile helper
	return copyFile(backupPath, settingsPath)
}

// RestoreLocalScopeBackup restores a local scope backup using project-specific naming
func RestoreLocalScopeBackup(homeDir, projectDir, settingsPath string) error {
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")

	// Derive the same hash used when saving
	hash := sha256.Sum256([]byte(projectDir))
	shortHash := hex.EncodeToString(hash[:])[:8]

	backupFileName := fmt.Sprintf("local-scope-%s.json", shortHash)
	backupPath := filepath.Join(backupDir, backupFileName)

	// Check backup exists
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNoBackup
		}
		return err
	}

	// Use copyFile helper
	return copyFile(backupPath, settingsPath)
}

// BackupInfo contains metadata about a backup
type BackupInfo struct {
	Exists      bool
	Path        string
	ModTime     time.Time
	PluginCount int
}

// GetBackupInfo returns information about a scope's backup
func GetBackupInfo(homeDir, scope string) (*BackupInfo, error) {
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")
	backupFileName := fmt.Sprintf("%s-scope.json", scope)
	backupPath := filepath.Join(backupDir, backupFileName)

	info := &BackupInfo{Path: backupPath}

	stat, err := os.Stat(backupPath)
	if os.IsNotExist(err) {
		return info, nil
	}
	if err != nil {
		return nil, err
	}

	info.Exists = true
	info.ModTime = stat.ModTime()

	// Count plugins
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return info, nil // Return partial info
	}

	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(content, &settings); err == nil {
		for _, enabled := range settings.EnabledPlugins {
			if enabled {
				info.PluginCount++
			}
		}
	}

	return info, nil
}

// GetLocalBackupInfo returns information about a local scope's backup using project-specific naming
func GetLocalBackupInfo(homeDir, projectDir string) (*BackupInfo, error) {
	backupDir := filepath.Join(homeDir, ".claudeup", "backups")

	// Derive the same hash used when saving
	hash := sha256.Sum256([]byte(projectDir))
	shortHash := hex.EncodeToString(hash[:])[:8]

	backupFileName := fmt.Sprintf("local-scope-%s.json", shortHash)
	backupPath := filepath.Join(backupDir, backupFileName)

	info := &BackupInfo{Path: backupPath}

	stat, err := os.Stat(backupPath)
	if os.IsNotExist(err) {
		return info, nil
	}
	if err != nil {
		return nil, err
	}

	info.Exists = true
	info.ModTime = stat.ModTime()

	// Count plugins
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return info, nil // Return partial info if file can't be read
	}

	var settings struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(content, &settings); err == nil {
		for _, enabled := range settings.EnabledPlugins {
			if enabled {
				info.PluginCount++
			}
		}
	}

	return info, nil
}
