// ABOUTME: Progress tracker for concurrent profile apply operations
// ABOUTME: Provides phased progress bars with sliding window of recent items
package ui

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// TrackerConfig configures a new ProgressTracker
type TrackerConfig struct {
	Phases []string // Phase names in order (e.g., "Marketplaces", "Plugins")
	Window int      // Number of recent items to show in sliding window
}

// ProgressTracker tracks progress across multiple phases of work
type ProgressTracker struct {
	phases        []*Phase
	phasesByName  map[string]*Phase
	window        int
	linesRendered int
	mu            sync.Mutex
}

// Phase represents a single phase of work (e.g., installing plugins)
type Phase struct {
	Name      string
	Total     int          // Items to process in this phase
	Completed int          // Items processed (success or failure)
	Skipped   int          // Items skipped (already installed)
	Items     []ItemResult // Sliding window of recent results
	Done      bool         // True when all items processed
}

// ItemResult represents the outcome of processing a single item
type ItemResult struct {
	Name    string
	Success bool
	Error   string // Empty if success
}

// NewProgressTracker creates a new progress tracker with the given configuration
func NewProgressTracker(config TrackerConfig) *ProgressTracker {
	phases := make([]*Phase, len(config.Phases))
	phasesByName := make(map[string]*Phase)

	for i, name := range config.Phases {
		phase := &Phase{
			Name:  name,
			Items: make([]ItemResult, 0),
		}
		phases[i] = phase
		phasesByName[name] = phase
	}

	window := config.Window
	if window <= 0 {
		window = 5 // Default window size
	}

	return &ProgressTracker{
		phases:       phases,
		phasesByName: phasesByName,
		window:       window,
	}
}

// SetPhaseTotals sets the total and skipped counts for a phase
// toProcess is the number of items that will be processed
// totalInProfile is the total number of items in the profile (including skipped)
func (t *ProgressTracker) SetPhaseTotals(phaseName string, toProcess, totalInProfile int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	phase, ok := t.phasesByName[phaseName]
	if !ok {
		return
	}

	phase.Total = toProcess
	phase.Skipped = totalInProfile - toProcess
}

// RecordResult records the result of processing an item
func (t *ProgressTracker) RecordResult(phaseName string, result ItemResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	phase, ok := t.phasesByName[phaseName]
	if !ok {
		return
	}

	phase.Completed++

	// Add to sliding window
	phase.Items = append(phase.Items, result)

	// Trim to window size
	if len(phase.Items) > t.window {
		phase.Items = phase.Items[len(phase.Items)-t.window:]
	}

	// Check if phase is complete
	if phase.Completed >= phase.Total {
		phase.Done = true
	}
}

// Rendering constants
const (
	barWidth      = 20 // Width of progress bar in characters
	barFilled     = '━'
	barEmpty      = '░'
	spinnerFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

// renderProgressBar renders a progress bar of the given width
func renderProgressBar(completed, total, width int) string {
	if total == 0 {
		// Empty bar if nothing to do
		return string(repeatRune(barEmpty, width))
	}

	filled := (completed * width) / total
	if filled > width {
		filled = width
	}

	return string(repeatRune(barFilled, filled)) + string(repeatRune(barEmpty, width-filled))
}

// renderPhaseLine renders a single phase line with name, bar, and counts
func renderPhaseLine(phase *Phase, barWidth int) string {
	bar := renderProgressBar(phase.Completed, phase.Total, barWidth)

	// Format: "Plugins      ━━━━━━━━━━░░░░░░░░░░ 12/44"
	name := padRight(phase.Name, 12)
	count := fmt.Sprintf("%d/%d", phase.Completed, phase.Total)

	status := ""
	if phase.Done {
		status = " " + Success(SymbolSuccess)
	}

	skipped := ""
	if phase.Skipped > 0 {
		skipped = fmt.Sprintf(" (%d already installed)", phase.Skipped)
	}

	return fmt.Sprintf("%s %s %s%s%s", name, bar, count, status, Muted(skipped))
}

// renderItemLine renders a single item result line
func renderItemLine(item ItemResult) string {
	if item.Success {
		return fmt.Sprintf("  %s %s", Success(SymbolSuccess), item.Name)
	}

	errMsg := ""
	if item.Error != "" {
		errMsg = fmt.Sprintf(" (%s)", item.Error)
	}
	return fmt.Sprintf("  %s %s%s", Error(SymbolError), item.Name, Muted(errMsg))
}

// repeatRune creates a string by repeating a rune n times
func repeatRune(r rune, n int) []rune {
	if n <= 0 {
		return []rune{}
	}
	result := make([]rune, n)
	for i := range result {
		result[i] = r
	}
	return result
}

// padRight pads a string to the given width with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + string(repeatRune(' ', width-len(s)))
}

