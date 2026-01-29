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

func TestPluginSearchIndex_Fields(t *testing.T) {
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
	if len(p.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(p.Skills))
	}
	if len(p.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(p.Commands))
	}
	if len(p.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(p.Agents))
	}
}
