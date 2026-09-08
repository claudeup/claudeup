// ABOUTME: Recognises whole-arg environment variable references in MCP args
// ABOUTME: Converts between the profile form ($KEY) and Claude's ${KEY} form
package profile

import "strings"

// envRefName reports whether arg is a whole-arg environment variable
// reference in either the profile form ($KEY) or the placeholder form
// Claude Code expands at launch (${KEY}), and returns the variable name.
//
// Only whole-arg references count: apply has always expanded only args that
// are entirely a reference, and Claude Code stores args verbatim, so an
// embedded reference such as "Bearer ${KEY}" is left untouched by both
// directions. The name must be a valid identifier so a lone "$" or a
// positional "$1" is treated as a literal.
func envRefName(arg string) (string, bool) {
	if !strings.HasPrefix(arg, "$") {
		return "", false
	}
	name := arg[1:]
	if strings.HasPrefix(name, "{") {
		if !strings.HasSuffix(name, "}") {
			return "", false
		}
		name = name[1 : len(name)-1]
	}
	if !isEnvIdentifier(name) {
		return "", false
	}
	return name, true
}

// isEnvPlaceholder reports whether arg is a whole-arg ${KEY} placeholder,
// the form Claude Code expands from its environment at launch.
func isEnvPlaceholder(arg string) bool {
	if !strings.HasPrefix(arg, "${") {
		return false
	}
	_, ok := envRefName(arg)
	return ok
}

// envPlaceholder returns the ${KEY} form Claude Code expands at launch.
func envPlaceholder(name string) string {
	return "${" + name + "}"
}

// envReference returns the $KEY form used in profile JSON.
func envReference(name string) string {
	return "$" + name
}

// isEnvIdentifier reports whether name is a shell-style variable name:
// letters, digits and underscores, not starting with a digit.
func isEnvIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// mcpArgsEqual compares two MCP arg lists, treating a whole-arg $KEY and
// ${KEY} as the same reference. Apply writes the braced form and profiles
// carry the bare form, so a literal comparison would never match.
func mcpArgsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		nameA, okA := envRefName(a[i])
		nameB, okB := envRefName(b[i])
		if !okA || !okB || nameA != nameB {
			return false
		}
	}
	return true
}

// placeholderArgs returns a copy of args with every whole-arg $KEY or ${KEY}
// reference written as ${KEY}. The input slice is not modified.
func placeholderArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, len(args))
	for i, arg := range args {
		if name, ok := envRefName(arg); ok {
			out[i] = envPlaceholder(name)
		} else {
			out[i] = arg
		}
	}
	return out
}
