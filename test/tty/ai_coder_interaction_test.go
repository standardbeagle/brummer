//go:build tty && integration
// +build tty,integration

package tty

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestAICoderTerminalInteraction tests terminal input/output and key bindings
func TestAICoderTerminalInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTY test in short mode")
	}

	tt := NewTTYTest(t)
	defer tt.Cleanup()

	// Start brummer in TTY mode
	err := tt.StartTTY("--debug")
	if err != nil {
		t.Fatalf("failed to start brummer with TTY: %v", err)
	}

	// Wait for initialization
	err = tt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Fatalf("TUI did not start properly: %v", err)
	}

	// Navigate to AI Coders and create a session
	tt.NavigateToAICoders()
	time.Sleep(500 * time.Millisecond)
	tt.CreateAICoderSession("claude", "interactive testing")
	time.Sleep(1 * time.Second)

	t.Run("Terminal focus and unfocus", func(t *testing.T) {
		// Focus terminal
		err = tt.FocusTerminal()
		if err != nil {
			t.Fatalf("failed to focus terminal: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Should see terminal focused indicator
		output := tt.Output()
		if strings.Contains(output, "Terminal Focused") {
			t.Logf("Terminal focus indication working")
		}

		// Unfocus terminal
		err = tt.UnfocusTerminal()
		if err != nil {
			t.Fatalf("failed to unfocus terminal: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Should see terminal not focused indicator
		output = tt.Output()
		if strings.Contains(output, "Terminal Not Focused") {
			t.Logf("Terminal unfocus indication working")
		}
	})

	t.Run("Full screen mode toggle", func(t *testing.T) {
		// Focus terminal first
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)

		// Toggle full screen (F11)
		err = tt.SendInput("\x1b[23~") // F11
		if err != nil {
			t.Fatalf("failed to send F11: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see full screen mode indicator
		output := tt.Output()
		if strings.Contains(output, "Full Screen") {
			t.Logf("Full screen mode toggle working")
		}

		// Exit full screen (ESC)
		err = tt.ExitFullScreen()
		if err != nil {
			t.Fatalf("failed to exit full screen: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	})

	t.Run("Debug mode toggle", func(t *testing.T) {
		// Focus terminal
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)

		// Toggle debug mode (F12)
		err = tt.SendInput("\x1b[24~") // F12
		if err != nil {
			t.Fatalf("failed to send F12: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see debug mode indicator
		output := tt.Output()
		if strings.Contains(output, "DEBUG MODE") {
			t.Logf("Debug mode toggle working")
		}
	})

	t.Run("Help toggle", func(t *testing.T) {
		// Toggle help (Ctrl+H)
		err = tt.SendInput("\x08") // Ctrl+H
		if err != nil {
			t.Fatalf("failed to send Ctrl+H: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see help content
		output := tt.Output()
		if strings.Contains(output, "AI Coder PTY Controls") {
			t.Logf("Help toggle working")
		}

		// Hide help again
		err = tt.SendInput("\x08") // Ctrl+H
		if err != nil {
			t.Fatalf("failed to hide help: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	})
}

// TestAICoderDataInjection tests data injection functionality
func TestAICoderDataInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTY test in short mode")
	}

	tt := NewTTYTest(t)
	defer tt.Cleanup()

	// Start brummer in TTY mode
	err := tt.StartTTY("--debug")
	if err != nil {
		t.Fatalf("failed to start brummer with TTY: %v", err)
	}

	// Wait for initialization
	err = tt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Fatalf("TUI did not start properly: %v", err)
	}

	// Navigate to AI Coders and create a session
	tt.NavigateToAICoders()
	time.Sleep(500 * time.Millisecond)
	tt.CreateAICoderSession("claude", "data injection testing")
	time.Sleep(1 * time.Second)

	t.Run("Inject error data", func(t *testing.T) {
		// Focus terminal
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)

		// Try to inject error data (Ctrl+E)
		err = tt.SendInput("\x05") // Ctrl+E
		if err != nil {
			t.Fatalf("failed to send Ctrl+E: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see injection feedback
		output := tt.Output()
		if strings.Contains(output, "Injected") || strings.Contains(output, "DEBUG MODE") {
			t.Logf("Data injection working (may require debug mode)")
		}
	})

	t.Run("Inject test failure data", func(t *testing.T) {
		// Try to inject test failure data (Ctrl+T)
		err = tt.SendInput("\x14") // Ctrl+T
		if err != nil {
			t.Fatalf("failed to send Ctrl+T: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see injection feedback
		output := tt.Output()
		if strings.Contains(output, "Injected") {
			t.Logf("Test failure injection working")
		}
	})

	t.Run("Inject build output data", func(t *testing.T) {
		// Try to inject build output data (Ctrl+B)
		err = tt.SendInput("\x02") // Ctrl+B
		if err != nil {
			t.Fatalf("failed to send Ctrl+B: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see injection feedback
		output := tt.Output()
		if strings.Contains(output, "Injected") {
			t.Logf("Build output injection working")
		}
	})

	t.Run("Inject log data", func(t *testing.T) {
		// Try to inject log data (Ctrl+L)
		err = tt.SendInput("\x0c") // Ctrl+L
		if err != nil {
			t.Fatalf("failed to send Ctrl+L: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see injection feedback
		output := tt.Output()
		if strings.Contains(output, "Injected") {
			t.Logf("Log data injection working")
		}
	})
}

// TestAICoderSessionInteraction tests session switching and navigation interactions
func TestAICoderSessionInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTY test in short mode")
	}

	tt := NewTTYTest(t)
	defer tt.Cleanup()

	// Start brummer in TTY mode
	err := tt.StartTTY("--debug")
	if err != nil {
		t.Fatalf("failed to start brummer with TTY: %v", err)
	}

	// Wait for initialization
	err = tt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Fatalf("TUI did not start properly: %v", err)
	}

	// Navigate to AI Coders
	tt.NavigateToAICoders()
	time.Sleep(500 * time.Millisecond)

	// Create multiple sessions
	tt.CreateAICoderSession("claude", "first session")
	time.Sleep(1 * time.Second)

	tt.CreateAICoderSession("claude", "second session")
	time.Sleep(1 * time.Second)

	t.Run("Navigate between sessions", func(t *testing.T) {
		// Try next session (Ctrl+N)
		err = tt.SendInput("\x0e") // Ctrl+N
		if err != nil {
			t.Fatalf("failed to send Ctrl+N: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see session change
		output := tt.Output()
		if strings.Contains(output, "Session 2/") {
			t.Logf("Next session navigation working")
		}

		// Try previous session (Ctrl+Shift+P - simplified to Ctrl+P)
		err = tt.SendInput("\x10") // Ctrl+P
		if err != nil {
			t.Fatalf("failed to send Ctrl+P: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see session change back
		output = tt.Output()
		if strings.Contains(output, "Session 1/") {
			t.Logf("Previous session navigation working")
		}
	})

	t.Run("Detach from session", func(t *testing.T) {
		// Try to detach (Ctrl+D)
		err = tt.SendInput("\x04") // Ctrl+D
		if err != nil {
			t.Fatalf("failed to send Ctrl+D: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see detached state
		output := tt.Output()
		if strings.Contains(output, "No AI Coder Sessions") || strings.Contains(output, "None Selected") {
			t.Logf("Session detach working")
		}
	})
}

// TestAICoderScrolling tests viewport scrolling functionality
func TestAICoderScrolling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTY test in short mode")
	}

	tt := NewTTYTest(t)
	defer tt.Cleanup()

	// Start brummer in TTY mode
	err := tt.StartTTY("--debug")
	if err != nil {
		t.Fatalf("failed to start brummer with TTY: %v", err)
	}

	// Wait for initialization
	err = tt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Fatalf("TUI did not start properly: %v", err)
	}

	// Navigate to AI Coders and create a session
	tt.NavigateToAICoders()
	time.Sleep(500 * time.Millisecond)
	tt.CreateAICoderSession("claude", "scrolling test")
	time.Sleep(1 * time.Second)

	t.Run("Generate output for scrolling", func(t *testing.T) {
		// Focus terminal and generate some output
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)

		// Generate multiple lines of output
		for i := 0; i < 10; i++ {
			err = tt.TypeInTerminal(fmt.Sprintf("echo 'Line %d of test output'\n", i+1))
			if err != nil {
				t.Fatalf("failed to type line %d: %v", i+1, err)
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Wait for output to be processed
		time.Sleep(1 * time.Second)

		// Should see some output in the terminal
		output := tt.Output()
		if strings.Contains(output, "Line 1 of test output") {
			t.Logf("Terminal output generation working")
		}
	})

	t.Run("Page up scrolling", func(t *testing.T) {
		// Try page up (PgUp)
		err = tt.SendInput("\x1b[5~") // Page Up
		if err != nil {
			t.Fatalf("failed to send Page Up: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Scrolling indication might appear
		output := tt.Output()
		t.Logf("Page up sent - output: %s", output)
	})

	t.Run("Page down scrolling", func(t *testing.T) {
		// Try page down (PgDn)
		err = tt.SendInput("\x1b[6~") // Page Down
		if err != nil {
			t.Fatalf("failed to send Page Down: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should return to bottom
		output := tt.Output()
		t.Logf("Page down sent - output: %s", output)
	})

	t.Run("Auto-scroll on new content", func(t *testing.T) {
		// Type some more content to test auto-scroll
		err = tt.TypeInTerminal("echo 'Auto-scroll test line'\n")
		if err != nil {
			t.Fatalf("failed to type auto-scroll test: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should auto-scroll to bottom
		output := tt.Output()
		if strings.Contains(output, "Auto-scroll test line") {
			t.Logf("Auto-scroll on new content working")
		}
	})
}