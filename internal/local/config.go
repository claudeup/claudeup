// ABOUTME: Manages enabled.json configuration file for extensions
// ABOUTME: Provides load/save operations for tracking enabled state in CLAUDEUP_HOME
package local

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config maps category -> item -> enabled status
type Config map[string]map[string]bool

// Manager handles local item operations
type Manager struct {
	claudeDir  string
	localDir   string
	configFile string
}

// NewManager creates a new Manager for managing local items.
// claudeDir is where Claude Code reads extensions (e.g., ~/.claude).
// claudeupHome is where claudeup stores its data (e.g., ~/.claudeup).
func NewManager(claudeDir, claudeupHome string) *Manager {
	return &Manager{
		claudeDir:  claudeDir,
		localDir:   filepath.Join(claudeupHome, "ext"),
		configFile: filepath.Join(claudeupHome, "enabled.json"),
	}
}

// LoadConfig reads the enabled.json config file
func (m *Manager) LoadConfig() (Config, error) {
	data, err := os.ReadFile(m.configFile)
	if os.IsNotExist(err) {
		return make(Config), nil
	}
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Ensure all categories have initialized maps
	if config == nil {
		config = make(Config)
	}
	return config, nil
}

// SaveConfig writes the enabled.json config file
func (m *Manager) SaveConfig(config Config) error {
	if err := os.MkdirAll(filepath.Dir(m.configFile), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.configFile, data, 0644)
}
