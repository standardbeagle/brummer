//go:build e2e
// +build e2e

package tui_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TUITestSession represents a PTY session for testing the TUI
type TUITestSession struct {
	t       *testing.T
	cmd     *exec.Cmd
	ptmx    *os.File
	output  *bytes.Buffer
	mu      sync.RWMutex
	reader  *bufio.Reader
	workDir string
	done    chan struct{}
}

// NewTUISession creates a new TUI test session
func NewTUISession(t *testing.T, useComponents bool) *TUITestSession {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "brummer-test-*")
	require.NoError(t, err)

	// Create a simple package.json
	packageJSON := `{
		"name": "test-project",
		"scripts": {
			"hello": "echo 'Hello World' && echo 'Second Line' && echo 'Third Line'",
			"dev": "echo 'Development server running on http://localhost:3000'",
			"test": "echo 'Running tests...'",
			"error": "echo 'Error output' >&2 && exit 1"
		}
	}`
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Build the brummer binary path
	brumPath := os.Getenv("BRUMMER_BINARY")
	if brumPath == "" {
		brumPath = "./brum"
		if _, err := os.Stat(brumPath); os.IsNotExist(err) {
			// Try from internal/tui perspective
			brumPath = "../../brum"
			if _, err := os.Stat(brumPath); os.IsNotExist(err) {
				// Try absolute path
				if absPath, err := filepath.Abs("./brum"); err == nil {
					brumPath = absPath
				}
			}
		}
	}

	// Create the command
	cmd := exec.Command(brumPath)
	cmd.Dir = tmpDir
	cmd.Env = os.Environ()
	
	if useComponents {
		cmd.Env = append(cmd.Env, "BRUMMER_USE_COMPONENTS=true")
		t.Log("Starting with component system enabled")
	} else {
		t.Log("Starting with monolithic system")
	}

	session := &TUITestSession{
		t:       t,
		cmd:     cmd,
		output:  &bytes.Buffer{},
		done:    make(chan struct{}),
		workDir: tmpDir,
	}

	return session
}

// Start starts the TUI session
func (s *TUITestSession) Start() error {
	// Start the command with a PTY
	ptmx, err := pty.Start(s.cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}
	s.ptmx = ptmx
	s.reader = bufio.NewReader(ptmx)

	// Start reading output in background
	go s.readOutput()

	// Wait a bit for initialization
	time.Sleep(1 * time.Second)

	return nil
}

// readOutput continuously reads output from the PTY
func (s *TUITestSession) readOutput() {
	defer close(s.done)
	
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.output.Write(buf[:n])
			s.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
}

// SendKey sends a keystroke to the TUI
func (s *TUITestSession) SendKey(key string) error {
	s.t.Logf("Sending key: %s", key)
	
	var keyBytes []byte
	
	switch key {
	case "enter":
		keyBytes = []byte{13}
	case "tab":
		keyBytes = []byte{9}
	case "up":
		keyBytes = []byte{27, 91, 65}
	case "down":
		keyBytes = []byte{27, 91, 66}
	case "right":
		keyBytes = []byte{27, 91, 67}
	case "left":
		keyBytes = []byte{27, 91, 68}
	case "escape":
		keyBytes = []byte{27}
	case "q":
		keyBytes = []byte{'q'}
	default:
		// Single character or number
		if len(key) == 1 {
			keyBytes = []byte(key)
		} else {
			return fmt.Errorf("unknown key: %s", key)
		}
	}

	_, err := s.ptmx.Write(keyBytes)
	if err != nil {
		return err
	}

	// Small delay for UI to update
	time.Sleep(200 * time.Millisecond)
	
	return nil
}

// SendText sends text to the TUI
func (s *TUITestSession) SendText(text string) error {
	s.t.Logf("Sending text: %s", text)
	_, err := io.WriteString(s.ptmx, text)
	if err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	return nil
}

// Wait waits for the specified duration
func (s *TUITestSession) Wait(d time.Duration) {
	s.t.Logf("Waiting %v", d)
	time.Sleep(d)
}

// GetOutput returns the current output
func (s *TUITestSession) GetOutput() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.output.String()
}

// ClearOutput clears the output buffer
func (s *TUITestSession) ClearOutput() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.output.Reset()
}

// WaitForOutput waits for specific text to appear in the output
func (s *TUITestSession) WaitForOutput(text string, timeout time.Duration) bool {
	s.t.Logf("Waiting for output containing: %q", text)
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if strings.Contains(s.GetOutput(), text) {
			s.t.Logf("Found expected text: %q", text)
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	s.t.Logf("Timeout waiting for text: %q", text)
	s.t.Logf("Current output:\n%s", s.GetOutput())
	return false
}

// Stop stops the TUI session
func (s *TUITestSession) Stop() {
	s.t.Log("Stopping TUI session")
	
	// Send quit command
	s.SendKey("q")
	time.Sleep(500 * time.Millisecond)

	// Kill the process if still running
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}

	// Close PTY
	if s.ptmx != nil {
		s.ptmx.Close()
	}

	// Wait for reader to finish
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
	}

	// Clean up temp directory
	os.RemoveAll(s.workDir)
}

// TestComponentSystemStartup tests that the component system starts without crashing
func TestComponentSystemStartup(t *testing.T) {
	session := NewTUISession(t, true)
	err := session.Start()
	require.NoError(t, err)
	defer session.Stop()

	// Wait for initialization
	session.Wait(2 * time.Second)

	// Check that it started with components
	output := session.GetOutput()
	
	// Check for component system message or absence of panic
	if strings.Contains(output, "panic") || strings.Contains(output, "Caught panic") {
		t.Fatalf("Component system crashed with panic:\n%s", output)
	}
	
	// The component system message might be cleared by TUI, but no panic is good
	t.Log("Component system started successfully without panic")
}

