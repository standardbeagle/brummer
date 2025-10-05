//go:build tty && integration
// +build tty,integration

package tty

import (
	"testing"
	"time"
)

// TestVerifySetup verifies that the TTY testing infrastructure works
func TestVerifySetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTY test in short mode")
	}

	tt := NewTTYTest(t)
	defer tt.Cleanup()

	// Test that we can start brummer in headless mode (no TTY required)
	err := tt.Start("--no-tui", "--debug")
	if err != nil {
		t.Fatalf("failed to start brummer in headless mode: %v", err)
	}

	// Wait for startup
	err = tt.WaitForOutput("🚀 Brummer started", 5*time.Second)
	if err != nil {
		t.Fatalf("brummer did not start properly: %v", err)
	}

	// Should see MCP server starting
	err = tt.WaitForOutput("MCP server started", 5*time.Second)
	if err != nil {
		t.Logf("MCP server may not have started (this is OK for setup verification)")
	}

	// Verify we can stop gracefully
	tt.Stop()

	// Verify process stopped
	if tt.IsRunning() {
		t.Errorf("Brummer should have stopped after Stop()")
	}

	t.Log("✅ TTY testing infrastructure verified successfully")
}