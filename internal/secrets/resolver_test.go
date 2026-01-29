// ABOUTME: Tests for secret resolution chain
// ABOUTME: Validates multi-backend resolution with priority ordering
package secrets

import (
	"errors"
	"testing"
)

// mockResolver is a test resolver
type mockResolver struct {
	name      string
	available bool
	value     string
	err       error
}

func (m *mockResolver) Name() string    { return m.name }
func (m *mockResolver) Available() bool { return m.available }
func (m *mockResolver) Resolve(_ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.value, nil
}

func TestChainResolvesFirstAvailable(t *testing.T) {
	chain := NewChain(
		&mockResolver{name: "first", available: false, value: "wrong"},
		&mockResolver{name: "second", available: true, value: "correct"},
		&mockResolver{name: "third", available: true, value: "also-wrong"},
	)

	value, source, err := chain.Resolve("any-ref")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != "correct" {
		t.Errorf("Expected 'correct', got %q", value)
	}

	if source != "second" {
		t.Errorf("Expected source 'second', got %q", source)
	}
}

func TestChainSkipsErrors(t *testing.T) {
	chain := NewChain(
		&mockResolver{name: "failing", available: true, err: errors.New("failed")},
		&mockResolver{name: "working", available: true, value: "success"},
	)

	value, source, err := chain.Resolve("any-ref")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != "success" {
		t.Errorf("Expected 'success', got %q", value)
	}

	if source != "working" {
		t.Errorf("Expected source 'working', got %q", source)
	}
}

func TestChainErrorsWhenAllFail(t *testing.T) {
	chain := NewChain(
		&mockResolver{name: "first", available: false},
		&mockResolver{name: "second", available: true, err: errors.New("failed")},
	)

	_, _, err := chain.Resolve("any-ref")
	if err == nil {
		t.Error("Expected error when all resolvers fail")
	}
}

func TestEmptyChainErrors(t *testing.T) {
	chain := NewChain()

	_, _, err := chain.Resolve("any-ref")
	if err == nil {
		t.Error("Expected error for empty chain")
	}
}

func TestChainReturnsEmptyStringAsSuccess(t *testing.T) {
	// This tests the current behavior: empty string with nil error counts as success
	// Note: apply.go has additional checks for non-empty values
	chain := NewChain(
		&mockResolver{name: "empty", available: true, value: ""},
		&mockResolver{name: "hasvalue", available: true, value: "real-value"},
	)

	value, source, err := chain.Resolve("any-ref")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Chain returns empty string as success (first available resolver that doesn't error)
	if value != "" {
		t.Errorf("Expected empty string, got %q", value)
	}
	if source != "empty" {
		t.Errorf("Expected source 'empty', got %q", source)
	}
}

func TestChainAllUnavailable(t *testing.T) {
	chain := NewChain(
		&mockResolver{name: "first", available: false},
		&mockResolver{name: "second", available: false},
		&mockResolver{name: "third", available: false},
	)

	_, _, err := chain.Resolve("any-ref")
	if err == nil {
		t.Error("Expected error when all resolvers unavailable")
	}
	if err.Error() != "no available resolvers could resolve the secret" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestChainMixedAvailabilityAndErrors(t *testing.T) {
	// Simulates: 1password unavailable, env fails, keychain succeeds
	chain := NewChain(
		&mockResolver{name: "1password", available: false},
		&mockResolver{name: "env", available: true, err: errors.New("not set")},
		&mockResolver{name: "keychain", available: true, value: "from-keychain"},
	)

	value, source, err := chain.Resolve("any-ref")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if value != "from-keychain" {
		t.Errorf("Expected 'from-keychain', got %q", value)
	}
	if source != "keychain" {
		t.Errorf("Expected source 'keychain', got %q", source)
	}
}

func TestChainPreservesLastError(t *testing.T) {
	// When all available resolvers error, should return the last error
	chain := NewChain(
		&mockResolver{name: "first", available: true, err: errors.New("first error")},
		&mockResolver{name: "second", available: true, err: errors.New("second error")},
	)

	_, _, err := chain.Resolve("any-ref")
	if err == nil {
		t.Fatal("Expected error")
	}
	if err.Error() != "second error" {
		t.Errorf("Expected 'second error', got %q", err.Error())
	}
}
