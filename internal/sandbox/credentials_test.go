// ABOUTME: Unit tests for credential resolution.
// ABOUTME: Tests credential type mapping and mount generation.
package sandbox

import (
	"testing"
)

func TestCredentialTypes(t *testing.T) {
	t.Run("git credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("git")
		if cred == nil {
			t.Fatal("git credential type not found")
		}
		if cred.SourceSuffix != ".gitconfig" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".gitconfig")
		}
		if cred.TargetPath != "/root/.gitconfig" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.gitconfig")
		}
	})

	t.Run("ssh credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("ssh")
		if cred == nil {
			t.Fatal("ssh credential type not found")
		}
		if cred.SourceSuffix != ".ssh" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".ssh")
		}
		if cred.TargetPath != "/root/.ssh" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.ssh")
		}
	})

	t.Run("gh credential has correct paths", func(t *testing.T) {
		cred := GetCredentialType("gh")
		if cred == nil {
			t.Fatal("gh credential type not found")
		}
		if cred.SourceSuffix != ".config/gh" {
			t.Errorf("wrong source: got %q, want %q", cred.SourceSuffix, ".config/gh")
		}
		if cred.TargetPath != "/root/.config/gh" {
			t.Errorf("wrong target: got %q, want %q", cred.TargetPath, "/root/.config/gh")
		}
	})

	t.Run("unknown credential returns nil", func(t *testing.T) {
		cred := GetCredentialType("unknown")
		if cred != nil {
			t.Error("expected nil for unknown credential type")
		}
	})
}
