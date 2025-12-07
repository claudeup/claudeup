// ABOUTME: Profile struct and Load/Save functionality for claudeup
// ABOUTME: Profiles define a desired state of Claude Code configuration
package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Profile represents a Claude Code configuration profile
type Profile struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	MCPServers   []MCPServer   `json:"mcpServers,omitempty"`
	Marketplaces []Marketplace `json:"marketplaces,omitempty"`
	Plugins      []string      `json:"plugins,omitempty"`
	Detect       DetectRules   `json:"detect,omitempty"`
	Sandbox      SandboxConfig `json:"sandbox,omitempty"`
}

// SandboxConfig defines sandbox-specific settings for a profile
type SandboxConfig struct {
	// Secrets are secret names to resolve and inject into the sandbox
	Secrets []string `json:"secrets,omitempty"`

	// Mounts are additional host:container path mappings
	Mounts []SandboxMount `json:"mounts,omitempty"`

	// Env are static environment variables to set
	Env map[string]string `json:"env,omitempty"`
}

// SandboxMount represents a host-to-container path mapping
type SandboxMount struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	ReadOnly  bool   `json:"readonly,omitempty"`
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name    string               `json:"name"`
	Command string               `json:"command"`
	Args    []string             `json:"args,omitempty"`
	Scope   string               `json:"scope,omitempty"`
	Secrets map[string]SecretRef `json:"secrets,omitempty"`
}

// Marketplace represents a plugin marketplace source
type Marketplace struct {
	Source string `json:"source"`
	Repo   string `json:"repo,omitempty"`
	URL    string `json:"url,omitempty"`
}

// SecretRef defines a secret requirement with multiple resolution sources
type SecretRef struct {
	Description string         `json:"description,omitempty"`
	Sources     []SecretSource `json:"sources"`
}

// SecretSource defines a single source for resolving a secret
type SecretSource struct {
	Type    string `json:"type"`              // env, 1password, keychain
	Key     string `json:"key,omitempty"`     // for env
	Ref     string `json:"ref,omitempty"`     // for 1password
	Service string `json:"service,omitempty"` // for keychain
	Account string `json:"account,omitempty"` // for keychain
}

// DetectRules defines how to auto-detect if a profile matches a project
type DetectRules struct {
	Files    []string          `json:"files,omitempty"`
	Contains map[string]string `json:"contains,omitempty"`
}

// Save writes a profile to the profiles directory
func Save(profilesDir string, p *Profile) error {
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return err
	}

	profilePath := filepath.Join(profilesDir, p.Name+".json")

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(profilePath, data, 0644)
}

// Load reads a profile from the profiles directory
func Load(profilesDir, name string) (*Profile, error) {
	profilePath := filepath.Join(profilesDir, name+".json")

	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, err
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// List returns all profiles in the profiles directory, sorted by name
func List(profilesDir string) ([]*Profile, error) {
	entries, err := os.ReadDir(profilesDir)
	if os.IsNotExist(err) {
		return []*Profile{}, nil
	}
	if err != nil {
		return nil, err
	}

	var profiles []*Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		p, err := Load(profilesDir, name)
		if err != nil {
			continue // Skip invalid profiles
		}
		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}
