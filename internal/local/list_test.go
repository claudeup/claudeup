// ABOUTME: Tests for listing library items
// ABOUTME: Verifies directory scanning and item discovery
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListItems(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create some agent files
	os.WriteFile(filepath.Join(agentsDir, "planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(agentsDir, "executor.md"), []byte("# Executor"), 0644)
	os.WriteFile(filepath.Join(agentsDir, ".hidden.md"), []byte("# Hidden"), 0644) // Should be excluded

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) != 2 {
		t.Errorf("ListItems() returned %d items, want 2", len(items))
	}

	// Should be sorted
	if items[0] != "executor.md" || items[1] != "planner.md" {
		t.Errorf("ListItems() = %v, want [executor.md, planner.md]", items)
	}
}

func TestListItemsWithGroups(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create library structure with groups (for agents)
	libraryDir := filepath.Join(tmpDir, ".library")
	agentsDir := filepath.Join(libraryDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")
	os.MkdirAll(groupDir, 0755)

	// Flat agents
	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# GSD Planner"), 0644)

	// Grouped agents
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)
	os.WriteFile(filepath.Join(groupDir, "strategist.md"), []byte("# Strategist"), 0644)

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	// Should include both flat and grouped items
	expected := []string{
		"business-product/analyst.md",
		"business-product/strategist.md",
		"gsd-planner.md",
	}

	if len(items) != len(expected) {
		t.Errorf("ListItems() returned %d items, want %d: %v", len(items), len(expected), items)
	}

	for i, want := range expected {
		if i < len(items) && items[i] != want {
			t.Errorf("ListItems()[%d] = %q, want %q", i, items[i], want)
		}
	}
}

func TestListItemsEmptyCategory(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	items, err := manager.ListItems("agents")
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("ListItems() returned %d items for empty category, want 0", len(items))
	}
}
