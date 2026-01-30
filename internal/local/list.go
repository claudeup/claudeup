// ABOUTME: Functions for listing items in the library
// ABOUTME: Handles both flat items and grouped items (like agents)
package local

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListItems returns all items in the library for a category.
// For agents, returns 'group/agent.md' format for grouped items.
// Excludes hidden files (starting with .) and CLAUDE.md.
func (m *Manager) ListItems(category string) ([]string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, err
	}

	libPath := filepath.Join(m.libraryDir, category)
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	var items []string

	if category == CategoryAgents {
		// Agents can have groups (subdirectories)
		items = m.listAgentItems(libPath)
	} else {
		// Other categories are flat
		items = m.listFlatItems(libPath)
	}

	sort.Strings(items)
	return items, nil
}

func (m *Manager) listFlatItems(dir string) []string {
	var items []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return items
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
			continue
		}
		items = append(items, name)
	}

	return items
}

func (m *Manager) listAgentItems(dir string) []string {
	var items []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return items
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

	return items
}
