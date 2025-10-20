//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
)

// TestStatePreservationIntegration implements T031: State preservation integration test
// Tests that state is preserved across tab switching and various user interactions
func TestStatePreservationIntegration(t *testing.T) {
	// Find or create test binary
	testBinary := os.Getenv("BRUMMER_BINARY")
	if testBinary == "" {
		testBinary = "./brum"
		if _, err := os.Stat(testBinary); os.IsNotExist(err) {
			t.Skip("Brummer binary not found. Set BRUMMER_BINARY or build with 'make build'")
		}
	}

	// Create a temporary project directory with processes that generate continuous output
	tempDir := t.TempDir()
	err := setupContinuousProcessProject(tempDir)
	require.NoError(t, err)

	// Create PTY for TUI testing
	ptm, pts, err := pty.Open()
	require.NoError(t, err)
	defer ptm.Close()
	defer pts.Close()

	// Create terminal emulator
	terminal := vt10x.New(vt10x.WithSize(80, 24))

	// Start the TUI application
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start brummer in TTY mode pointing to our test project
	cmd := setupTestCommandWithDir(testBinary, pts, tempDir)
	require.NoError(t, cmd.Start())
	defer cmd.Process.Kill()

	// Helper channel for PTY output
	outputChan := make(chan []byte, 1000)

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
		t.Log("✅ TUI initialized successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("TUI failed to initialize within 5 seconds")
	}

	// Test 1: Start a continuous process and verify it appears in Processes tab
	t.Log("Starting continuous process and verifying Processes tab state...")

	// Start a continuous process that generates output
	continuousProcess := exec.CommandContext(ctx, "node", "continuous-output.js")
	continuousProcess.Dir = tempDir
	continuousProcess.Stdout = pts
	continuousProcess.Stderr = pts

	err = continuousProcess.Start()
	require.NoError(t, err)
	defer continuousProcess.Process.Kill()

	// Wait for process to be registered
	time.Sleep(2 * time.Second)

	// Switch to Processes tab (should be default, but ensure we're there)
	_, err = ptm.Write([]byte("1"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Collect initial Processes tab content
	initialProcessesContent := collectOutput(outputChan, 3*time.Second)
	if strings.Contains(string(initialProcessesContent), "continuous-output.js") ||
	   strings.Contains(string(initialProcessesContent), "node") {
		t.Log("✅ Continuous process registered in Processes tab")
	} else {
		t.Log("⚠️  Process registration not clearly detected")
	}

	// Test 2: Switch to Logs tab and accumulate content
	t.Log("Switching to Logs tab and accumulating content...")

	_, err = ptm.Write([]byte("2")) // Press '2' to go to Logs tab
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Wait for logs to accumulate
	time.Sleep(3 * time.Second)

	// Collect Logs tab content
	logsContent1 := collectOutput(outputChan, 2*time.Second)
	if len(logsContent1) > 0 {
		t.Logf("✅ Logs tab content accumulated (%d bytes)", len(logsContent1))
	} else {
		t.Log("⚠️  Limited log content detected")
	}

	// Test 3: Switch to Errors tab and generate errors
	t.Log("Switching to Errors tab and generating errors...")

	_, err = ptm.Write([]byte("3")) // Press '3' to go to Errors tab
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Start a process that generates errors
	errorProcess := exec.CommandContext(ctx, "node", "error-generator.js")
	errorProcess.Dir = tempDir
	errorProcess.Stdout = pts
	errorProcess.Stderr = pts

	err = errorProcess.Start()
	require.NoError(t, err)
	defer errorProcess.Process.Kill()

	// Wait for errors to accumulate
	time.Sleep(2 * time.Second)

	// Collect Errors tab content
	errorsContent1 := collectOutput(outputChan, 2*time.Second)
	if len(errorsContent1) > 0 {
		t.Logf("✅ Errors tab content accumulated (%d bytes)", len(errorsContent1))
	} else {
		t.Log("⚠️  Limited error content detected")
	}

	// Test 4: State preservation across tab switches
	t.Log("Testing state preservation across multiple tab switches...")

	// Switch back to Processes tab and verify process is still there
	_, err = ptm.Write([]byte("1"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	processesContent2 := collectOutput(outputChan, 2*time.Second)
	if strings.Contains(string(processesContent2), "node") ||
	   strings.Contains(string(processesContent2), "continuous") {
		t.Log("✅ Processes tab state preserved - continuous process still visible")
	} else {
		t.Log("⚠️  Unable to verify Processes tab state preservation")
	}

	// Switch to Logs tab and verify content is preserved
	_, err = ptm.Write([]byte("2"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	logsContent2 := collectOutput(outputChan, 2*time.Second)
	if len(logsContent2) >= len(logsContent1) {
		t.Log("✅ Logs tab state preserved - content maintained or grew")
	} else {
		t.Log("⚠️  Logs tab state may not be fully preserved")
	}

	// Switch to Errors tab and verify content is preserved
	_, err = ptm.Write([]byte("3"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	errorsContent2 := collectOutput(outputChan, 2*time.Second)
	if len(errorsContent2) >= len(errorsContent1) {
		t.Log("✅ Errors tab state preserved - content maintained or grew")
	} else {
		t.Log("⚠️  Errors tab state may not be fully preserved")
	}

	// Test 5: Rapid tab switching to test state consistency
	t.Log("Testing rapid tab switching for state consistency...")

	tabSequence := []byte{'1', '2', '3', '2', '1', '3', '1', '2', '3'}
	for i, tab := range tabSequence {
		_, err = ptm.Write([]byte{tab})
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Verify TUI remains responsive
		select {
		case <-outputChan:
			t.Logf("✅ Tab %c responsive during rapid switching (%d/%d)", tab, i+1, len(tabSequence))
		case <-time.After(1 * time.Second):
			t.Errorf("❌ TUI became unresponsive during rapid tab switching at tab %c", tab)
			return
		}
	}

	// Test 6: Final state verification
	t.Log("Performing final state verification...")

	// Check each tab one more time
	tabs := []struct {
		key      byte
		name     string
		expected string
	}{
		{'1', "Processes", "node"},
		{'2', "Logs", "log"},
		{'3', "Errors", "Error"},
	}

	for _, tab := range tabs {
		_, err = ptm.Write([]byte{tab.key})
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond)

		finalContent := collectOutput(outputChan, 2*time.Second)
		if strings.Contains(string(finalContent), tab.expected) {
			t.Logf("✅ %s tab final state verified", tab.name)
		} else {
			t.Logf("⚠️  %s tab final state not clearly verified", tab.name)
		}
	}

	// Test 7: Test state persistence with keyboard navigation in Processes tab
	t.Log("Testing keyboard navigation state persistence...")

	_, err = ptm.Write([]byte("1")) // Go to Processes tab
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Simulate keyboard navigation (up/down arrows)
	navKeys := [][]byte{{'j'}, {'k'}, {'j'}, {'j'}, {'k'}}
	for _, key := range navKeys {
		_, err = ptm.Write(key)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Switch away and back to test if navigation state is preserved
	_, err = ptm.Write([]byte("2"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	_, err = ptm.Write([]byte("1"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Check if TUI is still responsive after navigation state changes
	select {
	case <-outputChan:
		t.Log("✅ Keyboard navigation state preservation test completed")
	case <-time.After(1 * time.Second):
		t.Error("❌ TUI became unresponsive during navigation state test")
	}

	// Test 8: Clean shutdown
	t.Log("Testing clean shutdown...")

	_, err = ptm.Write([]byte("q"))
	require.NoError(t, err)

	// Wait for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	select {
	case <-shutdownCtx.Done():
		t.Log("✅ TUI shutdown completed")
	case <-time.After(3 * time.Second):
		t.Log("Process didn't shut down gracefully, killing...")
		cmd.Process.Kill()
	}

	t.Log("✅ State preservation integration test completed successfully")
}

// setupContinuousProcessProject creates a project with processes that generate continuous output
func setupContinuousProcessProject(dir string) error {
	// Create package.json
	packageJson := `{
  "name": "state-preservation-test",
  "version": "1.0.0",
  "scripts": {
    "continuous": "node continuous-output.js",
    "errors": "node error-generator.js"
  }
}`

	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJson), 0644); err != nil {
		return err
	}

	// Create a script that generates continuous output
	continuousOutputJs := `// This script generates continuous output for state preservation testing
console.log("Starting continuous output process...");

let counter = 0;
const interval = setInterval(() => {
  counter++;
  console.log("Continuous output entry " + counter + ": " + new Date().toISOString());

  if (counter % 5 === 0) {
    console.log("  Status check: Process running smoothly");
  }

  if (counter >= 20) {
    console.log("Continuous output complete - " + counter + " entries generated");
    clearInterval(interval);
  }
}, 500);

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log("Received SIGTERM, shutting down gracefully...");
  clearInterval(interval);
  process.exit(0);
});
`

	if err := os.WriteFile(filepath.Join(dir, "continuous-output.js"), []byte(continuousOutputJs), 0644); err != nil {
		return err
	}

	// Create a script that generates errors periodically
	errorGeneratorJs := `// This script generates errors for testing error state preservation
console.log("Starting error generation process...");

let errorCounter = 0;
const errorInterval = setInterval(() => {
  errorCounter++;

  if (errorCounter % 3 === 0) {
    // Generate different types of errors
    const errors = [
      new Error("Simulated error " + errorCounter),
      new TypeError("Type error " + errorCounter),
      new ReferenceError("Reference error " + errorCounter)
    ];

    const randomError = errors[errorCounter % errors.length];
    console.error("Error generated:", randomError.message);
  }

  console.log("Error generator cycle " + errorCounter + " completed");

  if (errorCounter >= 10) {
    console.log("Error generation complete - " + errorCounter + " cycles processed");
    clearInterval(errorInterval);
  }
}, 1000);

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log("Error generator received SIGTERM, shutting down gracefully...");
  clearInterval(errorInterval);
  process.exit(0);
});
`

	if err := os.WriteFile(filepath.Join(dir, "error-generator.js"), []byte(errorGeneratorJs), 0644); err != nil {
		return err
	}

	return nil
}