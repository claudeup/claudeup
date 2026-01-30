// ABOUTME: Tests for wildcard pattern matching
// ABOUTME: Verifies prefix (gsd-*) and directory (gsd/*) wildcards
package local

import "testing"

func TestMatchWildcard(t *testing.T) {
	items := []string{
		"gsd-planner.md",
		"gsd-executor.md",
		"gsd-verifier.md",
		"other-agent.md",
		"business-product/analyst.md",
		"business-product/strategist.md",
		"gsd/new-project.md",
		"gsd/execute-phase.md",
	}

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			"prefix wildcard",
			"gsd-*",
			[]string{"gsd-executor.md", "gsd-planner.md", "gsd-verifier.md"},
		},
		{
			"directory wildcard",
			"business-product/*",
			[]string{"business-product/analyst.md", "business-product/strategist.md"},
		},
		{
			"directory wildcard for commands",
			"gsd/*",
			[]string{"gsd/execute-phase.md", "gsd/new-project.md"},
		},
		{
			"global wildcard",
			"*",
			[]string{
				"business-product/analyst.md",
				"business-product/strategist.md",
				"gsd-executor.md",
				"gsd-planner.md",
				"gsd-verifier.md",
				"gsd/execute-phase.md",
				"gsd/new-project.md",
				"other-agent.md",
			},
		},
		{
			"exact match",
			"gsd-planner.md",
			[]string{"gsd-planner.md"},
		},
		{
			"no match",
			"nonexistent-*",
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchWildcard(tt.pattern, items)
			if len(got) != len(tt.want) {
				t.Errorf("MatchWildcard(%q) returned %d items, want %d: %v", tt.pattern, len(got), len(tt.want), got)
				return
			}
			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("MatchWildcard(%q)[%d] = %q, want %q", tt.pattern, i, got[i], want)
				}
			}
		})
	}
}
