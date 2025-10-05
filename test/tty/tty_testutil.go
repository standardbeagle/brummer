//go:build tty && integration
// +build tty,integration

package tty

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/test/testutil"
)

// TTYTest provides a test harness for TTY-dependent brummer tests
type TTYTest struct {
	t          *testing.T
	binaryPath string
	workDir    string
	stdin      io.WriteCloser
	ctx        context.Context
	cancel     context.CancelFunc
	cmd        *exec.Cmd
	output     strings.Builder
}

// NewTTYTest creates a new TTY test harness
func NewTTYTest(t *testing.T) *TTYTest {
	t.Helper()

	// Find brummer binary
	binaryPath := os.Getenv("BRUMMER_BINARY")
	if binaryPath == "" {
		// Try to find it in project root
		projectRoot, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), "../.."))
		if err != nil {
			t.Fatal(err)
		}
		binaryPath = filepath.Join(projectRoot, "brum")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Fatalf("brummer binary not found at %s. Set BRUMMER_BINARY env var or build with 'make build'", binaryPath)
		}
	}

	// Create temp work directory
	workDir := t.TempDir()

	// Find fixtures directory
	projectRoot := filepath.Dir(binaryPath)
	fixturesDir := filepath.Join(projectRoot, "test", "fixtures")
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Fatalf("fixtures directory not found at %s", fixturesDir)
	}

	// Copy fixtures
	if err := testutil.CopyFixtures(fixturesDir, workDir); err != nil {
		t.Fatalf("failed to copy fixtures from %s: %v", fixturesDir, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TTYTest{
		t:          t,
		binaryPath: binaryPath,
		workDir:    workDir,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// StartTTY runs brummer with TTY support using the script command
func (tt *TTYTest) StartTTY(args ...string) error {
	tt.t.Helper()

	// Build command
	fullArgs := append([]string{"-d", tt.workDir}, args...)

	// Use script command to provide TTY
	scriptCmd := []string{"script", "-q", "-c", fmt.Sprintf("%s %s", tt.binaryPath, strings.Join(fullArgs, " ")), "/dev/null"}

	tt.cmd = exec.CommandContext(tt.ctx, scriptCmd[0], scriptCmd[1:]...)

	// Set up environment for proper TTY behavior
	tt.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"FORCE_TTY=1",
	)

	// Create pipes for stdin/stdout
	stdin, err := tt.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := tt.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := tt.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start capturing output
	go tt.captureOutput(stdout)
	go tt.captureOutput(stderr)

	// Start command
	if err := tt.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start brummer with TTY: %w", err)
	}

	// Store stdin for later input
	tt.stdin = stdin.(io.WriteCloser)

	// Give it time to start
	time.Sleep(1 * time.Second)

	return nil
}

// Start runs brummer without TTY (for comparison testing)
func (tt *TTYTest) Start(args ...string) error {
	tt.t.Helper()

	// Build command
	fullArgs := append([]string{"-d", tt.workDir}, args...)
	tt.cmd = exec.CommandContext(tt.ctx, tt.binaryPath, fullArgs...)

	// Capture output
	tt.cmd.Stdout = &tt.output
	tt.cmd.Stderr = &tt.output

	// Start command
	if err := tt.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start brummer: %w", err)
	}

	// Give it time to start
	time.Sleep(500 * time.Millisecond)

	return nil
}

// SendInput sends keystrokes to the TTY session
func (tt *TTYTest) SendInput(keys string) error {
	tt.t.Helper()

	if tt.stdin == nil {
		return fmt.Errorf("TTY session not started or no stdin available")
	}

	_, err := io.WriteString(tt.stdin, keys)
	return err
}

// SendKeySequence sends a key sequence with proper timing
func (tt *TTYTest) SendKeySequence(keys []string, interval time.Duration) error {
	tt.t.Helper()

	for _, key := range keys {
		if err := tt.SendInput(key); err != nil {
			return err
		}
		time.Sleep(interval)
	}
	return nil
}

// captureOutput captures output from a pipe
func (tt *TTYTest) captureOutput(pipe io.Reader) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		tt.output.WriteString(scanner.Text())
		tt.output.WriteString("\n")
	}
}

// Stop gracefully stops brummer
func (tt *TTYTest) Stop() {
	tt.t.Helper()

	if tt.cmd != nil && tt.cmd.Process != nil {
		// Try graceful shutdown first
		tt.cmd.Process.Signal(os.Interrupt)

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- tt.cmd.Wait()
		}()

		select {
		case <-done:
			// Graceful shutdown successful
		case <-time.After(3 * time.Second):
			// Force kill
			tt.cancel()
			tt.cmd.Process.Kill()
			<-done
		}
	}

	// Close stdin if it was opened
	if tt.stdin != nil {
		tt.stdin.Close()
	}
}

// Output returns the captured output
func (tt *TTYTest) Output() string {
	return tt.output.String()
}

// WaitForOutput waits for specific output to appear
func (tt *TTYTest) WaitForOutput(substr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if strings.Contains(tt.output.String(), substr) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for output containing %q\nGot: %s", substr, tt.output.String())
}

// Cleanup should be called with defer to ensure proper cleanup
func (tt *TTYTest) Cleanup() {
	tt.Stop()
}

// AssertContains checks if output contains substring
func (tt *TTYTest) AssertContains(substr string) {
	tt.t.Helper()
	if !strings.Contains(tt.output.String(), substr) {
		tt.t.Errorf("expected output to contain %q\nGot: %s", substr, tt.output.String())
	}
}

// AssertNotContains checks if output does not contain substring
func (tt *TTYTest) AssertNotContains(substr string) {
	tt.t.Helper()
	if strings.Contains(tt.output.String(), substr) {
		tt.t.Errorf("expected output to NOT contain %q\nGot: %s", substr, tt.output.String())
	}
}

// NavigateToAICoders sends key sequence to navigate to AI Coders tab
func (tt *TTYTest) NavigateToAICoders() error {
	// Tab 5 times to reach AI Coders (assuming it's the 5th tab)
	return tt.SendKeySequence([]string{"\t", "\t", "\t", "\t", "\t"}, 100*time.Millisecond)
}

// FocusTerminal sends Enter to focus the terminal
func (tt *TTYTest) FocusTerminal() error {
	return tt.SendInput("\n")
}

// UnfocusTerminal sends Ctrl+Q to unfocus the terminal
func (tt *TTYTest) UnfocusTerminal() error {
	return tt.SendInput("\x11") // Ctrl+Q
}

// ExitFullScreen sends ESC to exit full screen mode
func (tt *TTYTest) ExitFullScreen() error {
	return tt.SendInput("\x1b") // ESC
}

// CreateAICoderSession sends AI command to create a new session
func (tt *TTYTest) CreateAICoderSession(provider, task string) error {
	command := "/ai " + provider
	if task != "" {
		command += " " + task
	}
	return tt.SendInput(command + "\n")
}

// TypeInTerminal types text in the focused terminal
func (tt *TTYTest) TypeInTerminal(text string) error {
	return tt.SendInput(text)
}

// GetWorkDir returns the test work directory
func (tt *TTYTest) GetWorkDir() string {
	return tt.workDir
}

// IsRunning returns whether the brummer process is still running
func (tt *TTYTest) IsRunning() bool {
	return tt.cmd != nil && tt.cmd.ProcessState == nil
}