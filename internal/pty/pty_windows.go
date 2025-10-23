//go:build windows

package pty

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PTYLayerImpl is a stub implementation for Windows
// Windows PTY support can be added later using ConPTY or WinPTY
type PTYLayerImpl struct {
	ctx    context.Context
	cancel context.CancelFunc
	size   PTYSize
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

// NewPTYLayer creates a new PTY layer instance (stub for Windows)
func NewPTYLayer(config PTYConfig) (*PTYLayerImpl, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	return &PTYLayerImpl{
		ctx:    ctx,
		cancel: cancel,
		size:   config.Size,
	}, nil
}

// Allocate is not supported on Windows (stub)
func (p *PTYLayerImpl) Allocate() error {
	return fmt.Errorf("PTY allocation not supported on Windows (requires ConPTY or WinPTY)")
}

// IsAllocated returns false on Windows
func (p *PTYLayerImpl) IsAllocated() bool {
	return false
}

// GetSize returns the configured PTY size
func (p *PTYLayerImpl) GetSize() PTYSize {
	return p.size
}

// SetSize updates the size configuration but doesn't affect actual PTY
func (p *PTYLayerImpl) SetSize(size PTYSize) error {
	p.size = size
	return nil
}

// WriteInput is not supported on Windows (stub)
func (p *PTYLayerImpl) WriteInput(data []byte) error {
	return fmt.Errorf("PTY input not supported on Windows")
}

// ReadOutput is not supported on Windows (stub)
func (p *PTYLayerImpl) ReadOutput() ([]byte, error) {
	return nil, fmt.Errorf("PTY output not supported on Windows")
}

// GetContent returns empty string on Windows
func (p *PTYLayerImpl) GetContent() string {
	return ""
}

// GetCursorPosition returns 0,0 on Windows
func (p *PTYLayerImpl) GetCursorPosition() (row, col int) {
	return 0, 0
}

// SimulateKeyPress is not supported on Windows (stub)
func (p *PTYLayerImpl) SimulateKeyPress(key tea.KeyMsg) error {
	return fmt.Errorf("PTY key simulation not supported on Windows")
}

// GetInputBuffer returns empty buffer on Windows
func (p *PTYLayerImpl) GetInputBuffer() []byte {
	return []byte{}
}

// GetOutputBuffer returns empty buffer on Windows
func (p *PTYLayerImpl) GetOutputBuffer() []byte {
	return []byte{}
}

// ClearBuffers is a no-op on Windows
func (p *PTYLayerImpl) ClearBuffers() {
	// No-op
}

// Cleanup releases resources
func (p *PTYLayerImpl) Cleanup() error {
	if p.cancel != nil {
		p.cancel()
	}
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

// StartCommand is not supported on Windows (stub)
func (p *PTYLayerImpl) StartCommand(cmd *exec.Cmd) error {
	return fmt.Errorf("PTY command execution not supported on Windows")
}

// WaitForCommand is not supported on Windows (stub)
func (p *PTYLayerImpl) WaitForCommand(cmd *exec.Cmd) error {
	return fmt.Errorf("PTY command waiting not supported on Windows")
}

// Read implements io.Reader interface (stub)
func (p *PTYLayerImpl) Read(data []byte) (int, error) {
	return 0, fmt.Errorf("PTY read not supported on Windows")
}

// Write implements io.Writer interface (stub)
func (p *PTYLayerImpl) Write(data []byte) (int, error) {
	return 0, fmt.Errorf("PTY write not supported on Windows")
}
