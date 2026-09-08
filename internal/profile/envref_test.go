package profile

import "testing"

func TestEnvRefName(t *testing.T) {
	cases := []struct {
		arg  string
		name string
		ok   bool
	}{
		{"$API_KEY", "API_KEY", true},
		{"${API_KEY}", "API_KEY", true},
		{"$_x1", "_x1", true},
		{"$", "", false},
		{"${", "", false},
		{"${}", "", false},
		{"${API_KEY", "", false},
		{"$1", "", false},
		{"$API-KEY", "", false},
		{"prefix$API_KEY", "", false},
		{"Bearer ${API_KEY}", "", false},
		{"literal", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		name, ok := envRefName(c.arg)
		if ok != c.ok || name != c.name {
			t.Errorf("envRefName(%q) = (%q, %v), want (%q, %v)", c.arg, name, ok, c.name, c.ok)
		}
	}
}

func TestIsEnvPlaceholder(t *testing.T) {
	if !isEnvPlaceholder("${API_KEY}") {
		t.Error("expected ${API_KEY} to be a placeholder")
	}
	for _, arg := range []string{"$API_KEY", "${}", "${1}", "x${API_KEY}", "literal"} {
		if isEnvPlaceholder(arg) {
			t.Errorf("did not expect %q to be a placeholder", arg)
		}
	}
}

func TestPlaceholderArgs(t *testing.T) {
	in := []string{"-y", "$API_KEY", "${OTHER}", "$", "a$B"}
	got := placeholderArgs(in)
	want := []string{"-y", "${API_KEY}", "${OTHER}", "$", "a$B"}
	if len(got) != len(want) {
		t.Fatalf("placeholderArgs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
	if in[1] != "$API_KEY" {
		t.Errorf("placeholderArgs mutated its input: %v", in)
	}
	if placeholderArgs(nil) != nil {
		t.Error("expected nil for nil input")
	}
}
