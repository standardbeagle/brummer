//go:build windows

package aicoder

import (
	"os"
)

// terminalState stores the original terminal configuration for restoration
// On Windows, this is a no-op placeholder
type terminalState struct {
	saved bool
}

// configurePTYForResponsiveness is a no-op on Windows
// Windows PTY behavior is handled by the creack/pty library
func configurePTYForResponsiveness(ptmx *os.File) (*terminalState, error) {
	// On Windows, PTY responsiveness is handled by the underlying
	// Windows Console API through the creack/pty library.
	// No manual configuration is needed or possible.
	return &terminalState{saved: false}, nil
}

// restoreTerminalState is a no-op on Windows
func restoreTerminalState(state *terminalState) error {
	// No terminal state to restore on Windows
	return nil
}
