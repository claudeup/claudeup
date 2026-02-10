// ABOUTME: Removes items from local storage
// ABOUTME: Disables items, deletes files, and cleans up config entries
package local

import (
	"os"
	"path/filepath"
)

// Uninstall removes items matching the given patterns from local storage.
// For each matched item: removes the symlink, deletes from local storage,
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

	var removed []string
	var notFound []string

	for _, pattern := range patterns {
		matched, _, found := m.resolvePattern(category, pattern, allItems)
		if !found {
			notFound = append(notFound, pattern)
			continue
		}

		for _, item := range matched {
			// Remove from local storage
			itemPath := filepath.Join(m.localDir, category, item)
			os.RemoveAll(itemPath)

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
