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
	DisabledPlugins    map[string]DisabledPlugin `json:"disabledPlugins"`
	DisabledMCPServers []string                  `json:"disabledMcpServers"`
	ClaudeDir          string                    `json:"claudeDir,omitempty"`
	Preferences        Preferences               `json:"preferences"`
}

// DisabledPlugin stores metadata for a disabled plugin
type DisabledPlugin struct {
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	InstallPath  string `json:"installPath"`
	GitCommitSha string `json:"gitCommitSha"`
	IsLocal      bool   `json:"isLocal"`
}

// Preferences represents user preferences
type Preferences struct {
	AutoUpdate    bool   `json:"autoUpdate"`
	VerboseOutput bool   `json:"verboseOutput"`
	ActiveProfile string `json:"activeProfile,omitempty"`
	SecretBackend string `json:"secretBackend,omitempty"`
}

// DefaultConfig returns a new config with default values
func DefaultConfig() *GlobalConfig {
	homeDir, _ := os.UserHomeDir()
	return &GlobalConfig{
		DisabledPlugins:    make(map[string]DisabledPlugin),
		DisabledMCPServers: []string{},
		ClaudeDir:          filepath.Join(homeDir, ".claude"),
		Preferences: Preferences{
			AutoUpdate:    false,
			VerboseOutput: false,
		},
	}
}

// configPath returns the path to the global config file
func configPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".claudeup", "config.json")
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

// IsPluginDisabled checks if a plugin is in the disabled map
func (c *GlobalConfig) IsPluginDisabled(pluginName string) bool {
	_, exists := c.DisabledPlugins[pluginName]
	return exists
}

// GetDisabledPlugin retrieves metadata for a disabled plugin
func (c *GlobalConfig) GetDisabledPlugin(pluginName string) (DisabledPlugin, bool) {
	plugin, exists := c.DisabledPlugins[pluginName]
	return plugin, exists
}

// IsMCPServerDisabled checks if an MCP server is in the disabled list
func (c *GlobalConfig) IsMCPServerDisabled(serverRef string) bool {
	for _, ref := range c.DisabledMCPServers {
		if ref == serverRef {
			return true
		}
	}
	return false
}

// DisablePlugin adds a plugin to the disabled map with its metadata
func (c *GlobalConfig) DisablePlugin(pluginName string, metadata DisabledPlugin) bool {
	if c.IsPluginDisabled(pluginName) {
		return false // Already disabled
	}
	c.DisabledPlugins[pluginName] = metadata
	return true
}

// EnablePlugin removes a plugin from the disabled map and returns its metadata
func (c *GlobalConfig) EnablePlugin(pluginName string) (DisabledPlugin, bool) {
	metadata, exists := c.DisabledPlugins[pluginName]
	if !exists {
		return DisabledPlugin{}, false // Wasn't disabled
	}
	delete(c.DisabledPlugins, pluginName)
	return metadata, true
}

// DisableMCPServer adds an MCP server to the disabled list
func (c *GlobalConfig) DisableMCPServer(serverRef string) bool {
	if c.IsMCPServerDisabled(serverRef) {
		return false // Already disabled
	}
	c.DisabledMCPServers = append(c.DisabledMCPServers, serverRef)
	return true
}

// EnableMCPServer removes an MCP server from the disabled list
func (c *GlobalConfig) EnableMCPServer(serverRef string) bool {
	for i, ref := range c.DisabledMCPServers {
		if ref == serverRef {
			c.DisabledMCPServers = append(c.DisabledMCPServers[:i], c.DisabledMCPServers[i+1:]...)
			return true
		}
	}
	return false // Wasn't disabled
}
