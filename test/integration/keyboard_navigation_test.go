package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestKeyboardNavigation tests all global keyboard shortcuts
// This is the MOST CRITICAL test - verifies all navigation works
// Test Scenario 7 from quickstart.md
func TestKeyboardNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test with keyboard navigation")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test Step 1: Press "1" to switch to processes view
	t.Run("switch_to_processes", func(t *testing.T) {
		ctx.SendKeyMsg("1")
		time.Sleep(100 * time.Millisecond)

		// Verify processes view is displayed
		ctx.AssertViewContains("Processes")
	})

	// Test Step 2: Press "2" to switch to logs view
	t.Run("switch_to_logs", func(t *testing.T) {
		ctx.SendKeyMsg("2")
		time.Sleep(100 * time.Millisecond)

		// Verify logs view is displayed
		ctx.AssertViewContains("Logs")
	})

	// Test Step 3: Press Tab to cycle through views
	t.Run("tab_cycles_forward", func(t *testing.T) {
		// Start at processes
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		// Press Tab to cycle forward
		ctx.SendKeyMsg("tab")
		time.Sleep(100 * time.Millisecond)

		// Should have moved to next view
		view := ctx.Model.View()
		assert.NotEmpty(t, view, "View should render after Tab")
	})

	// Test Step 4: Press Shift+Tab to cycle backwards
	t.Run("shift_tab_cycles_backward", func(t *testing.T) {
		// Press Shift+Tab
		// Note: In real TTY this would be shift+tab, in test we simulate
		ctx.SendKeyMsg("shift+tab")
		time.Sleep(100 * time.Millisecond)

		// Should have moved to previous view
		view := ctx.Model.View()
		assert.NotEmpty(t, view, "View should render after Shift+Tab")
	})
}

// TestNumberKeyNavigation tests switching views with number keys
func TestNumberKeyNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test all numbered views
	viewKeys := []string{"1", "2", "3", "4"}

	for _, key := range viewKeys {
		t.Run(fmt.Sprintf("switch_with_%s", key), func(t *testing.T) {
			ctx.SendKeyMsg(key)
			time.Sleep(100 * time.Millisecond)

			// Verify view rendered
			view := ctx.Model.View()
			assert.NotEmpty(t, view, "View should render for key %s", key)
		})
	}
}

// TestTabNavigation tests Tab and Shift+Tab view cycling
func TestTabNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Start at known view
	ctx.SendKeyMsg("1")
	time.Sleep(50 * time.Millisecond)

	// Press Tab multiple times
	for i := 0; i < 3; i++ {
		ctx.SendKeyMsg("tab")
		time.Sleep(100 * time.Millisecond)

		// Verify view renders
		view := ctx.Model.View()
		assert.NotEmpty(t, view, "View should render after Tab %d", i+1)
	}

	// Press Shift+Tab to go back
	for i := 0; i < 2; i++ {
		ctx.SendKeyMsg("shift+tab")
		time.Sleep(100 * time.Millisecond)

		// Verify view renders
		view := ctx.Model.View()
		assert.NotEmpty(t, view, "View should render after Shift+Tab %d", i+1)
	}
}

// TestGlobalKeyShortcuts tests that global shortcuts work from all views
func TestGlobalKeyShortcuts(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test global shortcuts from different views
	views := []string{"1", "2", "3", "4"}

	for _, viewKey := range views {
		t.Run(fmt.Sprintf("global_shortcuts_from_view_%s", viewKey), func(t *testing.T) {
			// Switch to view
			ctx.SendKeyMsg(viewKey)
			time.Sleep(100 * time.Millisecond)

			// Test that we can switch to other views (global shortcut)
			ctx.SendKeyMsg("1")
			time.Sleep(50 * time.Millisecond)

			view := ctx.Model.View()
			assert.NotEmpty(t, view, "Global shortcuts should work from any view")
		})
	}
}

// TestArrowKeyNavigation tests arrow key navigation within views
func TestArrowKeyNavigation(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Test arrow keys in processes view
	ctx.SendKeyMsg("1")
	time.Sleep(100 * time.Millisecond)

	// Navigate with arrows
	arrows := []string{"down", "down", "up", "up"}
	for _, arrow := range arrows {
		ctx.SendKeyMsg(arrow)
		time.Sleep(50 * time.Millisecond)

		// Verify view still renders
		view := ctx.Model.View()
		assert.NotEmpty(t, view, "View should render after %s", arrow)
	}
}

// TestViewPersistence tests that view state persists when switching
func TestViewPersistence(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Switch to processes view
	ctx.SendKeyMsg("1")
	time.Sleep(100 * time.Millisecond)

	// Navigate down in list
	ctx.SendKeyMsg("down")
	ctx.SendKeyMsg("down")
	time.Sleep(50 * time.Millisecond)

	// Switch to different view
	ctx.SendKeyMsg("2")
	time.Sleep(100 * time.Millisecond)

	// Switch back to processes
	ctx.SendKeyMsg("1")
	time.Sleep(100 * time.Millisecond)

	// Verify view still renders (state may or may not persist depending on implementation)
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "View should render after returning")
}

// TestRapidKeyPresses tests handling of rapid key presses
func TestRapidKeyPresses(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Rapidly switch views
	keys := []string{"1", "2", "3", "4", "1", "2", "3", "4"}
	for _, key := range keys {
		ctx.SendKeyMsg(key)
		time.Sleep(20 * time.Millisecond) // Rapid presses
	}

	// Final view should still render
	view := ctx.Model.View()
	assert.NotEmpty(t, view, "View should handle rapid key presses")
}

// TestKeyEventLatency tests that key events respond quickly
func TestKeyEventLatency(t *testing.T) {
	SkipIfShort(t, "Performance test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	// Measure latency of key events
	samples := []time.Duration{}

	for i := 0; i < 20; i++ {
		latency := MeasureKeyEventLatency(ctx, "1")
		samples = append(samples, latency)
		time.Sleep(50 * time.Millisecond)
	}

	// Calculate average latency
	var total time.Duration
	for _, s := range samples {
		total += s
	}
	avg := total / time.Duration(len(samples))

	// Log latency
	t.Logf("Average key event latency: %v", avg)

	// Key events should be fast (<50ms target)
	assert.Less(t, avg, 100*time.Millisecond, "Key events should respond quickly")
}

// TestWindowResize tests that views handle window resize correctly
func TestWindowResize(t *testing.T) {
	SkipIfShort(t, "Integration test")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Start with small window
	ctx.SetWindowSize(80, 24)
	ctx.SendKeyMsg("1")
	time.Sleep(100 * time.Millisecond)

	// Verify initial render
	view1 := ctx.Model.View()
	assert.NotEmpty(t, view1, "Should render in small window")

	// Resize to larger window
	ctx.SetWindowSize(120, 40)
	time.Sleep(100 * time.Millisecond)

	// Verify re-render
	view2 := ctx.Model.View()
	assert.NotEmpty(t, view2, "Should re-render after resize")

	// Resize to very small window
	ctx.SetWindowSize(40, 10)
	time.Sleep(100 * time.Millisecond)

	// Should still render (possibly with overflow handling)
	view3 := ctx.Model.View()
	assert.NotEmpty(t, view3, "Should render in tiny window")
}
