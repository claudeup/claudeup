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

// testGumWizardIOWithOut is testGumWizardIO that also returns the stdout buffer,
// for tests that assert on user-facing feedback lines.
func testGumWizardIOWithOut(runner func(args ...string) ([]byte, error)) (WizardIO, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	wio := NewWizardIO(
		strings.NewReader(""),
		out,
		errBuf,
		func(name string) (string, error) { return "/usr/bin/gum", nil },
	)
	wio.GumRun = runner
	return wio, out, errBuf
}

// testFallbackWizardIO creates a WizardIO with gum unavailable and piped input,
// forcing the numbered-menu fallback paths. Returns the WizardIO and stdout buffer.
func testFallbackWizardIO(input string) (WizardIO, *bytes.Buffer) {
	out := &bytes.Buffer{}
	wio := NewWizardIO(
		strings.NewReader(input),
		out,
		&bytes.Buffer{},
		func(name string) (string, error) { return "", fmt.Errorf("executable file not found in $PATH") },
	)
	return wio, out
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
		{"trailing comma", "1,2,", 3, nil, "invalid selection: "},
		{"whitespace only", "   ", 3, nil, "no selection"},
		{"max zero rejects all", "1", 0, nil, "invalid selection: 1"},
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

func TestErrGumCanceled(t *testing.T) {
	t.Run("sentinel unwraps from wrapped error", func(t *testing.T) {
		wrapped := fmt.Errorf("marketplace selection cancelled: %w", ErrGumCanceled)
		if !errors.Is(wrapped, ErrGumCanceled) {
			t.Error("expected errors.Is to match ErrGumCanceled in wrapped error")
		}
	})

	t.Run("does not match unrelated errors", func(t *testing.T) {
		unrelated := fmt.Errorf("something else went wrong")
		if errors.Is(unrelated, ErrGumCanceled) {
			t.Error("expected errors.Is to NOT match ErrGumCanceled in unrelated error")
		}
	})
}

func TestSelectMarketplaces_CancelWrapsErrGumCanceled(t *testing.T) {
	exitErr := makeExitErrorWithCode(t, 1)
	wio, _ := testGumWizardIO(func(args ...string) ([]byte, error) {
		return nil, exitErr
	})

	available := []Marketplace{
		{Source: "github", Repo: "owner/first"},
	}
	_, err := SelectMarketplaces(wio, available)
	if err == nil {
		t.Fatal("expected error on cancel, got nil")
	}
	if !errors.Is(err, ErrGumCanceled) {
		t.Errorf("expected error to wrap ErrGumCanceled, got: %v", err)
	}
}

func TestSelectCategories_CancelWrapsErrGumCanceled(t *testing.T) {
	exitErr := makeExitErrorWithCode(t, 1)
	wio, _ := testGumWizardIO(func(args ...string) ([]byte, error) {
		return nil, exitErr
	})

	categories := []Category{
		{Name: "test-cat", Description: "A test category", Plugins: []string{"plugin-a"}},
	}
	_, err := selectCategories(wio, categories)
	if err == nil {
		t.Fatal("expected error on cancel, got nil")
	}
	if !errors.Is(err, ErrGumCanceled) {
		t.Errorf("expected error to wrap ErrGumCanceled, got: %v", err)
	}
}

func TestPromptForDescription_CancelReturnsDefault(t *testing.T) {
	exitErr := makeExitErrorWithCode(t, 1)
	wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
		return nil, exitErr // user says "no" to confirm
	})

	desc, err := PromptForDescription(wio, "Auto description")
	if err != nil {
		t.Fatalf("expected nil error on description cancel, got: %v", err)
	}
	if desc != "Auto description" {
		t.Errorf("expected auto description on cancel, got %q", desc)
	}
	if !strings.Contains(out.String(), "Using auto-generated description") {
		t.Errorf("expected feedback that the auto-generated description is used, got %q", out.String())
	}
}

func TestEditDescription_CancelReturnsPlaceholder(t *testing.T) {
	exitErr := makeExitErrorWithCode(t, 1)
	wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
		if args[0] == "confirm" {
			return nil, nil // user said "yes" to editing
		}
		return nil, exitErr // user cancelled gum write
	})

	desc, err := PromptForDescription(wio, "Auto description")
	if err != nil {
		t.Fatalf("expected nil error on editor cancel, got: %v", err)
	}
	if desc != "Auto description" {
		t.Errorf("expected placeholder on cancel, got %q", desc)
	}
	if !strings.Contains(out.String(), "Using auto-generated description") {
		t.Errorf("expected feedback that the auto-generated description is used, got %q", out.String())
	}
}

func TestEditDescription_EmptySubmitFeedback(t *testing.T) {
	wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
		if args[0] == "confirm" {
			return nil, nil // user said "yes" to editing
		}
		return []byte("  \n"), nil // user saved the editor with nothing typed
	})

	desc, err := PromptForDescription(wio, "Auto description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if desc != "Auto description" {
		t.Errorf("expected placeholder on empty submit, got %q", desc)
	}
	if !strings.Contains(out.String(), "Using auto-generated description") {
		t.Errorf("expected feedback that the auto-generated description is used, got %q", out.String())
	}
}

