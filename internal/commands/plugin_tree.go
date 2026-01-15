// ABOUTME: Tree generation for plugin directory display
// ABOUTME: Outputs unicode box-drawing formatted directory tree filtered to plugin components
package commands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Plugin-specific directories (per Claude Code plugin spec)
var pluginDirs = map[string]bool{
	".claude-plugin": true,
	"commands":       true,
	"agents":         true,
	"skills":         true,
	"hooks":          true,
	"scripts":        true,
}

// Plugin-specific files at root level
var pluginFiles = map[string]bool{
	".mcp.json": true,
	"README.md": true,
}

// generateTree creates a tree representation of a plugin directory
// Only shows plugin-specific directories and files at root level
// Returns the tree string, directory count, and file count
func generateTree(root string) (string, int, int) {
	return generateTreeWithPrefix(root, "", true)
}

// generateTreeWithPrefix creates a tree with a given prefix for nested content
func generateTreeWithPrefix(root string, prefix string, filterRoot bool) (string, int, int) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", 0, 0 // Directory couldn't be read
	}
	if len(entries) == 0 {
		return "", 0, 0 // Directory is empty
	}

	// Filter entries based on context
	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()

		if filterRoot {
			// At root: only show plugin-specific directories and files
			if entry.IsDir() {
				if pluginDirs[name] {
					filtered = append(filtered, entry)
				}
			} else {
				if pluginFiles[name] {
					filtered = append(filtered, entry)
				}
			}
		} else {
			// Inside plugin directories: show everything except .git
			if name != ".git" {
				filtered = append(filtered, entry)
			}
		}
	}
	entries = filtered

	if len(entries) == 0 {
		return "", 0, 0
	}

	var sb strings.Builder
	dirCount := 0
	fileCount := 0

	sort.Slice(entries, func(i, j int) bool {
		iDir := entries[i].IsDir()
		jDir := entries[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return entries[i].Name() < entries[j].Name()
	})

	for i, entry := range entries {
		isLast := i == len(entries)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := entry.Name()
		if entry.IsDir() {
			name += "/"
			dirCount++
			sb.WriteString(prefix + connector + name + "\n")

			childPrefix := prefix + "│   "
			if isLast {
				childPrefix = prefix + "    "
			}
			// Don't filter inside plugin directories
			childTree, childDirs, childFiles := generateTreeWithPrefix(filepath.Join(root, entry.Name()), childPrefix, false)
			sb.WriteString(childTree)
			dirCount += childDirs
			fileCount += childFiles
		} else {
			fileCount++
			sb.WriteString(prefix + connector + name + "\n")
		}
	}

	return sb.String(), dirCount, fileCount
}
