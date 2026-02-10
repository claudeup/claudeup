// ABOUTME: Global configuration management for claudeup
// ABOUTME: Handles loading and saving ~/.claudeup/config.json
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GlobalConfig represents the global configuration file structure
type GlobalConfig struct {
	Preferences Preferences `json:"preferences"`
}

// Preferences represents user preferences
type Preferences struct {
	ActiveProfile string `json:"activeProfile,omitempty"`
}

// DefaultConfig returns a new config with default values
func DefaultConfig() *GlobalConfig {
	return &GlobalConfig{
		Preferences: Preferences{},
	}
}

// configPath returns the path to the global config file
func configPath() string {
	return filepath.Join(MustClaudeupHome(), "config.json")
}

// Load reads the global config file, creating it with defaults if it doesn't exist
func Load() (*GlobalConfig, error) {
	cfgPath := configPath()

	// If config doesn't exist, create it with defaults
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := Save(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Read existing config
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the global config to disk
func Save(cfg *GlobalConfig) error {
	cfgPath := configPath()

	// Ensure directory exists
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write config
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cfgPath, data, 0644)
}
