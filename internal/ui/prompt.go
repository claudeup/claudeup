// ABOUTME: Interactive prompt UI functions for user input
// ABOUTME: Handles multi-select lists and yes/no confirmations
package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/claudeup/claudeup/internal/config"
)

// ErrUserCancelled is returned when user cancels a prompt with Ctrl+C
var ErrUserCancelled = errors.New("cancelled by user")

// SelectFromList prompts user to select items from a multi-select list
// All items are selected by default; press enter to confirm, space to toggle
func SelectFromList(prompt string, items []string) ([]string, error) {
	if config.YesFlag {
		return items, nil // Select all when --yes
	}

	if len(items) == 0 {
		return []string{}, nil
	}

	// Pre-select all items by default
	var selected []string
	multiSelect := &survey.MultiSelect{
		Message: prompt,
		Options: items,
		Default: items,
		Help:    "↑/↓ move, space toggle, enter confirm",
	}

	err := survey.AskOne(multiSelect, &selected)
	if err != nil {
		if err == terminal.InterruptErr {
			return nil, ErrUserCancelled
		}
		return nil, err
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
