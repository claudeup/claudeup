// ABOUTME: Installs items from external paths to the library
// ABOUTME: Copies files/directories and auto-enables them
package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Install copies items from sourcePath to the library and enables them.
// For single files/directories: copies as-is.
// For containers with multiple items: copies each item individually.
// Returns (installed items, skipped items, error).
func (m *Manager) Install(category string, sourcePath string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	source, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, nil, err
	}

	info, err := os.Stat(source)
	if err != nil {
		return nil, nil, fmt.Errorf("source not found: %s", sourcePath)
	}

	libraryDir := filepath.Join(m.libraryDir, category)
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		return nil, nil, err
	}

	var installed []string
	var skipped []string

	if info.IsDir() {
		// Determine if source is a single item or container of multiple items
		isSingleItem := m.isSingleItemDir(category, source)

		if isSingleItem {
			// Copy the directory as a single item
			itemName := filepath.Base(source)
			destPath := filepath.Join(libraryDir, itemName)

			if pathExists(destPath) {
				skipped = append(skipped, itemName)
			} else {
				if err := copyDir(source, destPath); err != nil {
					return nil, nil, err
				}
				installed = append(installed, itemName)
			}
		} else {
			// Container: copy each child item individually
			entries, err := os.ReadDir(source)
			if err != nil {
				return nil, nil, err
			}

			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				itemName := entry.Name()
				srcPath := filepath.Join(source, itemName)
				destPath := filepath.Join(libraryDir, itemName)

				if pathExists(destPath) {
					skipped = append(skipped, itemName)
					continue
				}

				if entry.IsDir() {
					if err := copyDir(srcPath, destPath); err != nil {
						return nil, nil, err
					}
				} else {
					if err := copyFile(srcPath, destPath); err != nil {
						return nil, nil, err
					}
				}
				installed = append(installed, itemName)
			}
		}
	} else {
		// Single file
		itemName := filepath.Base(source)
		destPath := filepath.Join(libraryDir, itemName)

		if pathExists(destPath) {
			skipped = append(skipped, itemName)
		} else {
			if err := copyFile(source, destPath); err != nil {
				return nil, nil, err
			}
			installed = append(installed, itemName)
		}
	}

	// Enable all installed items
	if len(installed) > 0 {
		_, _, err := m.Enable(category, installed)
		if err != nil {
			return nil, nil, err
		}
	}

	return installed, skipped, nil
}

// isSingleItemDir determines if a directory should be treated as a single item
// or as a container of multiple items.
func (m *Manager) isSingleItemDir(category string, dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return true // Treat errors as single item
	}

	// Filter out hidden files
	var children []os.DirEntry
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			children = append(children, e)
		}
	}

	// Skills: directory containing SKILL.md is a single skill
	if category == CategorySkills {
		for _, e := range children {
			if e.Name() == "SKILL.md" {
				return true
			}
		}
		// Check if children are skill directories (have SKILL.md inside)
		hasSkillChildren := false
		for _, e := range children {
			if e.IsDir() {
				skillMd := filepath.Join(dirPath, e.Name(), "SKILL.md")
				if pathExists(skillMd) {
					hasSkillChildren = true
					break
				}
			}
		}
		if hasSkillChildren {
			return false // Container of skills
		}
		return true // Single item (even if no SKILL.md)
	}

	// Agents: directory containing .md files (not inside subdirs) is an agent group
	if category == CategoryAgents {
		hasMdFiles := false
		for _, e := range children {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "CLAUDE.md" {
				hasMdFiles = true
				break
			}
		}
		if hasMdFiles {
			return true // Agent group (single item)
		}
		// Check if children are agent group directories
		hasAgentGroupChildren := false
		for _, e := range children {
			if e.IsDir() {
				groupDir := filepath.Join(dirPath, e.Name())
				groupEntries, _ := os.ReadDir(groupDir)
				for _, ge := range groupEntries {
					if !ge.IsDir() && strings.HasSuffix(ge.Name(), ".md") && ge.Name() != "CLAUDE.md" {
						hasAgentGroupChildren = true
						break
					}
				}
			}
		}
		if hasAgentGroupChildren {
			return false // Container of agent groups
		}
		return true
	}

	// Commands, hooks, rules, output-styles: if it contains .md/.sh/.py files at top level,
	// it's a container of items
	hasFlatItems := false
	for _, e := range children {
		if !e.IsDir() {
			ext := filepath.Ext(e.Name())
			if ext == ".md" || ext == ".sh" || ext == ".py" {
				hasFlatItems = true
				break
			}
		}
	}
	if hasFlatItems {
		return false // Container of items
	}

	return true // Single item (directory)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
