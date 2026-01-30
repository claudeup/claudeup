// ABOUTME: Manages enabled.json configuration file for local items
// ABOUTME: Provides load/save operations for tracking enabled state
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
	libraryDir string
	configFile string
}

// NewManager creates a new Manager for the given Claude directory
func NewManager(claudeDir string) *Manager {
	return &Manager{
		claudeDir:  claudeDir,
		libraryDir: filepath.Join(claudeDir, ".library"),
		configFile: filepath.Join(claudeDir, "enabled.json"),
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
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(m.configFile, data, 0644)
}
