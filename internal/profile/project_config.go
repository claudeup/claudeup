// ABOUTME: Manages .claudeup.json and .claudeup.local.json files for profile configuration
// ABOUTME: Mirrors Claude Code's scope structure (project vs local) for tracking applied profiles
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup/internal/events"
)

// ProjectConfigFile is the filename for project-level profile configuration
const ProjectConfigFile = ".claudeup.json"

// LocalConfigFile is the filename for local-level profile configuration
// Mirrors Claude Code's .claude/settings.local.json structure
const LocalConfigFile = ".claudeup.local.json"

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

// LoadLocalConfig reads a .claudeup.local.json file from the given directory
func LoadLocalConfig(projectDir string) (*ProjectConfig, error) {
	path := filepath.Join(projectDir, LocalConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", LocalConfigFile, err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", LocalConfigFile, err)
	}

	return &cfg, nil
}

// SaveLocalConfig writes a .claudeup.local.json file to the given directory
func SaveLocalConfig(projectDir string, cfg *ProjectConfig) error {
	cfg.Version = "1"
	cfg.AppliedAt = time.Now()

	path := filepath.Join(projectDir, LocalConfigFile)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Add trailing newline for cleaner git diffs
	data = append(data, '\n')

	// Wrap file write with event tracking
	return events.GlobalTracker().RecordFileWrite(
		"local config save",
		path,
		"local",
		func() error {
			return os.WriteFile(path, data, 0644)
		},
	)
}

// LocalConfigExists returns true if a .claudeup.local.json file exists in the directory
func LocalConfigExists(projectDir string) bool {
	path := filepath.Join(projectDir, LocalConfigFile)
	_, err := os.Stat(path)
	return err == nil
}

// LoadConfigForScope loads the appropriate config file based on scope
func LoadConfigForScope(projectDir string, scope Scope) (*ProjectConfig, error) {
	switch scope {
	case ScopeLocal:
		return LoadLocalConfig(projectDir)
	case ScopeProject:
		return LoadProjectConfig(projectDir)
	default:
		return nil, fmt.Errorf("invalid scope for config loading: %s (only project and local scopes have config files)", scope)
	}
}

// SaveConfigForScope saves the config to the appropriate file based on scope
func SaveConfigForScope(projectDir string, cfg *ProjectConfig, scope Scope) error {
	switch scope {
	case ScopeLocal:
		return SaveLocalConfig(projectDir, cfg)
	case ScopeProject:
		return SaveProjectConfig(projectDir, cfg)
	default:
		return fmt.Errorf("invalid scope for config saving: %s (only project and local scopes have config files)", scope)
	}
}

// ConfigExistsForScope returns true if a config file exists for the given scope
func ConfigExistsForScope(projectDir string, scope Scope) bool {
	switch scope {
	case ScopeLocal:
		return LocalConfigExists(projectDir)
	case ScopeProject:
		return ProjectConfigExists(projectDir)
	default:
		return false
	}
}
