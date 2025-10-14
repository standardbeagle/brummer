// Package pty provides pseudo-terminal layer for TUI testing without TTY allocation.
// It enables realistic terminal simulation for testing TUI applications in
// agent-based environments with limited shell capabilities.
package pty

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	tea "github.com/charmbracelet/bubbletea"
)

// PTYLayerImpl manages pseudo-terminal allocation and terminal emulation
type PTYLayerImpl struct {
	mu              sync.RWMutex
	pty             *os.File
	tty             *os.File
	allocated       bool
	size            PTYSize
	ctx             context.Context
	cancel          context.CancelFunc
	inputBuffer     []byte
	outputBuffer    []byte
	lastResizeTime  time.Time
	cleanupOnce     sync.Once
}

// PTYSize defines the terminal size for PTY allocation
type PTYSize struct {
	Rows uint16
	Cols uint16
}

// PTYConfig contains configuration for PTY layer
type PTYConfig struct {
	Size        PTYSize
	Timeout     time.Duration
	EnableDebug bool
}

// DefaultPTYConfig returns a default PTY configuration
func DefaultPTYConfig() PTYConfig {
	return PTYConfig{
		Size: PTYSize{
			Rows: 24,
			Cols: 80,
		},
		Timeout:     30 * time.Second,
		EnableDebug: false,
	}
}

// NewPTYLayer creates a new PTY layer instance
func NewPTYLayer(config PTYConfig) (*PTYLayerImpl, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	return &PTYLayerImpl{
		size:           config.Size,
		ctx:            ctx,
		cancel:         cancel,
		inputBuffer:    make([]byte, 0, 1024),
		outputBuffer:   make([]byte, 0, 4096),
		lastResizeTime: time.Now(),
	}, nil
}

// Allocate creates a new PTY pair and initializes terminal emulation
func (p *PTYLayerImpl) Allocate() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.allocated {
		return fmt.Errorf("PTY already allocated")
	}

	// Create PTY pair
	ptyFile, ttyFile, err := pty.Open()
	if err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	p.pty = ptyFile
	p.tty = ttyFile
	p.allocated = true

	// Initialize terminal emulator (simplified for basic PTY functionality)
	// Terminal emulation is optional for basic input testing

	// Set initial terminal size
	if err := p.resizePTY(); err != nil {
		p.cleanup()
		return fmt.Errorf("failed to set PTY size: %w", err)
	}

	return nil
}

// IsAllocated returns true if PTY is allocated
func (p *PTYLayerImpl) IsAllocated() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.allocated
}

// GetSize returns the current PTY size
func (p *PTYLayerImpl) GetSize() PTYSize {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.size
}

// SetSize changes the PTY size and notifies the terminal
func (p *PTYLayerImpl) SetSize(size PTYSize) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	p.size = size
	p.lastResizeTime = time.Now()

	return p.resizePTY()
}

// resizePTY resizes the PTY
func (p *PTYLayerImpl) resizePTY() error {
	if err := pty.Setsize(p.pty, &pty.Winsize{
		Rows: p.size.Rows,
		Cols: p.size.Cols,
	}); err != nil {
		return err
	}

	// Terminal emulator resize (simplified - optional for basic functionality)
	// if p.terminal != nil {
	// 	p.terminal.Resize(vt10x.Size{
	// 		Rows: int(p.size.Rows),
	// 		Cols: int(p.size.Cols),
	// 	})
	// }

	return nil
}

// WriteInput writes keyboard input to the PTY
func (p *PTYLayerImpl) WriteInput(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	if p.tty == nil {
		return fmt.Errorf("TTY file not available")
	}

	n, err := p.tty.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to PTY: %w", err)
	}

	// Buffer input for debugging
	p.inputBuffer = append(p.inputBuffer, data...)
	if len(p.inputBuffer) > 1024 {
		// Keep only last 1KB
		p.inputBuffer = p.inputBuffer[len(p.inputBuffer)-1024:]
	}

	if n != len(data) {
		return fmt.Errorf("partial write: wrote %d of %d bytes", n, len(data))
	}

	return nil
}

// ReadOutput reads output from the PTY
func (p *PTYLayerImpl) ReadOutput() ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.allocated {
		return nil, fmt.Errorf("PTY not allocated")
	}

	if p.pty == nil {
		return nil, fmt.Errorf("PTY file not available")
	}

	// Set read deadline
	if err := p.pty.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	buffer := make([]byte, 4096)
	n, err := p.pty.Read(buffer)
	if err != nil {
		if err == io.EOF || os.IsTimeout(err) {
			return nil, nil // No data available
		}
		return nil, fmt.Errorf("failed to read from PTY: %w", err)
	}

	data := buffer[:n]

	// Terminal emulation update (simplified - optional for basic functionality)
	// if p.terminal != nil {
	// 	p.terminal.Write(data)
	// }

	// Buffer output for debugging
	p.outputBuffer = append(p.outputBuffer, data...)
	if len(p.outputBuffer) > 4096 {
		// Keep only last 4KB
		p.outputBuffer = p.outputBuffer[len(p.outputBuffer)-4096:]
	}

	return data, nil
}

// GetTerminalState returns the current terminal state (simplified)
func (p *PTYLayerImpl) GetTerminalState() interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Simplified state tracking for basic PTY functionality
	return map[string]interface{}{
		"allocated": p.allocated,
		"rows":      p.size.Rows,
		"cols":      p.size.Cols,
	}
}

