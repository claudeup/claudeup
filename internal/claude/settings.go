// ABOUTME: Functions for reading Claude Code settings.json configuration
// ABOUTME: Provides access to enabled plugins and other user settings
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Settings represents the Claude Code settings.json file structure
type Settings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// LoadSettings reads the settings.json file from the Claude directory
func LoadSettings(claudeDir string) (*Settings, error) {
	// Check if Claude directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Claude CLI not found (directory %s does not exist)", claudeDir)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		// Claude installed but settings missing - suspicious
		return nil, &PathNotFoundError{
			Component:    "settings",
			ExpectedPath: settingsPath,
			ClaudeDir:    claudeDir,
		}
	}
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	// Validate settings structure
	if err := validateSettings(&settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// IsPluginEnabled checks if a plugin is enabled in the settings
func (s *Settings) IsPluginEnabled(pluginName string) bool {
	enabled, exists := s.EnabledPlugins[pluginName]
	return exists && enabled
}

// EnablePlugin enables a plugin in the settings
func (s *Settings) EnablePlugin(pluginName string) {
	if s.EnabledPlugins == nil {
		s.EnabledPlugins = make(map[string]bool)
	}
	s.EnabledPlugins[pluginName] = true
}

// DisablePlugin disables a plugin in the settings
func (s *Settings) DisablePlugin(pluginName string) {
	if s.EnabledPlugins == nil {
		return
	}
	s.EnabledPlugins[pluginName] = false
}

// RemovePlugin removes a plugin from the settings entirely
func (s *Settings) RemovePlugin(pluginName string) {
	if s.EnabledPlugins == nil {
		return
	}
	delete(s.EnabledPlugins, pluginName)
}

// SaveSettings writes the settings back to settings.json
func SaveSettings(claudeDir string, settings *Settings) error {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}
