// ABOUTME: Interactive wizard for creating profiles
// ABOUTME: Handles name validation, marketplace selection, plugin selection
package profile

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
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

// PromptForName prompts the user to enter a profile name
// Returns the validated name or an error
func PromptForName() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Profile name: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		name := strings.TrimSpace(input)
		if name == "" {
			continue // Re-prompt on empty input
		}

		if err := ValidateName(name); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue // Re-prompt on validation error
		}

		return name, nil
	}
}
