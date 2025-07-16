//go:build !windows

package process

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// setupProcessGroup sets up process group for Unix systems
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree kills a process and all its children on Unix
func killProcessTree(pid int) {
	// First, try to kill all child processes recursively
	killAllChildren(pid)

	// Kill the entire process group with SIGTERM first
	_ = syscall.Kill(-pid, syscall.SIGTERM) // Ignore errors during cleanup

	// Give processes more time to cleanup gracefully, especially for npm/pnpm
	time.Sleep(200 * time.Millisecond)

	// Try killing individual process in case process group didn't work
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}

	// Give a bit more time for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Now be more aggressive - kill the process group with SIGKILL
	_ = syscall.Kill(-pid, syscall.SIGKILL) // Ignore errors during cleanup

	// Also directly kill the main process
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}

	// Final cleanup: kill any remaining children with SIGKILL (non-blocking)
	go func() {
		time.Sleep(50 * time.Millisecond)
		killAllChildrenForce(pid)
		verifyProcessDead(pid)
	}()
}

// killAllChildren recursively finds and kills all child processes
func killAllChildren(parentPID int) {
	// Use pgrep to find all children of this process
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pgrep", "-P", strconv.Itoa(parentPID))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.TrimSpace(string(output))
	if lines == "" {
		return
	}

	var childPIDs []int
	for _, line := range strings.Split(lines, "\n") {
		if childPID, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			childPIDs = append(childPIDs, childPID)
		}
	}

	// First, recursively kill grandchildren
	for _, childPID := range childPIDs {
		killAllChildren(childPID)
	}

	// Then kill direct children
	for _, childPID := range childPIDs {
		// Try SIGTERM first
		if proc, err := os.FindProcess(childPID); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}

	// Give children time to exit gracefully
	time.Sleep(50 * time.Millisecond)

	// Kill any remaining children with SIGKILL
	for _, childPID := range childPIDs {
		if proc, err := os.FindProcess(childPID); err == nil {
			_ = proc.Kill()
		}
	}
}

// killAllChildrenForce kills all child processes with SIGKILL
func killAllChildrenForce(parentPID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pgrep", "-P", strconv.Itoa(parentPID))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.TrimSpace(string(output))
	if lines == "" {
		return
	}

	for _, line := range strings.Split(lines, "\n") {
		if childPID, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			// Recursively kill grandchildren first
			killAllChildrenForce(childPID)
			// Then force kill this child
			if proc, err := os.FindProcess(childPID); err == nil {
				_ = proc.Kill()
			}
		}
	}
}

// verifyProcessDead checks if a process is actually dead and force kills if needed
func verifyProcessDead(pid int) {
	time.Sleep(100 * time.Millisecond)

	if proc, err := os.FindProcess(pid); err == nil {
		// Check if process still exists by sending signal 0
		if err := proc.Signal(syscall.Signal(0)); err == nil {
			// Process still exists, force kill it one more time
			_ = proc.Kill()

			// Also try to kill the process group
			_ = syscall.Kill(-pid, syscall.SIGKILL)

			// Give it one final moment
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// killProcessByPID kills a single process on Unix
func killProcessByPID(pid int) {
	if proc, err := os.FindProcess(pid); err == nil {
		// First kill any children of this process
		killAllChildren(pid)

		// Try graceful termination first
		_ = proc.Signal(syscall.SIGTERM) // Ignore errors during cleanup
		time.Sleep(50 * time.Millisecond)

		// Check if process still exists
		if err := proc.Signal(syscall.Signal(0)); err == nil {
			// Process still exists, force kill it
			_ = proc.Kill() // Ignore errors during cleanup
		}
	}
}

// ensureProcessDead makes sure a process is really dead on Unix
func ensureProcessDead(pid int) {
	if proc, err := os.FindProcess(pid); err == nil {
		// Try to send signal 0 to check if process exists
		if err := proc.Signal(syscall.Signal(0)); err == nil {
			// Process still exists, kill it
			_ = proc.Kill() // Ignore errors during forced cleanup
		}
	}
}

// findProcessUsingPort finds the process ID using a specific port on Unix (stub)
func findProcessUsingPort(port int) (int, error) {
	// This is handled differently in the manager for Unix
	return 0, nil
}

// killProcessesByName kills all processes matching a name pattern on Unix (stub)
func killProcessesByName(pattern string) {
	// This is handled differently in the manager for Unix
}
