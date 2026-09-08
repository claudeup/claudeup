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

func TestMCPArgsEqual(t *testing.T) {
	cases := []struct {
		a, b []string
		want bool
	}{
		{[]string{"-y", "$KEY"}, []string{"-y", "${KEY}"}, true},
		{[]string{"${KEY}"}, []string{"$KEY"}, true},
		{[]string{"$KEY"}, []string{"$KEY"}, true},
		{[]string{"$KEY"}, []string{"$OTHER"}, false},
		{[]string{"$KEY"}, []string{"literal"}, false},
		{[]string{"$KEY"}, []string{"$KEY", "extra"}, false},
		{nil, []string{}, true},
	}
	for _, c := range cases {
		if got := mcpArgsEqual(c.a, c.b); got != c.want {
			t.Errorf("mcpArgsEqual(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// A live snapshot cannot carry secret descriptions or 1Password/keychain
// sources, so the live comparison must ignore the secrets map and accept
// either reference form; otherwise diff reports drift on every run.
func TestMCPServersEqualLiveIgnoresSecretMetadata(t *testing.T) {
	saved := MCPServer{
		Name:    "api",
		Command: "npx",
		Args:    []string{"--token", "$API_TOKEN"},
		Scope:   "user",
		Secrets: map[string]SecretRef{
			"API_TOKEN": {
				Description: "API token",
				Sources: []SecretSource{
					{Type: "env", Key: "API_TOKEN"},
					{Type: "1password", Ref: "op://vault/item/token"},
				},
			},
		},
	}
	live := MCPServer{
		Name:    "api",
		Command: "npx",
		Args:    []string{"--token", "${API_TOKEN}"},
		Scope:   "user",
	}
	if !mcpServersEqualLive(saved, live) {
		t.Error("expected saved $KEY with curated secrets to match live ${KEY}")
	}

	changed := live
	changed.Args = []string{"--token", "${OTHER}"}
	if mcpServersEqualLive(saved, changed) {
		t.Error("expected a different reference to be reported as drift")
	}
	if detail := mcpDiffDetail(saved, changed); detail != "args changed" {
		t.Errorf("mcpDiffDetail = %q, want %q", detail, "args changed")
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
