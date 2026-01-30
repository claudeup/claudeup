// ABOUTME: Test helpers for ui package
// ABOUTME: Provides synchronized access to global YesFlag during testing
package ui

import (
	"sync"
	"testing"

	"github.com/claudeup/claudeup/v4/internal/config"
)

// testYesFlagMutex ensures only one test modifies YesFlag at a time
// This prevents race conditions when tests run in parallel
var testYesFlagMutex sync.Mutex

// withYesFlag safely sets YesFlag for the duration of a test
// It ensures exclusive access and automatic cleanup
func withYesFlag(t *testing.T, value bool, fn func()) {
	t.Helper()

	testYesFlagMutex.Lock()
	defer testYesFlagMutex.Unlock()

	// Save and restore original value
	originalFlag := config.YesFlag
	defer func() { config.YesFlag = originalFlag }()

	config.YesFlag = value
	fn()
}
