// ABOUTME: Tests for profile include resolution (composable stacks)
// ABOUTME: Validates cycle detection, merge strategies, and pure stack validation
package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockLoader implements ProfileLoader for testing
type mockLoader struct {
	profiles map[string]*Profile
}

func (m *mockLoader) LoadProfile(name string) (*Profile, error) {
	p, ok := m.profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// errorLoader always returns the configured error for any profile load.
type errorLoader struct {
	err error
}

func (e *errorLoader) LoadProfile(string) (*Profile, error) {
	return nil, e.err
}

func TestResolveIncludes_NoIncludes(t *testing.T) {
	p := &Profile{
		Name:    "simple",
		Plugins: []string{"a"},
	}
	loader := &mockLoader{}

	resolved, err := ResolveIncludes(p, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Name != "simple" {
		t.Errorf("name: got %q, want %q", resolved.Name, "simple")
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0] != "a" {
		t.Errorf("plugins: got %v, want [a]", resolved.Plugins)
	}
}

func TestResolveIncludes_EmptyIncludes(t *testing.T) {
	p := &Profile{
		Name:     "empty-stack",
		Includes: []string{},
		Plugins:  []string{"a"},
	}
	loader := &mockLoader{}

	resolved, err := ResolveIncludes(p, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty includes slice is not a stack -- returned as-is
	if resolved.Name != "empty-stack" {
		t.Errorf("name: got %q, want %q", resolved.Name, "empty-stack")
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0] != "a" {
		t.Errorf("plugins: got %v, want [a]", resolved.Plugins)
	}
}

func TestResolveIncludes_NilProfile(t *testing.T) {
	loader := &mockLoader{}
	_, err := ResolveIncludes(nil, loader)
	if err == nil {
		t.Fatal("expected error for nil profile, got nil")
	}
}

func TestResolveIncludes_SingleInclude(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"base": {
				Name: "base",
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "org/plugins"},
				},
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Plugins: []string{"plugin-a"},
					},
				},
			},
		},
	}

	stack := &Profile{
		Name:        "my-stack",
		Description: "test stack",
		Includes:    []string{"base"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Name != "my-stack" {
		t.Errorf("name: got %q, want %q", resolved.Name, "my-stack")
	}
	if resolved.Description != "test stack" {
		t.Errorf("description: got %q, want %q", resolved.Description, "test stack")
	}
	if len(resolved.Marketplaces) != 1 {
		t.Fatalf("marketplaces: got %d, want 1", len(resolved.Marketplaces))
	}
	if resolved.Marketplaces[0].Repo != "org/plugins" {
		t.Errorf("marketplace repo: got %q, want %q", resolved.Marketplaces[0].Repo, "org/plugins")
	}
	if resolved.PerScope == nil || resolved.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be set")
	}
	if len(resolved.PerScope.User.Plugins) != 1 || resolved.PerScope.User.Plugins[0] != "plugin-a" {
		t.Errorf("user plugins: got %v, want [plugin-a]", resolved.PerScope.User.Plugins)
	}
}

func TestResolveIncludes_MultipleIncludes_LeftToRight(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"first": {
				Name: "first",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						MCPServers: []MCPServer{
							{Name: "server-a", Command: "cmd-first"},
						},
					},
				},
			},
			"second": {
				Name: "second",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						MCPServers: []MCPServer{
							{Name: "server-a", Command: "cmd-second"}, // same name, should win
							{Name: "server-b", Command: "cmd-b"},
						},
					},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "my-stack",
		Includes: []string{"first", "second"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.PerScope == nil || resolved.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be set")
	}

	servers := resolved.PerScope.User.MCPServers
	// Should have 2 servers: server-a (from second, last-wins) and server-b
	if len(servers) != 2 {
		t.Fatalf("MCP servers: got %d, want 2", len(servers))
	}

	serverMap := make(map[string]string)
	for _, s := range servers {
		serverMap[s.Name] = s.Command
	}
	if serverMap["server-a"] != "cmd-second" {
		t.Errorf("server-a command: got %q, want %q (last-wins)", serverMap["server-a"], "cmd-second")
	}
	if serverMap["server-b"] != "cmd-b" {
		t.Errorf("server-b command: got %q, want %q", serverMap["server-b"], "cmd-b")
	}
}

