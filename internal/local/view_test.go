// ABOUTME: Tests for viewing item contents
// ABOUTME: Verifies content retrieval for different item types
package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestView(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure
	localDir := filepath.Join(claudeupHome, "local")
	hooksDir := filepath.Join(localDir, "hooks")
	skillsDir := filepath.Join(localDir, "skills", "bash")
	os.MkdirAll(hooksDir, 0755)
	os.MkdirAll(skillsDir, 0755)

	os.WriteFile(filepath.Join(hooksDir, "format-on-save.sh"), []byte("#!/bin/bash\necho hello"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# Bash Skill\nDescription here"), 0644)

	// View hook
	content, err := manager.View("hooks", "format-on-save")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "#!/bin/bash\necho hello" {
		t.Errorf("View() content = %q, want script content", content)
	}

	// View skill (directory with SKILL.md)
	content, err = manager.View("skills", "bash")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "# Bash Skill\nDescription here" {
		t.Errorf("View() content = %q, want skill content", content)
	}
}

func TestViewAgent(t *testing.T) {
	claudeDir := t.TempDir()
	claudeupHome := t.TempDir()
	manager := NewManager(claudeDir, claudeupHome)

	// Create local directory structure with grouped agent
	localDir := filepath.Join(claudeupHome, "local")
	agentsDir := filepath.Join(localDir, "agents")
	groupDir := filepath.Join(agentsDir, "business-product")
	os.MkdirAll(groupDir, 0755)

	os.WriteFile(filepath.Join(agentsDir, "gsd-planner.md"), []byte("# GSD Planner"), 0644)
	os.WriteFile(filepath.Join(groupDir, "analyst.md"), []byte("# Analyst Agent"), 0644)

	// View flat agent
	content, err := manager.View("agents", "gsd-planner")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "# GSD Planner" {
		t.Errorf("View() content = %q, want '# GSD Planner'", content)
	}

	// View grouped agent
	content, err = manager.View("agents", "business-product/analyst")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if content != "# Analyst Agent" {
		t.Errorf("View() content = %q, want '# Analyst Agent'", content)
	}
}
