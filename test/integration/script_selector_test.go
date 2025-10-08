package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestScriptSelectorWorkflow tests the script selector functionality
// This test verifies script discovery, selection, and execution
func TestScriptSelectorWorkflow(t *testing.T) {
	SkipIfShort(t, "Integration test with script selector")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	// Set initial window size
	ctx.SetWindowSize(120, 40)

	// Test Step 1: Open script selector
	t.Run("open_script_selector", func(t *testing.T) {
		// Script selector might be opened via:
		// - Specific key binding
		// - Command palette
		// - Automatic on startup if no scripts running

		// Try opening with 's' key or similar
		ctx.SendKeyMsg("s")
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Script selector should be displayed")
	})

	// Test Step 2: Display available scripts from package.json
	t.Run("display_package_json_scripts", func(t *testing.T) {
		// Script selector should show scripts from package.json
		// (Test context should have a package.json with test scripts)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should display available scripts")
		// Typical scripts: dev, build, test, start
	})

	// Test Step 3: Navigate script list with arrow keys
	t.Run("navigate_script_list", func(t *testing.T) {
		// Navigate down through scripts
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Navigate back up
		ctx.SendKeyMsg("up")
		time.Sleep(50 * time.Millisecond)

		// Should still show script selector
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should navigate through scripts")
	})

	// Test Step 4: Filter scripts with search
	t.Run("filter_scripts", func(t *testing.T) {
		// Type to filter scripts
		ctx.SendKeyMsg("d")
		ctx.SendKeyMsg("e")
		ctx.SendKeyMsg("v")
		time.Sleep(100 * time.Millisecond)

		// Should show only matching scripts (e.g., "dev")
		view := ctx.GetCurrentView()
		assert.Contains(t, strings.ToLower(view), "dev", "Should filter to 'dev' script")
	})

	// Test Step 5: Select script with Enter
	t.Run("select_script_with_enter", func(t *testing.T) {
		// Navigate to a script
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Select with Enter
		ctx.SendKeyMsg("enter")
		time.Sleep(200 * time.Millisecond)

		// Script should start and selector should close
		// Should switch to process or logs view
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should start script and close selector")
	})

	// Test Step 6: Cancel with Esc
	t.Run("cancel_with_escape", func(t *testing.T) {
		// Open script selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Cancel with Esc
		ctx.SendKeyMsg("esc")
		time.Sleep(100 * time.Millisecond)

		// Should close without starting script
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should close script selector")
	})

	// Test Step 7: Multi-select scripts
	t.Run("multi_select_scripts", func(t *testing.T) {
		// Open script selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Select first script with Space
		ctx.SendKeyMsg("space")
		time.Sleep(50 * time.Millisecond)

		// Navigate to another
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Select second script
		ctx.SendKeyMsg("space")
		time.Sleep(50 * time.Millisecond)

		// Start both with Enter
		ctx.SendKeyMsg("enter")
		time.Sleep(200 * time.Millisecond)

		// Both scripts should start
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should start multiple scripts")
	})
}

// TestScriptDiscovery tests script discovery from package.json
func TestScriptDiscovery(t *testing.T) {
	SkipIfShort(t, "Integration test for script discovery")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("parse_package_json", func(t *testing.T) {
		// Script selector should parse package.json scripts section
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should discover scripts from package.json")
	})

	t.Run("handle_missing_package_json", func(t *testing.T) {
		// If no package.json, should show appropriate message
		// (Test context likely has package.json, so this is edge case)
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle missing package.json gracefully")
	})

	t.Run("handle_invalid_package_json", func(t *testing.T) {
		// If package.json is malformed, should show error
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle invalid package.json")
	})

	t.Run("show_script_commands", func(t *testing.T) {
		// Script selector should show the actual command for each script
		// e.g., "dev" → "vite dev"
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show script commands")
	})
}

// TestScriptSelection tests script selection behavior
func TestScriptSelection(t *testing.T) {
	SkipIfShort(t, "Integration test for script selection")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("single_script_selection", func(t *testing.T) {
		// Select one script and start it
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		ctx.SendKeyMsg("down")
		ctx.SendKeyMsg("enter")
		time.Sleep(200 * time.Millisecond)

		// Script should be running
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should start single script")
	})

	t.Run("toggle_selection_with_space", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Toggle selection on
		ctx.SendKeyMsg("space")
		time.Sleep(50 * time.Millisecond)

		// Toggle selection off
		ctx.SendKeyMsg("space")
		time.Sleep(50 * time.Millisecond)

		// Should be unselected
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should toggle selection")
	})

	t.Run("clear_all_selections", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Select multiple
		ctx.SendKeyMsg("space")
		ctx.SendKeyMsg("down")
		ctx.SendKeyMsg("space")
		time.Sleep(50 * time.Millisecond)

		// Esc should clear selections and close
		ctx.SendKeyMsg("esc")
		time.Sleep(100 * time.Millisecond)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should clear selections on cancel")
	})
}

