// ABOUTME: Tests for ext package core types and category validation
// ABOUTME: Verifies category constants and validation logic
package ext

import "testing"

func TestCategoryValidation(t *testing.T) {
	tests := []struct {
		name     string
		category string
		wantErr  bool
	}{
		{"valid agents", "agents", false},
		{"valid commands", "commands", false},
		{"valid skills", "skills", false},
		{"valid hooks", "hooks", false},
		{"valid rules", "rules", false},
		{"valid output-styles", "output-styles", false},
		{"invalid category", "invalid", true},
		{"empty category", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCategory(tt.category)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategory(%q) error = %v, wantErr %v", tt.category, err, tt.wantErr)
			}
		})
	}
}

func TestAllCategories(t *testing.T) {
	categories := AllCategories()
	if len(categories) != 6 {
		t.Errorf("AllCategories() returned %d categories, want 6", len(categories))
	}
}
