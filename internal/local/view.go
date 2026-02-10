// ABOUTME: View item contents from local storage
// ABOUTME: Handles files and skill directories
package local

import (
	"fmt"
	"os"
	"path/filepath"
)

// View returns the contents of an item.
// For skills (which are directories), returns SKILL.md content.
// For agents, returns the .md file content.
// For other categories, returns the file content.
func (m *Manager) View(category, item string) (string, error) {
	if err := ValidateCategory(category); err != nil {
		return "", err
	}

	if category == CategorySkills {
		// Skills are directories with SKILL.md inside
		skillDir := filepath.Join(m.localDir, category, item)
		if info, err := os.Stat(skillDir); err == nil && info.IsDir() {
			skillFile := filepath.Join(skillDir, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				return "", fmt.Errorf("skill not found: %s", item)
			}
			return string(data), nil
		}
		return "", fmt.Errorf("skill not found: %s", item)
	}

	// Resolve item name (handles missing extensions)
	resolved, err := m.ResolveItemName(category, item)
	if err != nil {
		return "", err
	}

	// Read the file
	filePath := filepath.Join(m.localDir, category, resolved)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("item not found: %s/%s", category, item)
	}

	return string(data), nil
}
