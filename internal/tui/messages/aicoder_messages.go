package messages

import (
	"time"
)

// AI Coder Session Messages
// These messages represent state changes and events from the AI coder service

// SessionStatus represents the state of an AI coding session
type SessionStatus string

const (
	SessionPending SessionStatus = "pending" // Created, not started
	SessionActive  SessionStatus = "active"  // Running with PTY
	SessionIdle    SessionStatus = "idle"    // Running but no recent activity
	SessionError   SessionStatus = "error"   // PTY failure or crash
	SessionStopped SessionStatus = "stopped" // Gracefully stopped
)

// Service → Component Messages
// These messages are published by the AI coder service for component consumption

// AICoderSessionCreatedMsg indicates a new AI coding session was created.
type AICoderSessionCreatedMsg struct {
	SessionID string
	Name      string
}

// AICoderSessionUpdatedMsg indicates an AI coding session state changed.
type AICoderSessionUpdatedMsg struct {
	SessionID    string
	Status       SessionStatus
	LastActivity time.Time
}

// AICoderSessionEndedMsg indicates an AI coding session was terminated.
type AICoderSessionEndedMsg struct {
	SessionID string
	Reason    string // "stopped", "crashed", "error"
}

// === NEW BATCH MESSAGES ===

// AICoderBatchUpdateMsg contains debounced PTY output ready for processing.
// This replaces individual AICoderPTYOutputMsg for better performance.
type AICoderBatchUpdateMsg struct {
	SessionID string
	Data      []byte    // Batched data from debouncer
	Timestamp time.Time // When batch was collected
}

// AICoderScreenSyncMsg updates the screen buffer in the model.
// Sent after processing batched PTY output through vt10x.
type AICoderScreenSyncMsg struct {
	SessionID    string
	ScreenBuffer [][]rune        // Rendered terminal screen
	CursorPos    CursorPosition // Cursor position
}

// CursorPosition represents cursor position in screen buffer
type CursorPosition struct {
	X int
	Y int
}

// AICoderSessionSwitchMsg indicates active session is switching.
type AICoderSessionSwitchMsg struct {
	OldSessionID string
	NewSessionID string
}

// === DEPRECATED MESSAGES ===

// AICoderPTYOutputMsg contains PTY output data from an AI coding session.
// DEPRECATED: Use AICoderBatchUpdateMsg instead for debounced updates.
type AICoderPTYOutputMsg struct {
	SessionID string
	Data      []byte
	Sequence  uint64
}

// AICoderErrorMsg represents an error that occurred in an AI coding session.
type AICoderErrorMsg struct {
	SessionID string
	Error     error
	Severity  string // "warning", "error", "critical"
}

// Component → Service Commands
// These messages represent commands from the component to the service

// StartAICoderSessionCmd requests creation and start of a new AI coding session.
type StartAICoderSessionCmd struct {
	Name       string
	Command    string
	WorkingDir string
	Width      int
	Height     int
}

// StopAICoderSessionCmd requests termination of an AI coding session.
type StopAICoderSessionCmd struct {
	SessionID string
}

// SendAICoderInputCmd sends keyboard input to an AI coding session's PTY.
type SendAICoderInputCmd struct {
	SessionID string
	Input     []byte
}

// ResizePTYCmd requests a terminal resize for an AI coding session.
type ResizePTYCmd struct {
	SessionID string
	Width     int
	Height    int
}

// Internal Component Messages
// These messages are used internally by the component

// SessionListMsg contains the list of all AI coding sessions.
type SessionListMsg struct {
	Sessions []SessionSummary
}

// SessionSummary provides a lightweight summary of an AI coding session.
type SessionSummary struct {
	ID           string
	Name         string
	Status       SessionStatus
	LastActivity time.Time
}

// TerminalStateMsg contains the terminal state for a specific session.
type TerminalStateMsg struct {
	SessionID string
	State     *TerminalState
}

// SwitchToPTYMsg indicates the UI should switch to PTY mode for a session.
type SwitchToPTYMsg struct {
	SessionID string
}

// AICoderPollPTYMsg is a tick message that triggers PTY output polling.
// DEPRECATED: No longer needed with proactive debounced updates.
type AICoderPollPTYMsg struct {
}

// TerminalState represents a read-only snapshot of terminal state.
type TerminalState struct {
	Lines       []string
	Cursor      CursorPos
	Width       int
	Height      int
	TotalLines  int
	ViewportTop int
	UpdatedAt   time.Time
}

// CursorPos represents the terminal cursor position.
type CursorPos struct {
	Row     int
	Col     int
	Visible bool
}
