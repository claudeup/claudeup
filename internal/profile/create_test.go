// ABOUTME: Unit tests for non-interactive profile creation
// ABOUTME: Tests validation and profile construction from specs
package profile

import (
	"strings"
	"testing"
)

func TestParseMarketplaceArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    Marketplace
		wantErr bool
	}{
		{
			name: "shorthand format",
			arg:  "anthropics/claude-code",
			want: Marketplace{Source: "github", Repo: "anthropics/claude-code"},
		},
		{
			name:    "empty string",
			arg:     "",
			wantErr: true,
		},
		{
			name:    "no slash",
			arg:     "invalid",
			wantErr: true,
		},
		// Edge cases documented per code review feedback
		{
			name: "whitespace around parts is trimmed",
			arg:  "  owner  /  repo  ",
			want: Marketplace{Source: "github", Repo: "owner/repo"},
		},
		{
			name:    "whitespace-only owner rejected",
			arg:     "   /repo",
			wantErr: true,
		},
		{
			name:    "whitespace-only repo rejected",
			arg:     "owner/   ",
			wantErr: true,
		},
		{
			name: "multiple slashes preserved in repo field",
			arg:  "owner/repo/extra/path",
			want: Marketplace{Source: "github", Repo: "owner/repo/extra/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMarketplaceArg(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarketplaceArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Source != tt.want.Source || got.Repo != tt.want.Repo) {
				t.Errorf("ParseMarketplaceArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePluginFormat(t *testing.T) {
	tests := []struct {
		name    string
		plugin  string
		wantErr bool
	}{
		{
			name:   "valid format",
			plugin: "plugin-dev@claude-code-plugins",
		},
		{
			name:   "valid with colons",
			plugin: "backend:api-design@claude-workflows",
		},
		// Edge case: plugin names can contain @ - LastIndex finds the separator
		{
			name:   "plugin name containing at sign uses last separator",
			plugin: "team@corp@marketplace-ref",
		},
		{
			name:    "empty string",
			plugin:  "",
			wantErr: true,
		},
		{
			name:    "no at sign",
			plugin:  "invalid-plugin",
			wantErr: true,
		},
		{
			name:    "empty marketplace ref",
			plugin:  "plugin@",
			wantErr: true,
		},
		{
			name:    "empty plugin name",
			plugin:  "@marketplace",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginFormat(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePluginFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateFromFlags(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		description string
		markets     []string
		plugins     []string
		scope       string
		wantErr     bool
	}{
		{
			name:        "creates profile with user scope by default",
			profileName: "test-profile",
			description: "Test description",
			markets:     []string{"anthropics/claude-code", "obra/superpowers"},
			plugins:     []string{"plugin-dev@claude-code-plugins"},
			scope:       "user",
			wantErr:     false,
		},
		{
			name:        "creates profile with project scope",
			profileName: "project-profile",
			description: "Project tools",
			markets:     []string{"anthropics/claude-code"},
			plugins:     []string{"plugin-dev@claude-code-plugins"},
			scope:       "project",
			wantErr:     false,
		},
		{
			name:        "creates profile with local scope",
			profileName: "local-profile",
			description: "Local tools",
			markets:     []string{"anthropics/claude-code"},
			plugins:     []string{"plugin-dev@claude-code-plugins"},
			scope:       "local",
			wantErr:     false,
		},
		{
			name:        "fails on validation error",
			profileName: "test",
			description: "",
			markets:     []string{"owner/repo"},
			scope:       "user",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := CreateFromFlags(tt.profileName, tt.description, tt.markets, tt.plugins, tt.scope)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFromFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if p.Name != tt.profileName {
					t.Errorf("CreateFromFlags() name = %v, want %v", p.Name, tt.profileName)
				}
				if p.Description != tt.description {
					t.Errorf("CreateFromFlags() description = %v, want %v", p.Description, tt.description)
				}
				if len(p.Marketplaces) != len(tt.markets) {
					t.Errorf("CreateFromFlags() marketplaces = %v, want %v", len(p.Marketplaces), len(tt.markets))
				}
				// Verify multi-scope format
				if !p.IsMultiScope() {
					t.Fatal("CreateFromFlags() should produce multi-scope profile")
				}
				// Verify flat fields are empty
				if len(p.Plugins) > 0 {
					t.Errorf("CreateFromFlags() flat Plugins should be empty, got %v", p.Plugins)
				}
				if len(p.MCPServers) > 0 {
					t.Errorf("CreateFromFlags() flat MCPServers should be empty, got %v", p.MCPServers)
				}
				// Verify plugins are placed under the correct scope
				var scopeSettings *ScopeSettings
				switch tt.scope {
				case "user":
					scopeSettings = p.PerScope.User
				case "project":
					scopeSettings = p.PerScope.Project
				case "local":
					scopeSettings = p.PerScope.Local
				}
				if scopeSettings == nil {
					t.Fatalf("CreateFromFlags() PerScope.%s should not be nil", tt.scope)
				}
				if len(scopeSettings.Plugins) != len(tt.plugins) {
					t.Errorf("CreateFromFlags() PerScope.%s.Plugins = %v, want %v", tt.scope, len(scopeSettings.Plugins), len(tt.plugins))
				}
			}
		})
	}
}

func TestValidateCreateSpec(t *testing.T) {
	tests := []struct {
		name        string
		description string
		markets     []string
		plugins     []string
		wantErr     string
	}{
		{
			name:        "valid minimal",
			description: "Test profile",
			markets:     []string{"owner/repo"},
			plugins:     []string{},
			wantErr:     "",
		},
		{
			name:        "valid with plugins",
			description: "Test profile",
			markets:     []string{"owner/repo"},
			plugins:     []string{"plugin@ref"},
			wantErr:     "",
		},
		{
			name:        "missing description",
			description: "",
			markets:     []string{"owner/repo"},
			wantErr:     "description is required",
		},
		{
			name:        "missing marketplaces",
			description: "Test",
			markets:     []string{},
			wantErr:     "at least one marketplace is required",
		},
		{
			name:        "invalid marketplace",
			description: "Test",
			markets:     []string{"invalid"},
			wantErr:     "invalid marketplace format",
		},
		{
			name:        "invalid plugin",
			description: "Test",
			markets:     []string{"owner/repo"},
			plugins:     []string{"no-at-sign"},
			wantErr:     "invalid plugin format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateSpec(tt.description, tt.markets, tt.plugins)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateCreateSpec() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateCreateSpec() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateCreateSpec() error = %v, want containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestCreateFromReader(t *testing.T) {
	tests := []struct {
		name         string
		profileName  string
		json         string
		descOverride string
		scope        string
		wantErr      string
	}{
		{
			name:        "flat input converts to multi-scope under user",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github", "repo": "owner/repo"}],
				"plugins": ["plugin@ref"]
			}`,
			scope:   "user",
			wantErr: "",
		},
		{
			name:        "flat input converts to multi-scope under project",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": ["owner/repo"],
				"plugins": ["plugin@ref"]
			}`,
			scope:   "project",
			wantErr: "",
		},
		{
			name:        "perScope input passes through directly",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": ["owner/repo"],
				"perScope": {
					"user": { "plugins": ["plugin@ref"] }
				}
			}`,
			scope:   "user",
			wantErr: "",
		},
		{
			name:        "rejects ambiguous input with both flat and perScope",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": ["owner/repo"],
				"plugins": ["plugin@ref"],
				"perScope": {
					"user": { "plugins": ["other@ref"] }
				}
			}`,
			scope:   "user",
			wantErr: "cannot specify both flat plugins/mcpServers and perScope",
		},
		// Object-format marketplace validation
		{
			name:        "object marketplace with empty repo rejected",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github", "repo": ""}]
			}`,
			scope:   "user",
			wantErr: "marketplace repo cannot be empty",
		},
		{
			name:        "object marketplace with missing repo rejected",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github"}]
			}`,
			scope:   "user",
			wantErr: "marketplace repo cannot be empty",
		},
		{
			name:        "valid JSON with shorthand marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": ["owner/repo"],
				"plugins": []
			}`,
			scope:   "user",
			wantErr: "",
		},
		{
			name:        "description override",
			profileName: "my-profile",
			json: `{
				"description": "Original",
				"marketplaces": ["owner/repo"]
			}`,
			descOverride: "Overridden",
			scope:        "user",
			wantErr:      "",
		},
		{
			name:        "invalid JSON",
			profileName: "my-profile",
			json:        `{invalid`,
			scope:       "user",
			wantErr:     "invalid JSON",
		},
		{
			name:        "missing description",
			profileName: "my-profile",
			json: `{
				"marketplaces": ["owner/repo"]
			}`,
			scope:   "user",
			wantErr: "description is required",
		},
		{
			name:        "missing marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test"
			}`,
			scope:   "user",
			wantErr: "at least one marketplace is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.json)
			p, err := CreateFromReader(tt.profileName, r, tt.descOverride, tt.scope)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("CreateFromReader() unexpected error = %v", err)
					return
				}
				if p.Name != tt.profileName {
					t.Errorf("CreateFromReader() name = %v, want %v", p.Name, tt.profileName)
				}
				if tt.descOverride != "" && p.Description != tt.descOverride {
					t.Errorf("CreateFromReader() description = %v, want %v", p.Description, tt.descOverride)
				}
				// Verify multi-scope format
				if !p.IsMultiScope() {
					t.Fatal("CreateFromReader() should produce multi-scope profile")
				}
			} else {
				if err == nil {
					t.Errorf("CreateFromReader() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CreateFromReader() error = %v, want containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestCreateFromReaderSizeLimit(t *testing.T) {
	// Create a reader that would exceed 10MB
	// We use a LimitedReader simulation by creating oversized input
	oversizedJSON := `{"description": "Test", "marketplaces": ["owner/repo"], "plugins": ["` +
		strings.Repeat("x", 11*1024*1024) + `@ref"]}`

	r := strings.NewReader(oversizedJSON)
	_, err := CreateFromReader("test", r, "", "user")

	if err == nil {
		t.Error("CreateFromReader() expected error for oversized input, got nil")
	}
	if !strings.Contains(err.Error(), "input too large") {
		t.Errorf("CreateFromReader() error = %v, want containing 'input too large'", err)
	}
}

func TestCreateFromReaderNilSlices(t *testing.T) {
	// Verify that omitting plugins/mcpServers results in empty slices in PerScope
	json := `{
		"description": "Test profile",
		"marketplaces": ["owner/repo"]
	}`

	r := strings.NewReader(json)
	p, err := CreateFromReader("test", r, "", "user")
	if err != nil {
		t.Fatalf("CreateFromReader() unexpected error = %v", err)
	}

	if !p.IsMultiScope() {
		t.Fatal("CreateFromReader() should produce multi-scope profile")
	}
	if p.PerScope.User == nil {
		t.Fatal("CreateFromReader() PerScope.User should not be nil")
	}
	if p.PerScope.User.Plugins == nil {
		t.Error("CreateFromReader() PerScope.User.Plugins should be empty slice, not nil")
	}
	if len(p.PerScope.User.Plugins) != 0 {
		t.Errorf("CreateFromReader() PerScope.User.Plugins should be empty, got %v", p.PerScope.User.Plugins)
	}
	if p.PerScope.User.MCPServers == nil {
		t.Error("CreateFromReader() PerScope.User.MCPServers should be empty slice, not nil")
	}
	if len(p.PerScope.User.MCPServers) != 0 {
		t.Errorf("CreateFromReader() PerScope.User.MCPServers should be empty, got %v", p.PerScope.User.MCPServers)
	}
}

func TestValidatePluginMarketplaces(t *testing.T) {
	tests := []struct {
		name         string
		plugins      []string
		marketplaces []Marketplace
		registryKeys []string
		wantErrs     []string
	}{
		{
			name:    "plugin matches marketplace in profile by suffix",
			plugins: []string{"my-tool@claude-code-plugins"},
			marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			registryKeys: nil,
		},
		{
			name:    "plugin matches marketplace in profile by full repo",
			plugins: []string{"my-tool@anthropics/claude-code-plugins"},
			marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			registryKeys: nil,
		},
		{
			name:         "plugin matches registry key",
			plugins:      []string{"my-tool@claude-code-plugins"},
			marketplaces: nil,
			registryKeys: []string{"claude-code-plugins"},
		},
		{
			name:         "plugin with no matching marketplace or registry key",
			plugins:      []string{"my-tool@nonexistent-marketplace"},
			marketplaces: nil,
			registryKeys: nil,
			wantErrs:     []string{"my-tool@nonexistent-marketplace"},
		},
		{
			name: "multiple plugins some matching some not",
			plugins: []string{
				"good-tool@claude-code-plugins",
				"bad-tool@fake-marketplace",
				"other-bad@also-fake",
			},
			marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			registryKeys: nil,
			wantErrs:     []string{"bad-tool@fake-marketplace", "other-bad@also-fake"},
		},
		{
			name:    "multiple unresolvable plugins listed in error",
			plugins: []string{"bad1@fake1", "bad2@fake2"},
			marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			registryKeys: nil,
			wantErrs:     []string{"bad1@fake1", "bad2@fake2"},
		},
		{
			name:         "empty plugins passes",
			plugins:      []string{},
			marketplaces: nil,
			registryKeys: nil,
		},
		{
			name:         "nil plugins passes",
			plugins:      nil,
			marketplaces: nil,
			registryKeys: nil,
		},
		{
			name:         "empty marketplaces and empty registry fails if plugins exist",
			plugins:      []string{"my-tool@some-marketplace"},
			marketplaces: []Marketplace{},
			registryKeys: []string{},
			wantErrs:     []string{"my-tool@some-marketplace"},
		},
		{
			name:    "plugin without @ separator is skipped",
			plugins: []string{"no-at-sign"},
			marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			registryKeys: nil,
		},
		{
			name:    "plugin with trailing @ is skipped",
			plugins: []string{"trailing-at@"},
			marketplaces: []Marketplace{
				{Source: "github", Repo: "anthropics/claude-code-plugins"},
			},
			registryKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginMarketplaces(tt.plugins, tt.marketplaces, tt.registryKeys)
			if len(tt.wantErrs) == 0 {
				if err != nil {
					t.Errorf("ValidatePluginMarketplaces() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidatePluginMarketplaces() expected error containing %v, got nil", tt.wantErrs)
				} else {
					for _, want := range tt.wantErrs {
						if !strings.Contains(err.Error(), want) {
							t.Errorf("ValidatePluginMarketplaces() error = %v, want containing %q", err, want)
						}
					}
				}
			}
		})
	}
}
