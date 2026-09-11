// ABOUTME: Tests for Claude CLI version checking functions
// ABOUTME: Ensures version parsing and comparison work correctly
package commands

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"1.0.72", []int{1, 0, 72}},
		{"1.0.80", []int{1, 0, 80}},
		{"2.1.0", []int{2, 1, 0}},
		{"v1.0.72", []int{1, 0, 72}},
		{"claude 1.0.72", []int{1, 0, 72}},
		{"1.0.72-beta", []int{1, 0, 72}},
		{"1.0", []int{1, 0}},
		{"1", []int{1}},
		{"", []int{}},
		// Edge cases for invalid/unusual formats
		{"abc", []int{}},
		{"v", []int{}},
		{"...", []int{}},
		{"1.2.3.4.5", []int{1, 2, 3, 4, 5}},
		{"0.0.0", []int{0, 0, 0}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseVersion(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("parseVersion(%q) = %v, expected %v", tc.input, result, tc.expected)
				return
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("parseVersion(%q) = %v, expected %v", tc.input, result, tc.expected)
					return
				}
			}
		})
	}
}

func TestIsVersionOutdated(t *testing.T) {
	tests := []struct {
		current  string
		minimum  string
		expected bool
	}{
		// Same version
		{"1.0.80", "1.0.80", false},

		// Outdated versions
		{"1.0.72", "1.0.80", true},
		{"1.0.79", "1.0.80", true},
		{"0.9.99", "1.0.0", true},
		{"1.0.0", "2.0.0", true},

		// Newer versions
		{"1.0.81", "1.0.80", false},
		{"1.1.0", "1.0.80", false},
		{"2.0.0", "1.0.80", false},
		{"1.0.100", "1.0.80", false},

		// Different length versions
		{"1.0", "1.0.80", true},
		{"1.0.80.1", "1.0.80", false},

		// With prefixes (should be stripped)
		{"v1.0.72", "1.0.80", true},
		{"claude 1.0.72", "1.0.80", true},
		{"1.0.81", "v1.0.80", false},

		// Edge cases for invalid/unusual formats
		{"", "1.0.80", true},        // Empty version is outdated
		{"abc", "1.0.80", true},     // Non-numeric is outdated
		{"unknown", "1.0.80", true}, // "unknown" version is outdated
		{"1.0.80", "", false},       // Anything beats empty minimum
		{"0.0.0", "0.0.1", true},    // Zero versions work
		{"0.0.1", "0.0.0", false},   // Zero versions work
	}

	for _, tc := range tests {
		t.Run(tc.current+"_vs_"+tc.minimum, func(t *testing.T) {
			result := isVersionOutdated(tc.current, tc.minimum)
			if result != tc.expected {
				t.Errorf("isVersionOutdated(%q, %q) = %v, expected %v",
					tc.current, tc.minimum, result, tc.expected)
			}
		})
	}
}
