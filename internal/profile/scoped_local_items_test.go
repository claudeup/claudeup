// ABOUTME: Tests for project-scoped LocalItems on ScopeSettings
// ABOUTME: Validates JSON marshaling, ForScope, CombinedScopes, Equal, Clone
package profile

import (
	"encoding/json"
	"testing"
)

func TestScopeSettingsLocalItemsJSONRoundTrip(t *testing.T) {
	p := &Profile{
		Name: "scoped-items",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@mp"},
				LocalItems: &LocalItemSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				LocalItems: &LocalItemSettings{
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
	if loaded.PerScope.User.LocalItems == nil {
		t.Fatal("expected PerScope.User.LocalItems to be non-nil")
	}
	if len(loaded.PerScope.User.LocalItems.Skills) != 1 || loaded.PerScope.User.LocalItems.Skills[0] != "session-notes" {
		t.Errorf("expected user skills [session-notes], got %v", loaded.PerScope.User.LocalItems.Skills)
	}
	if len(loaded.PerScope.User.LocalItems.Rules) != 1 || loaded.PerScope.User.LocalItems.Rules[0] != "coding-standards" {
		t.Errorf("expected user rules [coding-standards], got %v", loaded.PerScope.User.LocalItems.Rules)
	}

	if loaded.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be non-nil")
	}
	if loaded.PerScope.Project.LocalItems == nil {
		t.Fatal("expected PerScope.Project.LocalItems to be non-nil")
	}
	if len(loaded.PerScope.Project.LocalItems.Rules) != 1 || loaded.PerScope.Project.LocalItems.Rules[0] != "golang" {
		t.Errorf("expected project rules [golang], got %v", loaded.PerScope.Project.LocalItems.Rules)
	}
	if len(loaded.PerScope.Project.LocalItems.Agents) != 1 || loaded.PerScope.Project.LocalItems.Agents[0] != "reviewer" {
		t.Errorf("expected project agents [reviewer], got %v", loaded.PerScope.Project.LocalItems.Agents)
	}
}

func TestScopeSettingsLocalItemsOmittedWhenNil(t *testing.T) {
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
	if containsSubstring(jsonStr, `"localItems"`) {
		t.Errorf("expected localItems to be omitted when nil, got:\n%s", jsonStr)
	}
}

func TestForScopeReturnsLocalItems(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-p1"},
				LocalItems: &LocalItemSettings{
					Skills: []string{"session-notes"},
				},
			},
			Project: &ScopeSettings{
				LocalItems: &LocalItemSettings{
					Rules:  []string{"golang"},
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	// User scope should have user's local items
	userProfile := p.ForScope("user")
	if userProfile.LocalItems == nil {
		t.Fatal("expected user scope to have LocalItems")
	}
	if len(userProfile.LocalItems.Skills) != 1 || userProfile.LocalItems.Skills[0] != "session-notes" {
		t.Errorf("expected user skills [session-notes], got %v", userProfile.LocalItems.Skills)
	}

	// Project scope should have project's local items
	projectProfile := p.ForScope("project")
	if projectProfile.LocalItems == nil {
		t.Fatal("expected project scope to have LocalItems")
	}
	if len(projectProfile.LocalItems.Rules) != 1 || projectProfile.LocalItems.Rules[0] != "golang" {
		t.Errorf("expected project rules [golang], got %v", projectProfile.LocalItems.Rules)
	}
	if len(projectProfile.LocalItems.Agents) != 1 || projectProfile.LocalItems.Agents[0] != "reviewer" {
		t.Errorf("expected project agents [reviewer], got %v", projectProfile.LocalItems.Agents)
	}

	// Local scope (not set) should have nil LocalItems
	localProfile := p.ForScope("local")
	if localProfile.LocalItems != nil {
		t.Errorf("expected local scope to have nil LocalItems, got %v", localProfile.LocalItems)
	}
}

func TestCombinedScopesMergesLocalItems(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				LocalItems: &LocalItemSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				LocalItems: &LocalItemSettings{
					Rules:  []string{"golang", "coding-standards"}, // coding-standards is a duplicate
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	combined := p.CombinedScopes()

	if combined.LocalItems == nil {
		t.Fatal("expected combined to have LocalItems")
	}

	// Skills should come from user scope
	if len(combined.LocalItems.Skills) != 1 || combined.LocalItems.Skills[0] != "session-notes" {
		t.Errorf("expected skills [session-notes], got %v", combined.LocalItems.Skills)
	}

	// Rules should be union (coding-standards deduplicated)
	if len(combined.LocalItems.Rules) != 2 {
		t.Errorf("expected 2 rules (deduplicated), got %d: %v", len(combined.LocalItems.Rules), combined.LocalItems.Rules)
	}

	// Agents should come from project scope
	if len(combined.LocalItems.Agents) != 1 || combined.LocalItems.Agents[0] != "reviewer" {
		t.Errorf("expected agents [reviewer], got %v", combined.LocalItems.Agents)
	}
}

func TestCombinedScopesNoLocalItems(t *testing.T) {
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
	if combined.LocalItems != nil {
		t.Errorf("expected nil LocalItems when no scopes have them, got %v", combined.LocalItems)
	}
}

func TestEqualWithScopedLocalItems(t *testing.T) {
	p1 := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				LocalItems: &LocalItemSettings{
					Rules: []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				LocalItems: &LocalItemSettings{
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
				LocalItems: &LocalItemSettings{
					Rules: []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				LocalItems: &LocalItemSettings{
					Agents: []string{"reviewer"},
				},
			},
		},
	}

	if !p1.Equal(p2) {
		t.Error("expected profiles with identical scoped LocalItems to be equal")
	}

	// Different local items
	p3 := &Profile{
		Name: "test3",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				LocalItems: &LocalItemSettings{
					Rules: []string{"different-rule"},
				},
			},
		},
	}

	if p1.Equal(p3) {
		t.Error("expected profiles with different scoped LocalItems to not be equal")
	}
}

func TestCloneWithScopedLocalItems(t *testing.T) {
	original := &Profile{
		Name: "original",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"p1"},
				LocalItems: &LocalItemSettings{
					Skills: []string{"session-notes"},
					Rules:  []string{"coding-standards"},
				},
			},
			Project: &ScopeSettings{
				LocalItems: &LocalItemSettings{
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
	if clone.PerScope == nil || clone.PerScope.User == nil || clone.PerScope.User.LocalItems == nil {
		t.Fatal("expected clone to have PerScope.User.LocalItems")
	}
	clone.PerScope.User.LocalItems.Skills[0] = "modified"
	if original.PerScope.User.LocalItems.Skills[0] == "modified" {
		t.Error("modifying clone affected original - not a deep copy")
	}

	// Verify project scope local items are cloned
	if clone.PerScope.Project == nil || clone.PerScope.Project.LocalItems == nil {
		t.Fatal("expected clone to have PerScope.Project.LocalItems")
	}
	if len(clone.PerScope.Project.LocalItems.Rules) != 1 || clone.PerScope.Project.LocalItems.Rules[0] != "golang" {
		t.Errorf("expected clone project rules [golang], got %v", clone.PerScope.Project.LocalItems.Rules)
	}
}
