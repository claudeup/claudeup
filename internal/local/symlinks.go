// ABOUTME: Symlink-based enable/disable for local items
// ABOUTME: Creates relative symlinks from target dirs to .library
package local

import (
	"os"
	"path/filepath"
	"strings"
)

// Enable enables items matching the given patterns.
// Returns (enabled items, not found patterns, error).
func (m *Manager) Enable(category string, patterns []string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	config, err := m.LoadConfig()
	if err != nil {
		return nil, nil, err
	}

	// Initialize category map if needed
	if config[category] == nil {
		config[category] = make(map[string]bool)
	}

	allItems, err := m.ListItems(category)
	if err != nil {
		return nil, nil, err
	}

	var enabled []string
	var notFound []string

	for _, pattern := range patterns {
		matched := MatchWildcard(pattern, allItems)
		if len(matched) == 0 {
			// Try to resolve as a single item
			resolved, err := m.ResolveItemName(category, pattern)
			if err != nil {
				notFound = append(notFound, pattern)
				continue
			}
			matched = []string{resolved}
		}

		for _, item := range matched {
			config[category][item] = true
			enabled = append(enabled, item)
		}
	}

	if len(enabled) > 0 {
		if err := m.SaveConfig(config); err != nil {
			return nil, nil, err
		}
		if err := m.syncCategory(category, config); err != nil {
			return nil, nil, err
		}
	}

	return enabled, notFound, nil
}

// Disable disables items matching the given patterns.
// Returns (disabled items, not found patterns, error).
func (m *Manager) Disable(category string, patterns []string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	config, err := m.LoadConfig()
	if err != nil {
		return nil, nil, err
	}

	if config[category] == nil {
		config[category] = make(map[string]bool)
	}

	allItems, err := m.ListItems(category)
	if err != nil {
		return nil, nil, err
	}

	var disabled []string
	var notFound []string

	for _, pattern := range patterns {
		matched := MatchWildcard(pattern, allItems)
		if len(matched) == 0 {
			resolved, err := m.ResolveItemName(category, pattern)
			if err != nil {
				notFound = append(notFound, pattern)
				continue
			}
			matched = []string{resolved}
		}

		for _, item := range matched {
			config[category][item] = false
			disabled = append(disabled, item)
		}
	}

	if len(disabled) > 0 {
		if err := m.SaveConfig(config); err != nil {
			return nil, nil, err
		}
		if err := m.syncCategory(category, config); err != nil {
			return nil, nil, err
		}
	}

	return disabled, notFound, nil
}

// syncCategory creates/removes symlinks based on config state
func (m *Manager) syncCategory(category string, config Config) error {
	targetDir := filepath.Join(m.claudeDir, category)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	catConfig := config[category]
	if catConfig == nil {
		catConfig = make(map[string]bool)
	}

	if category == CategoryAgents {
		return m.syncAgents(targetDir, catConfig)
	}

	return m.syncFlatCategory(category, targetDir, catConfig)
}

func (m *Manager) syncFlatCategory(category string, targetDir string, catConfig map[string]bool) error {
	// Remove existing symlinks
	entries, _ := os.ReadDir(targetDir)
	for _, entry := range entries {
		path := filepath.Join(targetDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(path)
		}
	}

	// Create symlinks for enabled items
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}

		target := filepath.Join(targetDir, item)
		// Relative path: ../.library/{category}/{item}
		relSource := filepath.Join("..", ".library", category, item)
		if err := os.Symlink(relSource, target); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) syncAgents(targetDir string, catConfig map[string]bool) error {
	// Remove existing symlinks and empty group directories
	entries, _ := os.ReadDir(targetDir)
	for _, entry := range entries {
		path := filepath.Join(targetDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(path)
		} else if entry.IsDir() {
			// Remove symlinks inside group directories
			groupEntries, _ := os.ReadDir(path)
			for _, ge := range groupEntries {
				gePath := filepath.Join(path, ge.Name())
				geInfo, _ := os.Lstat(gePath)
				if geInfo != nil && geInfo.Mode()&os.ModeSymlink != 0 {
					os.Remove(gePath)
				}
			}
			// Remove group dir if empty
			remaining, _ := os.ReadDir(path)
			if len(remaining) == 0 {
				os.Remove(path)
			}
		}
	}

	// Create symlinks for enabled agents
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}

		if strings.Contains(item, "/") {
			// Grouped agent: group/agent.md
			parts := strings.SplitN(item, "/", 2)
			group, agent := parts[0], parts[1]

			groupTargetDir := filepath.Join(targetDir, group)
			if err := os.MkdirAll(groupTargetDir, 0755); err != nil {
				return err
			}

			target := filepath.Join(groupTargetDir, agent)
			// Relative path: ../../.library/agents/{group}/{agent}
			relSource := filepath.Join("..", "..", ".library", "agents", group, agent)
			if err := os.Symlink(relSource, target); err != nil {
				return err
			}
		} else {
			// Flat agent
			target := filepath.Join(targetDir, item)
			relSource := filepath.Join("..", ".library", "agents", item)
			if err := os.Symlink(relSource, target); err != nil {
				return err
			}
		}
	}

	return nil
}

// Sync synchronizes all categories from config to symlinks
func (m *Manager) Sync() error {
	config, err := m.LoadConfig()
	if err != nil {
		return err
	}

	for _, category := range AllCategories() {
		if err := m.syncCategory(category, config); err != nil {
			return err
		}
	}

	return nil
}
