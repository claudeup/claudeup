// ABOUTME: Unit tests for schema validation functions
// ABOUTME: Tests plugin registry and settings validation logic
package claude

import (
	"strings"
	"testing"
)

func TestValidatePluginRegistry(t *testing.T) {
	tests := []struct {
		name    string
		registry *PluginRegistry
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid V1 format",
			registry: &PluginRegistry{
				Version: 1,
				Plugins: make(map[string][]PluginMetadata),
			},
			wantErr: false,
		},
		{
			name: "valid V2 format with proper scope",
			registry: &PluginRegistry{
				Version: 2,
				Plugins: map[string][]PluginMetadata{
					"test-plugin": {{
						Scope:   "user",
						Version: "1.0.0",
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported version 3",
			registry: &PluginRegistry{
				Version: 3,
				Plugins: make(map[string][]PluginMetadata),
			},
			wantErr: true,
			errMsg:  "Found version: 3",
		},
		{
			name: "version 0 invalid",
			registry: &PluginRegistry{
				Version: 0,
				Plugins: make(map[string][]PluginMetadata),
			},
			wantErr: true,
			errMsg:  "Found version: 0",
		},
		{
			name: "V2 with empty metadata array",
			registry: &PluginRegistry{
				Version: 2,
				Plugins: map[string][]PluginMetadata{
					"test-plugin": {},
				},
			},
			wantErr: true,
			errMsg:  "has empty metadata array",
		},
		{
			name: "V2 with missing scope",
			registry: &PluginRegistry{
				Version: 2,
				Plugins: map[string][]PluginMetadata{
					"test-plugin": {{
						Scope:   "", // Missing scope
						Version: "1.0.0",
					}},
				},
			},
			wantErr: true,
			errMsg:  "missing required 'scope' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePluginRegistry(tt.registry)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePluginRegistry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validatePluginRegistry() error = %v, should contain %q", err, tt.errMsg)
			}
		})
	}
}
