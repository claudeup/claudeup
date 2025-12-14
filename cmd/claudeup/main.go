// ABOUTME: Entry point for the claudeup CLI tool
// ABOUTME: Initializes and executes the root command
package main

import (
	"fmt"
	"os"

	"github.com/claudeup/claudeup/internal/commands"
	"github.com/claudeup/claudeup/internal/ui"
)

var version = "dev" // Injected at build time via -ldflags

func main() {
	commands.SetVersion(version)

	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err))
		os.Exit(1)
	}
}
