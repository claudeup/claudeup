// ABOUTME: Symlink-based enable/disable for local items
// ABOUTME: Creates absolute symlinks from Claude's active dirs to the library
package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateItemPath checks that an item name doesn't contain path traversal sequences
func validateItemPath(item string) error {
	// Check for explicit path traversal
	if strings.Contains(item, "..") {
		return fmt.Errorf("path traversal detected in item name: %q", item)
	}
	// Check for absolute paths
	if filepath.IsAbs(item) {
		return fmt.Errorf("path traversal detected in item name: %q", item)
	}
	return nil
}

// resolvePattern resolves a pattern to matching items.
// If pattern matches via wildcard, returns those matches.
// If pattern resolves to a directory (not a skill), expands to all items inside.
// Returns (matched items, directory to clear from config, found).
func (m *Manager) resolvePattern(category, pattern string, allItems []string) ([]string, string, bool) {
	// Try wildcard match first
	matched := MatchWildcard(pattern, allItems)
	if len(matched) > 0 {
		return matched, "", true
	}

	// Try to resolve as a single item
	resolved, err := m.ResolveItemName(category, pattern)
	if err != nil {
		return nil, "", false
	}

	// Check if resolved item is a directory (not a skill)
	resolvedPath := filepath.Join(m.localDir, category, resolved)
	info, statErr := os.Stat(resolvedPath)
	if statErr != nil || !info.IsDir() {
		// Not a directory - return as single item
		return []string{resolved}, "", true
	}

	// Check if it's a skill directory (has SKILL.md)
	skillFile := filepath.Join(resolvedPath, "SKILL.md")
	if _, skillErr := os.Stat(skillFile); skillErr == nil {
		// It's a skill directory - treat as single item
		return []string{resolved}, "", true
	}

	// Not a skill - expand to contents using wildcard
	expandedPattern := resolved + "/*"
	matched = MatchWildcard(expandedPattern, allItems)
	if len(matched) == 0 {
		// Directory exists but is empty
		return nil, "", false
	}

	// Return matches and the directory name to clear from config
	return matched, resolved, true
}

// Enable enables items matching the given patterns.
// Supports wildcards (gsd-*, gsd/*) and directory names.
// When a directory name is given (without wildcard), it expands to enable all
// items inside, e.g., "vsphere-architect" becomes "vsphere-architect/*".
// Skill directories (containing SKILL.md) are treated as single items.
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
		matched, dirToClear, found := m.resolvePattern(category, pattern, allItems)
		if !found {
			notFound = append(notFound, pattern)
			continue
		}

		// Clear directory-level entry to prevent conflict with individual files
		if dirToClear != "" {
			config[category][dirToClear] = false
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
// Supports wildcards (gsd-*, gsd/*) and directory names.
// When a directory name is given (without wildcard), it expands to disable all
// items inside, e.g., "vsphere-architect" becomes "vsphere-architect/*".
// Skill directories (containing SKILL.md) are treated as single items.
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
		matched, dirToClear, found := m.resolvePattern(category, pattern, allItems)
		if !found {
			notFound = append(notFound, pattern)
			continue
		}

		// Clear directory-level entry if it exists (for config cleanup)
		if dirToClear != "" {
			config[category][dirToClear] = false
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
	// Validate all items before making any changes (fail fast)
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}
		if err := validateItemPath(item); err != nil {
			return err
		}
	}

	// Remove existing symlinks (including in subdirectories)
	m.cleanupSymlinksRecursive(targetDir)

	// Create symlinks for enabled items
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}

		target := filepath.Join(targetDir, item)

		// For nested items (e.g., gsd/new-project.md), create parent directories
		if strings.Contains(item, "/") {
			parentDir := filepath.Dir(target)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return err
			}
		}

		source := filepath.Join(m.localDir, category, item)
		if err := os.Symlink(source, target); err != nil {
			return err
		}
	}

	return nil
}

// cleanupSymlinksRecursive removes symlinks in a directory and its subdirectories
func (m *Manager) cleanupSymlinksRecursive(dir string) {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(path)
		} else if entry.IsDir() {
			// Recurse into subdirectories
			m.cleanupSymlinksRecursive(path)
			// Remove directory if empty
			remaining, _ := os.ReadDir(path)
			if len(remaining) == 0 {
				os.Remove(path)
			}
		}
	}
}

