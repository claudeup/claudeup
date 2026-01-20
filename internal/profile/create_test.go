// ABOUTME: Unit tests for non-interactive profile creation
// ABOUTME: Tests validation and profile construction from specs
package profile

import (
	"testing"
)

func TestParseMarketplaceArg(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		want    Marketplace
		wantErr bool
	}{
		{
			name: "shorthand format",
			arg:  "anthropics/claude-code",
			want: Marketplace{Source: "github", Repo: "anthropics/claude-code"},
		},
		{
			name:    "empty string",
			arg:     "",
			wantErr: true,
		},
		{
			name:    "no slash",
			arg:     "invalid",
			wantErr: true,
		},
		// Edge cases documented per code review feedback
		{
			name: "whitespace around parts is trimmed",
			arg:  "  owner  /  repo  ",
			want: Marketplace{Source: "github", Repo: "owner/repo"},
		},
		{
			name:    "whitespace-only owner rejected",
			arg:     "   /repo",
			wantErr: true,
		},
		{
			name:    "whitespace-only repo rejected",
			arg:     "owner/   ",
			wantErr: true,
		},
		{
			name: "multiple slashes preserved in repo field",
			arg:  "owner/repo/extra/path",
			want: Marketplace{Source: "github", Repo: "owner/repo/extra/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMarketplaceArg(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarketplaceArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Source != tt.want.Source || got.Repo != tt.want.Repo) {
				t.Errorf("ParseMarketplaceArg() = %v, want %v", got, tt.want)
			}
		})
	}
}
