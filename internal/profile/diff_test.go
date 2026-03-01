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

func TestAsPerScope_CopiesPerScope(t *testing.T) {
	p := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"a@market"},
			},
		},
	}

	result := p.AsPerScope()

	// PerScope should be a different pointer
	if result.PerScope == p.PerScope {
		t.Error("expected AsPerScope to copy PerScope, got same pointer")
	}
	// But content should match
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
					if item.Detail != "command changed" {
						t.Errorf("expected detail 'command changed', got %q", item.Detail)
					}
				}
			}
		}
	}
	if !found {
		t.Error("expected modified MCP server 'srv' in user scope")
	}
}

func TestMcpDiffDetail(t *testing.T) {
	tests := []struct {
		name   string
		saved  MCPServer
		live   MCPServer
		expect string
	}{
		{
			name:   "command changed",
			saved:  MCPServer{Name: "s", Command: "old"},
			live:   MCPServer{Name: "s", Command: "new"},
			expect: "command changed",
		},
		{
			name:   "args changed",
			saved:  MCPServer{Name: "s", Command: "cmd", Args: []string{"a"}},
			live:   MCPServer{Name: "s", Command: "cmd", Args: []string{"b"}},
			expect: "args changed",
		},
		{
			name:   "scope changed",
			saved:  MCPServer{Name: "s", Command: "cmd", Scope: "user"},
			live:   MCPServer{Name: "s", Command: "cmd", Scope: "project"},
			expect: "scope changed",
		},
		{
			name:   "multiple fields changed",
			saved:  MCPServer{Name: "s", Command: "old", Scope: "user"},
			live:   MCPServer{Name: "s", Command: "new", Scope: "project"},
			expect: "command, scope changed",
		},
		{
			name:   "secrets changed falls back to generic",
			saved:  MCPServer{Name: "s", Command: "cmd", Secrets: map[string]SecretRef{"k": {}}},
			live:   MCPServer{Name: "s", Command: "cmd"},
			expect: "config changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcpDiffDetail(tt.saved, tt.live)
			if got != tt.expect {
				t.Errorf("expected %q, got %q", tt.expect, got)
			}
		})
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

func TestFilterToScopes_KeepsAllMarketplacesWhenUserActive(t *testing.T) {
	// Marketplace matching via plugin @suffix vs DisplayName() is unreliable.
	// When user scope is active, all marketplaces should be preserved
	// regardless of whether plugin @suffixes match DisplayName() patterns.
	p := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "anthropics/claude-plugins-official"}, // suffix matches
			{Source: "github", Repo: "thedotmack/claude-mem"},              // prefix, not suffix
			{Source: "github", Repo: "anthropics/skills"},                  // no relationship to key
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{
					"code-simplifier@claude-plugins-official",
					"claude-mem@thedotmack",
					"document-skills@anthropic-agent-skills",
				},
			},
		},
	}

	result := FilterToScopes(p, map[string]bool{"user": true})

	if len(result.Marketplaces) != 3 {
		t.Errorf("expected 3 marketplaces, got %d", len(result.Marketplaces))
		for _, m := range result.Marketplaces {
			t.Logf("  kept: %s", m.DisplayName())
		}
	}
}

func TestFilterToScopes_DropsMarketplacesWhenUserNotActive(t *testing.T) {
	// When only project scope is active, marketplaces (always user-scoped)
	// should not be included.
	p := &Profile{
		Name: "test",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "org/repo"},
		},
		PerScope: &PerScopeSettings{
			Project: &ScopeSettings{
				Plugins: []string{"tool@repo"},
			},
		},
	}

	result := FilterToScopes(p, map[string]bool{"project": true})

	if len(result.Marketplaces) != 0 {
		t.Errorf("expected 0 marketplaces when user scope not active, got %d", len(result.Marketplaces))
	}
}

func TestUserScopeExtras(t *testing.T) {
	t.Run("returns plugins in live but not in saved", func(t *testing.T) {
		saved := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market", "plugin-b@market"},
				},
			},
		}
		live := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market", "plugin-b@market", "plugin-c@market", "plugin-d@market"},
				},
			},
		}

		extras := UserScopeExtras(saved, live)
		if len(extras) != 2 {
			t.Fatalf("expected 2 extras, got %d", len(extras))
		}

		names := map[string]bool{}
		for _, item := range extras {
			names[item.Name] = true
			if item.Op != DiffAdded {
				t.Errorf("expected DiffAdded, got %v", item.Op)
			}
		}
		if !names["plugin-c@market"] || !names["plugin-d@market"] {
			t.Errorf("expected plugin-c and plugin-d, got %v", names)
		}
	})

	t.Run("returns empty when no extras", func(t *testing.T) {
		saved := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market"},
				},
			},
		}
		live := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market"},
				},
			},
		}

		extras := UserScopeExtras(saved, live)
		if len(extras) != 0 {
			t.Fatalf("expected 0 extras, got %d", len(extras))
		}
	})

	t.Run("returns empty when saved has no user scope", func(t *testing.T) {
		saved := &Profile{
			Name:     "test",
			PerScope: &PerScopeSettings{},
		}
		live := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market"},
				},
			},
		}

		extras := UserScopeExtras(saved, live)
		if len(extras) != 0 {
			t.Fatalf("expected 0 extras (no user scope in profile), got %d", len(extras))
		}
	})

	t.Run("returns empty when live has no user scope", func(t *testing.T) {
		saved := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market"},
				},
			},
		}
		live := &Profile{
			PerScope: &PerScopeSettings{},
		}

		extras := UserScopeExtras(saved, live)
		if len(extras) != 0 {
			t.Fatalf("expected 0 extras (no live user scope), got %d", len(extras))
		}
	})

	t.Run("all live plugins are extras when saved has empty plugin list", func(t *testing.T) {
		saved := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{},
				},
			},
		}
		live := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market", "plugin-b@market"},
				},
			},
		}

		extras := UserScopeExtras(saved, live)
		if len(extras) != 2 {
			t.Fatalf("expected 2 extras, got %d", len(extras))
		}
	})

	t.Run("all extras have DiffPlugin kind", func(t *testing.T) {
		saved := &Profile{
			Name: "test",
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market"},
				},
			},
		}
		live := &Profile{
			PerScope: &PerScopeSettings{
				User: &ScopeSettings{
					Plugins: []string{"plugin-a@market", "plugin-b@market", "plugin-c@market"},
				},
			},
		}

		extras := UserScopeExtras(saved, live)
		if len(extras) != 2 {
			t.Fatalf("expected 2 extras, got %d", len(extras))
		}
		for _, item := range extras {
			if item.Kind != DiffPlugin {
				t.Errorf("expected DiffPlugin, got %v for %v", item.Kind, item.Name)
			}
		}
	})
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
