// ABOUTME: Tests for wizard functions
// ABOUTME: Validates name validation, description generation, gum error classification
package profile

import (
	"bytes"
	"errors"
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
// Returns the WizardIO and the stderr buffer for assertion.
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

// makeExitErrorWithCode returns an *exec.ExitError with the given exit code.
// Fails the test if the shell command does not produce an ExitError.
func makeExitErrorWithCode(t *testing.T, code int) *exec.ExitError {
	t.Helper()
	err := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("exec.Command(\"sh\", \"-c\", \"exit %d\").Run() returned %T, not *exec.ExitError", code, err)
	}
	return exitErr
}

func TestIsGumCancel(t *testing.T) {
	t.Run("true for exit code 1 (user declined)", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 1)
		if !isGumCancel(exitErr) {
			t.Error("expected isGumCancel to return true for exit code 1")
		}
	})

	t.Run("true for exit code 130 (SIGINT)", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 130)
		if !isGumCancel(exitErr) {
			t.Error("expected isGumCancel to return true for exit code 130")
		}
	})

	t.Run("false for exit code 2", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 2)
		if isGumCancel(exitErr) {
			t.Error("expected isGumCancel to return false for exit code 2")
		}
	})

	t.Run("false for non-ExitError", func(t *testing.T) {
		err := fmt.Errorf("permission denied")
		if isGumCancel(err) {
			t.Error("expected isGumCancel to return false for non-ExitError")
		}
	})

	t.Run("false for nil", func(t *testing.T) {
		if isGumCancel(nil) {
			t.Error("expected isGumCancel to return false for nil")
		}
	})
}

func TestParseNumberedSelection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		max     int
		want    []int
		wantErr string
	}{
		{"single valid", "2", 3, []int{1}, ""},
		{"multiple valid", "1,3", 3, []int{0, 2}, ""},
		{"with spaces", " 1 , 2 ", 3, []int{0, 1}, ""},
		{"deduplicates", "1,1,2", 3, []int{0, 1}, ""},
		{"zero invalid", "0", 3, nil, "invalid selection: 0"},
		{"over max invalid", "4", 3, nil, "invalid selection: 4"},
		{"non-numeric invalid", "abc", 3, nil, "invalid selection: abc"},
		{"empty invalid", "", 3, nil, "no selection"},
		{"negative invalid", "-1", 3, nil, "invalid selection: -1"},
		{"mixed valid and invalid", "1,abc", 3, nil, "invalid selection: abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNumberedSelection(tt.input, tt.max)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRefinePluginSelection_GumCrash(t *testing.T) {
	t.Run("returns error on gum crash", func(t *testing.T) {
		wio, errBuf := testGumWizardIO(func(args ...string) ([]byte, error) {
			return nil, fmt.Errorf("gum: version incompatible")
		})

		available := []string{"plugin-a", "plugin-b", "plugin-c"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err == nil {
			t.Fatal("expected error on gum crash, got nil")
		}
		if !strings.Contains(err.Error(), "plugin selection failed") {
			t.Errorf("expected error containing 'plugin selection failed', got %q", err.Error())
		}
		if result != nil {
			t.Errorf("expected nil result on crash, got %v", result)
		}
		if errBuf.String() != "" {
			t.Errorf("expected no stderr output (error propagated via return), got %q", errBuf.String())
		}
	})

	t.Run("returns error on non-cancel ExitError", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 2)
		wio, errBuf := testGumWizardIO(func(args ...string) ([]byte, error) {
			return nil, exitErr
		})

		available := []string{"plugin-a", "plugin-b"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err == nil {
			t.Fatal("expected error on non-cancel ExitError, got nil")
		}
		if !strings.Contains(err.Error(), "plugin selection failed") {
			t.Errorf("expected error containing 'plugin selection failed', got %q", err.Error())
		}
		if result != nil {
			t.Errorf("expected nil result on crash, got %v", result)
		}
		if errBuf.String() != "" {
			t.Errorf("expected no stderr output (error propagated via return), got %q", errBuf.String())
		}
		var unwrapped *exec.ExitError
		if !errors.As(err, &unwrapped) {
			t.Error("expected crash error to wrap *exec.ExitError for caller inspection")
		}
	})

	t.Run("silent on user cancellation", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 1)
		wio, errBuf := testGumWizardIO(func(args ...string) ([]byte, error) {
			return nil, exitErr
		})

		available := []string{"plugin-a", "plugin-b"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected pre-selected plugins on cancel, got empty")
		}
		if errBuf.String() != "" {
			t.Errorf("expected no warning for user cancel, got %q", errBuf.String())
		}
	})
}
