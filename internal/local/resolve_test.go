// ABOUTME: Tests for resolving item names with extension inference
// ABOUTME: Verifies partial name matching and agent group resolution
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveItemName(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create library structure
	localDir := filepath.Join(claudeupHome, "local")
	hooksDir := filepath.Join(localDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash"), 0644)
	os.WriteFile(filepath.Join(hooksDir, "gsd-check-update.js"), []byte("// js"), 0644)

	tests := []struct {
		name     string
		category string
		item     string
		want     string
		wantErr  bool
	}{
		{"exact match", "hooks", "format-on-save.sh", "format-on-save.sh", false},
		{"without extension .sh", "hooks", "format-on-save", "format-on-save.sh", false},
		{"without extension .js", "hooks", "gsd-check-update", "gsd-check-update.js", false},
		{"nonexistent", "hooks", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.ResolveItemName(tt.category, tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveItemName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveItemName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveAgentName(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create library structure with groups
	localDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(localDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")
	os.MkdirAll(groupDir, 0755)

	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# Planner"), 0644)
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst"), 0644)

	tests := []struct {
		name    string
		item    string
		want    string
		wantErr bool
	}{
		{"flat agent with ext", "gsd-planner.md", "gsd-planner.md", false},
		{"flat agent without ext", "gsd-planner", "gsd-planner.md", false},
		{"grouped agent full path", "business-product/analyst", "business-product/analyst.md", false},
		{"grouped agent with ext", "business-product/analyst.md", "business-product/analyst.md", false},
		{"agent name search", "analyst", "business-product/analyst.md", false},
		{"nonexistent", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.ResolveItemName("agents", tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveItemName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveItemName() = %q, want %q", got, tt.want)
			}
		})
	}
}
