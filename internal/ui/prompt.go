// ABOUTME: Interactive prompt UI functions for user input
// ABOUTME: Handles numbered selection lists and yes/no confirmations
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/claudeup/claudeup/internal/config"
)

// SelectFromList prompts user to select items from numbered list
// Supports: "1 3 5", "all", "none", "1-5"
func SelectFromList(prompt string, items []string) ([]string, error) {
	if config.YesFlag {
		return items, nil // Select all when --yes
	}

	if len(items) == 0 {
		return []string{}, nil
	}

	fmt.Println(prompt)
	for i, item := range items {
		fmt.Printf("  %d) %s\n", i+1, item)
	}
	fmt.Println()
	fmt.Print("Enter numbers (1 3 5), 'all', or 'none': ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "none" || input == "" {
		return []string{}, nil
	}

	if input == "all" {
		return items, nil
	}

	// Parse numbers
	selected := []string{}
	parts := strings.Fields(input)
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 1 || num > len(items) {
			fmt.Printf("Invalid selection: %s (skipping)\n", part)
			continue
		}
		selected = append(selected, items[num-1])
	}

	return selected, nil
}

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
