// +build integration

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/test/testutil"
)

// TestFullStackNoTUI tests all components working together in headless mode
func TestFullStackNoTUI(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with all features enabled
	err := bt.Start("--no-tui", "--debug", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for components to start
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check MCP server
	t.Run("MCP Server", func(t *testing.T) {
		mcpPort, err := bt.WaitForMCP(5 * time.Second)
		if err != nil {
			t.Errorf("MCP server did not start: %v", err)
			t.Logf("Output: %s", bt.Output())
			return
		}
		t.Logf("MCP server started on port %d", mcpPort)
	})

	// Check Proxy server
	t.Run("Proxy Server", func(t *testing.T) {
		proxyPort, err := bt.WaitForProxy(5 * time.Second)
		if err != nil {
			t.Errorf("Proxy server did not start: %v", err)
			t.Logf("Output: %s", bt.Output())
			return
		}
		t.Logf("Proxy server started on port %d", proxyPort)
	})

	// Check Process management
	t.Run("Process Management", func(t *testing.T) {
		err := bt.WaitForOutput("Started script 'dev'", 5*time.Second)
		if err != nil {
			t.Errorf("Process did not start: %v", err)
			return
		}

		err = bt.WaitForOutput("Dev server running", 5*time.Second)
		if err != nil {
			t.Errorf("Dev server output not found: %v", err)
			return
		}
	})

	// Check Logging
	t.Run("Logging System", func(t *testing.T) {
		output := bt.Output()
		testutil.AssertContains(t, output, "🚀 Brummer started")
		testutil.AssertContains(t, output, "[") // Timestamp check
	})

	// Give it time to run
	select {
	case <-ctx.Done():
		t.Logf("Test completed with timeout")
	case <-time.After(2 * time.Second):
		t.Logf("Test completed successfully")
	}
}

// TestFullStackTUI tests all components working together in TUI mode
func TestFullStackTUI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TUI test in short mode")
	}

	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer in TUI mode
	err := bt.Start("--debug", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// TUI mode should start successfully
	// We can't interact with TUI directly, but we can verify it starts
	time.Sleep(2 * time.Second)

	// Check that process is still running
	if bt.Cmd.ProcessState != nil {
		t.Errorf("TUI process exited unexpectedly")
		t.Logf("Output: %s", bt.Output())
	}
}

// TestMCPProxyIntegration tests MCP and Proxy working together
func TestMCPProxyIntegration(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with dev script
	err := bt.Start("--no-tui", "--debug", "dev")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for both MCP and Proxy
	mcpPort, err := bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Fatalf("MCP server did not start: %v", err)
	}

	proxyPort, err := bt.WaitForProxy(5 * time.Second)
	if err != nil {
		t.Fatalf("Proxy server did not start: %v", err)
	}

	t.Logf("MCP on port %d, Proxy on port %d - integration successful", mcpPort, proxyPort)

	// Verify both services are accessible
	err = testutil.WaitForHTTP(fmt.Sprintf("http://localhost:%d/mcp", mcpPort), 2*time.Second)
	if err != nil {
		t.Errorf("MCP server not accessible: %v", err)
	}
}

// TestProcessLoggingIntegration tests Process management and Logging working together
func TestProcessLoggingIntegration(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with test script
	err := bt.Start("--no-tui", "test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Check process startup
	err = bt.WaitForOutput("Started script 'test'", 5*time.Second)
	if err != nil {
		t.Fatalf("Process did not start: %v", err)
	}

	// Check process output is logged
	err = bt.WaitForOutput("Running tests...", 5*time.Second)
	if err != nil {
		t.Errorf("Process output not captured: %v", err)
	}

	// Check process completion
	err = bt.WaitForOutput("Tests completed!", 5*time.Second)
	if err != nil {
		t.Errorf("Process completion not logged: %v", err)
	}

	// Verify output has proper formatting
	output := bt.Output()
	testutil.AssertContains(t, output, "[test]") // Process prefix
	testutil.AssertContains(t, output, "[")      // Timestamp
}

// TestDebugModeIntegration tests debug mode features
func TestDebugModeIntegration(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	defer bt.Cleanup()

	// Start brummer with debug mode
	err := bt.Start("--no-tui", "--debug", "test")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// In debug mode, MCP should be enabled
	_, err = bt.WaitForMCP(5 * time.Second)
	if err != nil {
		t.Errorf("MCP server not started in debug mode: %v", err)
	}

	// Process management should work
	err = bt.WaitForOutput("Started script", 5*time.Second)
	if err != nil {
		t.Errorf("Process management not working in debug mode: %v", err)
	}

	// Timestamps should be present
	output := bt.Output()
	testutil.AssertContains(t, output, "[")
}

// TestMultipleScripts tests running multiple scripts simultaneously
func TestMultipleScripts(t *testing.T) {
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

	// Check output from both scripts
	err = bt.WaitForOutput("Running tests...", 5*time.Second)
	if err != nil {
		t.Errorf("Test script output not found: %v", err)
	}

	err = bt.WaitForOutput("Building project...", 5*time.Second)
	if err != nil {
		t.Errorf("Build script output not found: %v", err)
	}
}

// TestCleanupShutdown tests graceful shutdown
func TestCleanupShutdown(t *testing.T) {
	bt := testutil.NewBrummerTest(t)
	// Don't use defer here, we'll stop manually

	// Start brummer
	err := bt.Start("--no-tui", "long-running")
	if err != nil {
		t.Fatalf("failed to start brummer: %v", err)
	}

	// Wait for process to start
	err = bt.WaitForOutput("Long running process...", 5*time.Second)
	if err != nil {
		t.Fatalf("Process did not start: %v", err)
	}

	// Stop gracefully
	bt.Stop()

	// Check that shutdown was clean
	output := bt.Output()
	// We should not see the "Process done" message since we stopped early
	testutil.AssertNotContains(t, output, "Process done")
}

// TestErrorHandling tests error script handling
func TestErrorHandling(t *testing.T) {
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
		t.Errorf("Error output not captured: %v", err)
		t.Logf("Output: %s", bt.Output())
	}

	// Process should exit
	time.Sleep(2 * time.Second)
	if bt.Cmd.ProcessState == nil || bt.Cmd.ProcessState.Success() {
		t.Errorf("Expected process to exit with error")
	}
}