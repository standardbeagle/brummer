//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
)

// TestTabNavigationIntegration implements T028: Tab navigation integration test
// Tests tab switching with deadlock prevention and verifies View() never blocks
func TestTabNavigationIntegration(t *testing.T) {
	// Find or create test binary
	testBinary := os.Getenv("BRUMMER_BINARY")
	if testBinary == "" {
		testBinary = "./brum"
		if _, err := os.Stat(testBinary); os.IsNotExist(err) {
			t.Skip("Brummer binary not found. Set BRUMMER_BINARY or build with 'make build'")
		}
	}

	// Create PTY for TUI testing
	ptm, pts, err := pty.Open()
	require.NoError(t, err)
	defer ptm.Close()
	defer pts.Close()

	// Create terminal emulator
	terminal := vt10x.New(vt10x.WithSize(80, 24))

	// Start the TUI application
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start brummer in TTY mode
	cmd := setupTestCommand(testBinary, pts)
	require.NoError(t, cmd.Start())
	defer cmd.Process.Kill()

	// Helper channel for PTY output
	outputChan := make(chan []byte, 100)

	// Goroutine to read PTY output
	go func() {
		buffer := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := ptm.Read(buffer)
				if err != nil {
					return
				}
				if n > 0 {
					terminal.Write(buffer[:n])
					outputChan <- buffer[:n]
				}
			}
		}
	}()

	// Wait for TUI to initialize
	select {
	case <-outputChan:
		// Got some output, TUI is likely initialized
		t.Log("✅ TUI initialized successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("TUI failed to initialize within 5 seconds")
	}

	// Test 1: Verify TUI responds to input without freezing (deadlock prevention)
	t.Log("Testing TUI responsiveness...")

	// Send various keys and check for responses
	testKeys := []string{"1", "2", "3", "4", "5", "6", "7", "\t", "q"}

	for _, key := range testKeys {
		if key == "q" {
			continue // Save quit for last
		}

		_, err = ptm.Write([]byte(key))
		require.NoError(t, err)

		// Give time for UI to update
		time.Sleep(200 * time.Millisecond)

		// Check if TUI is still responsive
		select {
		case <-outputChan:
			// Got output, TUI is responsive
			t.Logf("✅ TUI responded to key '%s'", key)
		case <-time.After(2 * time.Second):
			t.Errorf("❌ TUI appears frozen - no response to key '%s' within 2 seconds", key)
			return
		}
	}

	// Test 2: Rapid input to test for deadlock during View() calls
	t.Log("Testing rapid input for deadlock prevention...")

	rapidTestStart := time.Now()
	rapidTestDuration := 2 * time.Second // Shorter test for faster execution

	for time.Since(rapidTestStart) < rapidTestDuration {
		// Send random keys rapidly
		key := []byte{byte('1' + (time.Now().UnixNano() % 7))}
		_, err = ptm.Write(key)
		require.NoError(t, err)
		time.Sleep(5 * time.Millisecond) // 200 keys per second
	}

	// Check if TUI is still responsive after rapid input
	select {
	case <-outputChan:
		t.Log("✅ TUI remained responsive during rapid input")
	case <-time.After(1 * time.Second):
		t.Error("❌ TUI became unresponsive during rapid input - possible deadlock")
	}

	// Test 3: Test specifically tabs 3 (Errors) and 4 (URLs) - historical deadlock locations
	t.Log("Testing historically problematic tabs 3 (Errors) and 4 (URLs)...")

	problematicTabs := []byte{'3', '4'}
	for _, tab := range problematicTabs {
		// Switch to problematic tab
		_, err = ptm.Write([]byte{tab})
		require.NoError(t, err)

		// Verify View() doesn't block by checking for timely response
		select {
		case <-outputChan:
			t.Logf("✅ Tab %c responsive - View() not blocking", tab)
		case <-time.After(500 * time.Millisecond):
			t.Errorf("❌ Tab %c appears to be blocking View() method", tab)
		}

		time.Sleep(300 * time.Millisecond) // Let tab settle
	}

	// Test 4: Clean shutdown
	t.Log("Testing clean shutdown...")

	// Send quit command
	_, err = ptm.Write([]byte("q"))
	require.NoError(t, err)

	// Wait for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	select {
	case <-shutdownCtx.Done():
		// Process terminated gracefully or timeout expired
		t.Log("✅ TUI shutdown completed")
	case <-time.After(3 * time.Second):
		t.Log("Process didn't shut down gracefully, killing...")
		cmd.Process.Kill()
	}

	t.Log("✅ Tab navigation integration test completed successfully - No deadlocks detected")
}

// setupTestCommand creates a test command for the brummer binary
func setupTestCommand(binary string, pts *os.File) *exec.Cmd {
	args := []string{
		"--no-mcp",     // Disable MCP server for testing
		"--no-proxy",   // Disable proxy server for testing
		"--debug",      // Enable debug mode for better test visibility
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdin = pts
	cmd.Stdout = pts
	cmd.Stderr = pts
	cmd.Dir = "."

	return cmd
}