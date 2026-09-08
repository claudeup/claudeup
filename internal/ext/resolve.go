// ABOUTME: Resolves item names with extension inference
// ABOUTME: Handles partial matches and agent group resolution
package ext

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveItemName resolves an item name, allowing partial matches without extension.
// Returns the full item name if found, error otherwise.
// For agents, returns 'group/agent.md' format.
func (m *Manager) ResolveItemName(category, item string) (string, error) {
	if err := ValidateCategory(category); err != nil {
		return "", err
	}

	if category == CategoryAgents {
		return m.resolveAgentName(item)
	}

	return m.resolveFlatItem(category, item)
}

// Resolve reports which patterns match items in extension storage using the
// same rules as Enable (wildcards, extension inference, directory expansion,
// skill directories). It changes nothing on disk.
// Returns (matched items, not found patterns, error).
func (m *Manager) Resolve(category string, patterns []string) ([]string, []string, error) {
	if err := ValidateCategory(category); err != nil {
		return nil, nil, err
	}

	allItems, err := m.ListItems(category)
	if err != nil {
		return nil, nil, fmt.Errorf("list %s items: %w", category, err)
	}

	var matched []string
	var notFound []string
	for _, pattern := range patterns {
		items, _, found := m.resolvePattern(category, pattern, allItems)
		if !found {
			notFound = append(notFound, pattern)
			continue
		}
		matched = append(matched, items...)
	}

	return matched, notFound, nil
}

func (m *Manager) resolveFlatItem(category, item string) (string, error) {
	libPath := filepath.Join(m.extDir, category, item)

	// Try exact match first
	if _, err := os.Stat(libPath); err == nil {
		return item, nil
	}

	// Try common extensions
	extensions := []string{".md", ".py", ".sh", ".js"}
	for _, ext := range extensions {
		fullName := item + ext
		fullPath := filepath.Join(m.extDir, category, fullName)
		if _, err := os.Stat(fullPath); err == nil {
			return fullName, nil
		}
	}

	return "", fmt.Errorf("item not found: %s/%s", category, item)
}

func (m *Manager) resolveAgentName(item string) (string, error) {
	agentsDir := filepath.Join(m.extDir, CategoryAgents)

	// Handle 'group/agent' format
	if strings.Contains(item, "/") {
		parts := strings.SplitN(item, "/", 2)
		group, agent := parts[0], parts[1]

		if !strings.HasSuffix(agent, ".md") {
			agent = agent + ".md"
		}

		fullPath := filepath.Join(agentsDir, group, agent)
		if _, err := os.Stat(fullPath); err == nil {
			return group + "/" + agent, nil
		}
		return "", fmt.Errorf("agent not found: agents/%s/%s", group, agent)
	}

	// Try as flat agent first
	agentName := item
	if !strings.HasSuffix(agentName, ".md") {
		agentName = agentName + ".md"
	}

	flatPath := filepath.Join(agentsDir, agentName)
	if _, err := os.Stat(flatPath); err == nil {
		return agentName, nil
	}

	// Search all groups for the agent
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return "", fmt.Errorf("agent not found: agents/%s", item)
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		groupPath := filepath.Join(agentsDir, entry.Name(), agentName)
		if _, err := os.Stat(groupPath); err == nil {
			return entry.Name() + "/" + agentName, nil
		}
	}

	return "", fmt.Errorf("agent not found: agents/%s", item)
}
