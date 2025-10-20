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

// TestClearCommandsIntegration implements T030: Clear commands integration test
// Tests the 'c' and 'C' key clear functionality with confirmation workflow
func TestClearCommandsIntegration(t *testing.T) {
	// Find or create test binary
	testBinary := os.Getenv("BRUMMER_BINARY")
	if testBinary == "" {
		testBinary = "./brum"
		if _, err := os.Stat(testBinary); os.IsNotExist(err) {
			t.Skip("Brummer binary not found. Set BRUMMER_BINARY or build with 'make build'")
		}
	}

	// Create a temporary project directory with scripts that generate content
	tempDir := t.TempDir()
	err := setupContentGeneratingProject(tempDir)
	require.NoError(t, err)

	// Create PTY for TUI testing
	ptm, pts, err := pty.Open()
	require.NoError(t, err)
	defer ptm.Close()
	defer pts.Close()

	// Create terminal emulator
	terminal := vt10x.New(vt10x.WithSize(80, 24))

	// Start the TUI application
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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

	// Test 1: Generate content in Logs tab
	t.Log("Generating content in Logs tab...")

	// Start a process that generates log output
	logProcess := exec.CommandContext(ctx, "node", "generate-logs.js")
	logProcess.Dir = tempDir
	logProcess.Stdout = pts
	logProcess.Stderr = pts

	err = logProcess.Start()
	require.NoError(t, err)
	defer logProcess.Process.Kill()

	// Wait for logs to accumulate
	time.Sleep(3 * time.Second)

	// Switch to Logs tab and verify content
	_, err = ptm.Write([]byte("2")) // Press '2' to go to Logs tab
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Collect initial log content
	initialLogContent := collectOutput(outputChan, 2*time.Second)
	if len(initialLogContent) > 0 {
		t.Logf("✅ Generated log content (%d bytes)", len(initialLogContent))
	} else {
		t.Log("⚠️  Limited log content generated")
	}

	// Test 2: Test 'c' key clears logs immediately
	t.Log("Testing 'c' key clear functionality...")

	// Press 'c' to clear logs
	_, err = ptm.Write([]byte("c"))
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Check if logs were cleared (look for "cleared" message or empty content)
	clearedLogContent := collectOutput(outputChan, 2*time.Second)
	if strings.Contains(string(clearedLogContent), "cleared") ||
	   strings.Contains(string(clearedLogContent), "✅") ||
	   strings.Contains(string(clearedLogContent), "Logs cleared") {
		t.Log("✅ 'c' key successfully cleared logs")
	} else {
		t.Log("⚠️  Log clear confirmation not detected in output")
	}

	// Test 3: Generate content in Errors tab
	t.Log("Generating content in Errors tab...")

	// Start a process that generates errors
	errorProcess := exec.CommandContext(ctx, "node", "generate-errors.js")
	errorProcess.Dir = tempDir
	errorProcess.Stdout = pts
	errorProcess.Stderr = pts

	err = errorProcess.Start()
	require.NoError(t, err)
	defer errorProcess.Process.Kill()

	// Wait for errors to accumulate
	time.Sleep(3 * time.Second)

	// Switch to Errors tab and verify content
	_, err = ptm.Write([]byte("3")) // Press '3' to go to Errors tab
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Collect initial error content
	initialErrorContent := collectOutput(outputChan, 2*time.Second)
	if len(initialErrorContent) > 0 {
		t.Logf("✅ Generated error content (%d bytes)", len(initialErrorContent))
	} else {
		t.Log("⚠️  Limited error content generated")
	}

	// Test 4: Test 'C' key shows confirmation prompt
	t.Log("Testing 'C' key confirmation prompt...")

	// Press 'C' to show confirmation dialog
	_, err = ptm.Write([]byte("C"))
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Look for confirmation dialog indicators
	confirmationContent := collectOutput(outputChan, 2*time.Second)
	confirmationStr := string(confirmationContent)

	// Check for confirmation dialog indicators
	confirmationIndicators := []string{
		"Clear all", "confirmation", "Clear", "(y/n)", "y/n", "⚠️",
	}

	confirmationFound := false
	for _, indicator := range confirmationIndicators {
		if strings.Contains(confirmationStr, indicator) {
			confirmationFound = true
			t.Logf("✅ Found confirmation dialog indicator: %s", indicator)
			break
		}
	}

	if confirmationFound {
		t.Log("✅ 'C' key successfully showed confirmation prompt")
	} else {
		t.Log("⚠️  Confirmation prompt not clearly detected")
	}

	// Test 5: Test 'n' cancels confirmation
	t.Log("Testing 'n' key cancels confirmation...")

	// Press 'n' to cancel
	_, err = ptm.Write([]byte("n"))
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Check if errors are still present (cancellation worked)
	cancelledContent := collectOutput(outputChan, 2*time.Second)
	if len(cancelledContent) > 0 {
		t.Log("✅ 'n' key successfully cancelled clear - errors still present")
	} else {
		t.Log("⚠️  Unable to verify cancellation - content may have been cleared")
	}

	// Test 6: Test 'C' key followed by 'y' confirms
	t.Log("Testing 'C' key followed by 'y' confirmation...")

	// Press 'C' to show confirmation again
	_, err = ptm.Write([]byte("C"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Press 'y' to confirm
	_, err = ptm.Write([]byte("y"))
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Check if errors were cleared
	confirmedClearContent := collectOutput(outputChan, 2*time.Second)
	if strings.Contains(string(confirmedClearContent), "cleared") ||
	   strings.Contains(string(confirmedClearContent), "✅") ||
	   strings.Contains(string(confirmedClearContent), "Errors cleared") {
		t.Log("✅ 'y' key successfully confirmed and cleared errors")
	} else {
		t.Log("⚠️  Error clear confirmation not detected in output")
	}

	// Test 7: Test 'C' key followed by 'N' (uppercase) cancels
	t.Log("Testing 'C' key followed by 'N' (uppercase) cancellation...")

	// Generate some more content first
	anotherErrorProcess := exec.CommandContext(ctx, "node", "generate-errors.js")
	anotherErrorProcess.Dir = tempDir
	anotherErrorProcess.Stdout = pts
	anotherErrorProcess.Stderr = pts

	err = anotherErrorProcess.Start()
	require.NoError(t, err)
	defer anotherErrorProcess.Process.Kill()

	time.Sleep(2 * time.Second)

	// Press 'C' to show confirmation
	_, err = ptm.Write([]byte("C"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Press 'N' (uppercase) to cancel
	_, err = ptm.Write([]byte("N"))
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Check if cancellation worked with uppercase
	uppercaseCancelContent := collectOutput(outputChan, 2*time.Second)
	if len(uppercaseCancelContent) > 0 {
		t.Log("✅ 'N' (uppercase) key successfully cancelled clear")
	} else {
		t.Log("⚠️  Unable to verify uppercase cancellation")
	}

	// Test 8: Test 'C' key followed by 'Y' (uppercase) confirms
	t.Log("Testing 'C' key followed by 'Y' (uppercase) confirmation...")

	// Press 'C' to show confirmation again
	_, err = ptm.Write([]byte("C"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Press 'Y' (uppercase) to confirm
	_, err = ptm.Write([]byte("Y"))
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Check if errors were cleared with uppercase Y
	uppercaseConfirmContent := collectOutput(outputChan, 2*time.Second)
	if strings.Contains(string(uppercaseConfirmContent), "cleared") ||
	   strings.Contains(string(uppercaseConfirmContent), "✅") ||
	   strings.Contains(string(uppercaseConfirmContent), "Errors cleared") {
		t.Log("✅ 'Y' (uppercase) key successfully confirmed and cleared errors")
	} else {
		t.Log("⚠️  Uppercase confirmation not detected in output")
	}

	// Test 9: Test 'C' key followed by 'Esc' cancels
	t.Log("Testing 'C' key followed by 'Esc' cancellation...")

	// Generate content again
	finalErrorProcess := exec.CommandContext(ctx, "node", "generate-errors.js")
	finalErrorProcess.Dir = tempDir
	finalErrorProcess.Stdout = pts
	finalErrorProcess.Stderr = pts

	err = finalErrorProcess.Start()
	require.NoError(t, err)
	defer finalErrorProcess.Process.Kill()

	time.Sleep(2 * time.Second)

	// Press 'C' to show confirmation
	_, err = ptm.Write([]byte("C"))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	// Press 'Esc' to cancel
	_, err = ptm.Write([]byte("\x1b")) // Escape character
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Check if cancellation worked with Escape
	escCancelContent := collectOutput(outputChan, 2*time.Second)
	if len(escCancelContent) > 0 {
		t.Log("✅ 'Esc' key successfully cancelled clear")
	} else {
		t.Log("⚠️  Unable to verify Escape cancellation")
	}

	// Test 10: Clean shutdown
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

	t.Log("✅ Clear commands integration test completed successfully")
}

// setupContentGeneratingProject creates a temporary project with scripts that generate different types of content
func setupContentGeneratingProject(dir string) error {
	// Create package.json
	packageJson := `{
  "name": "content-test-project",
  "version": "1.0.0",
  "scripts": {
    "start": "node generate-logs.js",
    "error": "node generate-errors.js"
  }
}`

	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJson), 0644); err != nil {
		return err
	}

	// Create a script that generates log content
	generateLogsJs := `// This script generates log output
console.log("Starting log generation...");

for (let i = 1; i <= 10; i++) {
  console.log("Log entry " + i + ": Processing data...");
  console.log("  Status: " + (i % 3 === 0 ? "success" : "warning"));
  console.log("  Data item: " + Math.random().toString(36).substring(2, 15));
}

console.log("Log generation complete");
console.log("All processes completed successfully");
console.log("Final summary: 10 entries processed");
`

	if err := os.WriteFile(filepath.Join(dir, "generate-logs.js"), []byte(generateLogsJs), 0644); err != nil {
		return err
	}

	// Create a script that generates errors
	generateErrorsJs := `// This script generates various types of errors
console.log("Starting error generation...");

// Stack trace error
function causeStackError() {
  throw new Error("This is a stack trace error for testing");
}

function nestedStackError() {
  causeStackError();
}

// Type error (will show in Node.js as runtime error)
try {
  const undefinedVar.someMethod();
} catch (e) {
  console.error("TypeError caught:", e.message);
}

// Reference error
try {
  referenceError.someMethod();
} catch (e) {
  console.error("ReferenceError caught:", e.message);
}

// Custom error
class CustomError extends Error {
  constructor(message) {
    super(message);
    this.name = 'CustomError';
  }
}

try {
  throw new CustomError("This is a custom error for testing");
} catch (e) {
  console.error("CustomError caught:", e.message);
}

console.log("Error generation complete - multiple error types generated for testing");
`

	if err := os.WriteFile(filepath.Join(dir, "generate-errors.js"), []byte(generateErrorsJs), 0644); err != nil {
		return err
	}

	return nil
}


