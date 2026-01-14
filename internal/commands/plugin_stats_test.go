// ABOUTME: Unit tests for plugin statistics calculation
// ABOUTME: Tests the extracted calculatePluginStatistics function
package commands

import (
	"testing"

	"github.com/claudeup/claudeup/v2/internal/claude"
)

func TestCalculatePluginStatistics(t *testing.T) {
	tests := []struct {
		name     string
		analysis map[string]*claude.PluginScopeInfo
		want     PluginStatistics
	}{
		{
			name:     "empty analysis returns zero counts",
			analysis: map[string]*claude.PluginScopeInfo{},
			want: PluginStatistics{
				Total:    0,
				Cached:   0,
				Local:    0,
				Enabled:  0,
				Disabled: 0,
				Stale:    0,
			},
		},
		{
			name: "counts enabled cached plugin",
			analysis: map[string]*claude.PluginScopeInfo{
				"test-plugin": {
					Name:         "test-plugin",
					ActiveSource: "user",
					EnabledAt:    []string{"user"},
					InstalledAt: []claude.PluginMetadata{
						{
							Scope:   "user",
							IsLocal: false,
						},
					},
				},
			},
			want: PluginStatistics{
				Total:    1,
				Cached:   1,
				Local:    0,
				Enabled:  1,
				Disabled: 0,
				Stale:    0,
			},
		},
		{
			name: "counts disabled local plugin",
			analysis: map[string]*claude.PluginScopeInfo{
				"local-plugin": {
					Name:         "local-plugin",
					ActiveSource: "",
					EnabledAt:    []string{},
					InstalledAt: []claude.PluginMetadata{
						{
							Scope:   "user",
							IsLocal: true,
						},
					},
				},
			},
			want: PluginStatistics{
				Total:    1,
				Cached:   0,
				Local:    1,
				Enabled:  0,
				Disabled: 1,
				Stale:    0,
			},
		},
		{
			name: "counts multiple plugins correctly",
			analysis: map[string]*claude.PluginScopeInfo{
				"enabled-cached": {
					Name:         "enabled-cached",
					ActiveSource: "user",
					EnabledAt:    []string{"user"},
					InstalledAt: []claude.PluginMetadata{
						{Scope: "user", IsLocal: false},
					},
				},
				"enabled-local": {
					Name:         "enabled-local",
					ActiveSource: "project",
					EnabledAt:    []string{"project"},
					InstalledAt: []claude.PluginMetadata{
						{Scope: "project", IsLocal: true},
					},
				},
				"disabled-cached": {
					Name:         "disabled-cached",
					ActiveSource: "",
					EnabledAt:    []string{},
					InstalledAt: []claude.PluginMetadata{
						{Scope: "user", IsLocal: false},
					},
				},
			},
			want: PluginStatistics{
				Total:    3,
				Cached:   2,
				Local:    1,
				Enabled:  2,
				Disabled: 1,
				Stale:    0,
			},
		},
		{
			name: "uses active source installation for type counting",
			analysis: map[string]*claude.PluginScopeInfo{
				"multi-scope": {
					Name:         "multi-scope",
					ActiveSource: "project",
					EnabledAt:    []string{"project"},
					InstalledAt: []claude.PluginMetadata{
						{Scope: "user", IsLocal: false},    // cached at user
						{Scope: "project", IsLocal: true},  // local at project (active)
					},
				},
			},
			want: PluginStatistics{
				Total:    1,
				Cached:   0,  // should use project's local, not user's cached
				Local:    1,
				Enabled:  1,
				Disabled: 0,
				Stale:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePluginStatistics(tt.analysis)

			if got.Total != tt.want.Total {
				t.Errorf("Total = %d, want %d", got.Total, tt.want.Total)
			}
			if got.Cached != tt.want.Cached {
				t.Errorf("Cached = %d, want %d", got.Cached, tt.want.Cached)
			}
			if got.Local != tt.want.Local {
				t.Errorf("Local = %d, want %d", got.Local, tt.want.Local)
			}
			if got.Enabled != tt.want.Enabled {
				t.Errorf("Enabled = %d, want %d", got.Enabled, tt.want.Enabled)
			}
			if got.Disabled != tt.want.Disabled {
				t.Errorf("Disabled = %d, want %d", got.Disabled, tt.want.Disabled)
			}
			// Note: Stale counting requires filesystem (PathExists checks os.Stat)
			// Stale detection is tested in acceptance tests with real files
		})
	}
}
