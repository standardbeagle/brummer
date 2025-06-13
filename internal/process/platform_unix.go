//go:build !windows

package process

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// setupProcessGroup sets up process group for Unix systems
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree kills a process and all its children on Unix
func killProcessTree(pid int) {
	// Kill the entire process group
	_ = syscall.Kill(-pid, syscall.SIGTERM) // Ignore errors during cleanup
	time.Sleep(50 * time.Millisecond)
	_ = syscall.Kill(-pid, syscall.SIGKILL) // Ignore errors during cleanup
}

// killProcessByPID kills a single process on Unix
func killProcessByPID(pid int) {
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM) // Ignore errors during cleanup
		time.Sleep(50 * time.Millisecond)
		_ = proc.Kill() // Ignore errors during cleanup
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
