// ABOUTME: Interactive wizard for creating profiles
// ABOUTME: Handles name validation, marketplace selection, plugin selection
package profile

import (
	"fmt"
	"regexp"
)

// validNameRegex matches valid profile names: alphanumeric, hyphens, underscores
var validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateName checks if a profile name is valid
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if name == "current" {
		return fmt.Errorf("'current' is a reserved name")
	}

	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("profile name contains invalid characters (use letters, numbers, hyphens, underscores only)")
	}

	return nil
}

// GetAvailableMarketplaces returns all available marketplaces for selection
// Currently returns only embedded marketplaces (wshobson/agents)
func GetAvailableMarketplaces() []Marketplace {
	// For V1, return hardcoded list of known marketplaces
	// Future: could load from registry or config
	return []Marketplace{
		{Source: "github", Repo: "wshobson/agents"},
	}
}
