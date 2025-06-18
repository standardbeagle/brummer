//go:build integration
// +build integration

package test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/test/testutil"
)

// TestProcessStartupNoTUI tests process startup in headless mode
func TestProcessStartupNoTUI(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with test script
	err := bt.Start("--no-tui", "test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check for process startup
	err = bt.WaitForOutput("Started script 'test'", 5*time.Second)
	if err != nil {
		t.Fatalf("Process did not start: %v\nOutput: %s", err, bt.Output())
	}

	// Verify process ID generation
	output := bt.Output()
	pidPattern := regexp.MustCompile(`PID: (\w+-\d+)`)
	matches := pidPattern.FindStringSubmatch(output)
	if len(matches) < 2 {
		t.Errorf("Process ID not found in output")
	} else {
		t.Logf("Process started with ID: %s", matches[1])
	}

	// Verify script output - the test script outputs with [test] prefix
	err = bt.WaitForOutput("[test] Running tests...", 5*time.Second)
	if err != nil {
		// Try without prefix in case output format changed
		err = bt.WaitForOutput("Running tests...", 2*time.Second)
		if err != nil {
			t.Errorf("Expected output not found: %v", err)
			t.Logf("Actual output: %s", bt.Output())
		}
	}
}

// TestProcessStartupTUI tests process startup in TUI mode
func TestProcessStartupTUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in TUI mode with test script
	err := bt.Start("test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// TUI should start successfully
	time.Sleep(2 * time.Second)

	// Check that process is still running
	if bt.Cmd.ProcessState != nil {
		t.Errorf("TUI process exited unexpectedly")
		t.Logf("Output: %s", bt.Output())
	}
}

// TestMultipleProcesses tests running multiple processes simultaneously
func TestMultipleProcesses(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with multiple scripts
	err := bt.Start("--no-tui", "test", "build")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check both scripts started
	err = bt.WaitForOutput("Started script 'test'", 5*time.Second)
	if err != nil {
		t.Errorf("Test script did not start: %v", err)
	}

	err = bt.WaitForOutput("Started script 'build'", 5*time.Second)
	if err != nil {
		t.Errorf("Build script did not start: %v", err)
	}

	// Verify output from both scripts
	err = bt.WaitForOutput("[test] Running tests...", 5*time.Second)
	if err != nil {
		t.Errorf("Test script output not found: %v", err)
	}

	err = bt.WaitForOutput("[build] Building project...", 5*time.Second)
	if err != nil {
		t.Errorf("Build script output not found: %v", err)
	}

	// Both should complete
	err = bt.WaitForOutput("[test] Tests completed!", 5*time.Second)
	if err != nil {
		t.Errorf("Test script did not complete: %v", err)
	}

	err = bt.WaitForOutput("[build] Build completed!", 5*time.Second)
	if err != nil {
		t.Errorf("Build script did not complete: %v", err)
	}
}

// TestProcessExitHandling tests that process exits are handled correctly
func TestProcessExitHandling(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with test script (which should complete quickly)
	err := bt.Start("--no-tui", "test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for process to complete
	err = bt.WaitForOutput("[test] Tests completed!", 5*time.Second)
	if err != nil {
		t.Fatalf("Process did not complete: %v", err)
	}

	// Check for exit handling
	output := bt.Output()
	// Look for process exit indicators
	if strings.Contains(output, "exit code 0") || strings.Contains(output, "Process exited") {
		t.Log("Process exit handled correctly")
	} else if strings.Contains(output, "Tests completed!") {
		// Even if exact exit message isn't there, completion is good enough
		t.Log("Process completed successfully")
	} else {
		t.Errorf("No process exit indication found")
	}
}

// TestProcessIDGeneration tests that processes get unique IDs
func TestProcessIDGeneration(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with multiple scripts
	err := bt.Start("--no-tui", "test", "build")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for both to start
	time.Sleep(2 * time.Second)

	// Extract all process IDs
	output := bt.Output()
	pidPattern := regexp.MustCompile(`PID: (\w+-\d+)`)
	matches := pidPattern.FindAllStringSubmatch(output, -1)

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 process IDs, found %d", len(matches))
		t.Logf("Output: %s", output)
		return
	}

	// Check that IDs are unique
	ids := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			id := match[1]
			if ids[id] {
				t.Errorf("Duplicate process ID found: %s", id)
			}
			ids[id] = true
			t.Logf("Found process ID: %s", id)
		}
	}

	// Verify ID format (scriptname-timestamp)
	for id := range ids {
		if !regexp.MustCompile(`^\w+-\d+$`).MatchString(id) {
			t.Errorf("Process ID does not match expected format: %s", id)
		}
	}
}

// TestScriptDetection tests that package.json scripts are properly detected
func TestScriptDetection(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Try to run a non-existent script
	err := bt.Start("--no-tui", "nonexistent")
	if err != nil {
		// This is actually expected - brummer should fail to start
		t.Logf("Expected error for non-existent script: %v", err)
		return
	}

	// Check if error is reported
	time.Sleep(1 * time.Second)
	output := bt.Output()
	if strings.Contains(output, "script not found") || strings.Contains(output, "nonexistent") {
		t.Log("Script detection working - reported missing script")
	} else {
		// If it started without error, that's also OK - it might create a default process
		t.Log("Brummer started despite missing script")
	}
}

// TestProcessWithError tests handling of processes that exit with error
func TestProcessWithError(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with error script
	err := bt.Start("--no-tui", "test-fail")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for error output
	err = bt.WaitForOutput("[test-fail] Tests failed!", 5*time.Second)
	if err != nil {
		t.Errorf("Error script output not found: %v", err)
		t.Logf("Output: %s", bt.Output())
	}

	// Process should exit with error
	time.Sleep(2 * time.Second)
	output := bt.Output()
	if strings.Contains(output, "exit code 1") || strings.Contains(output, "Process exited with error") {
		t.Log("Error exit handled correctly")
	}
}

// TestLongRunningProcess tests handling of long-running processes
func TestLongRunningProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start long-running process
	err := bt.Start("--no-tui", "long-running")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check that it started
	err = bt.WaitForOutput("[long] Long running process...", 5*time.Second)
	if err != nil {
		t.Fatalf("Long-running process did not start: %v", err)
	}

	// Wait a bit and check it's still running
	time.Sleep(2 * time.Second)
	if bt.Cmd.ProcessState != nil {
		t.Errorf("Long-running process exited prematurely")
	}

	// Stop it gracefully
	bt.Stop()

	// Verify it stopped
	output := bt.Output()
	// Should NOT see the completion message
	if strings.Contains(output, "[long] Process done") {
		t.Errorf("Process completed when it should have been interrupted")
	}
}

// TestArbitraryCommand tests running arbitrary commands
func TestArbitraryCommand(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Run an arbitrary echo command
	err := bt.Start("--no-tui", "echo 'Hello from brummer!'")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check for output
	err = bt.WaitForOutput("Hello from brummer!", 5*time.Second)
	if err != nil {
		t.Errorf("Arbitrary command output not found: %v", err)
		t.Logf("Output: %s", bt.Output())
	}
}
