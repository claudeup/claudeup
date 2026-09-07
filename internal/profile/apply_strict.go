// ABOUTME: Pre-flight check for extensions referenced by a profile
// ABOUTME: Reports missing extensions without changing anything on disk
package profile

import (
	"fmt"
	"path/filepath"

	"github.com/claudeup/claudeup/v5/internal/ext"
)

// extensionCategoryItems pairs an extension category with the patterns a
// profile lists for it.
type extensionCategoryItems struct {
	category string
	patterns []string
}

// extensionCategories flattens an ExtensionSettings into per-category pattern
// lists in the order apply processes them.
func extensionCategories(items *ExtensionSettings) []extensionCategoryItems {
	return []extensionCategoryItems{
		{ext.CategoryAgents, items.Agents},
		{ext.CategoryCommands, items.Commands},
		{ext.CategorySkills, items.Skills},
		{ext.CategoryHooks, items.Hooks},
		{ext.CategoryRules, items.Rules},
		{ext.CategoryOutputStyles, items.OutputStyles},
	}
}

// MissingExtensions returns every extension pattern the profile references
// that matches nothing in extension storage. It changes nothing on disk.
//
// Legacy profiles are checked with the user-scope (symlink) matching rules.
// Multi-scope profiles are checked per scope: user scope uses the symlink
// rules, project and local scopes use the copy rules, mirroring apply.
// Entries are "category/pattern"; multi-scope entries are prefixed "<scope>: ".
func MissingExtensions(p *Profile, claudeDir, claudeupHome string) ([]string, error) {
	if p == nil {
		return nil, nil
	}

	if !p.IsMultiScope() {
		return missingExtensionsForScope(p.Extensions, ScopeUser, claudeDir, claudeupHome)
	}

	scopes := []struct {
		scope    Scope
		settings *ScopeSettings
	}{
		{ScopeUser, p.PerScope.User},
		{ScopeProject, p.PerScope.Project},
		{ScopeLocal, p.PerScope.Local},
	}

	var missing []string
	for _, s := range scopes {
		if s.settings == nil {
			continue
		}
		scopeMissing, err := missingExtensionsForScope(s.settings.Extensions, s.scope, claudeDir, claudeupHome)
		if err != nil {
			return nil, err
		}
		for _, item := range scopeMissing {
			missing = append(missing, fmt.Sprintf("%s: %s", s.scope, item))
		}
	}

	return missing, nil
}

// missingExtensionsForScope resolves one scope's extension patterns with the
// matching rules that scope uses at apply time and returns the unmatched
// patterns in "category/pattern" format.
func missingExtensionsForScope(items *ExtensionSettings, scope Scope, claudeDir, claudeupHome string) ([]string, error) {
	if items == nil {
		return nil, nil
	}

	extDir := filepath.Join(claudeupHome, "ext")
	manager := ext.NewManager(claudeDir, claudeupHome)

	var missing []string
	for _, ci := range extensionCategories(items) {
		if len(ci.patterns) == 0 {
			continue
		}

		var notFound []string
		var err error
		if scope == ScopeProject || scope == ScopeLocal {
			_, notFound, err = ext.ResolveForProject(extDir, ci.category, ci.patterns)
		} else {
			_, notFound, err = manager.Resolve(ci.category, ci.patterns)
		}
		if err != nil {
			return nil, fmt.Errorf("resolve %s extensions: %w", ci.category, err)
		}

		for _, pattern := range notFound {
			missing = append(missing, ci.category+"/"+pattern)
		}
	}

	return missing, nil
}
