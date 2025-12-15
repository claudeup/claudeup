// ABOUTME: Functions for reading Claude Code settings.json configuration
// ABOUTME: Provides access to enabled plugins and other user settings
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings represents the Claude Code settings.json file structure
type Settings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// LoadSettings reads the settings.json file from the Claude directory
func LoadSettings(claudeDir string) (*Settings, error) {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// IsPluginEnabled checks if a plugin is enabled in the settings
func (s *Settings) IsPluginEnabled(pluginName string) bool {
	enabled, exists := s.EnabledPlugins[pluginName]
	return exists && enabled
}
