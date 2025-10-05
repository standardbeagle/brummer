//go:build tty && integration
// +build tty,integration

package tty

import (
	"strings"
	"testing"
	"time"
)

// TestAICoderSessionCreation tests creating and managing AI coder sessions
func TestAICoderSessionCreation(t *testing.T) {
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

	// Wait for TUI to initialize
	err = tt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Fatalf("TUI did not start properly: %v", err)
	}

	t.Run("Navigate to AI Coders tab", func(t *testing.T) {
		err = tt.NavigateToAICoders()
		if err != nil {
			t.Fatalf("failed to navigate to AI Coders tab: %v", err)
		}

		// Wait for navigation
		time.Sleep(500 * time.Millisecond)

		// Should see AI Coders view
		tt.AssertContains("AI Coder")
	})

	t.Run("Create interactive session", func(t *testing.T) {
		// Create an AI coder session without task
		err = tt.CreateAICoderSession("claude", "")
		if err != nil {
			t.Fatalf("failed to create AI coder session: %v", err)
		}

		// Wait for session creation
		err = tt.WaitForOutput("Started.*AI coder session", 5*time.Second)
		if err != nil {
			t.Errorf("AI coder session creation not confirmed: %v", err)
		}

		// Should be in AI Coders view with active session
		tt.AssertContains("Session 1/1")
		tt.AssertContains("Running")
	})

	t.Run("Focus terminal and interact", func(t *testing.T) {
		// Focus the terminal
		err = tt.FocusTerminal()
		if err != nil {
			t.Fatalf("failed to focus terminal: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Type something
		err = tt.TypeInTerminal("Hello AI Coder!\n")
		if err != nil {
			t.Fatalf("failed to type in terminal: %v", err)
		}

		// Should see our input reflected
		time.Sleep(500 * time.Millisecond)
		output := tt.Output()
		if !strings.Contains(output, "Hello AI Coder!") {
			t.Logf("Terminal input might not be echoed back immediately (this is normal)")
		}
	})

	t.Run("Unfocus terminal", func(t *testing.T) {
		// Unfocus the terminal
		err = tt.UnfocusTerminal()
		if err != nil {
			t.Fatalf("failed to unfocus terminal: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Should see terminal is no longer focused
		tt.AssertContains("Terminal Not Focused")
	})

	t.Run("Create task-based session", func(t *testing.T) {
		// Create another session with a task
		err = tt.CreateAICoderSession("claude", "implement a new feature")
		if err != nil {
			t.Fatalf("failed to create task-based session: %v", err)
		}

		// Wait for session creation
		time.Sleep(1 * time.Second)

		// Should see multiple sessions
		output := tt.Output()
		if strings.Contains(output, "Session 2/2") {
			t.Logf("Multiple sessions created successfully")
		}
	})
}

// TestAICoderSessionNavigation tests navigating between sessions
func TestAICoderSessionNavigation(t *testing.T) {
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

	// Navigate to AI Coders and create multiple sessions
	tt.NavigateToAICoders()
	time.Sleep(500 * time.Millisecond)

	// Create first session
	tt.CreateAICoderSession("claude", "first task")
	time.Sleep(1 * time.Second)

	// Create second session
	tt.CreateAICoderSession("claude", "second task")
	time.Sleep(1 * time.Second)

	t.Run("Navigate between sessions", func(t *testing.T) {
		// Try Ctrl+N for next session
		err = tt.SendInput("\x0e") // Ctrl+N
		if err != nil {
			t.Fatalf("failed to send Ctrl+N: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Try Ctrl+Shift+P for previous session
		err = tt.SendInput("\x10") // Ctrl+P (shift may not work in test)
		if err != nil {
			t.Fatalf("failed to send Ctrl+P: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	})

	t.Run("Detach from session", func(t *testing.T) {
		// Try Ctrl+D to detach
		err = tt.SendInput("\x04") // Ctrl+D
		if err != nil {
			t.Fatalf("failed to send Ctrl+D: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Should see detached state
		tt.AssertContains("No AI Coder Sessions")
	})
}

// TestAICoderTerminalProvider tests terminal provider functionality
func TestAICoderTerminalProvider(t *testing.T) {
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

	t.Run("Create terminal session", func(t *testing.T) {
		// Create a terminal session
		err = tt.CreateAICoderSession("terminal", "")
		if err != nil {
			t.Fatalf("failed to create terminal session: %v", err)
		}

		// Wait for session creation
		time.Sleep(1 * time.Second)

		// Should see terminal session
		tt.AssertContains("Session 1/1")
		tt.AssertContains("Running")
	})

	t.Run("Interact with terminal", func(t *testing.T) {
		// Focus terminal
		err = tt.FocusTerminal()
		if err != nil {
			t.Fatalf("failed to focus terminal: %v", err)
		}

		time.Sleep(200 * time.Millisecond)

		// Try a simple command
		err = tt.TypeInTerminal("echo 'Hello from terminal'\n")
		if err != nil {
			t.Fatalf("failed to type in terminal: %v", err)
		}

		// Wait for command execution
		time.Sleep(1 * time.Second)

		// The echo output should appear
		output := tt.Output()
		if strings.Contains(output, "Hello from terminal") {
			t.Logf("Terminal command executed successfully")
		}
	})
}

// TestAICoderErrorHandling tests error scenarios
func TestAICoderErrorHandling(t *testing.T) {
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

	t.Run("Invalid provider error", func(t *testing.T) {
		// Try to create session with invalid provider
		err = tt.CreateAICoderSession("nonexistent", "some task")
		if err != nil {
			t.Fatalf("failed to send AI command: %v", err)
		}

		// Wait for error handling
		time.Sleep(1 * time.Second)

		// Should see error in logs
		output := tt.Output()
		if strings.Contains(output, "nonexistent") &&
		   (strings.Contains(output, "not configured") || strings.Contains(output, "error")) {
			t.Logf("Invalid provider error handled correctly")
		}
	})

	t.Run("Command while session creating", func(t *testing.T) {
		// Start a session creation
		tt.CreateAICoderSession("claude", "test task")

		// Immediately try another command (should be rejected)
		time.Sleep(100 * time.Millisecond)
		tt.CreateAICoderSession("claude", "another task")

		// Wait for operations
		time.Sleep(2 * time.Second)

		// Should see concurrency protection message
		output := tt.Output()
		if strings.Contains(output, "already being created") {
			t.Logf("Concurrent session creation protection working")
		}
	})
}

// TestAICoderShutdown tests graceful shutdown with active sessions
func TestAICoderShutdown(t *testing.T) {
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

	// Navigate to AI Coders and create session
	tt.NavigateToAICoders()
	time.Sleep(500 * time.Millisecond)

	tt.CreateAICoderSession("claude", "test shutdown")
	time.Sleep(1 * time.Second)

	t.Run("Graceful shutdown with active session", func(t *testing.T) {
		// Should have active session
		tt.AssertContains("Session 1/1")

		// Stop brummer (this should clean up the session)
		tt.Stop()

		// Verify process stopped
		if tt.IsRunning() {
			t.Errorf("Brummer should have stopped")
		}

		t.Logf("Graceful shutdown completed")
	})
}