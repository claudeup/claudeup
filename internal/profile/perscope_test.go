// ABOUTME: Unit tests for per-scope profile settings
// ABOUTME: Tests JSON marshaling, backward compatibility, and type behavior
package profile

import (
	"encoding/json"
	"testing"
)

func TestPerScopeSettingsJSONMarshal(t *testing.T) {
	p := &Profile{
		Name:        "test",
		Description: "Test profile",
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "user/repo"},
		},
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-plugin@marketplace"},
				MCPServers: []MCPServer{
					{Name: "user-mcp", Command: "cmd"},
				},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-plugin@marketplace"},
			},
			Local: &ScopeSettings{
				Plugins: []string{"local-plugin@marketplace"},
			},
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify JSON contains expected structure
	jsonStr := string(data)
	if !containsSubstring(jsonStr, `"perScope"`) {
		t.Errorf("expected perScope field in JSON, got:\n%s", jsonStr)
	}
	if !containsSubstring(jsonStr, `"user"`) {
		t.Errorf("expected user field in perScope, got:\n%s", jsonStr)
	}
	if !containsSubstring(jsonStr, `"project"`) {
		t.Errorf("expected project field in perScope, got:\n%s", jsonStr)
	}
	if !containsSubstring(jsonStr, `"local"`) {
		t.Errorf("expected local field in perScope, got:\n%s", jsonStr)
	}
	if !containsSubstring(jsonStr, `"user-plugin@marketplace"`) {
		t.Errorf("expected user-plugin in JSON, got:\n%s", jsonStr)
	}
	if !containsSubstring(jsonStr, `"project-plugin@marketplace"`) {
		t.Errorf("expected project-plugin in JSON, got:\n%s", jsonStr)
	}
	if !containsSubstring(jsonStr, `"local-plugin@marketplace"`) {
		t.Errorf("expected local-plugin in JSON, got:\n%s", jsonStr)
	}
}

func TestPerScopeSettingsJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"name": "test",
		"description": "Test profile",
		"perScope": {
			"user": {
				"plugins": ["user-plugin@marketplace"],
				"mcpServers": [{"name": "user-mcp", "command": "cmd"}]
			},
			"project": {
				"plugins": ["project-plugin@marketplace"]
			},
			"local": {
				"plugins": ["local-plugin@marketplace"]
			}
		}
	}`

	var p Profile
	if err := json.Unmarshal([]byte(jsonData), &p); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify parsed structure
	if p.PerScope == nil {
		t.Fatal("expected PerScope to be non-nil")
	}
	if p.PerScope.User == nil {
		t.Fatal("expected PerScope.User to be non-nil")
	}
	if len(p.PerScope.User.Plugins) != 1 {
		t.Errorf("expected 1 user plugin, got %d", len(p.PerScope.User.Plugins))
	}
	if p.PerScope.User.Plugins[0] != "user-plugin@marketplace" {
		t.Errorf("expected user-plugin@marketplace, got %s", p.PerScope.User.Plugins[0])
	}
	if len(p.PerScope.User.MCPServers) != 1 {
		t.Errorf("expected 1 user MCP server, got %d", len(p.PerScope.User.MCPServers))
	}

	if p.PerScope.Project == nil {
		t.Fatal("expected PerScope.Project to be non-nil")
	}
	if len(p.PerScope.Project.Plugins) != 1 {
		t.Errorf("expected 1 project plugin, got %d", len(p.PerScope.Project.Plugins))
	}

	if p.PerScope.Local == nil {
		t.Fatal("expected PerScope.Local to be non-nil")
	}
	if len(p.PerScope.Local.Plugins) != 1 {
		t.Errorf("expected 1 local plugin, got %d", len(p.PerScope.Local.Plugins))
	}
}

func TestLegacyProfileBackwardCompatibility(t *testing.T) {
	// Legacy format without perScope
	jsonData := `{
		"name": "legacy",
		"description": "Legacy profile",
		"plugins": ["plugin-a@marketplace", "plugin-b@marketplace"],
		"mcpServers": [
			{"name": "mcp-server", "command": "cmd", "scope": "user"}
		]
	}`

	var p Profile
	if err := json.Unmarshal([]byte(jsonData), &p); err != nil {
		t.Fatalf("failed to unmarshal legacy profile: %v", err)
	}

	// Legacy profile should have PerScope as nil
	if p.PerScope != nil {
		t.Error("expected PerScope to be nil for legacy profiles")
	}

	// Legacy fields should be populated
	if len(p.Plugins) != 2 {
		t.Errorf("expected 2 plugins in legacy format, got %d", len(p.Plugins))
	}
	if len(p.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server in legacy format, got %d", len(p.MCPServers))
	}
}

func TestIsMultiScope(t *testing.T) {
	tests := []struct {
		name     string
		profile  *Profile
		expected bool
	}{
		{
			name:     "nil profile",
			profile:  nil,
			expected: false,
		},
		{
			name:     "legacy profile without PerScope",
			profile:  &Profile{Name: "legacy", Plugins: []string{"p1"}},
			expected: false,
		},
		{
			name:     "multi-scope profile with PerScope",
			profile:  &Profile{Name: "multi", PerScope: &PerScopeSettings{}},
			expected: true,
		},
		{
			name: "multi-scope profile with data",
			profile: &Profile{
				Name: "multi",
				PerScope: &PerScopeSettings{
					User: &ScopeSettings{Plugins: []string{"p1"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.profile.IsMultiScope()
			if result != tt.expected {
				t.Errorf("expected IsMultiScope()=%v, got %v", tt.expected, result)
			}
		})
	}
}

func TestForScope(t *testing.T) {
	multiProfile := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-p1", "user-p2"},
				MCPServers: []MCPServer{
					{Name: "user-mcp", Command: "cmd"},
				},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-p1"},
			},
			Local: &ScopeSettings{
				Plugins: []string{"local-p1"},
			},
		},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "user/repo"},
		},
	}

	// Test extracting user scope
	userProfile := multiProfile.ForScope("user")
	if userProfile == nil {
		t.Fatal("ForScope(user) returned nil")
	}
	if len(userProfile.Plugins) != 2 {
		t.Errorf("expected 2 user plugins, got %d", len(userProfile.Plugins))
	}
	if len(userProfile.MCPServers) != 1 {
		t.Errorf("expected 1 user MCP server, got %d", len(userProfile.MCPServers))
	}
	// Marketplaces should be included (they're always user-scoped)
	if len(userProfile.Marketplaces) != 1 {
		t.Errorf("expected 1 marketplace, got %d", len(userProfile.Marketplaces))
	}

	// Test extracting project scope
	projectProfile := multiProfile.ForScope("project")
	if projectProfile == nil {
		t.Fatal("ForScope(project) returned nil")
	}
	if len(projectProfile.Plugins) != 1 {
		t.Errorf("expected 1 project plugin, got %d", len(projectProfile.Plugins))
	}

	// Test extracting local scope
	localProfile := multiProfile.ForScope("local")
	if localProfile == nil {
		t.Fatal("ForScope(local) returned nil")
	}
	if len(localProfile.Plugins) != 1 {
		t.Errorf("expected 1 local plugin, got %d", len(localProfile.Plugins))
	}

	// Test extracting non-existent scope returns empty profile
	emptyProfile := multiProfile.ForScope("nonexistent")
	if emptyProfile == nil {
		t.Fatal("ForScope(nonexistent) returned nil")
	}
	if len(emptyProfile.Plugins) != 0 {
		t.Errorf("expected 0 plugins for nonexistent scope, got %d", len(emptyProfile.Plugins))
	}
}

func TestCombinedScopes(t *testing.T) {
	// Test multi-scope profile
	multiProfile := &Profile{
		Name: "test",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				Plugins: []string{"user-p1", "user-p2"},
				MCPServers: []MCPServer{
					{Name: "shared-mcp", Command: "user-cmd"},
					{Name: "user-only-mcp", Command: "user-cmd"},
				},
			},
			Project: &ScopeSettings{
				Plugins: []string{"project-p1", "user-p1"}, // user-p1 is a duplicate
				MCPServers: []MCPServer{
					{Name: "shared-mcp", Command: "project-cmd"}, // overrides user-scope
				},
			},
			Local: &ScopeSettings{
				Plugins: []string{"local-p1"},
			},
		},
		Marketplaces: []Marketplace{
			{Source: "github", Repo: "user/repo"},
		},
	}

	combined := multiProfile.CombinedScopes()

	// Should have unique plugins from all scopes (4 unique: user-p1, user-p2, project-p1, local-p1)
	if len(combined.Plugins) != 4 {
		t.Errorf("expected 4 unique plugins, got %d: %v", len(combined.Plugins), combined.Plugins)
	}

	// Should have 2 MCP servers (shared-mcp and user-only-mcp)
	// shared-mcp should have project-cmd (later scope overrides)
	if len(combined.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d: %v", len(combined.MCPServers), combined.MCPServers)
	}
	for _, server := range combined.MCPServers {
		if server.Name == "shared-mcp" && server.Command != "project-cmd" {
			t.Errorf("expected shared-mcp to have project-cmd (override), got %s", server.Command)
		}
	}

	// Marketplaces should be preserved
	if len(combined.Marketplaces) != 1 {
		t.Errorf("expected 1 marketplace, got %d", len(combined.Marketplaces))
	}

	// Test legacy profile (no PerScope)
	legacyProfile := &Profile{
		Name:       "legacy",
		Plugins:    []string{"legacy-p1", "legacy-p2"},
		MCPServers: []MCPServer{{Name: "legacy-mcp", Command: "cmd"}},
	}

	legacyCombined := legacyProfile.CombinedScopes()
	if len(legacyCombined.Plugins) != 2 {
		t.Errorf("expected 2 plugins for legacy profile, got %d", len(legacyCombined.Plugins))
	}
	if len(legacyCombined.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server for legacy profile, got %d", len(legacyCombined.MCPServers))
	}

	// Test nil profile
	var nilProfile *Profile
	nilCombined := nilProfile.CombinedScopes()
	if nilCombined == nil {
		t.Fatal("CombinedScopes() on nil profile should return empty profile, not nil")
	}
	if len(nilCombined.Plugins) != 0 {
		t.Errorf("expected 0 plugins for nil profile, got %d", len(nilCombined.Plugins))
	}
}

func TestOmitEmptyPerScope(t *testing.T) {
	// Profile with nil PerScope should not include perScope in JSON
	p := &Profile{
		Name:    "simple",
		Plugins: []string{"plugin1"},
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	if containsSubstring(jsonStr, `"perScope"`) {
		t.Errorf("expected perScope to be omitted when nil, got:\n%s", jsonStr)
	}
}

func TestEmptyPerScopeSections(t *testing.T) {
	// Test profile with PerScope but all sections nil
	p := &Profile{
		Name:     "empty-multi",
		PerScope: &PerScopeSettings{},
	}

	// IsMultiScope should return true (PerScope is non-nil)
	if !p.IsMultiScope() {
		t.Error("expected IsMultiScope() to return true for profile with empty PerScope")
	}

	// ForScope should return empty profile for all scopes
	for _, scope := range []string{"user", "project", "local"} {
		scopeProfile := p.ForScope(scope)
		if scopeProfile == nil {
			t.Errorf("ForScope(%s) returned nil", scope)
			continue
		}
		if len(scopeProfile.Plugins) != 0 {
			t.Errorf("expected 0 plugins for %s scope, got %d", scope, len(scopeProfile.Plugins))
		}
		if len(scopeProfile.MCPServers) != 0 {
			t.Errorf("expected 0 MCP servers for %s scope, got %d", scope, len(scopeProfile.MCPServers))
		}
	}

	// CombinedScopes should return empty profile
	combined := p.CombinedScopes()
	if combined == nil {
		t.Fatal("CombinedScopes() returned nil")
	}
	if len(combined.Plugins) != 0 {
		t.Errorf("expected 0 plugins in combined, got %d", len(combined.Plugins))
	}

	// JSON marshaling should work correctly
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	jsonStr := string(data)
	// Empty PerScope should be omitted due to omitempty on nested fields
	t.Logf("JSON output:\n%s", jsonStr)
}

func TestHasMCPServersWithSecrets(t *testing.T) {
	// Test nil profile
	var nilProfile *Profile
	if nilProfile.HasMCPServersWithSecrets() {
		t.Error("nil profile should not have MCP servers with secrets")
	}

	// Test profile without MCP servers
	noMCP := &Profile{
		Name:    "no-mcp",
		Plugins: []string{"plugin1"},
	}
	if noMCP.HasMCPServersWithSecrets() {
		t.Error("profile without MCP servers should return false")
	}

	// Test legacy profile with MCP servers without secrets
	noSecrets := &Profile{
		Name: "no-secrets",
		MCPServers: []MCPServer{
			{Name: "server1", Command: "cmd"},
		},
	}
	if noSecrets.HasMCPServersWithSecrets() {
		t.Error("profile with MCP servers without secrets should return false")
	}

	// Test legacy profile with MCP servers with secrets
	withSecrets := &Profile{
		Name: "with-secrets",
		MCPServers: []MCPServer{
			{Name: "server1", Command: "cmd", Secrets: map[string]SecretRef{
				"API_KEY": {Sources: []SecretSource{{Type: "env", Key: "MY_API_KEY"}}},
			}},
		},
	}
	if !withSecrets.HasMCPServersWithSecrets() {
		t.Error("profile with MCP servers with secrets should return true")
	}

	// Test multi-scope profile with secrets in user scope
	multiUserSecrets := &Profile{
		Name: "multi-user-secrets",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "server1", Command: "cmd", Secrets: map[string]SecretRef{
						"TOKEN": {Sources: []SecretSource{{Type: "env", Key: "TOKEN"}}},
					}},
				},
			},
		},
	}
	if !multiUserSecrets.HasMCPServersWithSecrets() {
		t.Error("multi-scope profile with secrets in user scope should return true")
	}

	// Test multi-scope profile with secrets in project scope
	multiProjectSecrets := &Profile{
		Name: "multi-project-secrets",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "no-secrets", Command: "cmd"},
				},
			},
			Project: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "with-secrets", Command: "cmd", Secrets: map[string]SecretRef{
						"DB_PASS": {Sources: []SecretSource{{Type: "keychain", Service: "db-password"}}},
					}},
				},
			},
		},
	}
	if !multiProjectSecrets.HasMCPServersWithSecrets() {
		t.Error("multi-scope profile with secrets in project scope should return true")
	}

	// Test multi-scope profile without any secrets
	multiNoSecrets := &Profile{
		Name: "multi-no-secrets",
		PerScope: &PerScopeSettings{
			User: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "server1", Command: "cmd"},
				},
			},
			Project: &ScopeSettings{
				MCPServers: []MCPServer{
					{Name: "server2", Command: "cmd"},
				},
			},
		},
	}
	if multiNoSecrets.HasMCPServersWithSecrets() {
		t.Error("multi-scope profile without secrets should return false")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (containsSubstringImpl(s, substr)))
}

func containsSubstringImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
