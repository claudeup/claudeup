// ABOUTME: Core types for managing local Claude Code extensions
// ABOUTME: Defines categories and provides validation
package local

import (
	"fmt"
	"sort"
)

// Category constants for local item types
const (
	CategoryAgents       = "agents"
	CategoryCommands     = "commands"
	CategorySkills       = "skills"
	CategoryHooks        = "hooks"
	CategoryRules        = "rules"
	CategoryOutputStyles = "output-styles"
)

var validCategories = map[string]bool{
	CategoryAgents:       true,
	CategoryCommands:     true,
	CategorySkills:       true,
	CategoryHooks:        true,
	CategoryRules:        true,
	CategoryOutputStyles: true,
}

// ValidateCategory checks if a category name is valid
func ValidateCategory(category string) error {
	if category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	if !validCategories[category] {
		return fmt.Errorf("invalid category %q, valid categories: %v", category, AllCategories())
	}
	return nil
}

// ProjectScopeCategories defines which categories Claude Code reads from project .claude/ directories
var ProjectScopeCategories = map[string]bool{
	CategoryAgents: true,
	CategoryRules:  true,
}

// ValidateProjectScope checks if a category is valid for project scope
func ValidateProjectScope(category string) error {
	if !ProjectScopeCategories[category] {
		return fmt.Errorf("category %q is not supported at project scope (only agents, rules)", category)
	}
	return nil
}

// AllCategories returns all valid category names sorted alphabetically
func AllCategories() []string {
	cats := make([]string, 0, len(validCategories))
	for cat := range validCategories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}
