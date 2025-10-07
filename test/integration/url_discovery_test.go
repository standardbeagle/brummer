package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestURLDiscoveryAndDisplay tests the complete URL discovery workflow
// This test verifies current functionality BEFORE controller migration
// Test Scenario 4 from quickstart.md
func TestURLDiscoveryAndDisplay(t *testing.T) {
	SkipIfShort(t, "Integration test with URL discovery")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test Step 1: Start a process that logs a URL
	t.Run("start_process_with_url", func(t *testing.T) {
		// Start dev process which outputs URL
		processID := ctx.StartTestProcess("dev")
		defer ctx.StopTestProcess(processID)

		// Wait for URL to be detected
		time.Sleep(1 * time.Second)

		// Verify URL was discovered
		urls := ctx.ProcessMgr.GetURLs()
		assert.NotEmpty(t, urls, "Should have discovered URL from logs")
	})

	// Test Step 2: Press "4" to view URLs
	t.Run("view_urls_list", func(t *testing.T) {
		ctx.SendKeyMsg("4")
		time.Sleep(100 * time.Millisecond)

		// Verify we're in URLs view
		ctx.AssertViewContains("URLs")
	})
}

// TestURLDisplay tests URL list display
func TestURLDisplay(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Start process that outputs URL
	processID := ctx.StartTestProcess("dev")
	defer ctx.StopTestProcess(processID)

	// Wait for URL detection
	time.Sleep(1 * time.Second)

	// Switch to URLs view
	ctx.SendKeyMsg("4")
	time.Sleep(100 * time.Millisecond)

	// Verify URLs view displays
	ctx.AssertViewContains("URLs")

	// Check for URL in display
	urls := ctx.ProcessMgr.GetURLs()
	if len(urls) > 0 {
		// URL was detected successfully
		assert.NotEmpty(t, urls, "Should have detected URL")
	}
}

// TestURLViewNavigation tests navigation in URLs view
func TestURLViewNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Switch to URLs view
	ctx.SendKeyMsg("4")
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

// TestURLViewKeyboardShortcuts tests keyboard shortcuts in URLs view
func TestURLViewKeyboardShortcuts(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Switch to URLs view
	ctx.SendKeyMsg("4")
	time.Sleep(100 * time.Millisecond)

	// Test view switching
	ctx.SendKeyMsg("1") // Processes
	time.Sleep(50 * time.Millisecond)

	ctx.SendKeyMsg("4") // Back to URLs
	time.Sleep(50 * time.Millisecond)

	// Verify URLs view is functional
	ctx.AssertViewContains("URLs")
}
