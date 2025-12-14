// ABOUTME: Custom help template for Cobra commands with lipgloss styling
// ABOUTME: Provides consistent, colorful help output across all commands
package ui

import (
	"strings"
	"text/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	// Help section styles
	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	helpHeadingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorInfo)

	helpCommandStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	helpFlagStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
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

// AddTemplateFuncs adds our custom template functions to a command
func AddTemplateFuncs(cmd *cobra.Command) {
	tmpl := template.New("help")
	tmpl.Funcs(template.FuncMap{
		"styleTitle":   styleTitle,
		"styleHeading": styleHeading,
		"styleCommand": styleCommand,
		"styleFlag":    styleFlag,
		"styleDesc":    styleDesc,
		"styleExample": styleExample,
	})
}

const helpTemplate = `{{if .Long}}{{.Long}}{{else}}{{.Short}}{{end}}

{{styleHeading "Usage:"}}
  {{styleCommand .UseLine}}{{if .HasAvailableSubCommands}}
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
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{styleHeading "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{styleHeading "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{styleCommand (rpad .CommandPath .CommandPathPadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{styleCommand (print .CommandPath " [command] --help")}}" for more information about a command.{{end}}
`

const usageTemplate = helpTemplate
