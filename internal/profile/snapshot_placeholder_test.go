package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// Apply writes $KEY secret references as ${KEY} placeholders (see #312), so
// the read-back path has to map them back to the profile form and record
// that the value comes from the environment. Otherwise diff and save would
// compare $KEY against ${KEY} and report drift forever.
func TestReadMCPServersForScopeNormalizesPlaceholders(t *testing.T) {
	tmpDir := t.TempDir()
	claudeJSONPath := filepath.Join(tmpDir, ".claude.json")
	content := `{"mcpServers":{"api":{"command":"npx","args":["-y","@my/mcp","--token","${API_TOKEN}","$BARE","literal${EMBEDDED}"]}}}`
	if err := os.WriteFile(claudeJSONPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	servers, err := ReadMCPServersForScope(claudeJSONPath, "", "user")
	if err != nil {
		t.Fatalf("ReadMCPServersForScope failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	srv := servers[0]

	want := []string{"-y", "@my/mcp", "--token", "$API_TOKEN", "$BARE", "literal${EMBEDDED}"}
	if len(srv.Args) != len(want) {
		t.Fatalf("args = %v, want %v", srv.Args, want)
	}
	for i := range want {
		if srv.Args[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, srv.Args[i], want[i])
		}
	}

	ref, ok := srv.Secrets["API_TOKEN"]
	if !ok {
		t.Fatalf("expected env SecretRef for API_TOKEN, got secrets: %v", srv.Secrets)
	}
	if len(ref.Sources) != 1 || ref.Sources[0].Type != "env" || ref.Sources[0].Key != "API_TOKEN" {
		t.Errorf("API_TOKEN sources = %v, want single env source", ref.Sources)
	}
	// A bare $BARE in the live config is a literal Claude Code does not
	// expand, so it must not be reported as an env-sourced secret.
	if _, ok := srv.Secrets["BARE"]; ok {
		t.Errorf("did not expect a SecretRef for bare $BARE, got: %v", srv.Secrets)
	}
}
