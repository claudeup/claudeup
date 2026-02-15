// ABOUTME: Tests for Extensions merging in include resolution
// ABOUTME: Validates that scoped Extensions are unioned across included profiles
package profile

import (
	"testing"
)

func TestResolveIncludes_ScopedExtensionsUnion(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Extensions: &ExtensionSettings{
							Skills: []string{"session-notes"},
							Rules:  []string{"shared-rule", "rule-a"},
						},
					},
					Project: &ScopeSettings{
						Extensions: &ExtensionSettings{
							Rules: []string{"golang"},
						},
					},
				},
			},
			"b": {
				Name: "b",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Extensions: &ExtensionSettings{
							Rules: []string{"shared-rule", "rule-b"},
						},
					},
					Project: &ScopeSettings{
						Extensions: &ExtensionSettings{
							Rules:  []string{"testing"},
							Agents: []string{"reviewer"},
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

	if resolved.PerScope == nil {
		t.Fatal("expected PerScope to be set")
	}

	// User scope: skills [session-notes], rules [shared-rule, rule-a, rule-b] (deduped)
	if resolved.PerScope.User == nil || resolved.PerScope.User.Extensions == nil {
		t.Fatal("expected PerScope.User.Extensions to be set")
	}
	userItems := resolved.PerScope.User.Extensions
	if len(userItems.Skills) != 1 || userItems.Skills[0] != "session-notes" {
		t.Errorf("user skills: got %v, want [session-notes]", userItems.Skills)
	}
	if len(userItems.Rules) != 3 {
		t.Errorf("user rules: got %v (len=%d), want 3 (shared-rule deduped)", userItems.Rules, len(userItems.Rules))
	}

	// Project scope: rules [golang, testing], agents [reviewer]
	if resolved.PerScope.Project == nil || resolved.PerScope.Project.Extensions == nil {
		t.Fatal("expected PerScope.Project.Extensions to be set")
	}
	projectItems := resolved.PerScope.Project.Extensions
	if len(projectItems.Rules) != 2 {
		t.Errorf("project rules: got %v (len=%d), want 2", projectItems.Rules, len(projectItems.Rules))
	}
	if len(projectItems.Agents) != 1 || projectItems.Agents[0] != "reviewer" {
		t.Errorf("project agents: got %v, want [reviewer]", projectItems.Agents)
	}
}

func TestResolveIncludes_ScopedExtensionsOneProfileHasNone(t *testing.T) {
	loader := &mockLoader{
		profiles: map[string]*Profile{
			"a": {
				Name: "a",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Plugins: []string{"plugin-a"},
					},
				},
			},
			"b": {
				Name: "b",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{
						Extensions: &ExtensionSettings{
							Rules: []string{"rule-b"},
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

	if resolved.PerScope == nil || resolved.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be set")
	}
	if resolved.PerScope.User.Extensions == nil {
		t.Fatal("expected PerScope.User.Extensions to be set")
	}
	if len(resolved.PerScope.User.Extensions.Rules) != 1 || resolved.PerScope.User.Extensions.Rules[0] != "rule-b" {
		t.Errorf("user rules: got %v, want [rule-b]", resolved.PerScope.User.Extensions.Rules)
	}
}