func TestResolveIncludes_NestedIncludes(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"leaf": {
				Name:    "leaf",
				Plugins: []string{"leaf-plugin"},
			},
			"middle": {
				Name:     "middle",
				Includes: []string{"leaf"},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"middle"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0] != "leaf-plugin" {
		t.Errorf("plugins: got %v, want [leaf-plugin]", resolved.Plugins)
	}
}

func TestResolveIncludes_CycleDetection(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {Name: "a", Includes: []string{"b"}},
			"b": {Name: "b", Includes: []string{"a"}},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle: %v", err)
	}
	// Should show full cycle path
	if !strings.Contains(err.Error(), "a -> b -> a") {
		t.Errorf("error should show full cycle path, got: %v", err)
	}
}

func TestResolveIncludes_SelfCycle(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"self": {Name: "self", Includes: []string{"self"}},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"self"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestResolveIncludes_TransitiveCycle(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {Name: "a", Includes: []string{"b"}},
			"b": {Name: "b", Includes: []string{"c"}},
			"c": {Name: "c", Includes: []string{"a"}},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	// Should show full transitive cycle path
	if !strings.Contains(err.Error(), "a -> b -> c -> a") {
		t.Errorf("error should show full cycle path, got: %v", err)
	}
}

func TestResolveIncludes_DiamondPattern(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"shared": {
				Name:    "shared",
				Plugins: []string{"shared-plugin"},
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "org/shared"},
				},
			},
			"left": {
				Name:     "left",
				Includes: []string{"shared"},
			},
			"right": {
				Name:     "right",
				Includes: []string{"shared"},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"left", "right"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// shared-plugin should appear only once
	if len(resolved.Plugins) != 1 {
		t.Errorf("plugins: got %v, want [shared-plugin] (deduped)", resolved.Plugins)
	}
	// marketplace should appear only once
	if len(resolved.Marketplaces) != 1 {
		t.Errorf("marketplaces: got %d, want 1 (deduped)", len(resolved.Marketplaces))
	}
}

func TestResolveIncludes_MissingInclude(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"nonexistent"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected error for missing include, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention missing profile name: %v", err)
	}
}

func TestResolveIncludes_StackWithConfigFields(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"base": {Name: "base", Plugins: []string{"a"}},
		},
	}

	// A stack with includes AND config fields is invalid
	stack := &Profile{
		Name:     "invalid",
		Includes: []string{"base"},
		Plugins:  []string{"extra-plugin"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected validation error for stack with config fields, got nil")
	}
	if !strings.Contains(err.Error(), "pure") || !strings.Contains(err.Error(), "config") {
		t.Errorf("error should mention pure stack violation: %v", err)
	}
}