// Render outputs the current progress state to the terminal
// Uses ANSI escape codes to update in-place on TTY, falls back to line-by-line otherwise
func (t *ProgressTracker) Render(w io.Writer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.renderLocked(w)
}

// renderLocked renders without acquiring the lock (caller must hold mutex)
func (t *ProgressTracker) renderLocked(w io.Writer) {
	if isTerminal(w) {
		t.renderTTY(w)
	} else {
		t.renderSimple(w)
	}
}

// renderTTY renders with ANSI cursor control for in-place updates
func (t *ProgressTracker) renderTTY(w io.Writer) {
	// Move cursor up to overwrite previous output
	if t.linesRendered > 0 {
		fmt.Fprintf(w, "\033[%dA", t.linesRendered)
	}

	lines := 0

	// Render each phase
	for _, phase := range t.phases {
		fmt.Fprintf(w, "\033[K%s\n", renderPhaseLine(phase, barWidth))
		lines++
	}

	// Find active phase (first incomplete, or last if all done)
	activePhase := t.getActivePhase()
	if activePhase != nil && len(activePhase.Items) > 0 {
		for _, item := range activePhase.Items {
			fmt.Fprintf(w, "\033[K%s\n", renderItemLine(item))
			lines++
		}
	}

	t.linesRendered = lines
}

// renderSimple renders line-by-line for non-TTY output (CI, pipes)
func (t *ProgressTracker) renderSimple(w io.Writer) {
	// Only output new items since last render
	// For simplicity, we just output phase completions and errors
	for _, phase := range t.phases {
		if phase.Done && phase.Completed > 0 {
			skipped := ""
			if phase.Skipped > 0 {
				skipped = fmt.Sprintf(" (%d skipped)", phase.Skipped)
			}
			fmt.Fprintf(w, "[%s] %d/%d complete%s\n",
				phase.Name, phase.Completed, phase.Total, skipped)
		}
	}
}

// RenderUpdate outputs a single item update
// For TTY: performs a full re-render with cursor control
// For non-TTY: streams individual updates line-by-line
// Uses mutex to prevent interleaved output from concurrent workers
func (t *ProgressTracker) RenderUpdate(w io.Writer, phaseName string, result ItemResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if isTerminal(w) {
		// TTY: full re-render handles it
		t.renderTTY(w)
	} else {
		// Non-TTY: stream individual updates
		status := SymbolSuccess
		suffix := ""
		if !result.Success {
			status = SymbolError
			if result.Error != "" {
				suffix = fmt.Sprintf(" (%s)", result.Error)
			}
		}
		fmt.Fprintf(w, "[%s] %s %s%s\n", phaseName, status, result.Name, suffix)
	}
}

// getActivePhase returns the first incomplete phase, or nil if all done
func (t *ProgressTracker) getActivePhase() *Phase {
	for _, phase := range t.phases {
		if !phase.Done && phase.Total > 0 {
			return phase
		}
	}
	// All done, return last phase with items for final display
	for i := len(t.phases) - 1; i >= 0; i-- {
		if len(t.phases[i].Items) > 0 {
			return t.phases[i]
		}
	}
	return nil
}

// Finish clears the sliding window display and shows final summary
func (t *ProgressTracker) Finish(w io.Writer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !isTerminal(w) {
		return // Non-TTY already streamed everything
	}

	// Move up and clear the item lines, keep phase bars
	if t.linesRendered > len(t.phases) {
		extraLines := t.linesRendered - len(t.phases)
		// Move to start of item lines and clear them
		fmt.Fprintf(w, "\033[%dA", extraLines)
		for i := 0; i < extraLines; i++ {
			fmt.Fprintf(w, "\033[K\n")
		}
		// Move back up to after phase bars
		fmt.Fprintf(w, "\033[%dA", extraLines)
	}

	t.linesRendered = len(t.phases)
}

// isTerminal checks if the writer is a terminal
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// PluginProgress returns a callback that prints plugin installation progress.
// Matches the profile.ProgressCallback signature for use with apply operations.
func PluginProgress() func(current, total int, item string) {
	return func(current, total int, item string) {
		fmt.Printf("  [%d/%d] Installing %s\n", current, total, item)
	}
}
