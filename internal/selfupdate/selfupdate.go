// ABOUTME: Self-update functionality for claudeup CLI
// ABOUTME: Downloads and replaces binary from GitHub releases
package selfupdate

import (
	"encoding/json"
	"fmt"
	"net/http"
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
