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
		wantErr     bool
	}{
		{
			name:        "creates valid profile",
			profileName: "test-profile",
			description: "Test description",
			markets:     []string{"anthropics/claude-code", "obra/superpowers"},
			plugins:     []string{"plugin-dev@claude-code-plugins"},
			wantErr:     false,
		},
		{
			name:        "fails on validation error",
			profileName: "test",
			description: "",
			markets:     []string{"owner/repo"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := CreateFromFlags(tt.profileName, tt.description, tt.markets, tt.plugins)
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
				if len(p.Plugins) != len(tt.plugins) {
					t.Errorf("CreateFromFlags() plugins = %v, want %v", len(p.Plugins), len(tt.plugins))
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
		wantErr      string
	}{
		{
			name:        "valid JSON with object marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github", "repo": "owner/repo"}],
				"plugins": ["plugin@ref"]
			}`,
			wantErr: "",
		},
		// Object-format marketplace validation
		{
			name:        "object marketplace with empty repo rejected",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github", "repo": ""}]
			}`,
			wantErr: "marketplace repo cannot be empty",
		},
		{
			name:        "object marketplace with missing repo rejected",
			profileName: "my-profile",
			json: `{
				"description": "Test profile",
				"marketplaces": [{"source": "github"}]
			}`,
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
			wantErr:      "",
		},
		{
			name:        "invalid JSON",
			profileName: "my-profile",
			json:        `{invalid`,
			wantErr:     "invalid JSON",
		},
		{
			name:        "missing description",
			profileName: "my-profile",
			json: `{
				"marketplaces": ["owner/repo"]
			}`,
			wantErr: "description is required",
		},
		{
			name:        "missing marketplaces",
			profileName: "my-profile",
			json: `{
				"description": "Test"
			}`,
			wantErr: "at least one marketplace is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.json)
			p, err := CreateFromReader(tt.profileName, r, tt.descOverride)
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
	_, err := CreateFromReader("test", r, "")

	if err == nil {
		t.Error("CreateFromReader() expected error for oversized input, got nil")
	}
	if !strings.Contains(err.Error(), "input too large") {
		t.Errorf("CreateFromReader() error = %v, want containing 'input too large'", err)
	}
}

func TestCreateFromReaderNilSlices(t *testing.T) {
	// Verify that omitting plugins/mcpServers results in empty slices, not nil
	json := `{
		"description": "Test profile",
		"marketplaces": ["owner/repo"]
	}`

	r := strings.NewReader(json)
	p, err := CreateFromReader("test", r, "")
	if err != nil {
		t.Fatalf("CreateFromReader() unexpected error = %v", err)
	}

	if p.Plugins == nil {
		t.Error("CreateFromReader() Plugins should be empty slice, not nil")
	}
	if len(p.Plugins) != 0 {
		t.Errorf("CreateFromReader() Plugins should be empty, got %v", p.Plugins)
	}
	if p.MCPServers == nil {
		t.Error("CreateFromReader() MCPServers should be empty slice, not nil")
	}
	if len(p.MCPServers) != 0 {
		t.Errorf("CreateFromReader() MCPServers should be empty, got %v", p.MCPServers)
	}
}
