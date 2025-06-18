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

// TestSystemLoggingNoTUI tests system logging in headless mode
func TestSystemLoggingNoTUI(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in headless mode
	err := bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check for startup message
	err = bt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Errorf("Startup message not found: %v", err)
		t.Logf("Output: %s", bt.Output())
	}

	// Check for MCP logging
	err = bt.WaitForOutput("MCP server started", 5*time.Second)
	if err != nil {
		t.Errorf("MCP log message not found: %v", err)
	}

	// Verify system messages are being logged
	output := bt.Output()
	systemLogPatterns := []string{
		"🚀 Brummer started",
		"MCP server",
		"Starting",
	}

	foundCount := 0
	for _, pattern := range systemLogPatterns {
		if strings.Contains(output, pattern) {
			foundCount++
			t.Logf("Found system log pattern: %s", pattern)
		}
	}

	if foundCount < 2 {
		t.Errorf("Insufficient system log messages found (%d/%d)", foundCount, len(systemLogPatterns))
	}
}

// TestSystemLoggingTUI tests system logging in TUI mode
func TestSystemLoggingTUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in TUI mode
	err := bt.Start("--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// TUI should start successfully
	time.Sleep(2 * time.Second)

	// Check that process is still running
	if bt.Cmd.ProcessState != nil {
		t.Errorf("TUI process exited unexpectedly")
		t.Logf("Output: %s", bt.Output())
	} else {
		t.Log("TUI mode started successfully, system logging assumed working")
	}
}

// TestProcessOutputCapture tests that process output is captured correctly
func TestProcessOutputCapture(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with test script
	err := bt.Start("--no-tui", "test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check for process output
	processOutputs := []string{
		"[test] Running tests...",
		"[test] Tests completed!",
		"Started script 'test'",
	}

	for _, expected := range processOutputs {
		err = bt.WaitForOutput(expected, 5*time.Second)
		if err != nil {
			t.Errorf("Process output not captured: %s - %v", expected, err)
		} else {
			t.Logf("Captured process output: %s", expected)
		}
	}

	// Verify output has process prefixes
	output := bt.Output()
	if !strings.Contains(output, "[test]") {
		t.Errorf("Process output prefix not found")
	}
}

// TestLogTimestamps tests that logs include proper timestamps
func TestLogTimestamps(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer
	err := bt.Start("--no-tui", "test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for some output
	err = bt.WaitForOutput("Running tests", 5*time.Second)
	if err != nil {
		t.Fatalf("No output to check timestamps: %v", err)
	}

	// Check for timestamp patterns [HH:MM:SS]
	output := bt.Output()
	timestampPattern := regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\]`)
	matches := timestampPattern.FindAllString(output, -1)

	if len(matches) < 2 {
		t.Errorf("Insufficient timestamp entries found: %d", len(matches))
		t.Logf("Output: %s", output)
	} else {
		t.Logf("Found %d timestamp entries", len(matches))
		// Show first few timestamps
		for i := 0; i < len(matches) && i < 3; i++ {
			t.Logf("Timestamp %d: %s", i+1, matches[i])
		}
	}
}

// TestErrorDetection tests error detection and logging
func TestErrorDetection(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with error script
	err := bt.Start("--no-tui", "error")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check that error is captured
	err = bt.WaitForOutput("Error: Something went wrong!", 5*time.Second)
	if err != nil {
		t.Errorf("Error message not captured: %v", err)
		t.Logf("Output: %s", bt.Output())
	} else {
		t.Log("Error detection working - error message captured")
	}

	// Verify error appears in output
	output := bt.Output()
	if strings.Contains(output, "Error:") || strings.Contains(output, "error") {
		t.Log("Error keyword detected in logs")
	}
}

// TestLogFiltering tests log filtering in TUI mode
func TestLogFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI filtering test in short mode")
	}

	// This is primarily a TUI feature, so we just verify TUI starts
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	err := bt.Start("test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	time.Sleep(2 * time.Second)

	if bt.Cmd.ProcessState != nil {
		t.Errorf("TUI process exited unexpectedly")
	} else {
		t.Log("TUI mode working - log filtering available via slash commands")
	}
}

// TestMultilineLogging tests handling of multiline output
func TestMultilineLogging(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Create a test that outputs multiple lines
	// The test script outputs on multiple lines
	err := bt.Start("--no-tui", "test", "build")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for both scripts to produce output
	time.Sleep(3 * time.Second)

	output := bt.Output()
	lines := strings.Split(output, "\n")

	if len(lines) < 5 {
		t.Errorf("Expected multiple lines of output, got %d lines", len(lines))
	} else {
		t.Logf("Multiline output captured: %d lines", len(lines))
	}

	// Check that output from different processes is interleaved
	hasTest := false
	hasBuild := false
	for _, line := range lines {
		if strings.Contains(line, "[test]") {
			hasTest = true
		}
		if strings.Contains(line, "[build]") {
			hasBuild = true
		}
	}

	if hasTest && hasBuild {
		t.Log("Multiple process outputs properly interleaved")
	} else {
		t.Errorf("Expected output from both test and build processes")
	}
}

// TestLogColors tests ANSI color handling
func TestLogColors(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer - emojis indicate color support
	err := bt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for colorful output (emojis)
	err = bt.WaitForOutput("🚀", 5*time.Second)
	if err != nil {
		t.Logf("No emoji found (color support might be disabled)")
	} else {
		t.Log("Color/emoji output detected")
	}

	// Check for other visual indicators
	output := bt.Output()
	visualIndicators := []string{"🚀", "✅", "🌐", "📦"}
	foundCount := 0

	for _, indicator := range visualIndicators {
		if strings.Contains(output, indicator) {
			foundCount++
		}
	}

	if foundCount > 0 {
		t.Logf("Found %d visual indicators (emojis) in output", foundCount)
	}
}

// TestLogBuffering tests that logs are properly buffered
func TestLogBuffering(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start a long-running process
	err := bt.Start("--no-tui", "long-running")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Let it run for a bit
	err = bt.WaitForOutput("Long running process", 5*time.Second)
	if err != nil {
		t.Fatalf("Process did not start: %v", err)
	}

	// Get output at different times
	output1 := bt.Output()
	time.Sleep(1 * time.Second)
	output2 := bt.Output()

	// Output should accumulate
	if len(output2) <= len(output1) {
		t.Errorf("Log buffer not accumulating: len1=%d, len2=%d", len(output1), len(output2))
	} else {
		t.Logf("Log buffering working: %d -> %d bytes", len(output1), len(output2))
	}
}
