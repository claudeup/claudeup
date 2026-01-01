// ABOUTME: Self-update functionality for claudeup CLI
// ABOUTME: Downloads and replaces binary from GitHub releases
package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultAPIURL   = "https://api.github.com/repos/claudeup/claudeup/releases/latest"
	DefaultAssetURL = "https://github.com/claudeup/claudeup/releases/download"
)

// CheckLatestVersion queries the GitHub API for the latest release version
func CheckLatestVersion(apiURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %w", err)
	}

	return release.TagName, nil
}

// DownloadBinary downloads the binary for the current platform to a temp file
func DownloadBinary(url, tempDir string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp(tempDir, "claudeup-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to set permissions: %w", err)
	}

	return tempFile.Name(), nil
}

// VerifyChecksum compares the SHA256 hash of a file against expected
func VerifyChecksum(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// GetBinaryURL returns the download URL for the current platform
func GetBinaryURL(version string) string {
	return fmt.Sprintf("%s/%s/claudeup-%s-%s", DefaultAssetURL, version, runtime.GOOS, runtime.GOARCH)
}

// GetChecksumsURL returns the checksums file URL for a version
func GetChecksumsURL(version string) string {
	return fmt.Sprintf("%s/%s/checksums.txt", DefaultAssetURL, version)
}

// ReplaceBinary atomically replaces the current binary with a new one.
// On failure, attempts to rollback to the original.
func ReplaceBinary(currentPath, newPath string) error {
	backupPath := currentPath + ".old"

	// Step 1: Rename current to backup
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Step 2: Move new binary to current location
	if err := os.Rename(newPath, currentPath); err != nil {
		// Rollback: restore backup
		if rbErr := os.Rename(backupPath, currentPath); rbErr != nil {
			return fmt.Errorf("failed to install new binary (%v) and rollback failed (%v)", err, rbErr)
		}
		return fmt.Errorf("failed to install new binary (rolled back): %w", err)
	}

	// Step 3: Remove backup
	os.Remove(backupPath) // Ignore error - not critical

	return nil
}

// UpdateResult contains the outcome of an update attempt
type UpdateResult struct {
	AlreadyUpToDate bool
	OldVersion      string
	NewVersion      string
	Error           error
}

// Update checks for and applies updates to the claudeup binary
func Update(currentVersion, latestVersion, binaryPath string) UpdateResult {
	result := UpdateResult{
		OldVersion: currentVersion,
		NewVersion: latestVersion,
	}

	// Check if update needed
	if !IsNewer(currentVersion, latestVersion) {
		result.AlreadyUpToDate = true
		return result
	}

	// If no binary path provided, detect it
	if binaryPath == "" {
		var err error
		binaryPath, err = os.Executable()
		if err != nil {
			result.Error = fmt.Errorf("failed to detect binary path: %w", err)
			return result
		}
		binaryPath, err = filepath.EvalSymlinks(binaryPath)
		if err != nil {
			result.Error = fmt.Errorf("failed to resolve binary path: %w", err)
			return result
		}
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "claudeup-update-*")
	if err != nil {
		result.Error = fmt.Errorf("failed to create temp directory: %w", err)
		return result
	}
	defer os.RemoveAll(tempDir)

	// Download new binary
	binaryURL := GetBinaryURL(latestVersion)
	newBinaryPath, err := DownloadBinary(binaryURL, tempDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to download binary: %w", err)
		return result
	}

	// Download and parse checksums
	checksumsURL := GetChecksumsURL(latestVersion)
	expectedHash, err := fetchExpectedChecksum(checksumsURL, latestVersion)
	if err != nil {
		result.Error = fmt.Errorf("failed to get checksum: %w", err)
		return result
	}

	// Verify checksum
	if err := VerifyChecksum(newBinaryPath, expectedHash); err != nil {
		result.Error = err
		return result
	}

	// Replace binary
	if err := ReplaceBinary(binaryPath, newBinaryPath); err != nil {
		result.Error = err
		return result
	}

	return result
}

// fetchExpectedChecksum downloads checksums.txt and extracts the hash for current platform
func fetchExpectedChecksum(checksumsURL, version string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(checksumsURL)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums download failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}

	// Parse checksums.txt format: "hash  filename"
	binaryName := fmt.Sprintf("claudeup-%s-%s", runtime.GOOS, runtime.GOARCH)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasSuffix(line, binaryName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				return parts[0], nil
			}
		}
	}

	return "", fmt.Errorf("checksum not found for %s", binaryName)
}