func TestResolveIncludes_MarketplaceDedup(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "org/shared"},
					{Source: "github", Repo: "org/a-only"},
				},
			},
			"b": {
				Name: "b",
				Marketplaces: []Marketplace{
					{Source: "github", Repo: "org/shared"},
					{Source: "github", Repo: "org/b-only"},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 3 unique marketplaces, not 4
	if len(resolved.Marketplaces) != 3 {
		repos := make([]string, len(resolved.Marketplaces))
		for i, m := range resolved.Marketplaces {
			repos[i] = m.Repo
		}
		t.Errorf("marketplaces: got %v (len=%d), want 3 unique", repos, len(resolved.Marketplaces))
	}
}

func TestResolveIncludes_PerScopePluginsDedup(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				PerScope: &PerScopeSettings{
					User:    &ScopeSettings{Plugins: []string{"shared", "a-only"}},
					Project: &ScopeSettings{Plugins: []string{"proj-a"}},
				},
			},
			"b": {
				Name: "b",
				PerScope: &PerScopeSettings{
					User:    &ScopeSettings{Plugins: []string{"shared", "b-only"}},
					Project: &ScopeSettings{Plugins: []string{"proj-b"}},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.PerScope == nil {
		t.Fatal("expected PerScope to be set")
	}

	// User plugins: shared (deduped), a-only, b-only = 3
	userPlugins := resolved.PerScope.User.Plugins
	if len(userPlugins) != 3 {
		t.Errorf("user plugins: got %v (len=%d), want 3", userPlugins, len(userPlugins))
	}

	// Project plugins: proj-a, proj-b = 2
	projPlugins := resolved.PerScope.Project.Plugins
	if len(projPlugins) != 2 {
		t.Errorf("project plugins: got %v (len=%d), want 2", projPlugins, len(projPlugins))
	}
}

func TestResolveIncludes_MCPServerLastWins(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						MCPServers: []MCPServer{
							{Name: "ctx7", Command: "old-cmd", Args: []string{"old-arg"}},
						},
					},
				},
			},
			"b": {
				Name: "b",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						MCPServers: []MCPServer{
							{Name: "ctx7", Command: "new-cmd", Args: []string{"new-arg"}},
						},
					},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	servers := resolved.PerScope.User.MCPServers
	if len(servers) != 1 {
		t.Fatalf("MCP servers: got %d, want 1", len(servers))
	}
	if servers[0].Command != "new-cmd" {
		t.Errorf("MCP server command: got %q, want %q (last-wins)", servers[0].Command, "new-cmd")
	}
}

