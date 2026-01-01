// ABOUTME: Manages project-to-profile mappings for local scope profiles
// ABOUTME: Stores ~/.claudeup/projects.json tracking which profile is used per project
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ProjectsFile is the filename for the projects registry
const ProjectsFile = "projects.json"

// ProjectEntry represents a project's profile configuration
type ProjectEntry struct {
	Profile   string    `json:"profile"`
	AppliedAt time.Time `json:"appliedAt"`
}

// ProjectsRegistry tracks which profiles are applied to which project directories
type ProjectsRegistry struct {
	Version  string                  `json:"version"`
	Projects map[string]ProjectEntry `json:"projects"`
}

// LoadProjectsRegistry reads the projects registry from ~/.claudeup/projects.json
func LoadProjectsRegistry() (*ProjectsRegistry, error) {
	path := projectsPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &ProjectsRegistry{
			Version:  "1",
			Projects: make(map[string]ProjectEntry),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var reg ProjectsRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}

	if reg.Projects == nil {
		reg.Projects = make(map[string]ProjectEntry)
	}

	return &reg, nil
}

// SaveProjectsRegistry writes the projects registry to disk
func SaveProjectsRegistry(reg *ProjectsRegistry) error {
	reg.Version = "1"
	path := projectsPath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}

	// Add trailing newline
	data = append(data, '\n')

	return os.WriteFile(path, data, 0644)
}

// SetProject associates a project directory with a profile
func (r *ProjectsRegistry) SetProject(projectPath, profile string) {
	r.Projects[projectPath] = ProjectEntry{
		Profile:   profile,
		AppliedAt: time.Now(),
	}
}

// GetProject returns the profile entry for a project directory
func (r *ProjectsRegistry) GetProject(projectPath string) (ProjectEntry, bool) {
	entry, ok := r.Projects[projectPath]
	return entry, ok
}

// RemoveProject removes a project from the registry
func (r *ProjectsRegistry) RemoveProject(projectPath string) bool {
	if _, ok := r.Projects[projectPath]; ok {
		delete(r.Projects, projectPath)
		return true
	}
	return false
}

func projectsPath() string {
	return filepath.Join(MustClaudeupHome(), ProjectsFile)
}
