// ABOUTME: Profile bootstrap functionality for sandbox containers.
// ABOUTME: Applies profile's Claude configuration on first sandbox run.
package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup/internal/profile"
)

const sentinelFile = ".bootstrapped"

// IsFirstRun returns true if the sandbox state directory has not been bootstrapped.
func IsFirstRun(stateDir string) bool {
	sentinel := filepath.Join(stateDir, sentinelFile)
	_, err := os.Stat(sentinel)
	return os.IsNotExist(err)
}

// WriteSentinel marks the sandbox as bootstrapped.
func WriteSentinel(stateDir string) error {
	sentinel := filepath.Join(stateDir, sentinelFile)
	timestamp := time.Now().Format(time.RFC3339)
	return os.WriteFile(sentinel, []byte(timestamp), 0644)
}

// BootstrapFromProfile writes Claude configuration files to the sandbox state directory.
// This applies the profile's marketplaces, plugins, and settings to the sandbox.
func BootstrapFromProfile(p *profile.Profile, stateDir string) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	// Write marketplaces.json if profile has marketplaces
	if len(p.Marketplaces) > 0 {
		if err := writeMarketplaces(p.Marketplaces, stateDir); err != nil {
			return err
		}
	}

	// Write settings.json with plugins if profile has plugins
	if len(p.Plugins) > 0 {
		if err := writeSettings(p.Plugins, stateDir); err != nil {
			return err
		}
	}

	// Mark as bootstrapped
	return WriteSentinel(stateDir)
}

func writeMarketplaces(marketplaces []profile.Marketplace, stateDir string) error {
	// Convert to Claude's marketplace format
	var data []map[string]interface{}
	for _, m := range marketplaces {
		entry := map[string]interface{}{
			"source": m.Source,
		}
		if m.Repo != "" {
			entry["repo"] = m.Repo
		}
		if m.URL != "" {
			entry["url"] = m.URL
		}
		data = append(data, entry)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(stateDir, "marketplaces.json"), jsonData, 0644)
}

func writeSettings(plugins []string, stateDir string) error {
	settingsPath := filepath.Join(stateDir, "settings.json")

	// Read existing settings to preserve user customizations
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			// If existing file is malformed, start fresh
			settings = make(map[string]interface{})
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Convert plugins slice to map format expected by Claude CLI
	enabledPlugins := make(map[string]bool)
	for _, plugin := range plugins {
		enabledPlugins[plugin] = true
	}
	settings["enabledPlugins"] = enabledPlugins

	jsonData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, jsonData, 0644)
}