func (m *Manager) syncAgents(targetDir string, catConfig map[string]bool) error {
	// Validate all items before making any changes (fail fast)
	for item, enabled := range catConfig {
		if !enabled {
			continue
		}
		if err := validateItemPath(item); err != nil {
			return err
		}
	}

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
			source := filepath.Join(m.localDir, "agents", group, agent)
			if err := os.Symlink(source, target); err != nil {
				return err
			}
		} else {
			// Flat agent
			target := filepath.Join(targetDir, item)
			source := filepath.Join(m.localDir, "agents", item)
			if err := os.Symlink(source, target); err != nil {
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

// Import moves items from active directory to the library and enables them.
// This is useful when tools like GSD install directly to active directories.
//
// Reconciliation: If an item already exists in the library, the local copy in the
// active directory is removed and a symlink to the library version is created.
// This ensures the state is consistent (no duplicate items, symlinks point to library).
//
// Returns (imported items, skipped/reconciled items, not found patterns, error).
func (m *Manager) Import(category string, patterns []string) ([]string, []string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, nil, err
	}

	activeDir := filepath.Join(m.claudeDir, category)
	localDir := filepath.Join(m.localDir, category)

	// Ensure library directory exists
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return nil, nil, nil, err
	}

	// Find all items in active directory that are NOT symlinks
	candidates, err := m.findImportCandidates(activeDir)
	if err != nil {
		return nil, nil, nil, err
	}

	var imported []string
	var skipped []string
	var notFound []string

	for _, pattern := range patterns {
		matched := MatchWildcard(pattern, candidates)
		if len(matched) == 0 {
			notFound = append(notFound, pattern)
			continue
		}

		for _, item := range matched {
			sourcePath := filepath.Join(activeDir, item)
			destPath := filepath.Join(localDir, item)

			// Check if destination already exists in library
			if _, err := os.Stat(destPath); err == nil {
				// Library already has this item - remove source and enable the library version
				// This reconciles the state (replaces local file with symlink to library)
				if err := os.RemoveAll(sourcePath); err != nil {
					return nil, nil, nil, fmt.Errorf("failed to remove %s: %w", sourcePath, err)
				}
				skipped = append(skipped, item)
				continue
			}

			// Move to library
			if err := os.Rename(sourcePath, destPath); err != nil {
				return nil, nil, nil, err
			}

			imported = append(imported, item)
		}
	}

	// Enable all items (creates symlinks) - both imported and skipped (already in library)
	toEnable := append(imported, skipped...)
	if len(toEnable) > 0 {
		_, _, err := m.Enable(category, toEnable)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return imported, skipped, notFound, nil
}

// findImportCandidates finds files and directories in activeDir that are not symlinks
func (m *Manager) findImportCandidates(activeDir string) ([]string, error) {
	var candidates []string

	entries, err := os.ReadDir(activeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return candidates, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files (like .gitkeep) and CLAUDE.md
		if strings.HasPrefix(name, ".") || name == "CLAUDE.md" {
			continue
		}

		path := filepath.Join(activeDir, name)
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}

		// Skip symlinks - they're already managed
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		candidates = append(candidates, name)
	}

	return candidates, nil
}

// ImportAll imports items from all categories that match the given patterns.
// If patterns is empty/nil, imports all non-symlink items.
// Returns maps of category -> imported items and category -> skipped items.
func (m *Manager) ImportAll(patterns []string) (map[string][]string, map[string][]string, error) {
	imported := make(map[string][]string)
	skipped := make(map[string][]string)

	// If no patterns provided, use "*" to match everything
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}

	for _, category := range AllCategories() {
		catImported, catSkipped, _, err := m.Import(category, patterns)
		if err != nil {
			return nil, nil, err
		}
		if len(catImported) > 0 {
			imported[category] = catImported
		}
		if len(catSkipped) > 0 {
			skipped[category] = catSkipped
		}
	}

	return imported, skipped, nil
}
