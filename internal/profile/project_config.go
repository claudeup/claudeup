// ABOUTME: Manages .claudeup.json file for project-level profile configuration
// ABOUTME: Local scope uses Claude Code's native .claude/settings.local.json
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup/internal/claude"
	"github.com/claudeup/claudeup/internal/events"
)

// ProjectConfigFile is the filename for project-level profile configuration
const ProjectConfigFile = ".claudeup.json"

// ProjectConfig represents the contents of a .claudeup.json file
type ProjectConfig struct {
	Version       string        `json:"version"`
	Profile       string        `json:"profile"`
	ProfileSource string        `json:"profileSource,omitempty"` // "embedded" or "custom"
	Marketplaces  []Marketplace `json:"marketplaces,omitempty"`
	Plugins       []string      `json:"plugins,omitempty"`
	AppliedAt     time.Time     `json:"appliedAt"`
}

// LoadProjectConfig reads a .claudeup.json file from the given directory
func LoadProjectConfig(projectDir string) (*ProjectConfig, error) {
	path := filepath.Join(projectDir, ProjectConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", ProjectConfigFile, err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", ProjectConfigFile, err)
	}

	return &cfg, nil
}

// Validate checks that required fields are present
func (c *ProjectConfig) Validate() error {
	if c.Profile == "" {
		return fmt.Errorf("missing required field: profile")
	}
	return nil
}

// SaveProjectConfig writes a .claudeup.json file to the given directory
func SaveProjectConfig(projectDir string, cfg *ProjectConfig) error {
	cfg.Version = "1"
	cfg.AppliedAt = time.Now()

	path := filepath.Join(projectDir, ProjectConfigFile)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Add trailing newline for cleaner git diffs
	data = append(data, '\n')

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"project config save",
		path,
		"project",
		func() error {
			return os.WriteFile(path, data, 0644)
		},
	)
}

// ProjectConfigExists returns true if a .claudeup.json file exists in the directory
func ProjectConfigExists(projectDir string) bool {
	path := filepath.Join(projectDir, ProjectConfigFile)
	_, err := os.Stat(path)
	return err == nil
}

// NewProjectConfig creates a ProjectConfig from a Profile
func NewProjectConfig(p *Profile) *ProjectConfig {
	source := "custom"
	if IsEmbeddedProfile(p.Name) {
		source = "embedded"
	}

	return &ProjectConfig{
		Profile:       p.Name,
		ProfileSource: source,
		Marketplaces:  p.Marketplaces,
		Plugins:       p.Plugins,
		AppliedAt:     time.Now(),
	}
}

// LoadConfigForScope loads the appropriate config file based on scope
// Only project scope has a .claudeup.json config file (local uses .claude/settings.local.json)
func LoadConfigForScope(projectDir string, scope Scope) (*ProjectConfig, error) {
	if scope == ScopeProject {
		return LoadProjectConfig(projectDir)
	}
	return nil, fmt.Errorf("invalid scope for config loading: %s (only project scope has config file)", scope)
}

// SaveConfigForScope saves the config to the appropriate file based on scope
// Only project scope has a .claudeup.json config file (local uses .claude/settings.local.json)
func SaveConfigForScope(projectDir string, cfg *ProjectConfig, scope Scope) error {
	if scope == ScopeProject {
		return SaveProjectConfig(projectDir, cfg)
	}
	return fmt.Errorf("invalid scope for config saving: %s (only project scope has config file)", scope)
}

// ConfigExistsForScope returns true if a config file exists for the given scope
// Only project scope has a .claudeup.json config file (local uses .claude/settings.local.json)
func ConfigExistsForScope(projectDir string, scope Scope) bool {
	if scope == ScopeProject {
		return ProjectConfigExists(projectDir)
	}
	return false
}

// DriftedPlugin represents a plugin that exists in config but is not installed
type DriftedPlugin struct {
	PluginName string
	Scope      Scope
}

// PluginChecker interface for checking if plugins are installed
type PluginChecker interface {
	IsPluginInstalled(name string) bool
}

// DetectConfigDrift finds plugins that are enabled but not installed
func DetectConfigDrift(claudeDir, projectDir string, pluginChecker PluginChecker) ([]DriftedPlugin, error) {
	var drift []DriftedPlugin
	var firstError error

	// Check project scope (.claudeup.json)
	if ProjectConfigExists(projectDir) {
		projectCfg, err := LoadProjectConfig(projectDir)
		if err != nil {
			// Record the error but continue checking other scopes
			if firstError == nil {
				firstError = fmt.Errorf("failed to load %s: %w", ProjectConfigFile, err)
			}
		} else {
			for _, pluginName := range projectCfg.Plugins {
				if !pluginChecker.IsPluginInstalled(pluginName) {
					drift = append(drift, DriftedPlugin{
						PluginName: pluginName,
						Scope:      ScopeProject,
					})
				}
			}
		}
	}

	// Check local scope (.claude/settings.local.json)
	localSettingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	if _, err := os.Stat(localSettingsPath); err == nil {
		localSettings, err := claude.LoadSettingsForScope("local", claudeDir, projectDir)
		if err != nil {
			// Record the error but continue checking other scopes
			if firstError == nil {
				firstError = fmt.Errorf("failed to load %s: %w", localSettingsPath, err)
			}
		} else {
			// Check each enabled plugin in local settings
			for pluginName := range localSettings.EnabledPlugins {
				if !pluginChecker.IsPluginInstalled(pluginName) {
					drift = append(drift, DriftedPlugin{
						PluginName: pluginName,
						Scope:      ScopeLocal,
					})
				}
			}
		}
	}

	return drift, firstError
}