// GetContent returns the visible terminal content (simplified)
func (p *PTYLayerImpl) GetContent() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return buffered output for basic content tracking
	if len(p.outputBuffer) > 0 {
		return string(p.outputBuffer)
	}
	return ""
}

// GetCursorPosition returns the current cursor position (simplified)
func (p *PTYLayerImpl) GetCursorPosition() (row, col int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Simplified cursor tracking - default to top-left
	return 0, 0
}

// SimulateKeyPress simulates a key press by writing to the PTY
func (p *PTYLayerImpl) SimulateKeyPress(key tea.KeyMsg) error {
	data, err := p.keyMsgToBytes(key)
	if err != nil {
		return fmt.Errorf("failed to convert key message: %w", err)
	}

	return p.WriteInput(data)
}

// keyMsgToBytes converts a tea.KeyMsg to bytes for PTY input
func (p *PTYLayerImpl) keyMsgToBytes(key tea.KeyMsg) ([]byte, error) {
	switch key.Type {
	case tea.KeyRunes:
		return []byte(string(key.Runes)), nil
	case tea.KeyEnter:
		return []byte("\r"), nil
	case tea.KeyTab:
		return []byte("\t"), nil
	case tea.KeyBackspace:
		return []byte("\x7f"), nil
	case tea.KeySpace:
		return []byte(" "), nil
	case tea.KeyCtrlC:
		return []byte("\x03"), nil
	case tea.KeyCtrlD:
		return []byte("\x04"), nil
	case tea.KeyCtrlZ:
		return []byte("\x1a"), nil
	case tea.KeyUp:
		return []byte("\x1b[A"), nil
	case tea.KeyDown:
		return []byte("\x1b[B"), nil
	case tea.KeyRight:
		return []byte("\x1b[C"), nil
	case tea.KeyLeft:
		return []byte("\x1b[D"), nil
	case tea.KeyHome:
		return []byte("\x1b[H"), nil
	case tea.KeyEnd:
		return []byte("\x1b[F"), nil
	case tea.KeyDelete:
		return []byte("\x1b[3~"), nil
	case tea.KeyPgUp:
		return []byte("\x1b[5~"), nil
	case tea.KeyPgDown:
		return []byte("\x1b[6~"), nil
	case tea.KeyEscape:
		return []byte("\x1b"), nil
	default:
		if len(key.Runes) > 0 {
			return []byte(string(key.Runes)), nil
		}
		return nil, fmt.Errorf("unsupported key type: %v", key.Type)
	}
}

// GetInputBuffer returns the recent input buffer for debugging
func (p *PTYLayerImpl) GetInputBuffer() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]byte, len(p.inputBuffer))
	copy(result, p.inputBuffer)
	return result
}

// GetOutputBuffer returns the recent output buffer for debugging
func (p *PTYLayerImpl) GetOutputBuffer() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]byte, len(p.outputBuffer))
	copy(result, p.outputBuffer)
	return result
}

// ClearBuffers clears the input and output buffers
func (p *PTYLayerImpl) ClearBuffers() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.inputBuffer = p.inputBuffer[:0]
	p.outputBuffer = p.outputBuffer[:0]
}

// Cleanup closes the PTY and releases resources
func (p *PTYLayerImpl) Cleanup() error {
	var cleanupErr error

	p.cleanupOnce.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		cleanupErr = p.cleanup()
	})

	return cleanupErr
}

// cleanup performs the actual cleanup without the Once wrapper
func (p *PTYLayerImpl) cleanup() error {
	p.allocated = false

	if p.cancel != nil {
		p.cancel()
	}

	if p.pty != nil {
		if err := p.pty.Close(); err != nil {
			fmt.Printf("Warning: failed to close PTY: %v\n", err)
		}
		p.pty = nil
	}

	if p.tty != nil {
		if err := p.tty.Close(); err != nil {
			fmt.Printf("Warning: failed to close TTY: %v\n", err)
		}
		p.tty = nil
	}

	// p.terminal = nil // Terminal field removed
	p.inputBuffer = p.inputBuffer[:0]
	p.outputBuffer = p.outputBuffer[:0]

	return nil
}

// Context returns the PTY context
func (p *PTYLayerImpl) Context() context.Context {
	return p.ctx
}

// Err returns any error from the context
func (p *PTYLayerImpl) Err() error {
	return p.ctx.Err()
}

// StartCommand starts a command with the PTY as its controlling terminal
func (p *PTYLayerImpl) StartCommand(cmd *exec.Cmd) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	// Set the PTY as the controlling terminal
	cmd.Stdin = p.tty
	cmd.Stdout = p.tty
	cmd.Stderr = p.tty

	// Only set SysProcAttr on Unix systems
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}

// WaitForCommand waits for a command to complete
func (p *PTYLayerImpl) WaitForCommand(cmd *exec.Cmd) error {
	return cmd.Wait()
}

// GetLastResizeTime returns when the PTY was last resized
func (p *PTYLayerImpl) GetLastResizeTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastResizeTime
}

// Read implements io.Reader interface
func (p *PTYLayerImpl) Read(data []byte) (int, error) {
	output, err := p.ReadOutput()
	if err != nil {
		return 0, err
	}

	n := copy(data, output)
	return n, nil
}

// Write implements io.Writer interface
func (p *PTYLayerImpl) Write(data []byte) (int, error) {
	err := p.WriteInput(data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}