// ABOUTME: Copies local items into project .claude/ directory for project-scope profiles
// ABOUTME: Uses file copy (not symlinks) so items are portable and git-committable
package local

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CopyToProject copies local items matching patterns into the project's .claude/{category}/ directory.
// Source items are read from {localDir}/{category}/{item}.
// Destination is {projectDir}/.claude/{category}/{item}.
// Skill directories (containing SKILL.md) are copied recursively.
// Returns (copied items, not found patterns, error).
func CopyToProject(localDir, category string, patterns []string, projectDir string) ([]string, []string, error) {
	sourceDir := filepath.Join(localDir, category)

	// List available items in local storage
	allItems, err := listLocalItems(sourceDir, category)
	if err != nil {
		return nil, nil, fmt.Errorf("list items for %s: %w", category, err)
	}

	var copied []string
	var notFound []string

	for _, pattern := range patterns {
		matched := MatchWildcard(pattern, allItems)
		if len(matched) == 0 {
			notFound = append(notFound, pattern)
			continue
		}

		destBase := filepath.Clean(filepath.Join(projectDir, ".claude", category))
		for _, item := range matched {
			srcPath := filepath.Join(sourceDir, item)
			destPath := filepath.Clean(filepath.Join(destBase, item))

			// Validate dest stays within the target directory (path traversal defense)
			if !strings.HasPrefix(destPath, destBase+string(filepath.Separator)) {
				return nil, nil, fmt.Errorf("item %q resolves outside target directory %s", item, destBase)
			}

			if err := copyItemToProject(srcPath, destPath); err != nil {
				return nil, nil, fmt.Errorf("copy %s/%s from %s to %s: %w", category, item, srcPath, destPath, err)
			}
			copied = append(copied, item)
		}
	}

	return copied, notFound, nil
}

// listLocalItems enumerates items in a local storage directory.
// Mirrors the listing logic from Manager.ListItems but without needing a Manager.
func listLocalItems(dir, category string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []string{}, nil
	}

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

			// Skill directories (containing SKILL.md) are single items
			if _, err := os.Stat(filepath.Join(subDir, "SKILL.md")); err == nil {
				items = append(items, name)
				continue
			}

			// Expand subdirectory contents
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				subName := subEntry.Name()
				if strings.HasPrefix(subName, ".") || subName == "CLAUDE.md" {
					continue
				}
				if subEntry.IsDir() {
					// For agents, subdirs are groups
					if category == CategoryAgents {
						items = append(items, name+"/"+subName)
					}
				} else {
					items = append(items, name+"/"+subName)
				}
			}
		} else {
			items = append(items, name)
		}
	}

	sort.Strings(items)
	return items, nil
}

// copyItemToProject copies a file or directory from src to dest, creating parent directories as needed.
func copyItemToProject(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dest)
	}

	// Create parent directories for the destination file
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	return copyFile(src, dest)
}
