// ABOUTME: Unit tests for profile diff logic
// ABOUTME: Tests ComputeProfileDiff and AsPerScope for comparing profiles against live state
package profile

import (
	"testing"
)

func TestAsPerScope_AlreadyPerScope(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}

	result := p.AsPerScope()

	if result.PerScope == nil {
		t.Fatal("expected PerScope to be set")
	}
	if len(result.PerScope.User.Plugins) != 1 || result.PerScope.User.Plugins[0] != "a@market" {
		t.Errorf("expected user plugins [a@market], got %v", result.PerScope.User.Plugins)
	}
}

func TestAsPerScope_LegacyProfile(t *testing.T) {
	p := &Profile{
		Name:        "test",
		Description: "legacy profile",
		Plugins:     []string{"a@market", "b@market"},
		MCPServers:  []MCPServer{{Name: "srv", Command: "cmd"}},
		Marketplaces: []Marketplace{{
			Source: "github",
			Repo:   "org/repo",
		}},
		Extensions: &ExtensionSettings{
			Agents: []string{"my-agent"},
		},
	}

	result := p.AsPerScope()

	if result.PerScope == nil {
		t.Fatal("expected PerScope to be set")
	}
	if result.PerScope.User == nil {
		t.Fatal("expected User scope to be set")
	}
	if len(result.PerScope.User.Plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(result.PerScope.User.Plugins))
	}
	if len(result.PerScope.User.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server, got %d", len(result.PerScope.User.MCPServers))
	}
	if result.PerScope.User.Extensions == nil || len(result.PerScope.User.Extensions.Agents) != 1 {
		t.Error("expected extensions to be lifted to user scope")
	}
	// Marketplaces stay at profile level
	if len(result.Marketplaces) != 1 {
		t.Errorf("expected 1 marketplace, got %d", len(result.Marketplaces))
	}
	// Description preserved
	if result.Description != "legacy profile" {
		t.Errorf("expected description 'legacy profile', got %q", result.Description)
	}
}

func TestAsPerScope_NilProfile(t *testing.T) {
	var p *Profile
	result := p.AsPerScope()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.PerScope == nil {
		t.Fatal("expected PerScope to be set")
	}
}

func TestComputeProfileDiff_IdenticalProfiles(t *testing.T) {
	saved := &Profile{
		Name:        "test",
		Description: "desc",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}
	live := &Profile{
		Name:        "live",
		Description: "desc",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	if !diff.IsEmpty() {
		t.Errorf("expected empty diff, got %+v", diff)
	}
}

func TestComputeProfileDiff_AddedPlugin(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market", "b@market"},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffAdded && item.Kind == DiffPlugin && item.Name == "b@market" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected added plugin b@market in user scope")
	}
}

func TestComputeProfileDiff_RemovedPlugin(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market", "b@market"},
			},
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffRemoved && item.Kind == DiffPlugin && item.Name == "b@market" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected removed plugin b@market in user scope")
	}
}

func TestComputeProfileDiff_MCPServerAdded(t *testing.T) {
	saved := &Profile{
		Name:     "test",
		PerScope: &PerScopeSettings{User: &ScopeSettings{}},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{{Name: "srv", Command: "cmd"}},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffAdded && item.Kind == DiffMCP && item.Name == "srv" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected added MCP server 'srv' in user scope")
	}
}

func TestComputeProfileDiff_MCPServerRemoved(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{{Name: "srv", Command: "cmd"}},
			},
		},
	}
	live := &Profile{
		Name:     "live",
		PerScope: &PerScopeSettings{User: &ScopeSettings{}},
	}

	diff := ComputeProfileDiff(saved, live)
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffRemoved && item.Kind == DiffMCP && item.Name == "srv" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected removed MCP server 'srv' in user scope")
	}
}

func TestComputeProfileDiff_MCPServerModified(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{{Name: "srv", Command: "old-cmd"}},
			},
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{{Name: "srv", Command: "new-cmd"}},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffModified && item.Kind == DiffMCP && item.Name == "srv" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected modified MCP server 'srv' in user scope")
	}
}

