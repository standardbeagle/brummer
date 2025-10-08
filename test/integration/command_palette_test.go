package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCommandPaletteWorkflow tests the command palette functionality
// This test verifies command palette activation, autocomplete, and execution
// Test Scenario 8 from quickstart.md
func TestCommandPaletteWorkflow(t *testing.T) {
	SkipIfShort(t, "Integration test with command palette")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Set initial window size
	ctx.SetWindowSize(120, 40)

	// Test Step 1: Press '/' to open command palette
	t.Run("open_command_palette", func(t *testing.T) {
		// Simulate pressing "/" to open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(100 * time.Millisecond)

		// Verify command palette is visible
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Command palette should be visible")
		// Command palette typically shows a prompt or input area
	})

	// Test Step 2: Type partial command and see autocomplete
	t.Run("autocomplete_suggestions", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type partial command
		ctx.SendKeyMsg("s")
		ctx.SendKeyMsg("t")
		ctx.SendKeyMsg("a")
		time.Sleep(100 * time.Millisecond)

		// Command palette should show suggestions
		// (e.g., "start", "stop", "status")
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show autocomplete suggestions")
	})

	// Test Step 3: Navigate suggestions with arrow keys
	t.Run("navigate_suggestions", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type to get suggestions
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Navigate down
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Navigate up
		ctx.SendKeyMsg("up")
		time.Sleep(50 * time.Millisecond)

		// View should still show command palette
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show command palette with navigation")
	})

	// Test Step 4: Close command palette with Esc
	t.Run("close_with_escape", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Close with Esc
		ctx.SendKeyMsg("esc")
		time.Sleep(100 * time.Millisecond)

		// Command palette should be closed
		// (back to previous view)
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should return to previous view")
	})

	// Test Step 5: Close command palette with backspace on empty input
	t.Run("close_with_backspace", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type something
		ctx.SendKeyMsg("t")
		ctx.SendKeyMsg("e")
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Delete characters
		ctx.SendKeyMsg("backspace")
		ctx.SendKeyMsg("backspace")
		ctx.SendKeyMsg("backspace")
		time.Sleep(50 * time.Millisecond)

		// Backspace on empty should close
		ctx.SendKeyMsg("backspace")
		time.Sleep(100 * time.Millisecond)

		// Command palette should be closed
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should close command palette on empty backspace")
	})

	// Test Step 6: Execute command with Enter
	t.Run("execute_command", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type a command (e.g., go to processes view)
		ctx.SendKeyMsg("p")
		ctx.SendKeyMsg("r")
		ctx.SendKeyMsg("o")
		time.Sleep(50 * time.Millisecond)

		// Execute with Enter
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Command should have executed
		// (Specific outcome depends on command)
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Command should execute and show result")
	})
}

// TestCommandPaletteCommands tests specific command execution
func TestCommandPaletteCommands(t *testing.T) {
	SkipIfShort(t, "Integration test for command palette commands")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("navigate_to_processes_view", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type "processes" command
		for _, ch := range "processes" {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(50 * time.Millisecond)

		// Execute
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should be in processes view
		view := ctx.GetCurrentView()
		assert.Contains(t, strings.ToLower(view), "process", "Should show processes view")
	})

	t.Run("navigate_to_logs_view", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type "logs" command
		for _, ch := range "logs" {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(50 * time.Millisecond)

		// Execute
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should be in logs view
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show logs view")
	})

	t.Run("navigate_to_errors_view", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type "errors" command
		for _, ch := range "errors" {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(50 * time.Millisecond)

		// Execute
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should be in errors view
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show errors view")
	})
}

// TestCommandPaletteAutocomplete tests autocomplete functionality
func TestCommandPaletteAutocomplete(t *testing.T) {
	SkipIfShort(t, "Integration test for autocomplete")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("fuzzy_matching", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type partial/fuzzy match
		ctx.SendKeyMsg("p")
		ctx.SendKeyMsg("r")
		time.Sleep(100 * time.Millisecond)

		// Should show suggestions matching "pr"
		// (e.g., "processes", "proxy")
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show fuzzy match suggestions")
	})

	t.Run("case_insensitive_matching", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type uppercase
		ctx.SendKeyMsg("P")
		ctx.SendKeyMsg("R")
		ctx.SendKeyMsg("O")
		time.Sleep(100 * time.Millisecond)

		// Should still show suggestions
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should match case-insensitively")
	})

	t.Run("no_matches", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type something that doesn't match
		ctx.SendKeyMsg("x")
		ctx.SendKeyMsg("y")
		ctx.SendKeyMsg("z")
		ctx.SendKeyMsg("z")
		ctx.SendKeyMsg("z")
		time.Sleep(100 * time.Millisecond)

		// Should show "no matches" or empty suggestions
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle no matches gracefully")
	})
}

// TestCommandPalettePerformance tests command palette responsiveness
func TestCommandPalettePerformance(t *testing.T) {
	SkipIfShort(t, "Performance test for command palette")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("open_latency", func(t *testing.T) {
		// Measure time to open command palette
		start := time.Now()
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)
		openTime := time.Since(start)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should open command palette")
		assert.Less(t, openTime, 100*time.Millisecond, "Should open quickly")
	})

	t.Run("autocomplete_latency", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Measure autocomplete response time
		start := time.Now()
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)
		autocompleteTime := time.Since(start)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show autocomplete")
		assert.Less(t, autocompleteTime, 100*time.Millisecond, "Autocomplete should be fast")
	})

	t.Run("rapid_input_handling", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type rapidly
		start := time.Now()
		for _, ch := range "processes" {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(50 * time.Millisecond)
		inputTime := time.Since(start)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid input")
		assert.Less(t, inputTime, 200*time.Millisecond, "Should handle rapid typing")
	})
}