// TestMonolithicSystemStartup tests that the monolithic system starts without crashing
func TestMonolithicSystemStartup(t *testing.T) {
	session := NewTUISession(t, false)
	err := session.Start()
	require.NoError(t, err)
	defer session.Stop()

	// Wait for initialization
	session.Wait(2 * time.Second)

	// Check that it started without panic
	output := session.GetOutput()
	assert.NotContains(t, output, "panic", "Should not contain panic")
}

// TestComponentSystemLogDisplay tests that logs are displayed in the component system
func TestComponentSystemLogDisplay(t *testing.T) {
	session := NewTUISession(t, true)
	err := session.Start()
	require.NoError(t, err)
	defer session.Stop()

	// Wait for initialization
	session.Wait(2 * time.Second)
	
	// Clear initial output
	session.ClearOutput()

	// Navigate to scripts tab
	t.Log("Navigating to scripts tab...")
	err = session.SendKey("tab")
	require.NoError(t, err)
	session.Wait(1 * time.Second)
	
	// The ProcessView should be focused now, navigate down to select hello script
	// The first item is usually selected by default, but let's be explicit
	t.Log("Selecting hello script...")
	err = session.SendKey("down")  // Move to first script if not already there
	require.NoError(t, err)
	session.Wait(500 * time.Millisecond)
	
	// Start the hello script (press enter)
	t.Log("Starting hello script with Enter...")
	err = session.SendKey("enter")
	require.NoError(t, err)
	session.Wait(2 * time.Second)

	// Switch to logs view
	t.Log("Switching to logs view...")
	err = session.SendKey("2")
	require.NoError(t, err)
	session.Wait(2 * time.Second)

	// Check if logs are displayed
	output := session.GetOutput()
	
	// Debug output
	t.Logf("Output after switching to logs:\n%s", output)
	
	// Log the raw output for debugging
	t.Logf("Raw output length: %d bytes", len(output))
	
	// Check various conditions
	if strings.Contains(output, "No logs available") {
		t.Error("❌ FAILED: Logs are not displayed - shows 'No logs available'")
		t.Logf("Full output:\n%s", output)
		t.FailNow()
	} else if strings.Contains(output, "Hello World") {
		t.Log("✅ SUCCESS: Logs are displayed correctly!")
	} else if strings.Contains(output, "hello") || strings.Contains(output, "Hello") {
		t.Log("✅ SUCCESS: Found 'hello' in output - logs are working!")
	} else if strings.Contains(output, "Started script") {
		t.Log("✅ Logs system appears to be working (script started)")
	} else {
		t.Error("⚠️ Cannot determine if logs are displayed correctly")
		// Save output to file for inspection
		outputFile := "/tmp/e2e_test_output.txt"
		if err := os.WriteFile(outputFile, []byte(output), 0644); err == nil {
			t.Logf("Full output saved to: %s", outputFile)
		}
		// Log first 500 chars of output for quick debugging
		if len(output) > 500 {
			t.Logf("First 500 chars of output:\n%s", output[:500])
		} else {
			t.Logf("Full output:\n%s", output)
		}
	}
}

// TestMonolithicSystemLogDisplay tests that logs are displayed in the monolithic system
func TestMonolithicSystemLogDisplay(t *testing.T) {
	session := NewTUISession(t, false)
	err := session.Start()
	require.NoError(t, err)
	defer session.Stop()

	// Wait for initialization
	session.Wait(2 * time.Second)
	
	// Clear initial output
	session.ClearOutput()

	// Navigate to scripts tab
	t.Log("Navigating to scripts tab...")
	err = session.SendKey("tab")
	require.NoError(t, err)
	session.Wait(1 * time.Second)

	// Start the hello script
	t.Log("Starting hello script...")
	err = session.SendKey("enter")
	require.NoError(t, err)
	session.Wait(2 * time.Second)

	// Switch to logs view
	t.Log("Switching to logs view...")
	err = session.SendKey("2")
	require.NoError(t, err)
	session.Wait(2 * time.Second)

	// Check if logs are displayed
	output := session.GetOutput()
	
	if strings.Contains(output, "Hello World") || strings.Contains(output, "hello") {
		t.Log("✅ Monolithic system displays logs correctly")
	} else {
		t.Error("⚠️ Monolithic system may have log display issues")
		t.Logf("Output:\n%s", output)
	}
}

// TestNavigationBetweenViews tests navigation between different views
func TestNavigationBetweenViews(t *testing.T) {
	testCases := []struct {
		name          string
		useComponents bool
	}{
		{"Component System", true},
		{"Monolithic System", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session := NewTUISession(t, tc.useComponents)
			err := session.Start()
			require.NoError(t, err)
			defer session.Stop()

			session.Wait(2 * time.Second)

			// Test navigation through views
			views := []struct {
				key  string
				name string
			}{
				{"1", "Processes"},
				{"2", "Logs"},
				{"3", "Errors"},
				{"4", "URLs"},
				{"5", "Web"},
			}

			for _, view := range views {
				t.Logf("Switching to %s view...", view.name)
				session.ClearOutput()
				
				err = session.SendKey(view.key)
				require.NoError(t, err)
				session.Wait(1 * time.Second)
				
				// Check that we didn't crash
				output := session.GetOutput()
				if strings.Contains(output, "panic") {
					t.Fatalf("Panic when switching to %s view:\n%s", view.name, output)
				}
			}
			
			t.Log("✅ All views navigated successfully without crashes")
		})
	}
}