// ABOUTME: Removes items from extension storage
// ABOUTME: Disables items, deletes files, and cleans up config entries
package ext

import (
	"fmt"
	"os"
	"path/filepath"
)

// Uninstall removes items matching the given patterns from extension storage.
// For each matched item: removes the symlink, deletes from extension storage,
// and removes the config entry.
// Returns (removed items, not found patterns, error).
func (m *Manager) Uninstall(category string, patterns []string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	allItems, err := m.ListItems(category)
	if err != nil {
		return nil, nil, err
	}

	config, err := m.LoadConfig()
	if err != nil {
		return nil, nil, err
	}

	seen := make(map[string]bool)
	var removed []string
	var notFound []string

	for _, pattern := range patterns {
		matched, dirToClear, found := m.resolvePattern(category, pattern, allItems)
		if !found {
			notFound = append(notFound, pattern)
			continue
		}

		// Clean up directory-level config entry
		if dirToClear != "" && config[category] != nil {
			delete(config[category], dirToClear)
		}

		for _, item := range matched {
			if seen[item] {
				continue
			}
			seen[item] = true

			if err := validateItemPath(item); err != nil {
				return nil, nil, err
			}

			// Remove from extension storage
			itemPath := filepath.Join(m.extDir, category, item)
			if err := os.RemoveAll(itemPath); err != nil {
				return nil, nil, fmt.Errorf("failed to remove %s: %w", item, err)
			}

			// Remove config entry
			if config[category] != nil {
				delete(config[category], item)
			}

			removed = append(removed, item)
		}
	}

	if len(removed) > 0 {
		if err := m.SaveConfig(config); err != nil {
			return nil, nil, err
		}
		if err := m.syncCategory(category, config); err != nil {
			return nil, nil, err
		}
	}

	return removed, notFound, nil
}
