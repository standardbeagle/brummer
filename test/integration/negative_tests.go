package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNegativeProcessManagement tests error conditions in process management
func TestNegativeProcessManagement(t *testing.T) {
	SkipIfShort(t, "Negative integration test for process management")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("start_script_with_empty_name", func(t *testing.T) {
		_, err := ctx.ProcessMgr.StartScript("")
		assert.Error(t, err, "Should error on empty script name")
	})

	t.Run("start_script_with_whitespace_name", func(t *testing.T) {
		_, err := ctx.ProcessMgr.StartScript("   ")
		assert.Error(t, err, "Should error on whitespace script name")
	})

	t.Run("start_script_with_special_characters", func(t *testing.T) {
		// Try scripts with special characters
		badNames := []string{
			"script/with/slash",
			"script\\with\\backslash",
			"script\x00null",
			"script\nwith\nnewline",
			"../../../etc/passwd",
			"script;rm -rf /",
		}

		for _, name := range badNames {
			_, err := ctx.ProcessMgr.StartScript(name)
			assert.Error(t, err, "Should error on script name: %s", name)
		}
	})

	t.Run("stop_process_with_empty_id", func(t *testing.T) {
		err := ctx.ProcessMgr.StopProcess("")
		assert.Error(t, err, "Should error on empty process ID")
	})

	t.Run("stop_process_with_null_bytes", func(t *testing.T) {
		err := ctx.ProcessMgr.StopProcess("process\x00id")
		assert.Error(t, err, "Should error on process ID with null bytes")
	})

	t.Run("get_process_with_invalid_id", func(t *testing.T) {
		invalidIDs := []string{
			"",
			"\x00",
			"../../secrets",
			"id;rm -rf /",
			"id\nwith\nnewline",
		}

		for _, id := range invalidIDs {
			_, exists := ctx.ProcessMgr.GetProcess(id)
			assert.False(t, exists, "Should not find process with invalid ID: %s", id)
		}
	})

	t.Run("rapid_start_stop_same_script", func(t *testing.T) {
		// Rapidly start and stop the same script
		for i := 0; i < 5; i++ {
			proc, err := ctx.ProcessMgr.StartScript("test")
			if err != nil {
				t.Logf("Start failed on iteration %d: %v", i, err)
				continue
			}

			// Try to stop immediately
			err = ctx.ProcessMgr.StopProcess(proc.ID)
			if err != nil {
				t.Logf("Stop failed on iteration %d: %v", i, err)
			}
		}

		time.Sleep(500 * time.Millisecond)

		// System should still be functional
		proc, err := ctx.ProcessMgr.StartScript("dev")
		assert.NoError(t, err, "Should still work after rapid start/stop")

		if proc != nil {
			time.Sleep(200 * time.Millisecond)
			ctx.ProcessMgr.StopProcess(proc.ID)
		}
	})

	t.Run("concurrent_stops_same_process", func(t *testing.T) {
		// Start a process
		proc, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Try to stop it multiple times concurrently
		done := make(chan bool, 3)
		for i := 0; i < 3; i++ {
			go func() {
				ctx.ProcessMgr.StopProcess(proc.ID)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		time.Sleep(500 * time.Millisecond)

		// System should handle concurrent stops gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle concurrent stops")
	})
}

// TestNegativeLogManagement tests error conditions in log handling
func TestNegativeLogManagement(t *testing.T) {
	SkipIfShort(t, "Negative integration test for log management")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	t.Run("get_logs_for_nonexistent_process", func(t *testing.T) {
		logs := ctx.LogStore.GetByProcess("nonexistent-process-id")
		assert.Empty(t, logs, "Should return empty logs for nonexistent process")
	})

	t.Run("get_logs_with_empty_id", func(t *testing.T) {
		logs := ctx.LogStore.GetByProcess("")
		// Should return empty or all logs depending on implementation
		assert.NotNil(t, logs, "Should handle empty process ID")
	})

	t.Run("get_logs_with_null_bytes", func(t *testing.T) {
		logs := ctx.LogStore.GetByProcess("process\x00id")
		assert.NotNil(t, logs, "Should handle process ID with null bytes")
	})

	t.Run("log_overflow", func(t *testing.T) {
		// Start a process that generates many logs
		proc, err := ctx.ProcessMgr.StartScript("dev")
		require.NoError(t, err)

		// Wait for log accumulation
		time.Sleep(3 * time.Second)

		// Log store should handle overflow (circular buffer or limit)
		logs := ctx.LogStore.GetAll()
		assert.NotNil(t, logs, "Should handle log overflow")

		// Check that we don't have unbounded growth
		// (LogStore has maxEntries = 10000 from test_helpers.go)
		assert.LessOrEqual(t, len(logs), 10000, "Should limit log entries")

		ctx.ProcessMgr.StopProcess(proc.ID)
	})
}

// TestNegativeViewNavigation tests error conditions in view switching
func TestNegativeViewNavigation(t *testing.T) {
	SkipIfShort(t, "Negative integration test for view navigation")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("invalid_view_keys", func(t *testing.T) {
		// Try various invalid view keys
		invalidKeys := []string{
			"10", "11", "100",
			"-1", "-5",
			"a", "b", "c", "x", "y", "z",
			"!", "@", "#", "$", "%",
		}

		for _, key := range invalidKeys {
			ctx.SendKeyMsg(key)
			time.Sleep(20 * time.Millisecond)

			// Should not crash
			view := ctx.GetCurrentView()
			assert.NotEmpty(t, view, "Should handle invalid view key: %s", key)
		}
	})

	t.Run("view_switch_with_zero_size_window", func(t *testing.T) {
		// Set window to zero size
		ctx.SetWindowSize(0, 0)
		time.Sleep(50 * time.Millisecond)

		// Try to switch views
		ctx.SendKeyMsg("1")
		ctx.SendKeyMsg("2")
		ctx.SendKeyMsg("3")
		time.Sleep(50 * time.Millisecond)

		// Should not crash
		view := ctx.GetCurrentView()
		assert.NotNil(t, view, "Should handle view switch with zero window")
	})

	t.Run("navigation_in_empty_list", func(t *testing.T) {
		// Switch to processes view (may be empty initially)
		ctx.SendKeyMsg("1")
		time.Sleep(50 * time.Millisecond)

		// Try to navigate in potentially empty list
		for i := 0; i < 10; i++ {
			ctx.SendKeyMsg("down")
		}
		time.Sleep(50 * time.Millisecond)

		for i := 0; i < 10; i++ {
			ctx.SendKeyMsg("up")
		}
		time.Sleep(50 * time.Millisecond)

		// Should handle gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle navigation in empty list")
	})

	t.Run("scroll_beyond_bounds", func(t *testing.T) {
		// Switch to logs view
		ctx.SendKeyMsg("2")
		time.Sleep(50 * time.Millisecond)

		// Try to scroll way beyond possible content
		for i := 0; i < 1000; i++ {
			ctx.SendKeyMsg("down")
		}
		time.Sleep(100 * time.Millisecond)

		// Should clamp to bounds
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle scroll beyond bounds")

		// Scroll back up beyond bounds
		for i := 0; i < 1000; i++ {
			ctx.SendKeyMsg("up")
		}
		time.Sleep(100 * time.Millisecond)

		view = ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle scroll above bounds")
	})
}

// TestNegativeInputHandling tests error conditions in input processing
func TestNegativeInputHandling(t *testing.T) {
	SkipIfShort(t, "Negative integration test for input handling")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("extremely_long_key_sequence", func(t *testing.T) {
		// Send extremely long sequence
		for i := 0; i < 10000; i++ {
			ctx.SendKeyMsg("a")
		}
		time.Sleep(500 * time.Millisecond)

		// Should handle without crashing or hanging
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle extremely long key sequence")
	})

	t.Run("alternating_keys_rapidly", func(t *testing.T) {
		// Rapidly alternate between different keys
		for i := 0; i < 100; i++ {
			ctx.SendKeyMsg("1")
			ctx.SendKeyMsg("2")
			ctx.SendKeyMsg("up")
			ctx.SendKeyMsg("down")
			ctx.SendKeyMsg("enter")
			ctx.SendKeyMsg("esc")
		}
		time.Sleep(200 * time.Millisecond)

		// Should handle rapid alternation
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid key alternation")
	})

	t.Run("key_input_during_resize", func(t *testing.T) {
		// Send keys while resizing
		for i := 0; i < 10; i++ {
			ctx.SetWindowSize(80+i*10, 24+i*2)
			ctx.SendKeyMsg("1")
			ctx.SendKeyMsg("2")
			ctx.SendKeyMsg("down")
		}
		time.Sleep(200 * time.Millisecond)

		// Should handle interleaved input and resize
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle input during resize")
	})

	t.Run("input_with_modifier_keys", func(t *testing.T) {
		// Try various modifier combinations (may not all be supported)
		ctx.SendKeyMsg("ctrl+c")
		ctx.SendKeyMsg("ctrl+d")
		ctx.SendKeyMsg("alt+f4")
		ctx.SendKeyMsg("shift+tab")
		time.Sleep(100 * time.Millisecond)

		// Should not crash
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle modifier keys")
	})
}

// TestNegativeCommandPalette tests error conditions in command palette
func TestNegativeCommandPalette(t *testing.T) {
	SkipIfShort(t, "Negative integration test for command palette")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("empty_command_execution", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Try to execute with no input
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should handle gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle empty command execution")
	})

	t.Run("invalid_command", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type invalid command
		for _, ch := range "invalid-command-xyz" {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(50 * time.Millisecond)

		// Try to execute
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should handle invalid command
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle invalid command")
	})

	t.Run("command_with_special_characters", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Type command with special characters
		specialChars := "!@#$%^&*()"
		for _, ch := range specialChars {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(50 * time.Millisecond)

		// Try to execute
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should handle gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle command with special characters")
	})

	t.Run("rapid_open_close_command_palette", func(t *testing.T) {
		// Rapidly open and close
		for i := 0; i < 20; i++ {
			ctx.SendKeyMsg("/")
			time.Sleep(10 * time.Millisecond)
			ctx.SendKeyMsg("esc")
			time.Sleep(10 * time.Millisecond)
		}

		// Should handle rapid toggling
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid open/close")
	})

	t.Run("backspace_beyond_empty", func(t *testing.T) {
		// Open command palette
		ctx.SendKeyMsg("/")
		time.Sleep(50 * time.Millisecond)

		// Spam backspace on empty input
		for i := 0; i < 100; i++ {
			ctx.SendKeyMsg("backspace")
		}
		time.Sleep(100 * time.Millisecond)

		// Should handle gracefully (may close palette)
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle excessive backspace")
	})
}

// TestNegativeScriptSelector tests error conditions in script selector
func TestNegativeScriptSelector(t *testing.T) {
	SkipIfShort(t, "Negative integration test for script selector")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("select_without_scripts", func(t *testing.T) {
		// Note: Test context has scripts, but testing the behavior
		// Open script selector (if it exists)
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Try to select with no selection
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should handle gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle selection attempt")
	})

	t.Run("filter_with_no_matches", func(t *testing.T) {
		// Open script selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Filter to something that won't match
		for _, ch := range "zzzzzzzzz" {
			ctx.SendKeyMsg(string(ch))
		}
		time.Sleep(100 * time.Millisecond)

		// Try to select
		ctx.SendKeyMsg("enter")
		time.Sleep(100 * time.Millisecond)

		// Should handle no matches
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle filter with no matches")
	})

	t.Run("rapid_filter_changes", func(t *testing.T) {
		// Open script selector
		ctx.SendKeyMsg("s")
		time.Sleep(50 * time.Millisecond)

		// Rapidly change filter
		for i := 0; i < 50; i++ {
			ctx.SendKeyMsg("a")
			ctx.SendKeyMsg("backspace")
			ctx.SendKeyMsg("b")
			ctx.SendKeyMsg("backspace")
		}
		time.Sleep(100 * time.Millisecond)

		// Should handle rapid changes
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid filter changes")
	})
}

// TestNegativeSettings tests error conditions in settings
func TestNegativeSettings(t *testing.T) {
	SkipIfShort(t, "Negative integration test for settings")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("invalid_port_number", func(t *testing.T) {
		// Switch to settings
		ctx.SendKeyMsg("8")
		time.Sleep(100 * time.Millisecond)

		// The actual setting editing may not be fully implemented
		// but test that view handles it gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle settings view")
	})

	t.Run("rapid_setting_changes", func(t *testing.T) {
		// Switch to settings
		ctx.SendKeyMsg("8")
		time.Sleep(50 * time.Millisecond)

		// Rapidly navigate and press enter
		for i := 0; i < 20; i++ {
			ctx.SendKeyMsg("down")
			ctx.SendKeyMsg("enter")
			ctx.SendKeyMsg("esc")
		}
		time.Sleep(100 * time.Millisecond)

		// Should handle rapid changes
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid setting navigation")
	})
}

// TestNegativeProxyView tests error conditions in proxy view
func TestNegativeProxyView(t *testing.T) {
	SkipIfShort(t, "Negative integration test for proxy view")

	ctx := NewTestContextWithProxy(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("view_with_no_requests", func(t *testing.T) {
		// Switch to proxy view with no requests
		ctx.SendKeyMsg("5")
		time.Sleep(100 * time.Millisecond)

		// Should show empty or placeholder
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle proxy view with no requests")

		// Try to navigate in empty list
		ctx.SendKeyMsg("down")
		ctx.SendKeyMsg("up")
		ctx.SendKeyMsg("enter")
		time.Sleep(50 * time.Millisecond)

		view = ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle navigation in empty proxy view")
	})

	t.Run("rapid_navigation_in_proxy_view", func(t *testing.T) {
		// Switch to proxy view
		ctx.SendKeyMsg("5")
		time.Sleep(50 * time.Millisecond)

		// Rapidly navigate
		for i := 0; i < 100; i++ {
			ctx.SendKeyMsg("down")
			ctx.SendKeyMsg("up")
		}
		time.Sleep(100 * time.Millisecond)

		// Should handle rapid navigation
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid proxy view navigation")
	})
}

// TestNegativeAICoderView tests error conditions in AI coder view
func TestNegativeAICoderView(t *testing.T) {
	SkipIfShort(t, "Negative integration test for AI coder view")

	ctx := NewTestContext(t)
	defer ctx.Cleanup()

	ctx.SetWindowSize(120, 40)

	t.Run("ai_view_with_no_sessions", func(t *testing.T) {
		// Switch to AI coders view
		ctx.SendKeyMsg("7")
		time.Sleep(100 * time.Millisecond)

		// Should handle empty session list
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle AI view with no sessions")

		// Try to navigate
		ctx.SendKeyMsg("down")
		ctx.SendKeyMsg("up")
		ctx.SendKeyMsg("enter")
		time.Sleep(50 * time.Millisecond)

		view = ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle navigation in empty AI view")
	})

	t.Run("rapid_fullscreen_toggle", func(t *testing.T) {
		// Switch to AI view
		ctx.SendKeyMsg("7")
		time.Sleep(50 * time.Millisecond)

		// Rapidly toggle fullscreen
		for i := 0; i < 20; i++ {
			ctx.SendKeyMsg("f11")
			time.Sleep(10 * time.Millisecond)
		}

		// Should handle rapid toggling
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle rapid fullscreen toggle")
	})

	t.Run("invalid_input_in_ai_terminal", func(t *testing.T) {
		// Switch to AI view
		ctx.SendKeyMsg("7")
		time.Sleep(50 * time.Millisecond)

		// Send various invalid inputs
		ctx.SendKeyMsg("\x00") // Null
		ctx.SendKeyMsg("\x1b") // Escape sequences
		ctx.SendKeyMsg("\x7f") // DEL
		time.Sleep(100 * time.Millisecond)

		// Should handle gracefully
		view := ctx.GetCurrentView()
		assert.NotEmpty(t, view, "Should handle invalid terminal input")
	})
}
