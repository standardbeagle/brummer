package tui

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
)

// DirectPTYPassthrough provides zero-copy, lock-free PTY to terminal passthrough
// This bypasses vt10x parsing and BubbleTea updates for maximum performance
type DirectPTYPassthrough struct {
	// Core components (atomic access only)
	session    atomic.Value // *aicoder.PTYSession - atomic for safe concurrent access
	terminalFD atomic.Int64 // Direct terminal file descriptor - atomic for safe concurrent access

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

// getSession safely retrieves the current session with type checking
// Returns nil if session is not available or invalid
func (d *DirectPTYPassthrough) getSession() *aicoder.PTYSession {
	sessionInterface := d.session.Load()
	if sessionInterface == nil {
		return nil
	}

	session, ok := sessionInterface.(*aicoder.PTYSession)
	if !ok || session == nil {
		return nil
	}

	return session
}

// NewDirectPTYPassthrough creates a new direct passthrough system
func NewDirectPTYPassthrough(ctx context.Context, session *aicoder.PTYSession, logStore *logs.Store) *DirectPTYPassthrough {
	passthroughCtx, cancel := context.WithCancel(ctx)

	passthrough := &DirectPTYPassthrough{
		ctx:      passthroughCtx,
		cancel:   cancel,
		logStore: logStore,
	}

	// Store session atomically for safe concurrent access
	passthrough.session.Store(session)

	// Initialize terminal file descriptor atomically (stdout for now)
	passthrough.terminalFD.Store(int64(os.Stdout.Fd()))

	return passthrough
}

// Start begins the direct passthrough operation
func (d *DirectPTYPassthrough) Start() error {
	if !d.active.CompareAndSwap(false, true) {
		return fmt.Errorf("passthrough already active")
	}

	d.logStore.Add("system", "System", "Starting direct PTY passthrough (command-driven)", false)

	// NO MORE GOROUTINES - Passthrough will be command-driven
	// through BubbleTea Update() cycle instead of background goroutine

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

// ProcessPTYOutput creates a command to process PTY passthrough output
// This replaces the goroutine-based passthroughLoop with command-driven processing
func (d *DirectPTYPassthrough) ProcessPTYOutput() tea.Cmd {
	return func() tea.Msg {
		if !d.active.Load() || !d.passthroughMode.Load() {
			return nil // Not active or not in direct mode
		}

		// Get session safely to prevent race conditions
		session := d.getSession()
		if session == nil {
			d.active.Store(false)
			return nil
		}

		// Process output with timeout to prevent blocking
		timeout := time.After(10 * time.Millisecond)

		for {
			select {
			case output, ok := <-session.OutputChan:
				if !ok {
					// Channel closed - signal passthrough ended
					d.active.Store(false)
					return DirectPTYMessage{
						SessionID: session.ID,
						Data:      nil,
						Direct:    false,
					}
				}

				// Direct mode: write raw bytes to terminal and create message
				d.writeDirectToTerminal(output)
				return CreateDirectPTYCmd(session.ID, output, true)()

			case <-timeout:
				// Timeout reached - stop processing to maintain responsiveness
				return nil

			case <-d.ctx.Done():
				// Context cancelled
				d.active.Store(false)
				return nil
			}
		}
	}
}

// writeDirectToTerminal writes raw bytes directly to terminal FD
// This is the fastest possible path - zero parsing, zero copying
func (d *DirectPTYPassthrough) writeDirectToTerminal(data []byte) {
	// Get terminal file descriptor atomically to prevent race conditions
	fd := int(d.terminalFD.Load())
	if fd <= 0 {
		d.logStore.Add("system", "System", "Invalid terminal file descriptor", true)
		return
	}

	// Use syscall.Write for maximum performance
	n, err := syscall.Write(fd, data)
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
// Uses deep copy to prevent data races on shared byte slices
func CreateDirectPTYCmd(sessionID string, data []byte, direct bool) tea.Cmd {
	return func() tea.Msg {
		// Create a deep copy to prevent data races on shared data
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		return DirectPTYMessage{
			SessionID: sessionID,
			Data:      dataCopy, // Use deep copied data
			Direct:    direct,
		}
	}
}
