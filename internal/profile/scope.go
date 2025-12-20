// ABOUTME: Defines the Scope type for profile application targets
// ABOUTME: Supports user, project, and local scopes for Claude Code configuration
package profile

import "fmt"

// Scope represents where a profile should be applied
type Scope string

const (
	// ScopeUser applies profile at user level (~/.claude/)
	ScopeUser Scope = "user"
	// ScopeProject applies profile at project level (.mcp.json + .claudeup.json)
	ScopeProject Scope = "project"
	// ScopeLocal applies profile locally for this project only (~/.claudeup/projects.json)
	ScopeLocal Scope = "local"
)

func (s Scope) String() string {
	return string(s)
}

// IsValid returns true if the scope is a recognized value
func (s Scope) IsValid() bool {
	switch s {
	case ScopeUser, ScopeProject, ScopeLocal:
		return true
	default:
		return false
	}
}

// ParseScope converts a string to a Scope, returning an error for invalid values
func ParseScope(s string) (Scope, error) {
	scope := Scope(s)
	if !scope.IsValid() {
		return "", fmt.Errorf("invalid scope: %q (must be user, project, or local)", s)
	}
	return scope, nil
}
