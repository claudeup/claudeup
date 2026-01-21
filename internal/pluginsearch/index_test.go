// ABOUTME: Unit tests for plugin search index types
// ABOUTME: Tests data structures used for indexing plugins
package pluginsearch

import (
	"testing"
)

func TestComponentInfo_Fields(t *testing.T) {
	c := ComponentInfo{
		Name:        "test-skill",
		Description: "A test skill",
		Path:        "/path/to/skill",
	}

	if c.Name != "test-skill" {
		t.Errorf("expected Name 'test-skill', got '%s'", c.Name)
	}
	if c.Description != "A test skill" {
		t.Errorf("expected Description 'A test skill', got '%s'", c.Description)
	}
	if c.Path != "/path/to/skill" {
		t.Errorf("expected Path '/path/to/skill', got '%s'", c.Path)
	}
}

func TestPluginSearchIndex_HasSkills(t *testing.T) {
	p := PluginSearchIndex{
		Name:        "test-plugin",
		Description: "A test plugin",
		Marketplace: "test-marketplace",
		Skills: []ComponentInfo{
			{Name: "skill1"},
		},
	}

	if !p.HasSkills() {
		t.Error("expected HasSkills() to return true when skills are present")
	}
	if p.HasCommands() {
		t.Error("expected HasCommands() to return false when commands are empty")
	}
	if p.HasAgents() {
		t.Error("expected HasAgents() to return false when agents are empty")
	}
}

func TestPluginSearchIndex_HasCommands(t *testing.T) {
	p := PluginSearchIndex{
		Name: "test-plugin",
		Commands: []ComponentInfo{
			{Name: "cmd1"},
			{Name: "cmd2"},
		},
	}

	if p.HasSkills() {
		t.Error("expected HasSkills() to return false when skills are empty")
	}
	if !p.HasCommands() {
		t.Error("expected HasCommands() to return true when commands are present")
	}
	if p.HasAgents() {
		t.Error("expected HasAgents() to return false when agents are empty")
	}
}

func TestPluginSearchIndex_HasAgents(t *testing.T) {
	p := PluginSearchIndex{
		Name: "test-plugin",
		Agents: []ComponentInfo{
			{Name: "agent1"},
		},
	}

	if p.HasSkills() {
		t.Error("expected HasSkills() to return false when skills are empty")
	}
	if p.HasCommands() {
		t.Error("expected HasCommands() to return false when commands are empty")
	}
	if !p.HasAgents() {
		t.Error("expected HasAgents() to return true when agents are present")
	}
}

func TestPluginSearchIndex_AllEmpty(t *testing.T) {
	p := PluginSearchIndex{
		Name: "empty-plugin",
	}

	if p.HasSkills() {
		t.Error("expected HasSkills() to return false for empty plugin")
	}
	if p.HasCommands() {
		t.Error("expected HasCommands() to return false for empty plugin")
	}
	if p.HasAgents() {
		t.Error("expected HasAgents() to return false for empty plugin")
	}
}

func TestPluginSearchIndex_AllPopulated(t *testing.T) {
	p := PluginSearchIndex{
		Name:        "full-plugin",
		Description: "A full plugin",
		Keywords:    []string{"test", "full"},
		Marketplace: "example-marketplace",
		Version:     "1.0.0",
		Path:        "/path/to/plugin",
		Skills: []ComponentInfo{
			{Name: "skill1", Description: "First skill"},
		},
		Commands: []ComponentInfo{
			{Name: "cmd1", Description: "First command"},
		},
		Agents: []ComponentInfo{
			{Name: "agent1", Description: "First agent"},
		},
	}

	if !p.HasSkills() {
		t.Error("expected HasSkills() to return true")
	}
	if !p.HasCommands() {
		t.Error("expected HasCommands() to return true")
	}
	if !p.HasAgents() {
		t.Error("expected HasAgents() to return true")
	}

	// Verify all fields are accessible
	if p.Name != "full-plugin" {
		t.Errorf("expected Name 'full-plugin', got '%s'", p.Name)
	}
	if len(p.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(p.Keywords))
	}
	if p.Marketplace != "example-marketplace" {
		t.Errorf("expected Marketplace 'example-marketplace', got '%s'", p.Marketplace)
	}
	if p.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", p.Version)
	}
}