func TestComputeProfileDiff_ExtensionDiffs(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Agents: []string{"agent-a"},
					Rules:  []string{"rule-a"},
				},
			},
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Extensions: &ExtensionSettings{
					Agents: []string{"agent-a", "agent-b"},
					// rule-a removed in live
				},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}

	addedAgent := false
	removedRule := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffAdded && item.Kind == DiffExtension && item.Name == "agent-b" && item.Detail == "agents" {
					addedAgent = true
				}
				if item.Op == DiffRemoved && item.Kind == DiffExtension && item.Name == "rule-a" && item.Detail == "rules" {
					removedRule = true
				}
			}
		}
	}
	if !addedAgent {
		t.Error("expected added extension agent-b")
	}
	if !removedRule {
		t.Error("expected removed extension rule-a")
	}
}

func TestComputeProfileDiff_MarketplaceDiffs(t *testing.T) {
	saved := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/repo-a"},
		},
		PerScope: &PerScopeSettings{},
	}
	live := &Profile{
		Name: "live",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/repo-a"},
			{Source: "github", Repo: "org/repo-b"},
		},
		PerScope: &PerScopeSettings{},
	}

	diff := ComputeProfileDiff(saved, live)
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "user" {
			for _, item := range sd.Items {
				if item.Op == DiffAdded && item.Kind == DiffMarketplace && item.Name == "org/repo-b" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected added marketplace org/repo-b in user scope")
	}
}

func TestComputeProfileDiff_DescriptionChange(t *testing.T) {
	saved := &Profile{
		Name:        "test",
		Description: "old desc",
		PerScope:    &PerScopeSettings{},
	}
	live := &Profile{
		Name:        "live",
		Description: "new desc",
		PerScope:    &PerScopeSettings{},
	}

	diff := ComputeProfileDiff(saved, live)
	if diff.DescriptionChange == nil {
		t.Fatal("expected description change")
	}
	if diff.DescriptionChange[0] != "old desc" || diff.DescriptionChange[1] != "new desc" {
		t.Errorf("expected [old desc, new desc], got %v", diff.DescriptionChange)
	}
}

func TestComputeProfileDiff_NilScopeInOneProfile(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
			// No project scope
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"proj@market"},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)
	if diff.IsEmpty() {
		t.Fatal("expected non-empty diff")
	}
	found := false
	for _, sd := range diff.Scopes {
		if sd.Scope == "project" {
			for _, item := range sd.Items {
				if item.Op == DiffAdded && item.Kind == DiffPlugin && item.Name == "proj@market" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected added plugin proj@market in project scope")
	}
}

func TestComputeProfileDiff_MultiScopeDiffs(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market", "b@market"},
			},
			Project: &ScopeSettings{
				Plugins: []string{"proj@market"},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)

	scopeCount := 0
	for _, sd := range diff.Scopes {
		if len(sd.Items) > 0 {
			scopeCount++
		}
	}
	if scopeCount != 2 {
		t.Errorf("expected diffs in 2 scopes, got %d", scopeCount)
	}
}

func TestProfileDiff_IsEmpty(t *testing.T) {
	// Empty diff
	d := &ProfileDiff{ProfileName: "test"}
	if !d.IsEmpty() {
		t.Error("expected IsEmpty to be true for empty diff")
	}

	// Diff with description change
	d = &ProfileDiff{
		ProfileName:       "test",
		DescriptionChange: &[2]string{"a", "b"},
	}
	if d.IsEmpty() {
		t.Error("expected IsEmpty to be false when description changed")
	}

	// Diff with scope items
	d = &ProfileDiff{
		ProfileName: "test",
		Scopes: []ScopeDiff{
			{Scope: "user", Items: []DiffItem{{Op: DiffAdded, Kind: DiffPlugin, Name: "x"}}},
		},
	}
	if d.IsEmpty() {
		t.Error("expected IsEmpty to be false when scope items exist")
	}
}

func TestComputeProfileDiff_TotalCounts(t *testing.T) {
	saved := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"removed@market"},
			},
		},
	}
	live := &Profile{
		Name: "live",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"added-a@market", "added-b@market"},
			},
		},
	}

	diff := ComputeProfileDiff(saved, live)

	added, removed, modified := diff.Counts()
	if added != 2 {
		t.Errorf("expected 2 additions, got %d", added)
	}
	if removed != 1 {
		t.Errorf("expected 1 removal, got %d", removed)
	}
	if modified != 0 {
		t.Errorf("expected 0 modifications, got %d", modified)
	}
}
