// ABOUTME: Resolves and displays individual file contents within a plugin
// ABOUTME: Handles extension inference and skill directory conventions
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup/v4/internal/ui"
)

// resolvePluginFile resolves a relative file path within a plugin root directory.
// It tries: exact match, SKILL.md inside directory, then common extensions.
func resolvePluginFile(pluginRoot, filePath string) (string, error) {
	fullPath := filepath.Join(pluginRoot, filePath)

	// Exact match (file)
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		return fullPath, nil
	}

	// If it's a directory, look for SKILL.md inside (skill convention)
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		skillFile := filepath.Join(fullPath, "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			return skillFile, nil
		}
		return "", fmt.Errorf("directory %q has no SKILL.md -- specify a file inside it", filePath)
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
