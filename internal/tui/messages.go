package tui

import (
	"time"
)

// Message types for the TUI system

// Process-related messages
// ProcessUpdateMsg is now defined in model_messages.go to centralize message types

type processStoppedMsg struct {
	ProcessID string
}

type processStartedMsg struct {
	ProcessID string
	Name      string
}

// Log-related messages
// LogUpdateMsg is now defined in model_messages.go to centralize message types

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

// Note: AI Coder and PTY message types have been moved to messages/aicoder_messages.go
// to consolidate message definitions and avoid duplication


// URL detection messages
type urlDetectedMsg struct {
	ProcessID string
	URL       string
}

// Refresh messages
type refreshMsg struct{}
