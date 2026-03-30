// ABOUTME: Tests for wizard functions
// ABOUTME: Validates name validation, description generation, gum error classification
package profile

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid alphanumeric", "myprofile", false},
		{"valid with hyphen", "my-profile", false},
		{"valid with underscore", "my_profile", false},
		{"valid mixed", "My-Profile_123", false},
		{"empty string", "", true},
		{"reserved name", "current", true},
		{"invalid spaces", "my profile", true},
		{"invalid special chars", "my@profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateWizardDescription(t *testing.T) {
	tests := []struct {
		name             string
		marketplaceCount int
		pluginCount      int
		want             string
	}{
		{
			name:             "single marketplace single plugin",
			marketplaceCount: 1,
			pluginCount:      1,
			want:             "Custom profile with 1 plugin from 1 marketplace",
		},
		{
			name:             "multiple marketplaces multiple plugins",
			marketplaceCount: 3,
			pluginCount:      10,
			want:             "Custom profile with 10 plugins from 3 marketplaces",
		},
		{
			name:             "zero plugins",
			marketplaceCount: 0,
			pluginCount:      0,
			want:             "Custom profile with 0 plugins from 0 marketplaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWizardDescription(tt.marketplaceCount, tt.pluginCount)
			if got != tt.want {
				t.Errorf("GenerateWizardDescription(%d, %d) = %q, want %q",
					tt.marketplaceCount, tt.pluginCount, got, tt.want)
			}
		})
	}
}

// testGumWizardIO creates a WizardIO with gum available and a custom GumRun.
func testGumWizardIO(runner func(args ...string) ([]byte, error)) (WizardIO, *bytes.Buffer) {
	errBuf := &bytes.Buffer{}
	wio := NewWizardIO(
		strings.NewReader(""),
		&bytes.Buffer{},
		errBuf,
		func(name string) (string, error) { return "/usr/bin/gum", nil },
	)
	wio.GumRun = runner
	return wio, errBuf
}

func TestWarnIfGumCrash(t *testing.T) {
	t.Run("writes warning for non-ExitError", func(t *testing.T) {
		var buf bytes.Buffer
		warnIfGumCrash(fmt.Errorf("permission denied"), &buf, "editor failed")
		if !strings.Contains(buf.String(), "Warning:") {
			t.Errorf("expected warning, got %q", buf.String())
		}
		if !strings.Contains(buf.String(), "permission denied") {
			t.Errorf("expected error detail in warning, got %q", buf.String())
		}
	})

	t.Run("silent for ExitError (user cancel)", func(t *testing.T) {
		var buf bytes.Buffer
		exitErr := exec.Command("false").Run()
		warnIfGumCrash(exitErr, &buf, "editor failed")
		if buf.String() != "" {
			t.Errorf("expected no output for user cancel, got %q", buf.String())
		}
	})
}

func TestRefinePluginSelection_GumCrash(t *testing.T) {
	t.Run("warns on gum crash and returns pre-selected plugins", func(t *testing.T) {
		wio, errBuf := testGumWizardIO(func(args ...string) ([]byte, error) {
			return nil, fmt.Errorf("gum: version incompatible")
		})

		available := []string{"plugin-a", "plugin-b", "plugin-c"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return pre-selected plugins (installed ones)
		if len(result) == 0 {
			t.Fatal("expected pre-selected plugins, got empty")
		}
		if !strings.Contains(errBuf.String(), "Warning:") {
			t.Errorf("expected warning on stderr, got %q", errBuf.String())
		}
		if !strings.Contains(errBuf.String(), "version incompatible") {
			t.Errorf("expected error detail in warning, got %q", errBuf.String())
		}
	})

	t.Run("silent on user cancellation", func(t *testing.T) {
		exitErr := exec.Command("false").Run()
		wio, errBuf := testGumWizardIO(func(args ...string) ([]byte, error) {
			return nil, exitErr
		})

		available := []string{"plugin-a", "plugin-b"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		_, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if errBuf.String() != "" {
			t.Errorf("expected no warning for user cancel, got %q", errBuf.String())
		}
	})
}