func TestPromptForDescription_EditedDescriptionHasNoFallbackFeedback(t *testing.T) {
	wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
		if args[0] == "confirm" {
			return nil, nil // user said "yes" to editing
		}
		return []byte("My custom description\n"), nil
	})

	desc, err := PromptForDescription(wio, "Auto description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if desc != "My custom description" {
		t.Errorf("expected edited description, got %q", desc)
	}
	if strings.Contains(out.String(), "Using auto-generated description") {
		t.Errorf("expected no fallback feedback when the description was edited, got %q", out.String())
	}
}

func TestFallbackDescriptionPrompt_DeclineFeedback(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"user answers n", "n\n"},
		{"user presses enter", "\n"},
		{"input ends (EOF)", ""},
		{"user answers y then submits empty description", "y\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wio, out := testFallbackWizardIO(tt.input)

			desc, err := PromptForDescription(wio, "Auto description")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if desc != "Auto description" {
				t.Errorf("expected auto description, got %q", desc)
			}
			if !strings.Contains(out.String(), "Using auto-generated description") {
				t.Errorf("expected feedback that the auto-generated description is used, got %q", out.String())
			}
		})
	}
}

func TestFallbackDescriptionPrompt_EditedDescriptionHasNoFallbackFeedback(t *testing.T) {
	wio, out := testFallbackWizardIO("y\nMy custom description\n")

	desc, err := PromptForDescription(wio, "Auto description")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if desc != "My custom description" {
		t.Errorf("expected edited description, got %q", desc)
	}
	if strings.Contains(out.String(), "Using auto-generated description") {
		t.Errorf("expected no fallback feedback when the description was edited, got %q", out.String())
	}
}

func TestRefinePluginSelection_CancelFeedback(t *testing.T) {
	t.Run("announces pre-selected plugins on cancel", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 1)
		wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
			return nil, exitErr
		})

		available := []string{"plugin-a", "plugin-b", "plugin-c"}
		installed := map[string]bool{"plugin-a@marketplace": true, "plugin-c@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 pre-selected plugins on cancel, got %v", result)
		}
		if !strings.Contains(out.String(), "Using pre-selected plugins (2)") {
			t.Errorf("expected feedback naming the pre-selected plugin count, got %q", out.String())
		}
	})

	t.Run("announces no plugins when nothing is pre-selected", func(t *testing.T) {
		exitErr := makeExitErrorWithCode(t, 1)
		wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
			return nil, exitErr
		})

		available := []string{"plugin-a", "plugin-b"}
		result, err := refinePluginSelection(wio, available, map[string]bool{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected no plugins on cancel with nothing pre-selected, got %v", result)
		}
		if !strings.Contains(out.String(), "No plugins selected") {
			t.Errorf("expected feedback that no plugins were selected, got %q", out.String())
		}
		if strings.Contains(out.String(), "Using pre-selected plugins") {
			t.Errorf("expected no pre-selected feedback when nothing is pre-selected, got %q", out.String())
		}
	})

	t.Run("no fallback feedback when user confirms a selection", func(t *testing.T) {
		wio, out, _ := testGumWizardIOWithOut(func(args ...string) ([]byte, error) {
			return []byte("plugin-b\n"), nil
		})

		available := []string{"plugin-a", "plugin-b"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "plugin-b" {
			t.Errorf("expected [plugin-b], got %v", result)
		}
		if out.String() != "" {
			t.Errorf("expected no output when a selection was confirmed, got %q", out.String())
		}
	})
}

func TestFallbackPluginRefinement_EmptyInputFeedback(t *testing.T) {
	t.Run("announces pre-selected plugins on empty input", func(t *testing.T) {
		wio, out := testFallbackWizardIO("\n")

		available := []string{"plugin-a", "plugin-b"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "plugin-a" {
			t.Errorf("expected [plugin-a], got %v", result)
		}
		if !strings.Contains(out.String(), "Using pre-selected plugins (1)") {
			t.Errorf("expected feedback naming the pre-selected plugin count, got %q", out.String())
		}
	})

	t.Run("announces no plugins when nothing is pre-selected", func(t *testing.T) {
		wio, out := testFallbackWizardIO("\n")

		available := []string{"plugin-a", "plugin-b"}
		result, err := refinePluginSelection(wio, available, map[string]bool{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected no plugins with nothing pre-selected, got %v", result)
		}
		if !strings.Contains(out.String(), "No plugins selected") {
			t.Errorf("expected feedback that no plugins were selected, got %q", out.String())
		}
	})

	t.Run("no fallback feedback when user enters a selection", func(t *testing.T) {
		wio, out := testFallbackWizardIO("2\n")

		available := []string{"plugin-a", "plugin-b"}
		installed := map[string]bool{"plugin-a@marketplace": true}
		result, err := refinePluginSelection(wio, available, installed)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "plugin-b" {
			t.Errorf("expected [plugin-b], got %v", result)
		}
		if strings.Contains(out.String(), "Using pre-selected plugins") ||
			strings.Contains(out.String(), "No plugins selected") {
			t.Errorf("expected no fallback feedback when a selection was entered, got %q", out.String())
		}
	})
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

	t.Run("no stderr warning on user cancellation", func(t *testing.T) {
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
		// User cancel is not a failure: feedback goes to stdout
		// (see TestRefinePluginSelection_CancelFeedback), never stderr.
		if errBuf.String() != "" {
			t.Errorf("expected no warning for user cancel, got %q", errBuf.String())
		}
	})
}
