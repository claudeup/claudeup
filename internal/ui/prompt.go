// ABOUTME: Interactive prompt UI functions for user input
// ABOUTME: Handles yes/no confirmations
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/claudeup/claudeup/v5/internal/config"
)

// ConfirmYesNo prompts for Y/n confirmation
func ConfirmYesNo(prompt string) (bool, error) {
	if config.YesFlag {
		return true, nil
	}

	fmt.Printf("%s [Y/n]: ", prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" || input == "y" || input == "yes" {
		return true, nil
	}

	return false, nil
}

// PromptYesNo prompts for yes/no confirmation with configurable default
func PromptYesNo(prompt string, defaultYes bool) bool {
	if config.YesFlag {
		return defaultYes
	}

	var hint string
	if defaultYes {
		hint = "[Y/n]"
	} else {
		hint = "[y/N]"
	}

	fmt.Printf("%s %s: ", prompt, hint)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// ValidateTypedConfirmation checks if input matches expected (case-insensitive)
func ValidateTypedConfirmation(input, expected string) bool {
	return strings.EqualFold(strings.TrimSpace(input), expected)
}
