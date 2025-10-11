//go:build integration
// +build integration

package test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

// TestTTY_KeyHandling_q tests the q key in a real TTY environment
func TestTTY_KeyHandling_q(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the binary first
	require.NoError(t, exec.Command("make", "build").Run(), "Failed to build brummer")

	// Start the TUI application in a PTY
	cmd := exec.Command("./brum")
	ptmx, err := pty.Start(cmd)
	require.NoError(t, err, "Failed to start brummer in PTY")
	defer func() {
		cmd.Process.Kill()
		ptmx.Close()
	}()

	// Wait for TUI to initialize
	output := make([]byte, 1024)
	n, err := ptmx.Read(output)
	require.NoError(t, err, "Failed to read from PTY")

	initialOutput := string(output[:n])
	t.Logf("Initial TUI output: %q", initialOutput)

	// Look for the TUI interface
	if !strings.Contains(initialOutput, "Brummer") {
		t.Fatal("TUI did not start properly - no 'Brummer' found in output")
	}

	// Send the 'q' key
	_, err = ptmx.Write([]byte("q"))
	require.NoError(t, err, "Failed to send 'q' key")

	// Read the response after sending 'q'
	time.Sleep(100 * time.Millisecond) // Give the app time to respond

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

	// Force quit with double Ctrl+C
	ptmx.Write([]byte("\003\003"))
	time.Sleep(100 * time.Millisecond)
}

// TestTTY_KeyHandling_arrows tests arrow keys in a real TTY environment
func TestTTY_KeyHandling_arrows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the binary first
	require.NoError(t, exec.Command("make", "build").Run(), "Failed to build brummer")

	// Start the TUI application in a PTY
	cmd := exec.Command("./brum")
	ptmx, err := pty.Start(cmd)
	require.NoError(t, err, "Failed to start brummer in PTY")
	defer func() {
		cmd.Process.Kill()
		ptmx.Close()
	}()

	// Wait for TUI to initialize
	output := make([]byte, 1024)
	n, err := ptmx.Read(output)
	require.NoError(t, err, "Failed to read from PTY")

	initialOutput := string(output[:n])
	t.Logf("Initial TUI output: %q", initialOutput)

	// Look for the TUI interface
	if !strings.Contains(initialOutput, "Brummer") {
		t.Fatal("TUI did not start properly - no 'Brummer' found in output")
	}

	// Note current view (should be Processes)
	currentView := "Processes"
	if strings.Contains(initialOutput, "Logs") {
		currentView = "Logs"
	}
	t.Logf("Starting view: %s", currentView)

	// Send left arrow key
	leftArrow := []byte("\033[D")
	_, err = ptmx.Write(leftArrow)
	require.NoError(t, err, "Failed to send left arrow key")

	// Read the response after sending left arrow
	time.Sleep(100 * time.Millisecond) // Give the app time to respond

	response := make([]byte, 1024)
	n, err = ptmx.Read(response)
	require.NoError(t, err, "Failed to read response after left arrow")

	responseOutput := string(response[:n])
	t.Logf("Response after left arrow: %q", responseOutput)

	// Check if left arrow was handled (should show different view)
	arrowHandled := strings.Contains(responseOutput, "Processes") && currentView != "Processes" ||
		strings.Contains(responseOutput, "Logs") && currentView != "Logs" ||
		strings.Contains(responseOutput, "Errors") && currentView != "Errors" ||
		strings.Contains(responseOutput, "URLs") && currentView != "URLs"

	t.Logf("Left arrow handled: %v", arrowHandled)

	if !arrowHandled {
		t.Error("❌ Left arrow was NOT handled by the TUI application (matches user experience)")
	} else {
		t.Log("✅ Left arrow was handled by the TUI application")
	}

	// Force quit with double Ctrl+C
	ptmx.Write([]byte("\003\003"))
	time.Sleep(100 * time.Millisecond)
}

// TestTTY_KeyHandling_numbers tests number keys in a real TTY environment
func TestTTY_KeyHandling_numbers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the binary first
	require.NoError(t, exec.Command("make", "build").Run(), "Failed to build brummer")

	// Start the TUI application in a PTY
	cmd := exec.Command("./brum")
	ptmx, err := pty.Start(cmd)
	require.NoError(t, err, "Failed to start brummer in PTY")
	defer func() {
		cmd.Process.Kill()
		ptmx.Close()
	}()

	// Wait for TUI to initialize
	output := make([]byte, 1024)
	n, err := ptmx.Read(output)
	require.NoError(t, err, "Failed to read from PTY")

	initialOutput := string(output[:n])
	t.Logf("Initial TUI output: %q", initialOutput)

	// Look for the TUI interface
	if !strings.Contains(initialOutput, "Brummer") {
		t.Fatal("TUI did not start properly - no 'Brummer' found in output")
	}

	// Send number key '2' (should switch to Logs view)
	_, err = ptmx.Write([]byte("2"))
	require.NoError(t, err, "Failed to send '2' key")

	// Read the response after sending '2'
	time.Sleep(100 * time.Millisecond) // Give the app time to respond

	response := make([]byte, 1024)
	n, err = ptmx.Read(response)
	require.NoError(t, err, "Failed to read response after '2' key")

	responseOutput := string(response[:n])
	t.Logf("Response after '2' key: %q", responseOutput)

	// Check if number key was handled (should show Logs view)
	numberHandled := strings.Contains(responseOutput, "Logs") ||
		strings.Contains(responseOutput, "PROCESS") ||
		strings.Contains(responseOutput, "STATUS")

	t.Logf("Number key '2' handled: %v", numberHandled)

	if !numberHandled {
		t.Error("❌ Number key '2' was NOT handled by the TUI application (matches user experience)")
	} else {
		t.Log("✅ Number key '2' was handled by the TUI application")
	}

	// Force quit with double Ctrl+C
	ptmx.Write([]byte("\003\003"))
	time.Sleep(100 * time.Millisecond)
}
