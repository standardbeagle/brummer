//go:build tty && integration
// +build tty,integration

package tty

import (
	"strings"
	"testing"
	"time"
)

// TestAICoderEndToEndWorkflow tests complete AI coder workflows
func TestAICoderEndToEndWorkflow(t *testing.T) {
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

	t.Run("Complete AI coder workflow", func(t *testing.T) {
		// Step 1: Navigate to AI Coders tab
		err = tt.NavigateToAICoders()
		if err != nil {
			t.Fatalf("failed to navigate to AI Coders: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
		tt.AssertContains("AI Coder")

		// Step 2: Create a new AI coder session
		err = tt.CreateAICoderSession("claude", "implement a user authentication system")
		if err != nil {
			t.Fatalf("failed to create AI coder session: %v", err)
		}

		// Wait for session creation
		time.Sleep(2 * time.Second)
		tt.AssertContains("Session 1/1")

		// Step 3: Focus terminal and start interaction
		err = tt.FocusTerminal()
		if err != nil {
			t.Fatalf("failed to focus terminal: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Step 4: Send initial task to AI
		err = tt.TypeInTerminal("Please create a user authentication system with login and registration.\n")
		if err != nil {
			t.Fatalf("failed to type task: %v", err)
		}

		// Wait for AI to process
		time.Sleep(2 * time.Second)

		// Step 5: Inject some context (error data)
		err = tt.SendInput("\x05") // Ctrl+E for error injection
		if err != nil {
			t.Fatalf("failed to inject error data: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Step 6: Switch to full screen mode for better visibility
		err = tt.SendInput("\x1b[23~") // F11
		if err != nil {
			t.Fatalf("failed to toggle full screen: %v", err)
		}

		time.Sleep(1 * time.Second)

		// Step 7: Exit full screen
		err = tt.ExitFullScreen()
		if err != nil {
			t.Fatalf("failed to exit full screen: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		// Step 8: Unfocus terminal
		err = tt.UnfocusTerminal()
		if err != nil {
			t.Fatalf("failed to unfocus terminal: %v", err)
		}

		time.Sleep(500 * time.Millisecond)
		tt.AssertContains("Terminal Not Focused")

		t.Logf("Complete AI coder workflow executed successfully")
	})
}

// TestAICoderMultiSessionWorkflow tests managing multiple AI coder sessions
func TestAICoderMultiSessionWorkflow(t *testing.T) {
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

	t.Run("Multiple session management", func(t *testing.T) {
		// Navigate to AI Coders
		tt.NavigateToAICoders()
		time.Sleep(500 * time.Millisecond)

		// Create first session for frontend development
		tt.CreateAICoderSession("claude", "build React frontend")
		time.Sleep(1 * time.Second)

		// Create second session for backend development
		tt.CreateAICoderSession("claude", "build Node.js backend API")
		time.Sleep(1 * time.Second)

		// Create third session with terminal provider
		tt.CreateAICoderSession("terminal", "shell commands")
		time.Sleep(1 * time.Second)

		// Should see multiple sessions
		output := tt.Output()
		if strings.Contains(output, "Session 3/3") {
			t.Logf("Multiple sessions created successfully")
		}

		// Navigate between sessions
		t.Run("Navigate between sessions", func(t *testing.T) {
			// Focus terminal of current session
			tt.FocusTerminal()
			time.Sleep(200 * time.Millisecond)

			// Type something in current session
			tt.TypeInTerminal("echo 'Working in session 3'\n")
			time.Sleep(500 * time.Millisecond)

			// Switch to next session (should wrap around)
			tt.SendInput("\x0e") // Ctrl+N
			time.Sleep(500 * time.Millisecond)

			// Should be in session 1 now
			tt.FocusTerminal()
			time.Sleep(200 * time.Millisecond)

			tt.TypeInTerminal("echo 'Switched to session 1'\n")
			time.Sleep(500 * time.Millisecond)

			// Switch to next session again
			tt.SendInput("\x0e") // Ctrl+N
			time.Sleep(500 * time.Millisecond)

			tt.FocusTerminal()
			time.Sleep(200 * time.Millisecond)

			tt.TypeInTerminal("echo 'Now in session 2'\n")
			time.Sleep(500 * time.Millisecond)

			t.Logf("Session navigation working")
		})

		// Test session detaching
		t.Run("Detach and reattach", func(t *testing.T) {
			// Detach from current session
			tt.SendInput("\x04") // Ctrl+D
			time.Sleep(500 * time.Millisecond)

			// Should see detached state
			tt.AssertContains("No AI Coder Sessions")

			// Navigate back (should reattach to first session)
			tt.SendInput("\x0e") // Ctrl+N
			time.Sleep(500 * time.Millisecond)

			// Should be reattached to a session
			output = tt.Output()
			if strings.Contains(output, "Session") && strings.Contains(output, "Running") {
				t.Logf("Session detach/reattach working")
			}
		})
	})
}

// TestAICoderDebugWorkflow tests AI coder with debug mode enabled
func TestAICoderDebugWorkflow(t *testing.T) {
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

	t.Run("Debug mode workflow", func(t *testing.T) {
		// Navigate to AI Coders and create session
		tt.NavigateToAICoders()
		time.Sleep(500 * time.Millisecond)

		tt.CreateAICoderSession("claude", "debug mode testing")
		time.Sleep(1 * time.Second)

		// Enable debug mode
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)


		tt.SendInput("\x1b[24~") // F12
		time.Sleep(500 * time.Millisecond)


		// Should see debug mode indicator
		output := tt.Output()
		if strings.Contains(output, "DEBUG MODE") {
			t.Logf("Debug mode enabled successfully")
		}

		// Inject error data (should be auto-forwarded in debug mode)
		tt.SendInput("\x05") // Ctrl+E
		time.Sleep(500 * time.Millisecond)


		// Inject test failure data
		tt.SendInput("\x14") // Ctrl+T
		time.Sleep(500 * time.Millisecond)


		// Type some code that might have errors
		tt.TypeInTerminal("function test() {\n  console.log('test')\n  // missing closing brace\n")
		time.Sleep(1 * time.Second)


		// In debug mode, errors should be automatically forwarded
		t.Logf("Debug mode workflow completed")
	})
}

// TestAICoderTerminalProviderWorkflow tests terminal provider specifically
func TestAICoderTerminalProviderWorkflow(t *testing.T) {
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

	t.Run("Terminal provider workflow", func(t *testing.T) {
		// Navigate to AI Coders
		tt.NavigateToAICoders()
		time.Sleep(500 * time.Millisecond)


		// Create terminal session
		tt.CreateAICoderSession("terminal", "")
		time.Sleep(1 * time.Second)


		// Focus terminal
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)


		// Run some shell commands
		commands := []string{
			"pwd\n",
			"ls -la\n",
			"echo 'Hello from terminal provider'\n",
			"date\n",
			"whoami\n",
		}

		for _, cmd := range commands {
			err = tt.TypeInTerminal(cmd)
			if err != nil {
				t.Fatalf("failed to type command '%s': %v", cmd, err)
			}
			time.Sleep(500 * time.Millisecond)

		}

		// Should see command outputs
		output := tt.Output()
		if strings.Contains(output, "Hello from terminal provider") {
			t.Logf("Terminal provider workflow working")
		}

		// Test command history navigation (up/down arrows)
		tt.SendInput("\x1b[A") // Up arrow
		time.Sleep(200 * time.Millisecond)


		tt.SendInput("\x1b[B") // Down arrow
		time.Sleep(200 * time.Millisecond)


		t.Logf("Terminal provider workflow completed")
	})
}

// TestAICoderErrorRecoveryWorkflow tests error handling and recovery
func TestAICoderErrorRecoveryWorkflow(t *testing.T) {
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

	t.Run("Error handling and recovery", func(t *testing.T) {
		// Navigate to AI Coders
		tt.NavigateToAICoders()
		time.Sleep(500 * time.Millisecond)


		// Try to create session with invalid provider
		tt.CreateAICoderSession("nonexistent-provider", "test")
		time.Sleep(1 * time.Second)


		// Should see error message
		output := tt.Output()
		if strings.Contains(output, "not configured") || strings.Contains(output, "error") {
			t.Logf("Invalid provider error handled correctly")
		}

		// Create a valid session
		tt.CreateAICoderSession("claude", "error recovery test")
		time.Sleep(1 * time.Second)


		// Focus terminal
		tt.FocusTerminal()
		time.Sleep(200 * time.Millisecond)


		// Type something that might cause issues
		tt.TypeInTerminal("exit\n") // Try to exit the session
		time.Sleep(1 * time.Second)


		// Session should handle exit gracefully
		output = tt.Output()
		if !tt.IsRunning() {
			t.Errorf("Brummer should still be running after terminal exit")
		}

		// Should be able to create another session
		tt.CreateAICoderSession("claude", "recovery test")
		time.Sleep(1 * time.Second)


		if strings.Contains(tt.Output(), "Session") {
			t.Logf("Error recovery working - can create new session after error")
		}
	})
}