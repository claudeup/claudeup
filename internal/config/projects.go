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

// ProjectEntry represents a project's profile configuration.
// Profile/AppliedAt track local scope; ProjectProfile/ProjectAppliedAt track project scope.
type ProjectEntry struct {
	Profile          string    `json:"profile"`
	AppliedAt        time.Time `json:"appliedAt"`
	ProjectProfile   string    `json:"projectProfile,omitempty"`
	ProjectAppliedAt time.Time `json:"projectAppliedAt,omitzero"`
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

// SetProject associates a project directory with a profile at local scope.
// Preserves any existing project-scope fields on the entry.
func (r *ProjectsRegistry) SetProject(projectPath, profile string) {
	entry := r.Projects[projectPath]
	entry.Profile = profile
	entry.AppliedAt = time.Now()
	r.Projects[projectPath] = entry
}

// GetProject returns the profile entry for a project directory at local scope.
// Returns false if no local-scope profile is set, even if a project-scope profile exists.
func (r *ProjectsRegistry) GetProject(projectPath string) (ProjectEntry, bool) {
	entry, ok := r.Projects[projectPath]
	if !ok || entry.Profile == "" {
		return entry, false
	}
	return entry, true
}

// SetProjectScope associates a project directory with a profile at project scope
func (r *ProjectsRegistry) SetProjectScope(projectPath, profile string) {
	entry := r.Projects[projectPath]
	entry.ProjectProfile = profile
	entry.ProjectAppliedAt = time.Now()
	r.Projects[projectPath] = entry
}

// GetProjectScope returns the project-scope profile name for a project directory
func (r *ProjectsRegistry) GetProjectScope(projectPath string) (string, bool) {
	entry, ok := r.Projects[projectPath]
	if !ok || entry.ProjectProfile == "" {
		return "", false
	}
	return entry.ProjectProfile, true
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
