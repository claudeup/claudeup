// ABOUTME: Resolves and displays individual file contents within a plugin
// ABOUTME: Handles extension inference and skill directory conventions
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/ui"
)

// resolvePluginFile resolves a relative file path within a plugin root directory.
// It tries: exact match, SKILL.md inside skill directories, then common extensions.
// Returns an error for paths that escape the plugin root.
func resolvePluginFile(pluginRoot, filePath string) (string, error) {
	// Reject absolute paths
	if filepath.IsAbs(filePath) {
		return "", fmt.Errorf("path traversal not allowed: %s", filePath)
	}

	// Clean the path and verify it stays within the plugin root
	cleaned := filepath.Clean(filePath)
	fullPath := filepath.Join(pluginRoot, cleaned)

	// After joining and cleaning, verify the result is still under pluginRoot
	absRoot, err := filepath.Abs(pluginRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve plugin path: %w", err)
	}
	absTarget, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}
	if !strings.HasPrefix(absTarget, absRoot+string(filepath.Separator)) && absTarget != absRoot {
		return "", fmt.Errorf("path traversal not allowed: %s", filePath)
	}

	// Exact match (file)
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		return fullPath, nil
	}

	// If it's a directory, apply SKILL.md convention only for skills/ subtree
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		rel, relErr := filepath.Rel(pluginRoot, fullPath)
		if relErr == nil {
			parts := strings.SplitN(rel, string(filepath.Separator), 2)
			if parts[0] == "skills" {
				skillFile := filepath.Join(fullPath, "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					return skillFile, nil
				}
				return "", fmt.Errorf("skill directory %q has no SKILL.md", filePath)
			}
		}
		return "", fmt.Errorf("%q is a directory; specify a file inside it", filePath)
	}

	// Try common extensions
	extensions := []string{".md", ".py", ".sh", ".js"}
	for _, ext := range extensions {
		candidate := fullPath + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("file not found: %s", filePath)
}

// showPluginFile reads and displays a file from within a plugin.
// Markdown files are rendered with glamour unless raw is true.
func showPluginFile(pluginRoot, filePath string, raw bool) error {
	resolved, err := resolvePluginFile(pluginRoot, filePath)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	if strings.HasSuffix(resolved, ".md") {
		fmt.Print(ui.RenderMarkdown(content, raw))
	} else {
		fmt.Print(content)
	}

	return nil
}
