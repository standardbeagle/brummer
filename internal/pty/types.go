// Package pty provides pseudo-terminal functionality for TUI applications.
// This package abstracts PTY operations across Unix and Windows platforms.
package pty

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PTYSize represents the dimensions of a pseudo-terminal
type PTYSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// PTYLayer defines the interface for pseudo-terminal layer functionality
type PTYLayer interface {
	// PTY allocation and management
	Allocate() error
	IsAllocated() bool
	GetSize() PTYSize
	SetSize(size PTYSize) error
	Cleanup() error

	// PTY input/output
	WriteInput(data []byte) error
	ReadOutput() ([]byte, error)
	SimulateKeyPress(key tea.KeyMsg) error

	// PTY state management
	GetContent() string
	GetCursorPosition() (row, col int)
	GetInputBuffer() []byte
	GetOutputBuffer() []byte
	ClearBuffers()

	// PTY context management
	Context() context.Context
	Err() error

	// io.Reader and io.Writer interface implementation
	Read(data []byte) (int, error)
	Write(data []byte) (int, error)
}

// PTYDescriptor represents a PTY file descriptor
type PTYDescriptor struct {
	ID       string    `json:"id"`
	PID      int       `json:"pid"`
	MasterFD int       `json:"master_fd"`
	SlaveFD  int       `json:"slave_fd"`
	Created  time.Time `json:"created"`
	Size     PTYSize   `json:"size"`
}

// PTYStatus represents the status of a PTY
type PTYStatus string

const (
	PTYStatusUnallocated PTYStatus = "unallocated"
	PTYStatusAllocated   PTYStatus = "allocated"
	PTYStatusActive      PTYStatus = "active"
	PTYStatusError       PTYStatus = "error"
	PTYStatusReleased    PTYStatus = "released"
)

// BridgeConfig represents configuration for PTY-to-TTY bridge
type BridgeConfig struct {
	TermType      string        `json:"term_type"`
	InputEncoding string        `json:"input_encoding"`
	OutputFormat  string        `json:"output_format"`
	Buffersize    int           `json:"buffer_size"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// BridgeStatus represents the status of PTY-to-TTY bridge
type BridgeStatus struct {
	IsActive       bool          `json:"is_active"`
	InputBuffer    int           `json:"input_buffer"`
	OutputBuffer   int           `json:"output_buffer"`
	LastActivity   time.Time     `json:"last_activity"`
	BytesProcessed int64         `json:"bytes_processed"`
	Errors         []BridgeError `json:"errors"`
}

// BridgeError represents an error in the PTY bridge
type BridgeError struct {
	Timestamp   time.Time `json:"timestamp"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Recoverable bool      `json:"recoverable"`
}

// PTYDiagnostic represents diagnostic information about PTY layer
type PTYDiagnostic struct {
	PTYID     string                 `json:"pty_id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Metadata  map[string]interface{} `json:"metadata"`
}
