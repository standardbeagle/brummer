//go:build windows

package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// setupProcessGroup sets up process group for Windows systems
func setupProcessGroup(cmd *exec.Cmd) {
	// On Windows, we use CREATE_NEW_PROCESS_GROUP to allow proper termination
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}

// killProcessTree kills a process and all its children on Windows
func killProcessTree(pid int) {
	// Use taskkill with /T flag to kill the process tree
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	_ = cmd.Run() // Ignore errors, process might already be dead

	// Give processes a moment to die
	time.Sleep(50 * time.Millisecond)

	// Double-check with direct kill
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill() // Ignore errors during forced cleanup
	}
}

// killProcessByPID kills a single process on Windows
func killProcessByPID(pid int) {
	// First try taskkill for cleaner shutdown
	cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
	_ = cmd.Run() // Ignore errors

	// Fallback to direct kill
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill() // Ignore errors during cleanup
	}
}

// ensureProcessDead makes sure a process is really dead on Windows
func ensureProcessDead(pid int) {
	// Check if process exists using tasklist
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
	output, err := cmd.Output()
	if err != nil {
		return // Process probably doesn't exist
	}

	// If the process is still in the output, kill it
	if strings.Contains(string(output), strconv.Itoa(pid)) {
		killProcessByPID(pid)
	}
}

// findProcessUsingPort finds the process ID using a specific port on Windows
func findProcessUsingPort(port int) (int, error) {
	// Use netstat to find the process using the port
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Look for lines containing the port
	lines := strings.Split(string(output), "\n")
	portStr := fmt.Sprintf(":%d", port)

	for _, line := range lines {
		if strings.Contains(line, portStr) && strings.Contains(line, "LISTENING") {
			// Extract PID from the last column
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				pid, err := strconv.Atoi(fields[len(fields)-1])
				if err == nil && pid > 0 {
					return pid, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("no process found using port %d", port)
}

// killProcessesByName kills all processes matching a name pattern on Windows
func killProcessesByName(pattern string) {
	// Use taskkill with image name filter
	cmd := exec.Command("taskkill", "/F", "/IM", pattern+"*")
	_ = cmd.Run() // Ignore errors, processes might not exist
}