func TestResolveIncludes_ExtensionsUnion(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				Extensions: &ExtensionSettings{
					Agents:   []string{"agent-a"},
					Commands: []string{"shared-cmd", "cmd-a"},
				},
			},
			"b": {
				Name: "b",
				Extensions: &ExtensionSettings{
					Agents:   []string{"agent-b"},
					Commands: []string{"shared-cmd", "cmd-b"},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Extensions == nil {
		t.Fatal("expected Extensions to be set")
	}

	// Agents: agent-a, agent-b = 2
	if len(resolved.Extensions.Agents) != 2 {
		t.Errorf("agents: got %v, want 2 items", resolved.Extensions.Agents)
	}

	// Commands: shared-cmd (deduped), cmd-a, cmd-b = 3
	if len(resolved.Extensions.Commands) != 3 {
		t.Errorf("commands: got %v, want 3 items (shared-cmd deduped)", resolved.Extensions.Commands)
	}
}

func TestResolveIncludes_SettingsHooksDedup(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				SettingsHooks: map[string][]HookEntry{
					"PreToolUse": {
						{Type: "command", Command: "shared-hook"},
						{Type: "command", Command: "hook-a"},
					},
				},
			},
			"b": {
				Name: "b",
				SettingsHooks: map[string][]HookEntry{
					"PreToolUse": {
						{Type: "command", Command: "shared-hook"},
						{Type: "command", Command: "hook-b"},
					},
					"PostToolUse": {
						{Type: "command", Command: "post-hook"},
					},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved.SettingsHooks) != 2 {
		t.Fatalf("settings hooks event types: got %d, want 2", len(resolved.SettingsHooks))
	}

	preHooks := resolved.SettingsHooks["PreToolUse"]
	// shared-hook (deduped), hook-a, hook-b = 3
	if len(preHooks) != 3 {
		t.Errorf("PreToolUse hooks: got %v (len=%d), want 3", preHooks, len(preHooks))
	}

	postHooks := resolved.SettingsHooks["PostToolUse"]
	if len(postHooks) != 1 {
		t.Errorf("PostToolUse hooks: got %v (len=%d), want 1", postHooks, len(postHooks))
	}
}

func TestResolveIncludes_DetectFilesUnion(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				Detect: DetectRules{
					Files: []string{"go.mod", "shared.txt"},
				},
			},
			"b": {
				Name: "b",
				Detect: DetectRules{
					Files: []string{"package.json", "shared.txt"},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// go.mod, shared.txt (deduped), package.json = 3
	if len(resolved.Detect.Files) != 3 {
		t.Errorf("detect files: got %v (len=%d), want 3", resolved.Detect.Files, len(resolved.Detect.Files))
	}
}

func TestResolveIncludes_DetectContainsMerge(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				Detect: DetectRules{
					Contains: map[string]string{
						"go.mod":      "module",
						"shared-file": "old-value",
					},
				},
			},
			"b": {
				Name: "b",
				Detect: DetectRules{
					Contains: map[string]string{
						"package.json": "name",
						"shared-file":  "new-value", // later wins
					},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resolved.Detect.Contains) != 3 {
		t.Errorf("detect contains: got %v (len=%d), want 3", resolved.Detect.Contains, len(resolved.Detect.Contains))
	}
	if resolved.Detect.Contains["shared-file"] != "new-value" {
		t.Errorf("shared-file: got %q, want %q (later wins)", resolved.Detect.Contains["shared-file"], "new-value")
	}
}

func TestResolveIncludes_PostApplyLastWins(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name:      "a",
				PostApply: &PostApplyHook{Command: "first-cmd", Condition: "always"},
			},
			"b": {
				Name:      "b",
				PostApply: &PostApplyHook{Command: "last-cmd", Condition: "first-run"},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.PostApply == nil {
		t.Fatal("expected PostApply to be set")
	}
	if resolved.PostApply.Command != "last-cmd" {
		t.Errorf("PostApply command: got %q, want %q (last-wins)", resolved.PostApply.Command, "last-cmd")
	}
	if resolved.PostApply.Condition != "first-run" {
		t.Errorf("PostApply condition: got %q, want %q", resolved.PostApply.Condition, "first-run")
	}
}

func TestResolveIncludes_ClearsIncludes(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"base": {Name: "base", Plugins: []string{"a"}},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"base"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved.Includes) != 0 {
		t.Errorf("resolved Includes: got %v, want nil/empty", resolved.Includes)
	}
}

func TestResolveIncludes_PreservesRootNameAndDescription(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"base": {
				Name:        "base",
				Description: "base description",
				Plugins:     []string{"a"},
			},
		},
	}

	stack := &Profile{
		Name:        "my-stack",
		Description: "stack description",
		Includes:    []string{"base"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Name != "my-stack" {
		t.Errorf("name: got %q, want %q", resolved.Name, "my-stack")
	}
	if resolved.Description != "stack description" {
		t.Errorf("description: got %q, want %q", resolved.Description, "stack description")
	}
}

func TestResolveIncludes_PathQualifiedName(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"languages/go": {
				Name: "go",
				PerScope: &PerScopeSettings{
					Project: &ScopeSettings{Plugins: []string{"gopls"}},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"languages/go"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.PerScope == nil || resolved.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be set")
	}
	if len(resolved.PerScope.Project.Plugins) != 1 || resolved.PerScope.Project.Plugins[0] != "gopls" {
		t.Errorf("project plugins: got %v, want [gopls]", resolved.PerScope.Project.Plugins)
	}
}

func TestResolveIncludes_MixedShortAndPathQualified(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"base": {
				Name:    "base",
				Plugins: []string{"base-plugin"},
			},
			"languages/go": {
				Name: "go",
				PerScope: &PerScopeSettings{
					Project: &ScopeSettings{Plugins: []string{"gopls"}},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"base", "languages/go"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0] != "base-plugin" {
		t.Errorf("plugins: got %v, want [base-plugin]", resolved.Plugins)
	}
	if resolved.PerScope == nil || resolved.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be set")
	}
}

func TestResolveIncludes_SkipPluginDiffOR(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {Name: "a", SkipPluginDiff: false},
			"b": {Name: "b", SkipPluginDiff: true},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved.SkipPluginDiff {
		t.Error("SkipPluginDiff: got false, want true (OR semantics)")
	}
}

func TestResolveIncludes_LegacyFlatPluginsDedup(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {Name: "a", Plugins: []string{"shared", "a-only"}},
			"b": {Name: "b", Plugins: []string{"shared", "b-only"}},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// shared (deduped), a-only, b-only = 3
	if len(resolved.Plugins) != 3 {
		t.Errorf("plugins: got %v (len=%d), want 3", resolved.Plugins, len(resolved.Plugins))
	}
}

func TestResolveIncludes_LegacyFlatMCPLastWins(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				MCPServers: []MCPServer{
					{Name: "srv", Command: "old"},
				},
			},
			"b": {
				Name: "b",
				MCPServers: []MCPServer{
					{Name: "srv", Command: "new"},
				},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"a", "b"},
	}

	resolved, err := ResolveIncludes(stack, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved.MCPServers) != 1 {
		t.Fatalf("MCP servers: got %d, want 1", len(resolved.MCPServers))
	}
	if resolved.MCPServers[0].Command != "new" {
		t.Errorf("MCP server command: got %q, want %q", resolved.MCPServers[0].Command, "new")
	}
}

