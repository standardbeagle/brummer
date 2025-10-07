package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestErrorDisplayAndManagement tests the complete error management workflow
// This test verifies current functionality BEFORE controller migration
// Test Scenario 3 from quickstart.md
func TestErrorDisplayAndManagement(t *testing.T) {
	SkipIfShort(t, "Integration test with error handling")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test Step 1: Trigger an error (start invalid process)
	t.Run("trigger_error", func(t *testing.T) {
		// Try to start a process that will fail
		_, err := ctx.ProcessMgr.StartScript("error")
		// Error script exists and starts, but exits with error
		// We handle this in next steps
		_ = err
	})

	// Test Step 2: Press "3" to view errors
	t.Run("view_errors_list", func(t *testing.T) {
		ctx.SendKeyMsg("3")
		time.Sleep(100 * time.Millisecond)

		// Verify we're in errors view
		ctx.AssertViewContains("Errors")
	})
}

// TestErrorList tests error list display
func TestErrorList(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Add some test errors via model
	// Note: In real usage, errors would be added through error handling system

	// Switch to errors view
	ctx.SendKeyMsg("3")
	time.Sleep(100 * time.Millisecond)

	// Verify errors view is displayed
	ctx.AssertViewContains("Errors")
}

// TestErrorNavigation tests navigation in errors list
func TestErrorNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Switch to errors view
	ctx.SendKeyMsg("3")
	time.Sleep(100 * time.Millisecond)

	// Navigate with arrow keys
	ctx.SendKeyMsg("down")
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("up")
	time.Sleep(50 * time.Millisecond)

	// View should still render
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "View should render after navigation")
}

// TestErrorViewKeyboardShortcuts tests keyboard shortcuts in errors view
func TestErrorViewKeyboardShortcuts(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Switch to errors view
	ctx.SendKeyMsg("3")
	time.Sleep(100 * time.Millisecond)

	// Test view switching
	ctx.SendKeyMsg("1") // Processes
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("3") // Back to errors
	time.Sleep(50 * time.Millisecond)

	// Verify errors view is functional
	ctx.AssertViewContains("Errors")
}
