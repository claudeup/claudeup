// ABOUTME: Scope validation and constants for Claude settings
// ABOUTME: Provides shared scope validation logic to avoid duplication

package claude

import "fmt"

// Valid scope constants
const (
	ScopeUser    = "user"
	ScopeProject = "project"
	ScopeLocal   = "local"
)

// ValidScopes is the ordered list of scopes (lowest to highest precedence)
var ValidScopes = []string{ScopeUser, ScopeProject, ScopeLocal}

// ScopePrecedence returns the precedence of a scope (higher = takes priority).
// Returns -1 for unknown scopes.
func ScopePrecedence(scope string) int {
	for i, s := range ValidScopes {
		if s == scope {
			return i
		}
	}
	return -1
}

// ValidateScope validates that the given scope is one of the valid values
func ValidateScope(scope string) error {
	for _, valid := range ValidScopes {
		if scope == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid scope %q: must be one of %v", scope, ValidScopes)
}
