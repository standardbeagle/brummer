//go:build !windows

package aicoder

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// terminalState stores the original terminal configuration for restoration
type terminalState struct {
	fd       int
	original unix.Termios
	saved    bool
}

// configurePTYForResponsiveness configures the PTY for immediate output
// This fixes the 1-second delay using non-canonical mode while preserving output formatting
func configurePTYForResponsiveness(ptmx *os.File) (*terminalState, error) {
	if ptmx == nil {
		return nil, fmt.Errorf("PTY file is nil")
	}

	fd := int(ptmx.Fd())
	if fd < 0 {
		return nil, fmt.Errorf("invalid PTY file descriptor: %d", fd)
	}

	// Check if the file descriptor is still valid
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		return nil, fmt.Errorf("PTY file descriptor is not valid: %w", err)
	}

	// Get current terminal attributes
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal attributes: %w", err)
	}

	// Save original state for restoration
	state := &terminalState{
		fd:       fd,
		original: *termios,
		saved:    true,
	}

	// REFINED fix: Use non-canonical mode instead of disabling OPOST
	// This addresses buffering without breaking output formatting like newlines

	// Input flags: Keep ECHO for character display
	termios.Iflag |= unix.ECHO

	// Local flags: Disable canonical mode (ICANON) for immediate character processing
	// Keep ECHO enabled, disable ECHONL (echo newline) to prevent double newlines
	termios.Lflag &^= unix.ICANON
	termios.Lflag |= unix.ECHO
	termios.Lflag &^= unix.ECHONL

	// Output flags: Keep OPOST enabled to preserve newline conversion and formatting
	// This is critical - OPOST handles \n to \r\n conversion and other formatting
	termios.Oflag |= unix.OPOST

	// Control characters for non-canonical mode:
	// VMIN=1: Return after reading 1 character (immediate response)
	// VTIME=0: No timeout, return immediately when character available
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	// Apply with flush to ensure immediate effect
	if err := unix.IoctlSetTermios(fd, unix.TCSETSF, termios); err != nil {
		return nil, fmt.Errorf("failed to set terminal attributes: %w", err)
	}

	return state, nil
}

// restoreTerminalState restores the original terminal configuration
func restoreTerminalState(state *terminalState) error {
	if state == nil || !state.saved {
		return nil // Nothing to restore
	}

	// Validate file descriptor is still valid
	var stat syscall.Stat_t
	if err := syscall.Fstat(state.fd, &stat); err != nil {
		return nil // File descriptor is no longer valid, nothing to restore
	}

	if err := unix.IoctlSetTermios(state.fd, unix.TCSETSF, &state.original); err != nil {
		return fmt.Errorf("failed to restore terminal attributes: %w", err)
	}

	state.saved = false
	return nil
}
