package test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

// TestRealTTY_KeyHandling tests key handling in a real TTY environment
// This test runs the actual TUI application and sends keys to it
func TestRealTTY_KeyHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTY integration test in short mode")
	}

	// Build the binary first (run from project root)
	cmdBuild := exec.Command("make", "build")
	cmdBuild.Dir = ".." // Go up to project root
	require.NoError(t, cmdBuild.Run(), "Failed to build brummer")

	// Start the TUI application in a PTY
	cmd := exec.Command("../brum")
	ptmx, err := pty.Start(cmd)
	require.NoError(t, err, "Failed to start brummer in PTY")
	defer func() {
		cmd.Process.Kill()
		ptmx.Close()
	}()

	// Wait for TUI to initialize
	output := make([]byte, 2048)
	n, err := ptmx.Read(output)
	require.NoError(t, err, "Failed to read from PTY")

	initialOutput := string(output[:n])
	t.Logf("Initial TUI output length: %d", len(initialOutput))
	t.Logf("Initial TUI output snippet: %q", initialOutput[:min(200, len(initialOutput))])

	// Look for the TUI interface
	if !strings.Contains(initialOutput, "Brummer") {
		t.Fatal("TUI did not start properly - no 'Brummer' found in output")
	}

	t.Log("✅ TUI started successfully")

	// Test 1: q key
	t.Run("q_key", func(t *testing.T) {
		// Send the 'q' key
		_, err = ptmx.Write([]byte("q"))
		require.NoError(t, err, "Failed to send 'q' key")

		// Read the response after sending 'q'
		time.Sleep(200 * time.Millisecond) // Give the app time to respond

		response := make([]byte, 1024)
		n, err = ptmx.Read(response)
		require.NoError(t, err, "Failed to read response after 'q' key")

		responseOutput := string(response[:n])
		t.Logf("Response after 'q' key: %q", responseOutput)

		// Check if q key was handled (should show quit warning or signal)
		qHandled := strings.Contains(responseOutput, "again") ||
					strings.Contains(responseOutput, "Interrupt") ||
					strings.Contains(responseOutput, "exit") ||
					strings.Contains(responseOutput, "quit")

		t.Logf("q key handled: %v", qHandled)

		if !qHandled {
			t.Error("❌ q key was NOT handled by the TUI application (matches user experience)")
		} else {
			t.Log("✅ q key was handled by the TUI application")
		}
	})

	// Test 2: Left arrow key
	t.Run("left_arrow", func(t *testing.T) {
		// Send left arrow key
		leftArrow := []byte("\033[D")
		_, err = ptmx.Write(leftArrow)
		require.NoError(t, err, "Failed to send left arrow key")

		// Read the response after sending left arrow
		time.Sleep(200 * time.Millisecond) // Give the app time to respond

		response := make([]byte, 1024)
		n, err = ptmx.Read(response)
		require.NoError(t, err, "Failed to read response after left arrow")

		responseOutput := string(response[:n])
		t.Logf("Response after left arrow: %q", responseOutput)

		// Check if left arrow was handled (should show different view or navigation)
		arrowHandled := strings.Contains(responseOutput, "Processes") ||
						strings.Contains(responseOutput, "Logs") ||
						strings.Contains(responseOutput, "Errors") ||
						strings.Contains(responseOutput, "URLs")

		t.Logf("Left arrow handled: %v", arrowHandled)

		if !arrowHandled {
			t.Error("❌ Left arrow was NOT handled by the TUI application (matches user experience)")
		} else {
			t.Log("✅ Left arrow was handled by the TUI application")
		}
	})

	// Test 3: Ctrl+C
	t.Run("ctrl_c", func(t *testing.T) {
		// Send Ctrl+C
		_, err = ptmx.Write([]byte("\003"))
		require.NoError(t, err, "Failed to send Ctrl+C")

		// Read the response after sending Ctrl+C
		time.Sleep(200 * time.Millisecond) // Give the app time to respond

		response := make([]byte, 1024)
		n, err = ptmx.Read(response)
		require.NoError(t, err, "Failed to read response after Ctrl+C")

		responseOutput := string(response[:n])
		t.Logf("Response after Ctrl+C: %q", responseOutput)

		// Check if Ctrl+C was handled (should show quit warning or signal)
		ctrlCHandled := strings.Contains(responseOutput, "again") ||
						strings.Contains(responseOutput, "Interrupt") ||
						strings.Contains(responseOutput, "exit") ||
						strings.Contains(responseOutput, "quit")

		t.Logf("Ctrl+C handled: %v", ctrlCHandled)

		if !ctrlCHandled {
			t.Error("❌ Ctrl+C was NOT handled by the TUI application (matches user experience)")
		} else {
			t.Log("✅ Ctrl+C was handled by the TUI application")
		}
	})

	// Force quit with double Ctrl+C
	ptmx.Write([]byte("\003\003"))
	time.Sleep(100 * time.Millisecond)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}