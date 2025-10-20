package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TTY helpers for integration testing with terminal allocation

// AllocateTTY creates a pseudo-terminal for testing
func AllocateTTY() (*os.File, error) {
	// Use /dev/tty if available
	if _, err := os.Stat("/dev/tty"); err == nil {
		return os.OpenFile("/dev/tty", os.O_RDWR, 0)
	}

	// Fallback: Create a PTY using os.Open on stdin/stdout/stderr
	// This is a simplified approach for CI environments
	return os.Stdin, nil
}

// CleanupTTY closes the TTY file
func CleanupTTY(tty *os.File) {
	if tty != nil {
		tty.Close()
	}
}

// RunWithTTY executes a command with TTY allocation
func RunWithTTY(cmd *exec.Cmd) error {
	// Set up standard streams
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up a pseudo-terminal if available
	if _, err := os.Stat("/dev/tty"); err == nil {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err == nil {
			defer tty.Close()
			// Try to set up controlling terminal
			if sysProcAttr := cmd.SysProcAttr; sysProcAttr != nil {
				sysProcAttr.Setctty = true
				sysProcAttr.Setsid = true
				cmd.SysProcAttr = sysProcAttr
			}
		}
	}

	return cmd.Run()
}

// RunCommandWithTimeout runs a command with timeout and TTY allocation
func RunCommandWithTimeout(command string, args []string, timeout time.Duration) (string, error) {
	cmd := exec.Command(command, args...)

	// Create pipes for output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Wait for completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Command completed
		if err != nil {
			return "", err
		}
		// Read output
		output := make([]byte, 0)
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		return string(output), nil
	case <-time.After(timeout):
		// Timeout
		cmd.Process.Kill()
		return "", fmt.Errorf("command timed out after %v", timeout)
	}
}

// StartBrummerTUI starts the Brummer TUI for testing
func StartBrummerTUI(binaryPath string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(binaryPath, args...)

	// Set up pseudo-terminal
	if tty, err := AllocateTTY(); err == nil {
		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty
	}

	// Set up process group for cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// StopBrummerTUI stops a running Brummer TUI process
func StopBrummerTUI(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Try graceful shutdown first
	if err := cmd.Process.Signal(syscall.SIGTERM); err == nil {
		// Wait a bit for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			// Force kill if graceful shutdown fails
		}
	}

	// Force kill
	return cmd.Process.Kill()
}

// WithTTY is a test helper that runs a test function with TTY allocation
func WithTTY(t *testing.T, testFunc func(*testing.T, *os.File)) {
	tty, err := AllocateTTY()
	if err != nil {
		t.Skipf("TTY not available: %v", err)
		return
	}
	defer CleanupTTY(tty)

	testFunc(t, tty)
}

// CheckTTYAvailable checks if TTY is available for testing
func CheckTTYAvailable() bool {
	if _, err := os.Stat("/dev/tty"); err != nil {
		return false
	}
	return true
}