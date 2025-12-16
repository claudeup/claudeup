// ABOUTME: Interactive wizard for creating profiles
// ABOUTME: Handles name validation, marketplace selection, plugin selection
package profile

import (
	"fmt"
	"regexp"
)

// ValidateName checks if a profile name is valid
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if name == "current" {
		return fmt.Errorf("'current' is a reserved name")
	}

	// Valid names: alphanumeric, hyphens, underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("profile name contains invalid characters (use letters, numbers, hyphens, underscores only)")
	}

	return nil
}
