//go:build windows

package process

import (
	"os"
	"os/exec"
)

// setupProcessGroup sets up process group for Windows systems
func setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use process groups in the same way
	// No special setup needed
}

// killProcessTree kills a process and all its children on Windows
func killProcessTree(pid int) {
	// On Windows, try to kill the process directly
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill() // Ignore errors during forced cleanup
	}
}

// killProcessByPID kills a single process on Windows
func killProcessByPID(pid int) {
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill() // Ignore errors during cleanup
	}
}

// ensureProcessDead makes sure a process is really dead on Windows
func ensureProcessDead(pid int) {
	if proc, err := os.FindProcess(pid); err == nil {
		// On Windows, just try to kill it directly
		_ = proc.Kill() // Ignore errors during forced cleanup
	}
}
