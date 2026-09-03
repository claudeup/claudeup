// ABOUTME: Tests for redacting MCP server secrets by matching env var values
// ABOUTME: Covers whole-arg matching, filtering, ambiguity, and warnings
package profile

import (
	"strings"
	"testing"
)

func TestRedactMCPSecretsFromEnv(t *testing.T) {
	t.Run("replaces whole-arg match with $VAR and records env source", func(t *testing.T) {
		p := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					MCPServers: []MCPServer{
						{
							Name:    "my-server",
							Command: "npx",
							Args:    []string{"-y", "@my/mcp", "--token", "sk-secret-12345"},
						},
					},
				},
			},
		}

		warnings := p.RedactMCPSecretsFromEnv([]string{"MY_TOKEN=sk-secret-12345"})

		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[3] != "$MY_TOKEN" {
			t.Errorf("expected arg[3] to be $MY_TOKEN, got %q", srv.Args[3])
		}
		ref, ok := srv.Secrets["MY_TOKEN"]
		if !ok {
			t.Fatalf("expected MY_TOKEN secret entry, got %v", srv.Secrets)
		}
		if len(ref.Sources) != 1 || ref.Sources[0].Type != "env" || ref.Sources[0].Key != "MY_TOKEN" {
			t.Errorf("expected single env source for MY_TOKEN, got %+v", ref.Sources)
		}
	})

	t.Run("redacts legacy flat MCPServers and every scope", func(t *testing.T) {
		p := &Profile{
			Name: "test",
			MCPServers: []MCPServer{
				{Name: "flat", Command: "cmd", Args: []string{"flat-secret-value"}},
			},
			PerScope: &PerScopeSettings{
				User:    &ScopeSettings{MCPServers: []MCPServer{{Name: "u", Command: "cmd", Args: []string{"user-secret-value"}}}},
				Project: &ScopeSettings{MCPServers: []MCPServer{{Name: "p", Command: "cmd", Args: []string{"project-secret-value"}}}},
				Local:   &ScopeSettings{MCPServers: []MCPServer{{Name: "l", Command: "cmd", Args: []string{"local-secret-value"}}}},
			},
		}

		p.RedactMCPSecretsFromEnv([]string{
			"FLAT_SECRET=flat-secret-value",
			"USER_SECRET=user-secret-value",
			"PROJECT_SECRET=project-secret-value",
			"LOCAL_SECRET=local-secret-value",
		})

		checks := []struct {
			name string
			srv  MCPServer
			want string
		}{
			{"flat", p.MCPServers[0], "$FLAT_SECRET"},
			{"user", p.PerScope.User.MCPServers[0], "$USER_SECRET"},
			{"project", p.PerScope.Project.MCPServers[0], "$PROJECT_SECRET"},
			{"local", p.PerScope.Local.MCPServers[0], "$LOCAL_SECRET"},
		}
		for _, c := range checks {
			if c.srv.Args[0] != c.want {
				t.Errorf("%s: expected %q, got %q", c.name, c.want, c.srv.Args[0])
			}
			if _, ok := c.srv.Secrets[strings.TrimPrefix(c.want, "$")]; !ok {
				t.Errorf("%s: expected secrets entry for %s", c.name, c.want)
			}
		}
	})

	t.Run("leaves existing $VAR references and their metadata untouched", func(t *testing.T) {
		p := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					MCPServers: []MCPServer{
						{
							Name:    "my-server",
							Command: "npx",
							Args:    []string{"--token", "$MY_TOKEN"},
							Secrets: map[string]SecretRef{
								"MY_TOKEN": {
									Description: "curated",
									Sources:     []SecretSource{{Type: "1password", Ref: "op://vault/item/field"}},
								},
							},
						},
					},
				},
			},
		}

		// Even though MY_TOKEN is in the environment, the curated entry must win.
		p.RedactMCPSecretsFromEnv([]string{"MY_TOKEN=sk-secret-12345"})

		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[1] != "$MY_TOKEN" {
			t.Errorf("expected $MY_TOKEN to be kept, got %q", srv.Args[1])
		}
		ref := srv.Secrets["MY_TOKEN"]
		if ref.Description != "curated" || len(ref.Sources) != 1 || ref.Sources[0].Type != "1password" {
			t.Errorf("expected curated 1password source to be preserved, got %+v", ref)
		}
	})

	t.Run("does not clobber a preserved secret entry when adding an env-inferred one", func(t *testing.T) {
		p := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					MCPServers: []MCPServer{
						{
							Name:    "my-server",
							Command: "npx",
							Args:    []string{"--a", "$A_TOKEN", "--b", "b-secret-value"},
							Secrets: map[string]SecretRef{
								"A_TOKEN": {Sources: []SecretSource{{Type: "keychain", Service: "svc", Account: "acct"}}},
							},
						},
					},
				},
			},
		}

		p.RedactMCPSecretsFromEnv([]string{"A_TOKEN=a-secret-value", "B_TOKEN=b-secret-value"})

		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[3] != "$B_TOKEN" {
			t.Errorf("expected arg[3] to be $B_TOKEN, got %q", srv.Args[3])
		}
		if len(srv.Secrets) != 2 {
			t.Fatalf("expected 2 secret entries, got %d: %v", len(srv.Secrets), srv.Secrets)
		}
		if srv.Secrets["A_TOKEN"].Sources[0].Type != "keychain" {
			t.Errorf("expected keychain source for A_TOKEN to survive, got %+v", srv.Secrets["A_TOKEN"])
		}
	})

	t.Run("ignores env values shorter than the minimum length", func(t *testing.T) {
		p := singleServerProfile([]string{"--port", "8080", "--flag", "true"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"PORT=8080", "FLAG=true"})

		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[1] != "8080" || srv.Args[3] != "true" {
			t.Errorf("short values must not be redacted, got %v", srv.Args)
		}
		if len(srv.Secrets) != 0 {
			t.Errorf("expected no secrets entries, got %v", srv.Secrets)
		}
	})

	t.Run("ignores env values that are absolute paths", func(t *testing.T) {
		p := singleServerProfile([]string{"--dir", "/Users/mark/projects"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"PROJECT_ROOT=/Users/mark/projects"})

		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
		if p.PerScope.User.MCPServers[0].Args[1] != "/Users/mark/projects" {
			t.Errorf("absolute path must not be redacted, got %v", p.PerScope.User.MCPServers[0].Args)
		}
	})

	t.Run("ignores well-known shell variables", func(t *testing.T) {
		p := singleServerProfile([]string{"--host", "build-host-01", "--lang", "en_US.UTF-8"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"HOSTNAME=build-host-01", "LANG=en_US.UTF-8"})

		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[1] != "build-host-01" || srv.Args[3] != "en_US.UTF-8" {
			t.Errorf("denylisted vars must not be redacted, got %v", srv.Args)
		}
	})

	t.Run("refuses ambiguous matches and warns with candidate names", func(t *testing.T) {
		p := singleServerProfile([]string{"--token", "shared-secret-value"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"TOKEN_A=shared-secret-value", "TOKEN_B=shared-secret-value"})

		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[1] != "shared-secret-value" {
			t.Errorf("ambiguous match must be left as-is, got %q", srv.Args[1])
		}
		if len(srv.Secrets) != 0 {
			t.Errorf("ambiguous match must not add secrets, got %v", srv.Secrets)
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
		}
		if !strings.Contains(warnings[0], "my-server") || !strings.Contains(warnings[0], "TOKEN_A") || !strings.Contains(warnings[0], "TOKEN_B") {
			t.Errorf("warning should name server and both candidates, got %q", warnings[0])
		}
		if strings.Contains(warnings[0], "shared-secret-value") {
			t.Errorf("warning must not echo the secret value, got %q", warnings[0])
		}
	})

	t.Run("warns on substring match instead of rewriting", func(t *testing.T) {
		p := singleServerProfile([]string{"--header", "Authorization: Bearer sk-secret-12345"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"MY_TOKEN=sk-secret-12345"})

		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[1] != "Authorization: Bearer sk-secret-12345" {
			t.Errorf("substring match must not rewrite the arg, got %q", srv.Args[1])
		}
		if len(srv.Secrets) != 0 {
			t.Errorf("substring match must not add secrets, got %v", srv.Secrets)
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
		}
		if !strings.Contains(warnings[0], "my-server") || !strings.Contains(warnings[0], "MY_TOKEN") {
			t.Errorf("warning should name server and env var, got %q", warnings[0])
		}
		if strings.Contains(warnings[0], "sk-secret-12345") {
			t.Errorf("warning must not echo the secret value, got %q", warnings[0])
		}
	})

	t.Run("warns on secret-looking arg with no env match", func(t *testing.T) {
		p := singleServerProfile([]string{"--token", "ghp_abcdefghijklmnop123456"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"UNRELATED=some-other-value"})

		srv := p.PerScope.User.MCPServers[0]
		if srv.Args[1] != "ghp_abcdefghijklmnop123456" {
			t.Errorf("unmatched arg must be left as-is, got %q", srv.Args[1])
		}
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
		}
		if !strings.Contains(warnings[0], "my-server") || !strings.Contains(warnings[0], "ghp_") {
			t.Errorf("warning should name server and matched prefix, got %q", warnings[0])
		}
		if strings.Contains(warnings[0], "ghp_abcdefghijklmnop123456") {
			t.Errorf("warning must not echo the secret value, got %q", warnings[0])
		}
	})

	t.Run("does not warn on ordinary args", func(t *testing.T) {
		p := singleServerProfile([]string{"-y", "@modelcontextprotocol/server-github", "--verbose"})

		warnings := p.RedactMCPSecretsFromEnv([]string{"UNRELATED=some-other-value"})

		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
	})

	t.Run("composes with PreserveMCPSecrets on re-save", func(t *testing.T) {
		existing := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					MCPServers: []MCPServer{
						{
							Name:    "my-server",
							Command: "npx",
							Args:    []string{"--token", "$MY_TOKEN", "--other", "$OTHER"},
							Secrets: map[string]SecretRef{
								"MY_TOKEN": {Sources: []SecretSource{{Type: "1password", Ref: "op://vault/item/field"}}},
							},
						},
					},
				},
			},
		}
		snapshot := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					MCPServers: []MCPServer{
						{
							Name:    "my-server",
							Command: "npx",
							Args:    []string{"--token", "sk-secret-12345", "--other", "other-secret-value"},
						},
					},
				},
			},
		}

		snapshot.PreserveMCPSecrets(existing)
		snapshot.RedactMCPSecretsFromEnv([]string{"MY_TOKEN=sk-secret-12345", "OTHER=other-secret-value"})

		srv := snapshot.PerScope.User.MCPServers[0]
		if srv.Args[1] != "$MY_TOKEN" || srv.Args[3] != "$OTHER" {
			t.Errorf("expected both args redacted, got %v", srv.Args)
		}
		if srv.Secrets["MY_TOKEN"].Sources[0].Type != "1password" {
			t.Errorf("preserved 1password source must win over env inference, got %+v", srv.Secrets["MY_TOKEN"])
		}
		if src := srv.Secrets["OTHER"].Sources; len(src) != 1 || src[0].Type != "env" || src[0].Key != "OTHER" {
			t.Errorf("expected env source for OTHER, got %+v", srv.Secrets["OTHER"])
		}
	})

	t.Run("nil profile is a no-op", func(t *testing.T) {
		var nilProfile *Profile
		if warnings := nilProfile.RedactMCPSecretsFromEnv([]string{"X=long-enough-value"}); warnings != nil {
			t.Errorf("expected nil warnings, got %v", warnings)
		}
	})
}

func singleServerProfile(args []string) *Profile {
	return &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "my-server", Command: "cmd", Args: args},
				},
			},
		},
	}
}
