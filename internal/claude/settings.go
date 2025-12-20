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
	raw            map[string]interface{} // Preserves all fields from settings.json
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

	// Unmarshal into raw map first to preserve all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Extract enabledPlugins with type safety
	settings := &Settings{
		raw:            raw,
		EnabledPlugins: make(map[string]bool),
	}

	if enabledPlugins, ok := raw["enabledPlugins"].(map[string]interface{}); ok {
		for key, val := range enabledPlugins {
			if enabled, ok := val.(bool); ok {
				settings.EnabledPlugins[key] = enabled
			}
		}
	}

	// Validate settings structure
	if err := validateSettings(settings); err != nil {
		return nil, err
	}

	return settings, nil
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

	// Initialize raw map if not present
	if settings.raw == nil {
		settings.raw = make(map[string]interface{})
	}

	// Update enabledPlugins in raw map
	settings.raw["enabledPlugins"] = settings.EnabledPlugins

	// Marshal the full raw map to preserve all fields
	data, err := json.MarshalIndent(settings.raw, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}
