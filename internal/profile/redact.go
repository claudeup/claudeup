// ABOUTME: Redacts MCP server secrets on save by matching env var values
// ABOUTME: Replaces whole-arg matches with $VAR refs and warns on the rest
package profile

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// minEnvSecretLen is the shortest env var value considered for redaction.
// Shorter values (ports, flags, short words) are too likely to collide with
// ordinary args.
const minEnvSecretLen = 8

// ignoredEnvVars are well-known shell and tooling variables whose values are
// never secrets but routinely show up in args (paths, hostnames, locales).
var ignoredEnvVars = map[string]bool{
	"HOME": true, "PATH": true, "PWD": true, "OLDPWD": true, "SHELL": true,
	"USER": true, "LOGNAME": true, "HOSTNAME": true,
	"TMPDIR": true, "TMP": true, "TEMP": true,
	"LANG": true, "LANGUAGE": true, "TERM": true, "COLORTERM": true,
	"EDITOR": true, "VISUAL": true, "PAGER": true, "DISPLAY": true,
	"SHLVL": true, "MAIL": true,
	"SSH_AUTH_SOCK": true, "SSH_CONNECTION": true, "SSH_CLIENT": true, "SSH_TTY": true,
	"GOPATH": true, "GOROOT": true, "GOFLAGS": true, "GOMODCACHE": true,
	"CLAUDE_CONFIG_DIR": true, "CLAUDEUP_HOME": true, "NO_COLOR": true,
}

// ignoredEnvPrefixes extend ignoredEnvVars to whole families of variables.
var ignoredEnvPrefixes = []string{"LC_", "XDG_", "TERM_"}

// secretPrefixes are well-known token formats. An arg starting with one of
// these that matched no env var is reported so the user can redact it by hand.
var secretPrefixes = []string{"sk-", "ghp_", "github_pat_", "AKIA", "xox"}

// RedactMCPSecretsFromEnv replaces MCP server args whose value is exactly the
// value of an environment variable with a $VAR reference and records an env
// SecretRef for it, so a freshly saved profile never carries the plaintext
// secret. environ holds "KEY=value" entries as returned by os.Environ().
//
// Only whole-arg matches are rewritten, because apply only expands args that
// are entirely $VAR. Args already holding a $VAR reference are left alone, and
// an existing Secrets entry is never overwritten, so curated sources survive
// when this runs after PreserveMCPSecrets on a re-save.
//
// Returns warnings for args left in plaintext that deserve a look: values
// matching several env vars, values that appear inside a longer arg, and
// values with a well-known secret prefix that matched nothing. Warnings never
// include the arg value itself.
func (p *Profile) RedactMCPSecretsFromEnv(environ []string) []string {
	if p == nil {
		return nil
	}

	candidates := envSecretCandidates(environ)

	var warnings []string
	warnings = append(warnings, redactServerSecrets(p.MCPServers, candidates)...)
	if p.PerScope != nil {
		for _, s := range []*ScopeSettings{p.PerScope.User, p.PerScope.Project, p.PerScope.Local} {
			if s != nil {
				warnings = append(warnings, redactServerSecrets(s.MCPServers, candidates)...)
			}
		}
	}
	return warnings
}

// envSecretCandidates maps env var values to the sorted names holding them,
// skipping values that are too short, absolute paths, or from ignored vars.
func envSecretCandidates(environ []string) map[string][]string {
	byValue := make(map[string][]string)
	for _, entry := range environ {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || name == "" || len(value) < minEnvSecretLen {
			continue
		}
		if isIgnoredEnvVar(name) || filepath.IsAbs(value) {
			continue
		}
		byValue[value] = append(byValue[value], name)
	}
	for _, names := range byValue {
		sort.Strings(names)
	}
	return byValue
}

func isIgnoredEnvVar(name string) bool {
	if ignoredEnvVars[name] {
		return true
	}
	for _, prefix := range ignoredEnvPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// redactServerSecrets rewrites whole-arg env matches in place and collects
// warnings for args that stay in plaintext.
func redactServerSecrets(servers []MCPServer, candidates map[string][]string) []string {
	var warnings []string
	for i := range servers {
		srv := &servers[i]
		for j, arg := range srv.Args {
			if strings.HasPrefix(arg, "$") {
				continue
			}
			names := candidates[arg]
			switch {
			case len(names) == 1:
				name := names[0]
				srv.Args[j] = "$" + name
				if srv.Secrets == nil {
					srv.Secrets = make(map[string]SecretRef)
				}
				if _, exists := srv.Secrets[name]; !exists {
					srv.Secrets[name] = SecretRef{Sources: []SecretSource{{Type: "env", Key: name}}}
				}
			case len(names) > 1:
				warnings = append(warnings, fmt.Sprintf(
					"server %q arg %d matches multiple environment variables (%s); left as plaintext -- review saved profile",
					srv.Name, j, strings.Join(names, ", ")))
			default:
				if w := plaintextArgWarning(srv.Name, j, arg, candidates); w != "" {
					warnings = append(warnings, w)
				}
			}
		}
	}
	return warnings
}

// plaintextArgWarning reports an arg that stays in plaintext but looks like it
// carries a secret: it contains an env var's value, or starts with a known
// token prefix. The value itself is never included in the message.
func plaintextArgWarning(server string, index int, arg string, candidates map[string][]string) string {
	var contained []string
	for value, names := range candidates {
		if strings.Contains(arg, value) {
			contained = append(contained, names...)
		}
	}
	if len(contained) > 0 {
		sort.Strings(contained)
		return fmt.Sprintf(
			"server %q arg %d contains the value of environment variable %s but is not an exact match; left as plaintext -- review saved profile",
			server, index, strings.Join(contained, ", "))
	}

	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(arg, prefix) {
			return fmt.Sprintf(
				"server %q arg %d looks like a secret (%s...) but matches no environment variable; left as plaintext -- see docs on redacting secrets",
				server, index, prefix)
		}
	}
	return ""
}
