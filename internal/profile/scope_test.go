package profile

import (
	"testing"
)

func TestScope_String(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeUser, "user"},
		{ScopeProject, "project"},
		{ScopeLocal, "local"},
	}

	for _, tt := range tests {
		if got := tt.scope.String(); got != tt.want {
			t.Errorf("Scope.String() = %q, want %q", got, tt.want)
		}
	}
}

func TestScope_IsValid(t *testing.T) {
	tests := []struct {
		scope Scope
		want  bool
	}{
		{ScopeUser, true},
		{ScopeProject, true},
		{ScopeLocal, true},
		{Scope("invalid"), false},
		{Scope(""), false},
		{Scope("USER"), false}, // Case sensitive
	}

	for _, tt := range tests {
		if got := tt.scope.IsValid(); got != tt.want {
			t.Errorf("Scope(%q).IsValid() = %v, want %v", tt.scope, got, tt.want)
		}
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		input   string
		want    Scope
		wantErr bool
	}{
		{"user", ScopeUser, false},
		{"project", ScopeProject, false},
		{"local", ScopeLocal, false},
		{"invalid", "", true},
		{"", "", true},
		{"USER", "", true}, // Case sensitive
	}

	for _, tt := range tests {
		got, err := ParseScope(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseScope(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseScope(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
