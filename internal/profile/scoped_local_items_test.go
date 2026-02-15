// ABOUTME: Tests for project-scoped Extensions on ScopeSettings
// ABOUTME: Validates JSON marshaling, ForScope, CombinedScopes, Equal, Clone
package profile

import (
	"encoding/json"
	"testing"
)

func TestScopeSettingsExtensionsJSONRoundTrip(t *testing.T) {
	p := &Profile{
		Name: "scoped-items",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@mp"},
				Extensions: &ExtensionSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules:  []string{"golang"},
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded Profile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.PerScope == nil {
		t.Fatal("expected PerScope to be non-nil")
	}
	if loaded.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be non-nil")
	}
	if loaded.PerScope.User.Extensions == nil {
		t.Fatal("expected PerScope.User.Extensions to be non-nil")
	}
	if len(loaded.PerScope.User.Extensions.Skills) != 1 || loaded.PerScope.User.Extensions.Skills[0] != "session-notes" {
		t.Errorf("expected user skills [session-notes], got %v", loaded.PerScope.User.Extensions.Skills)
	}
	if len(loaded.PerScope.User.Extensions.Rules) != 1 || loaded.PerScope.User.Extensions.Rules[0] != "coding-standards" {
		t.Errorf("expected user rules [coding-standards], got %v", loaded.PerScope.User.Extensions.Rules)
	}

	if loaded.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be non-nil")
	}
	if loaded.PerScope.Project.Extensions == nil {
		t.Fatal("expected PerScope.Project.Extensions to be non-nil")
	}
	if len(loaded.PerScope.Project.Extensions.Rules) != 1 || loaded.PerScope.Project.Extensions.Rules[0] != "golang" {
		t.Errorf("expected project rules [golang], got %v", loaded.PerScope.Project.Extensions.Rules)
	}
	if len(loaded.PerScope.Project.Extensions.Agents) != 1 || loaded.PerScope.Project.Extensions.Agents[0] != "reviewer" {
		t.Errorf("expected project agents [reviewer], got %v", loaded.PerScope.Project.Extensions.Agents)
	}
}

func TestScopeSettingsExtensionsOmittedWhenNil(t *testing.T) {
	p := &Profile{
		Name: "no-items",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"plugin1"},
			},
		},
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	if containsSubstring(jsonStr, `"extensions"`) {
		t.Errorf("expected extensions to be omitted when nil, got:\n%s", jsonStr)
	}
}

func TestForScopeReturnsExtensions(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-p1"},
				Extensions: &ExtensionSettings{
					Skills: []string{"session-notes"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules:  []string{"golang"},
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	// User scope should have user's local items
	userProfile := p.ForScope("user")
	if userProfile.Extensions == nil {
		t.Fatal("expected user scope to have Extensions")
	}
	if len(userProfile.Extensions.Skills) != 1 || userProfile.Extensions.Skills[0] != "session-notes" {
		t.Errorf("expected user skills [session-notes], got %v", userProfile.Extensions.Skills)
	}

	// Project scope should have project's local items
	projectProfile := p.ForScope("project")
	if projectProfile.Extensions == nil {
		t.Fatal("expected project scope to have Extensions")
	}
	if len(projectProfile.Extensions.Rules) != 1 || projectProfile.Extensions.Rules[0] != "golang" {
		t.Errorf("expected project rules [golang], got %v", projectProfile.Extensions.Rules)
	}
	if len(projectProfile.Extensions.Agents) != 1 || projectProfile.Extensions.Agents[0] != "reviewer" {
		t.Errorf("expected project agents [reviewer], got %v", projectProfile.Extensions.Agents)
	}

	// Local scope (not set) should have nil Extensions
	localProfile := p.ForScope("local")
	if localProfile.Extensions != nil {
		t.Errorf("expected local scope to have nil Extensions, got %v", localProfile.Extensions)
	}
}

func TestCombinedScopesMergesExtensions(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules:  []string{"golang", "coding-standards"}, // coding-standards is a duplicate
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	combined := p.CombinedScopes()

	if combined.Extensions == nil {
		t.Fatal("expected combined to have Extensions")
	}

	// Skills should come from user scope
	if len(combined.Extensions.Skills) != 1 || combined.Extensions.Skills[0] != "session-notes" {
		t.Errorf("expected skills [session-notes], got %v", combined.Extensions.Skills)
	}

	// Rules should be union (coding-standards deduplicated)
	if len(combined.Extensions.Rules) != 2 {
		t.Errorf("expected 2 rules (deduplicated), got %d: %v", len(combined.Extensions.Rules), combined.Extensions.Rules)
	}

	// Agents should come from project scope
	if len(combined.Extensions.Agents) != 1 || combined.Extensions.Agents[0] != "reviewer" {
		t.Errorf("expected agents [reviewer], got %v", combined.Extensions.Agents)
	}
}

func TestCombinedScopesNoExtensions(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"p1"},
			},
		},
	}

	combined := p.CombinedScopes()

	// No local items anywhere - should be nil
	if combined.Extensions != nil {
		t.Errorf("expected nil Extensions when no scopes have them, got %v", combined.Extensions)
	}
}

func TestEqualWithScopedExtensions(t *testing.T) {
	p1 := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules: []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	// Identical profile
	p2 := &Profile{
		Name: "test2",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules: []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	if !p1.Equal(p2) {
		t.Error("expected profiles with identical scoped Extensions to be equal")
	}

	// Different local items
	p3 := &Profile{
		Name: "test3",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules: []string{"different-rule"},
				},
			},
		},
	}

	if p1.Equal(p3) {
		t.Error("expected profiles with different scoped Extensions to not be equal")
	}
}

func TestCloneWithScopedExtensions(t *testing.T) {
	original := &Profile{
		Name: "original",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"p1"},
				Extensions: &ExtensionSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Rules:  []string{"golang"},
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	clone := original.Clone("clone")

	if clone.Name != "clone" {
		t.Errorf("expected clone name 'clone', got %q", clone.Name)
	}

	// Verify deep copy - modifying clone should not affect original
	if clone.PerScope == nil || clone.PerScope.User == nil || clone.PerScope.User.Extensions == nil {
		t.Fatal("expected clone to have PerScope.User.Extensions")
	}
	clone.PerScope.User.Extensions.Skills[0] = "modified"
	if original.PerScope.User.Extensions.Skills[0] == "modified" {
		t.Error("modifying clone affected original - not a deep copy")
	}

	// Verify project scope local items are cloned
	if clone.PerScope.Project == nil || clone.PerScope.Project.Extensions == nil {
		t.Fatal("expected clone to have PerScope.Project.Extensions")
	}
	if len(clone.PerScope.Project.Extensions.Rules) != 1 || clone.PerScope.Project.Extensions.Rules[0] != "golang" {
		t.Errorf("expected clone project rules [golang], got %v", clone.PerScope.Project.Extensions.Rules)
	}
}
