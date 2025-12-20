// ABOUTME: Manages .claudeup.json files for project-level profile configuration
// ABOUTME: Stores profile metadata, marketplaces, and plugins for team sharing
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

	return os.WriteFile(path, data, 0644)
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
	}
}