// Validation tests for HasConfigFields and IsStack

func TestIsStack(t *testing.T) {
	tests := []struct {
		name    string
		profile *Profile
		want    bool
	}{
		{"nil profile", nil, false},
		{"no includes", &Profile{Name: "a"}, false},
		{"empty includes", &Profile{Name: "a", Includes: []string{}}, false},
		{"with includes", &Profile{Name: "a", Includes: []string{"b"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.IsStack()
			if got != tt.want {
				t.Errorf("IsStack(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasConfigFields(t *testing.T) {
	tests := []struct {
		name    string
		profile *Profile
		want    bool
	}{
		{"empty", &Profile{}, false},
		{"name only", &Profile{Name: "a"}, false},
		{"description only", &Profile{Name: "a", Description: "desc"}, false},
		{"includes only", &Profile{Name: "a", Includes: []string{"b"}}, false},
		{"with plugins", &Profile{Plugins: []string{"a"}}, true},
		{"with marketplaces", &Profile{Marketplaces: []Marketplace{{Source: "github"}}}, true},
		{"with mcp servers", &Profile{MCPServers: []MCPServer{{Name: "a"}}}, true},
		{"with perScope", &Profile{PerScope: &PerScopeSettings{}}, true},
		{"with localItems", &Profile{Extensions: &ExtensionSettings{}}, true},
		{"with settingsHooks", &Profile{SettingsHooks: map[string][]HookEntry{"a": {}}}, true},
		{"with detect files", &Profile{Detect: DetectRules{Files: []string{"a"}}}, true},
		{"with detect contains", &Profile{Detect: DetectRules{Contains: map[string]string{"a": "b"}}}, true},
		{"with postApply", &Profile{PostApply: &PostApplyHook{Command: "a"}}, true},
		{"with skipPluginDiff", &Profile{SkipPluginDiff: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.HasConfigFields()
			if got != tt.want {
				t.Errorf("HasConfigFields(): got %v, want %v", got, tt.want)
			}
		})
	}
}

// Validation: stack with nested includes that has config fields
func TestResolveIncludes_NestedStackWithConfigFields(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"base": {Name: "base", Plugins: []string{"a"}},
			// This is an invalid stack: has includes AND config fields
			"bad-middle": {
				Name:     "bad-middle",
				Includes: []string{"base"},
				Plugins:  []string{"extra"},
			},
		},
	}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"bad-middle"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected validation error for nested stack with config fields, got nil")
	}
}

// Test that AmbiguousProfileError propagates through ResolveIncludes
func TestResolveIncludes_AmbiguousInclude(t *testing.T) {
	ambigErr := &AmbiguousProfileError{
		Name:  "ambiguous",
		Paths: []string{"team/ambiguous", "personal/ambiguous"},
	}
	loader := &errorLoader{err: ambigErr}

	stack := &Profile{
		Name:     "top",
		Includes: []string{"ambiguous"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected AmbiguousProfileError to propagate, got nil")
	}

	var gotErr *AmbiguousProfileError
	if !errors.As(err, &gotErr) {
		t.Fatalf("expected AmbiguousProfileError, got: %v", err)
	}
	if gotErr.Name != "ambiguous" {
		t.Errorf("error name: got %q, want %q", gotErr.Name, "ambiguous")
	}
}

func TestResolveIncludes_NilLoaderReturnsError(t *testing.T) {
	stack := &Profile{
		Name:     "my-stack",
		Includes: []string{"leaf"},
	}

	_, err := ResolveIncludes(stack, nil)
	if err == nil {
		t.Fatal("expected error for nil loader, got nil")
	}
	if !strings.Contains(err.Error(), "loader is nil") {
		t.Errorf("expected 'loader is nil' error, got: %v", err)
	}
}

func TestDirLoader_PropagatesNonNotFoundErrors(t *testing.T) {
	// DirLoader should only fall back to embedded profiles for not-found errors.
	// Other errors like AmbiguousProfileError should propagate.
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(filepath.Join(profilesDir, "sub1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(profilesDir, "sub2"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create two profiles with the same name in different subdirectories
	for _, sub := range []string{"sub1", "sub2"} {
		data := []byte(`{"name": "dup"}`)
		if err := os.WriteFile(filepath.Join(profilesDir, sub, "dup.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	loader := &DirLoader{ProfilesDir: profilesDir}
	_, err := loader.LoadProfile("dup")
	if err == nil {
		t.Fatal("expected AmbiguousProfileError to propagate, got nil")
	}

	var ambigErr *AmbiguousProfileError
	if !errors.As(err, &ambigErr) {
		t.Errorf("expected AmbiguousProfileError, got: %v", err)
	}
}

func TestResolveIncludes_DepthLimitExceeded(t *testing.T) {
	// Build a chain deeper than MaxIncludeDepth: level-0 -> level-1 -> ... -> level-N
	profiles := make(map[string]*Profile)
	for i := 0; i <= MaxIncludeDepth; i++ {
		name := fmt.Sprintf("level-%d", i)
		if i == MaxIncludeDepth {
			// Leaf profile at the bottom
			profiles[name] = &Profile{
				Name:    name,
				Plugins: []string{"deep-plugin"},
			}
		} else {
			// Stack that includes the next level
			profiles[name] = &Profile{
				Name:     name,
				Includes: []string{fmt.Sprintf("level-%d", i+1)},
			}
		}
	}
	loader := &mockLoader{profiles: profiles}

	stack := &Profile{
		Name:     "deep-stack",
		Includes: []string{"level-0"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected depth limit error, got nil")
	}
	if !strings.Contains(err.Error(), "depth limit exceeded") {
		t.Errorf("expected depth limit error, got: %v", err)
	}
}

func TestResolveIncludes_PathTraversalRejected(t *testing.T) {
	// Verify that path traversal in include names is rejected by the DirLoader.
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a secret file outside profilesDir
	secretData := []byte(`{"name":"secret","plugins":["stolen-plugin"]}`)
	if err := os.WriteFile(filepath.Join(dir, "secret.json"), secretData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a leaf profile for the stack to include
	leafData := []byte(`{"name":"safe","plugins":["good-plugin"]}`)
	if err := os.WriteFile(filepath.Join(profilesDir, "safe.json"), leafData, 0644); err != nil {
		t.Fatal(err)
	}

	loader := &DirLoader{ProfilesDir: profilesDir}

	stack := &Profile{
		Name:     "evil-stack",
		Includes: []string{"../secret"},
	}

	_, err := ResolveIncludes(stack, loader)
	if err == nil {
		t.Fatal("expected error for path traversal include, got nil")
	}
	if !strings.Contains(err.Error(), "escapes profiles directory") {
		t.Errorf("expected 'escapes profiles directory' error, got: %v", err)
	}
}

func TestDirLoader_FallsBackOnNotFound(t *testing.T) {
	// DirLoader should fall back to embedded profiles when the profile is not found on disk.
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatal(err)
	}

	loader := &DirLoader{ProfilesDir: profilesDir}
	// "default" is an embedded profile that should be found via fallback
	p, err := loader.LoadProfile("default")
	if err != nil {
		t.Fatalf("expected embedded fallback, got error: %v", err)
	}
	if p.Name != "default" {
		t.Errorf("name: got %q, want %q", p.Name, "default")
	}
}
