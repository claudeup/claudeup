package claude

import "testing"

func TestScopePrecedence(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  int
	}{
		{"user is lowest", ScopeUser, 0},
		{"project is middle", ScopeProject, 1},
		{"local is highest", ScopeLocal, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScopePrecedence(tt.scope)
			if got != tt.want {
				t.Errorf("ScopePrecedence(%q) = %d, want %d", tt.scope, got, tt.want)
			}
		})
	}
}

func TestScopePrecedenceOrdering(t *testing.T) {
	// local > project > user
	if ScopePrecedence(ScopeLocal) <= ScopePrecedence(ScopeProject) {
		t.Error("local should have higher precedence than project")
	}
	if ScopePrecedence(ScopeProject) <= ScopePrecedence(ScopeUser) {
		t.Error("project should have higher precedence than user")
	}
}

func TestScopePrecedenceUnknown(t *testing.T) {
	got := ScopePrecedence("unknown")
	if got != -1 {
		t.Errorf("ScopePrecedence(%q) = %d, want -1", "unknown", got)
	}
}
