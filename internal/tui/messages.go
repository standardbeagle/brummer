package tui

import "time"

// Message types for the TUI system

// Process-related messages
// processUpdateMsg already defined in process_view_controller.go

type processStoppedMsg struct {
	ProcessID string
}

type processStartedMsg struct {
	ProcessID string
	Name      string
}

// Log-related messages
// logUpdateMsg already defined in model.go

type logsClearedMsg struct {
	ProcessID string
}

// MCP-related messages
type mcpRequestMsg struct {
	Method    string
	Params    interface{}
	Timestamp time.Time
}

type mcpResponseMsg struct {
	ID        string
	Result    interface{}
	Error     error
	Timestamp time.Time
}

type mcpErrorMsg struct {
	Error     error
	Timestamp time.Time
}

// AI Coder messages
type aiCoderSessionMsg struct {
	SessionID string
	Event     string
	Data      interface{}
}

type aiCoderOutputMsg struct {
	SessionID string
	Output    string
}

type aiCoderErrorMsg struct {
	SessionID string
	Error     error
}

// URL detection messages
type urlDetectedMsg struct {
	ProcessID string
	URL       string
}

// Refresh messages
type refreshMsg struct{}
