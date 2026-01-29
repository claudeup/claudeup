// ABOUTME: Custom help template for Cobra commands with lipgloss styling
// ABOUTME: Provides consistent, colorful help output across all commands
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	// Help section styles
	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorInfo)

	helpHeadingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorInfo)

	helpCommandStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	helpFlagStyle = lipgloss.NewStyle().
			Foreground(ColorFlags)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	helpErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)
)

// SetupHelpTemplate configures custom help templates for the root command
// and all its subcommands
func SetupHelpTemplate(cmd *cobra.Command) {
	// Set custom usage template
	cmd.SetUsageTemplate(usageTemplate)

	// Set custom help template
	cmd.SetHelpTemplate(helpTemplate)

	// Add template functions
	cobra.AddTemplateFunc("styleTitle", styleTitle)
	cobra.AddTemplateFunc("styleHeading", styleHeading)
	cobra.AddTemplateFunc("styleCommand", styleCommand)
	cobra.AddTemplateFunc("styleFlag", styleFlag)
	cobra.AddTemplateFunc("styleDesc", styleDesc)
	cobra.AddTemplateFunc("styleExample", styleExample)
	cobra.AddTemplateFunc("styleError", styleError)
	cobra.AddTemplateFunc("styleLong", styleLong)
	cobra.AddTemplateFunc("styleFlagUsages", styleFlagUsages)
	cobra.AddTemplateFunc("styleUseLine", styleUseLine)

	// Set custom error message prefix with styling
	cmd.SetErrPrefix(helpErrorStyle.Render("Error:"))

	// Silence the default error printing - we'll handle it ourselves
	cmd.SilenceErrors = true

	// Set custom flag error function for styled flag errors
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fmt.Errorf("%s", err.Error())
	})
}

// FormatError returns a styled error message for CLI output
func FormatError(err error) string {
	return helpErrorStyle.Render("Error:") + " " + err.Error()
}

func styleTitle(s string) string {
	return helpTitleStyle.Render(s)
}

func styleHeading(s string) string {
	return helpHeadingStyle.Render(s)
}

func styleCommand(s string) string {
	return helpCommandStyle.Render(s)
}

func styleFlag(s string) string {
	return helpFlagStyle.Render(s)
}

func styleDesc(s string) string {
	return helpDescStyle.Render(s)
}

func styleError(s string) string {
	return helpErrorStyle.Render(s)
}

func styleLong(s string) string {
	// Style the long description with structure awareness
	lines := strings.Split(s, "\n")
	var styled []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed != "" {
			// First non-empty line is the summary - style as title
			styled = append(styled, helpTitleStyle.Render(line))
		} else if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
			// Lines ending with colon are sub-headings (e.g., "Shows:")
			styled = append(styled, helpHeadingStyle.Render(line))
		} else if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "•") {
			// Bullet points - keep muted
			styled = append(styled, helpDescStyle.Render(line))
		} else if trimmed == "" {
			// Empty lines
			styled = append(styled, line)
		} else {
			// Other text - muted
			styled = append(styled, helpDescStyle.Render(line))
		}
	}
	return strings.Join(styled, "\n")
}

func styleExample(s string) string {
	// Indent and style example lines
	lines := strings.Split(s, "\n")
	var styled []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Comment line
			styled = append(styled, helpDescStyle.Render(line))
		} else if strings.TrimSpace(line) != "" {
			// Command line
			styled = append(styled, helpCommandStyle.Render(line))
		} else {
			styled = append(styled, line)
		}
	}
	return strings.Join(styled, "\n")
}

func styleUseLine(s string) string {
	// Style the usage line, making [flags] muted like [command]
	if strings.HasSuffix(s, " [flags]") {
		cmdPart := s[:len(s)-len(" [flags]")]
		return helpCommandStyle.Render(cmdPart) + " " + helpDescStyle.Render("[flags]")
	}
	return helpCommandStyle.Render(s)
}

func styleFlagUsages(s string) string {
	// Style flag usage lines from Cobra's FlagUsages output
	// Format: "  -f, --flag type   Description text"
	lines := strings.Split(s, "\n")
	var styled []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			styled = append(styled, line)
			continue
		}

		// Find where the description starts (after multiple spaces following the flag)
		// Flags start with spaces, then -, the description follows after 2+ spaces
		trimmed := strings.TrimLeft(line, " ")
		indent := line[:len(line)-len(trimmed)]

		// Find the split point: look for 2+ consecutive spaces after the flag definition
		splitIdx := -1
		inSpaces := false
		spaceStart := 0
		for i, c := range trimmed {
			if c == ' ' {
				if !inSpaces {
					inSpaces = true
					spaceStart = i
				}
				if i-spaceStart >= 2 {
					splitIdx = spaceStart
					break
				}
			} else {
				inSpaces = false
			}
		}

		if splitIdx > 0 {
			flagPart := trimmed[:splitIdx]
			descPart := strings.TrimLeft(trimmed[splitIdx:], " ")
			styled = append(styled, indent+helpFlagStyle.Render(flagPart)+"  "+helpDescStyle.Render(descPart))
		} else {
			// No description found, just style the whole line as a flag
			styled = append(styled, indent+helpFlagStyle.Render(trimmed))
		}
	}
	return strings.Join(styled, "\n")
}

const helpTemplate = `{{if .Long}}{{styleLong .Long}}{{else}}{{styleTitle .Short}}{{end}}

{{styleHeading "Usage:"}}
  {{styleUseLine .UseLine}}{{if .HasAvailableSubCommands}}
  {{styleCommand .CommandPath}} {{styleDesc "[command]"}}{{end}}{{if gt (len .Aliases) 0}}

{{styleHeading "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{styleHeading "Examples:"}}
{{styleExample .Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{styleHeading "Available Commands:"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{styleCommand (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{styleHeading .Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{styleCommand (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{styleHeading "Additional Commands:"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{styleCommand (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{styleHeading "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces | styleFlagUsages}}{{end}}{{if .HasAvailableInheritedFlags}}

{{styleHeading "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces | styleFlagUsages}}{{end}}{{if .HasHelpSubCommands}}

{{styleHeading "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{styleCommand (rpad .CommandPath .CommandPathPadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{styleCommand (print .CommandPath " [command] --help")}}" for more information about a command.{{end}}
`

const usageTemplate = helpTemplate
