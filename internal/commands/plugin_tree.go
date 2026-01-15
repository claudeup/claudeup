// ABOUTME: Tree generation for plugin directory display
// ABOUTME: Outputs unicode box-drawing formatted directory tree
package commands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// generateTree creates a tree representation of a directory
// Returns the tree string, directory count, and file count
func generateTree(root string) (string, int, int) {
	return generateTreeWithPrefix(root, "")
}

// generateTreeWithPrefix creates a tree with a given prefix for nested content
func generateTreeWithPrefix(root string, prefix string) (string, int, int) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", 0, 0 // Directory couldn't be read
	}
	if len(entries) == 0 {
		return "", 0, 0 // Directory is empty
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
			childTree, childDirs, childFiles := generateTreeWithPrefix(filepath.Join(root, entry.Name()), childPrefix)
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
