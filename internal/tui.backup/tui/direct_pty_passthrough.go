package tui

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
)

// DirectPTYPassthrough provides zero-copy, lock-free PTY to terminal passthrough
// This bypasses vt10x parsing and BubbleTea updates for maximum performance
type DirectPTYPassthrough struct {
	// Core components
	session    *aicoder.PTYSession
	terminalFD int // Direct terminal file descriptor

	// Atomic state
	active          atomic.Bool
	passthroughMode atomic.Bool // Toggle between direct and parsed modes

	// Performance counters
	bytesPassedThrough atomic.Uint64
	directWrites       atomic.Uint64

	// Control
	ctx    context.Context
	cancel context.CancelFunc

	// Dependencies
	logStore *logs.Store
}

// DirectPTYMessage represents a message that can be sent directly to terminal
type DirectPTYMessage struct {
	SessionID string
	Data      []byte
	Direct    bool // If true, send directly to terminal without parsing
}

// NewDirectPTYPassthrough creates a new direct passthrough system
func NewDirectPTYPassthrough(ctx context.Context, session *aicoder.PTYSession, logStore *logs.Store) *DirectPTYPassthrough {
	passthroughCtx, cancel := context.WithCancel(ctx)

	passthrough := &DirectPTYPassthrough{
		session:  session,
		ctx:      passthroughCtx,
		cancel:   cancel,
		logStore: logStore,
	}

	// Initialize terminal file descriptor (stdout for now)
	passthrough.terminalFD = int(os.Stdout.Fd())

	return passthrough
}

// Start begins the direct passthrough operation
func (d *DirectPTYPassthrough) Start() error {
	if !d.active.CompareAndSwap(false, true) {
		return fmt.Errorf("passthrough already active")
	}

	d.logStore.Add("system", "System", "Starting direct PTY passthrough", false)

	// Start the passthrough loop
	go d.passthroughLoop()

	return nil
}

// Stop ends the direct passthrough operation
func (d *DirectPTYPassthrough) Stop() {
	if !d.active.CompareAndSwap(true, false) {
		return // Already stopped
	}

	d.cancel()
	d.logStore.Add("system", "System", "Stopped direct PTY passthrough", false)
}

// SetPassthroughMode toggles between direct and parsed modes
func (d *DirectPTYPassthrough) SetPassthroughMode(enabled bool) {
	d.passthroughMode.Store(enabled)
	mode := "parsed"
	if enabled {
		mode = "direct"
	}
	d.logStore.Add("system", "System", fmt.Sprintf("Switched to %s mode", mode), false)
}

// passthroughLoop is the main loop for direct PTY passthrough
func (d *DirectPTYPassthrough) passthroughLoop() {
	defer d.active.Store(false)

	for {
		select {
		case <-d.ctx.Done():
			return

		case output, ok := <-d.session.OutputChan:
			if !ok {
				// Channel closed
				return
			}

			if d.passthroughMode.Load() {
				// Direct mode: write raw bytes to terminal
				d.writeDirectToTerminal(output)
			}
		}
	}
}

// writeDirectToTerminal writes raw bytes directly to terminal FD
// This is the fastest possible path - zero parsing, zero copying
func (d *DirectPTYPassthrough) writeDirectToTerminal(data []byte) {
	// Use syscall.Write for maximum performance
	n, err := syscall.Write(d.terminalFD, data)
	if err != nil {
		d.logStore.Add("system", "System",
			fmt.Sprintf("Direct terminal write error: %v", err), true)
		return
	}

	// Update metrics
	d.bytesPassedThrough.Add(uint64(n))
	d.directWrites.Add(1)
}

// GetMetrics returns performance metrics
func (d *DirectPTYPassthrough) GetMetrics() PassthroughMetrics {
	return PassthroughMetrics{
		Active:             d.active.Load(),
		PassthroughMode:    d.passthroughMode.Load(),
		BytesPassedThrough: d.bytesPassedThrough.Load(),
		DirectWrites:       d.directWrites.Load(),
	}
}

// PassthroughMetrics contains performance statistics
type PassthroughMetrics struct {
	Active             bool
	PassthroughMode    bool
	BytesPassedThrough uint64
	DirectWrites       uint64
}

// CreateDirectPTYCmd creates a BubbleTea command for direct PTY output
func CreateDirectPTYCmd(sessionID string, data []byte, direct bool) tea.Cmd {
	return func() tea.Msg {
		return DirectPTYMessage{
			SessionID: sessionID,
			Data:      make([]byte, len(data)), // Copy data
			Direct:    direct,
		}
	}
}
