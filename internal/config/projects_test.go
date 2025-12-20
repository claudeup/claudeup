package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectsRegistry_SetAndGet(t *testing.T) {
	reg := &ProjectsRegistry{
		Projects: make(map[string]ProjectEntry),
	}

	// Set a project
	reg.SetProject("/path/to/project", "frontend")

	// Get it back
	entry, ok := reg.GetProject("/path/to/project")
	if !ok {
		t.Fatal("GetProject returned false for existing project")
	}
	if entry.Profile != "frontend" {
		t.Errorf("Profile = %q, want %q", entry.Profile, "frontend")
	}
	if entry.AppliedAt.IsZero() {
		t.Error("AppliedAt should be set")
	}

	// Get nonexistent
	_, ok = reg.GetProject("/nonexistent")
	if ok {
		t.Error("GetProject should return false for nonexistent project")
	}
}

func TestProjectsRegistry_Remove(t *testing.T) {
	reg := &ProjectsRegistry{
		Projects: make(map[string]ProjectEntry),
	}

	reg.SetProject("/path/to/project", "frontend")

	// Remove existing
	if !reg.RemoveProject("/path/to/project") {
		t.Error("RemoveProject should return true for existing project")
	}

	// Verify it's gone
	_, ok := reg.GetProject("/path/to/project")
	if ok {
		t.Error("GetProject should return false after removal")
	}

	// Remove nonexistent
	if reg.RemoveProject("/nonexistent") {
		t.Error("RemoveProject should return false for nonexistent project")
	}
}

func TestLoadProjectsRegistry_CreatesEmpty(t *testing.T) {
	// Save original home and restore after test
	origHome := os.Getenv("HOME")
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tempDir)

	// Load should return empty registry when file doesn't exist
	reg, err := LoadProjectsRegistry()
	if err != nil {
		t.Fatalf("LoadProjectsRegistry failed: %v", err)
	}

	if reg.Version != "1" {
		t.Errorf("Version = %q, want %q", reg.Version, "1")
	}
	if reg.Projects == nil {
		t.Error("Projects map should be initialized")
	}
	if len(reg.Projects) != 0 {
		t.Errorf("len(Projects) = %d, want 0", len(reg.Projects))
	}
}

func TestSaveAndLoadProjectsRegistry(t *testing.T) {
	// Save original home and restore after test
	origHome := os.Getenv("HOME")
	tempDir, err := os.MkdirTemp("", "claudeup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tempDir)

	// Create and save registry
	reg := &ProjectsRegistry{
		Projects: make(map[string]ProjectEntry),
	}
	reg.SetProject("/project/one", "frontend")
	reg.SetProject("/project/two", "backend")

	if err := SaveProjectsRegistry(reg); err != nil {
		t.Fatalf("SaveProjectsRegistry failed: %v", err)
	}

	// Verify file was created
	path := filepath.Join(tempDir, ".claudeup", ProjectsFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("projects.json was not created")
	}

	// Load and verify
	loaded, err := LoadProjectsRegistry()
	if err != nil {
		t.Fatalf("LoadProjectsRegistry failed: %v", err)
	}

	if loaded.Version != "1" {
		t.Errorf("Version = %q, want %q", loaded.Version, "1")
	}
	if len(loaded.Projects) != 2 {
		t.Errorf("len(Projects) = %d, want 2", len(loaded.Projects))
	}

	entry, ok := loaded.GetProject("/project/one")
	if !ok {
		t.Error("Project /project/one not found after load")
	}
	if entry.Profile != "frontend" {
		t.Errorf("Profile = %q, want %q", entry.Profile, "frontend")
	}
}
