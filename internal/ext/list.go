// ABOUTME: Functions for listing items in extension storage
// ABOUTME: Handles both flat items and grouped items (like agents)
package ext

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListItems returns all items in extension storage for a category.
// For agents, returns 'group/agent.md' format for grouped items.
// Excludes hidden files (starting with .) and CLAUDE.md.
func (m *Manager) ListItems(category string) ([]string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}

	libPath := filepath.Join(m.extDir, category)
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	var items []string
	var err error

	if category == CategoryAgents {
		// Agents can have groups (subdirectories)
		items, err = m.listAgentItems(libPath)
	} else {
		// Other categories support nested structure
		items, err = m.listFlatItems(libPath)
	}

	if err != nil {
		return nil, err
	}

	sort.Strings(items)
	return items, nil
}

func (m *Manager) listFlatItems(dir string) ([]string, error) {
	var items []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
			continue
		}

		if entry.IsDir() {
			subDir := filepath.Join(dir, name)

			// Check if this is a skill directory (contains SKILL.md)
			// Skills are directories containing a SKILL.md file - the directory name IS the item
			skillFile := filepath.Join(subDir, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				items = append(items, name)
				continue
			}

			// Walk subdirectory for nested items (e.g., commands/gsd/*)
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				// Subdirectory unreadable - skip it (non-fatal)
				continue
			}
			for _, subEntry := range subEntries {
				subName := subEntry.Name()
				if strings.HasPrefix(subName, ".") || subName == "CLAUDE.md" {
					continue
				}
				if !subEntry.IsDir() {
					items = append(items, name+"/"+subName)
				}
			}
		} else {
			items = append(items, name)
		}
	}

	return items, nil
}

func (m *Manager) listAgentItems(dir string) ([]string, error) {
	var items []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
			continue
		}

		if entry.IsDir() {
			// This is a group directory - list agents inside
			groupPath := filepath.Join(dir, name)
			groupEntries, err := os.ReadDir(groupPath)
			if err != nil {
				// Group directory unreadable - skip it (non-fatal)
				continue
			}
			for _, groupEntry := range groupEntries {
				agentName := groupEntry.Name()
				if strings.HasPrefix(agentName, ".") || agentName == "CLAUDE.md" {
					continue
				}
				if strings.HasSuffix(agentName, ".md") {
					items = append(items, name+"/"+agentName)
				}
			}
		} else if strings.HasSuffix(name, ".md") {
			// Flat agent file
			items = append(items, name)
		}
	}

	return items, nil
}
