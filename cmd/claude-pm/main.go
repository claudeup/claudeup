// ABOUTME: Entry point for the claude-pm CLI tool
// ABOUTME: Initializes and executes the root command
package main

import (
	"fmt"
	"os"

	"github.com/malston/claude-pm/internal/commands"
)

var version = "dev" // Injected at build time via -ldflags

func main() {
	commands.SetVersion(version)

	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