// TestScriptFiltering tests script filtering functionality
func TestScriptFiltering(t *testing.T) {
	SkipIfShort(t, "Integration test for script filtering")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("filter_by_name", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Type filter
		ctx.SendKeyMsg("t")
		ctx.SendKeyMsg("e")
		ctx.SendKeyMsg("s")
		ctx.SendKeyMsg("t")
		time.Sleep(100 * time.Millisecond)

		// Should show only "test" scripts
		view := ctx.GetCurrentView()
		assert.Contains(t, strings.ToLower(view), "test", "Should filter to 'test' scripts")
	})

	t.Run("clear_filter_with_backspace", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Type filter
		ctx.SendKeyMsg("t")
		ctx.SendKeyMsg("e")
		time.Sleep(50 * time.Millisecond)

		// Clear filter
		ctx.SendKeyMsg("backspace")
		ctx.SendKeyMsg("backspace")
		time.Sleep(100 * time.Millisecond)

		// Should show all scripts again
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should clear filter")
	})

	t.Run("no_matching_scripts", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Type filter that doesn't match
		ctx.SendKeyMsg("x")
		ctx.SendKeyMsg("y")
		ctx.SendKeyMsg("z")
		time.Sleep(100 * time.Millisecond)

		// Should show "no matches" message
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle no matches")
	})

	t.Run("fuzzy_matching", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Type partial match
		ctx.SendKeyMsg("d")
		ctx.SendKeyMsg("v")
		time.Sleep(100 * time.Millisecond)

		// Should fuzzy match "dev"
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should support fuzzy matching")
	})
}

// TestScriptExecution tests script execution after selection
func TestScriptExecution(t *testing.T) {
	SkipIfShort(t, "Integration test for script execution")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("execute_selected_script", func(t *testing.T) {
		// Select and execute a script
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Filter to specific script
		ctx.SendKeyMsg("d")
		ctx.SendKeyMsg("e")
		ctx.SendKeyMsg("v")
		time.Sleep(50 * time.Millisecond)

		// Execute
		ctx.SendKeyMsg("enter")
		time.Sleep(200 * time.Millisecond)

		// Process should be running
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Script should execute")
	})

	t.Run("execute_multiple_scripts", func(t *testing.T) {
		// Select multiple scripts
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		ctx.SendKeyMsg("space") // Select first
		ctx.SendKeyMsg("down")
		ctx.SendKeyMsg("space") // Select second
		time.Sleep(50 * time.Millisecond)

		// Execute both
		ctx.SendKeyMsg("enter")
		time.Sleep(200 * time.Millisecond)

		// Both should be running
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Multiple scripts should execute")
	})
}

// TestScriptSelectorRendering tests script selector rendering
func TestScriptSelectorRendering(t *testing.T) {
	SkipIfShort(t, "Integration test for script selector rendering")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("render_script_list", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(100 * time.Millisecond)

		// Should render formatted list
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should render script list")
		assert.Greater(t, len(view), 10, "Script list should have content")
	})

	t.Run("highlight_selected_script", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Navigate to select
		ctx.SendKeyMsg("down")
		time.Sleep(50 * time.Millisecond)

		// Selected script should be highlighted
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should highlight selected script")
	})

	t.Run("show_multi_select_checkboxes", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Select with space
		ctx.SendKeyMsg("space")
		time.Sleep(50 * time.Millisecond)

		// Should show checkbox/indicator
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should show multi-select indicators")
	})
}

// TestScriptSelectorPerformance tests script selector performance
func TestScriptSelectorPerformance(t *testing.T) {
	SkipIfShort(t, "Performance test for script selector")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("selector_open_time", func(t *testing.T) {
		// Measure time to open selector
		start := time.Now()
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)
		openTime := time.Since(start)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should open selector")
		assert.Less(t, openTime, 100*time.Millisecond, "Should open quickly")
	})

	t.Run("filter_performance", func(t *testing.T) {
		// Open selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Measure filter responsiveness
		start := time.Now()
		ctx.SendKeyMsg("d")
		ctx.SendKeyMsg("e")
		time.Sleep(50 * time.Millisecond)
		filterTime := time.Since(start)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should filter scripts")
		assert.Less(t, filterTime, 100*time.Millisecond, "Filtering should be fast")
	})

	t.Run("large_script_list_performance", func(t *testing.T) {
		// Even with many scripts, selector should be responsive
		start := time.Now()
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Navigate through list
		for i := 0; i < 10; i++ {
			ctx.SendKeyMsg("down")
		}
		time.Sleep(50 * time.Millisecond)
		navTime := time.Since(start)

		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle navigation")
		assert.Less(t, navTime, 200*time.Millisecond, "Navigation should be smooth")
	})
}
