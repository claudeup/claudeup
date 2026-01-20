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
