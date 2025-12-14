// ABOUTME: Tests for UI output helper functions
// ABOUTME: Verifies print helpers format messages correctly
package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintSuccess(t *testing.T) {
	output := captureOutput(func() {
		PrintSuccess("Operation completed")
	})

	if !strings.Contains(output, SymbolSuccess) {
		t.Errorf("Expected output to contain success symbol, got: %s", output)
	}
	if !strings.Contains(output, "Operation completed") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestPrintError(t *testing.T) {
	output := captureOutput(func() {
		PrintError("Something failed")
	})

	if !strings.Contains(output, SymbolError) {
		t.Errorf("Expected output to contain error symbol, got: %s", output)
	}
	if !strings.Contains(output, "Something failed") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestPrintWarning(t *testing.T) {
	output := captureOutput(func() {
		PrintWarning("Be careful")
	})

	if !strings.Contains(output, SymbolWarning) {
		t.Errorf("Expected output to contain warning symbol, got: %s", output)
	}
}

func TestPrintInfo(t *testing.T) {
	output := captureOutput(func() {
		PrintInfo("FYI")
	})

	if !strings.Contains(output, SymbolInfo) {
		t.Errorf("Expected output to contain info symbol, got: %s", output)
	}
}

func TestMutedReturnsString(t *testing.T) {
	result := Muted("secondary text")
	if result == "" {
		t.Error("Muted should return non-empty string")
	}
	if !strings.Contains(result, "secondary text") {
		t.Errorf("Muted should contain original text, got: %s", result)
	}
}

func TestBoldReturnsString(t *testing.T) {
	result := Bold("important")
	if result == "" {
		t.Error("Bold should return non-empty string")
	}
	if !strings.Contains(result, "important") {
		t.Errorf("Bold should contain original text, got: %s", result)
	}
}

func TestPrintMuted(t *testing.T) {
	output := captureOutput(func() {
		PrintMuted("secondary info")
	})

	if !strings.Contains(output, "secondary info") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestSuccessReturnsString(t *testing.T) {
	result := Success("done")
	if result == "" {
		t.Error("Success should return non-empty string")
	}
	if !strings.Contains(result, "done") {
		t.Errorf("Success should contain original text, got: %s", result)
	}
}

func TestErrorReturnsString(t *testing.T) {
	result := Error("failed")
	if result == "" {
		t.Error("Error should return non-empty string")
	}
	if !strings.Contains(result, "failed") {
		t.Errorf("Error should contain original text, got: %s", result)
	}
}

func TestWarningReturnsString(t *testing.T) {
	result := Warning("caution")
	if result == "" {
		t.Error("Warning should return non-empty string")
	}
	if !strings.Contains(result, "caution") {
		t.Errorf("Warning should contain original text, got: %s", result)
	}
}

func TestInfoReturnsString(t *testing.T) {
	result := Info("note")
	if result == "" {
		t.Error("Info should return non-empty string")
	}
	if !strings.Contains(result, "note") {
		t.Errorf("Info should contain original text, got: %s", result)
	}
}
