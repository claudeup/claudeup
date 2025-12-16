// ABOUTME: Interactive wizard for creating profiles
// ABOUTME: Handles name validation, marketplace selection, plugin selection
package profile

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
			fmt.Println("Error: profile name cannot be empty")
			continue // Re-prompt on empty input
		}

		if err := ValidateName(name); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue // Re-prompt on validation error
		}

		return name, nil
	}
}

// SelectMarketplaces prompts user to select marketplaces
// Returns selected marketplaces or error
func SelectMarketplaces(available []Marketplace) ([]Marketplace, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no marketplaces available")
	}

	// Check if gum is available
	if _, err := exec.LookPath("gum"); err != nil {
		return fallbackMarketplaceSelection(available)
	}

	// Build gum command with marketplace choices
	args := []string{"choose", "--no-limit", "--header=Select marketplaces:"}
	for _, m := range available {
		args = append(args, m.DisplayName())
	}

	cmd := exec.Command("gum", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		// User cancelled or gum error
		return nil, fmt.Errorf("marketplace selection cancelled")
	}

	// Parse selected marketplace names
	selectedNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(selectedNames) == 0 || selectedNames[0] == "" {
		return nil, fmt.Errorf("no marketplaces selected")
	}

	// Map selected names back to Marketplace structs
	selected := make([]Marketplace, 0)
	for _, name := range selectedNames {
		for _, m := range available {
			if m.DisplayName() == name {
				selected = append(selected, m)
				break
			}
		}
	}

	return selected, nil
}

// fallbackMarketplaceSelection provides simple numbered menu when gum unavailable
func fallbackMarketplaceSelection(available []Marketplace) ([]Marketplace, error) {
	fmt.Println("\nSelect marketplaces (enter numbers separated by commas):")
	for i, m := range available {
		fmt.Printf("  %d) %s\n", i+1, m.DisplayName())
	}
	fmt.Print("\nYour selection: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// Parse comma-separated numbers
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("no marketplaces selected")
	}

	parts := strings.Split(input, ",")
	selected := make([]Marketplace, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		idx, err := strconv.Atoi(part)
		if err != nil || idx < 1 || idx > len(available) {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}
		selected = append(selected, available[idx-1])
	}

	return selected, nil
}
