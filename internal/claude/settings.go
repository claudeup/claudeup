// ABOUTME: Functions for reading Claude Code settings.json configuration
// ABOUTME: Provides access to enabled plugins and other user settings
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup/internal/events"
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

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"settings update",
		settingsPath,
		"user",
		func() error {
			return os.WriteFile(settingsPath, data, 0644)
		},
	)
}

// SettingsPathForScope returns the settings.json path for a given scope
func SettingsPathForScope(scope string, claudeDir string, projectDir string) (string, error) {
	// Validate scope (allow empty string as alias for "user")
	if scope != "" {
		if err := ValidateScope(scope); err != nil {
			return "", err
		}
	}

	switch scope {
	case "user", "":
		return filepath.Join(claudeDir, "settings.json"), nil
	case "project":
		if projectDir == "" {
			return "", fmt.Errorf("project directory required for project scope")
		}
		// Project scope: ./.claude/settings.json
		return filepath.Join(projectDir, ".claude", "settings.json"), nil
	case "local":
		if projectDir == "" {
			return "", fmt.Errorf("project directory required for local scope")
		}
		// Local scope: ./.claude/settings.local.json (machine-specific, gitignored)
		return filepath.Join(projectDir, ".claude", "settings.local.json"), nil
	default:
		// This should never be reached due to ValidateScope above
		return "", fmt.Errorf("invalid scope: %s", scope)
	}
}

// LoadSettingsForScope reads settings from a specific scope
func LoadSettingsForScope(scope string, claudeDir string, projectDir string) (*Settings, error) {
	path, err := SettingsPathForScope(scope, claudeDir, projectDir)
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty settings (not an error)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Settings{
			EnabledPlugins: make(map[string]bool),
			raw:            make(map[string]interface{}),
		}, nil
	}

	// Read and parse
	data, err := os.ReadFile(path)
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

	return settings, nil
}

// SaveSettingsForScope writes settings to a specific scope
func SaveSettingsForScope(scope string, claudeDir string, projectDir string, settings *Settings) error {
	path, err := SettingsPathForScope(scope, claudeDir, projectDir)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

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

	// Normalize empty scope to "user"
	normalizedScope := scope
	if normalizedScope == "" {
		normalizedScope = "user"
	}

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"settings update",
		path,
		normalizedScope,
		func() error {
			return os.WriteFile(path, data, 0644)
		},
	)
}

// LoadMergedSettings loads settings from all scopes and merges them
// Precedence: local > project > user (later scopes override earlier ones)
func LoadMergedSettings(claudeDir string, projectDir string) (*Settings, error) {
	merged := &Settings{
		EnabledPlugins: make(map[string]bool),
		raw:            make(map[string]interface{}),
	}

	// Load settings in precedence order (lowest to highest priority)
	// ValidScopes is ordered: [user, project, local]
	// This means local settings override project, which override user
	for _, scope := range ValidScopes {
		settings, err := LoadSettingsForScope(scope, claudeDir, projectDir)
		if err != nil {
			// Only return error for user scope (required), others are optional
			if scope == ScopeUser {
				return nil, err
			}
			continue
		}

		// Merge enabled plugins - later scopes override earlier ones
		// This implements the precedence: local > project > user
		for plugin, enabled := range settings.EnabledPlugins {
			merged.EnabledPlugins[plugin] = enabled
		}
	}

	return merged, nil
}
