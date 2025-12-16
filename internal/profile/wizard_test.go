// ABOUTME: Tests for wizard functions
// ABOUTME: Validates name validation, description generation
package profile

import (
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid alphanumeric", "myprofile", false},
		{"valid with hyphen", "my-profile", false},
		{"valid with underscore", "my_profile", false},
		{"valid mixed", "My-Profile_123", false},
		{"empty string", "", true},
		{"reserved name", "current", true},
		{"invalid spaces", "my profile", true},
		{"invalid special chars", "my@profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateWizardDescription(t *testing.T) {
	tests := []struct {
		name             string
		marketplaceCount int
		pluginCount      int
		want             string
	}{
		{
			name:             "single marketplace single plugin",
			marketplaceCount: 1,
			pluginCount:      1,
			want:             "Custom profile with 1 plugin from 1 marketplace",
		},
		{
			name:             "multiple marketplaces multiple plugins",
			marketplaceCount: 3,
			pluginCount:      10,
			want:             "Custom profile with 10 plugins from 3 marketplaces",
		},
		{
			name:             "zero plugins",
			marketplaceCount: 0,
			pluginCount:      0,
			want:             "Custom profile with 0 plugins from 0 marketplaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWizardDescription(tt.marketplaceCount, tt.pluginCount)
			if got != tt.want {
				t.Errorf("GenerateWizardDescription(%d, %d) = %q, want %q",
					tt.marketplaceCount, tt.pluginCount, got, tt.want)
			}
		})
	}
}
