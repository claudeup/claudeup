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
	"runtime"
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
