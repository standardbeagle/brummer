package messages

import (
	"time"

	"github.com/standardbeagle/brummer/internal/domain/processes"
)

// CommandResultMsg is now defined in commands/slash_commands.go to avoid circular imports

// NavigateToViewMsg is a message that requests a view change.
type NavigateToViewMsg struct {
	View string
}

// ShowHelpMsg is a message that requests to show the help view.
type ShowHelpMsg struct{}

// ServiceStatusUpdateMsg is a message that updates the status of a service in the header.
type ServiceStatusUpdateMsg struct {
	Service string // "mcp", "proxy", "ai_coders", "processes"
	Status  string // "running", "stopped", "error", "starting"
	Details string // Optional details like port count, error message, etc.
}

// AICoderSessionInfo holds information about an AI coder session for display
type AICoderSessionInfo struct {
	ID             string
	Name           string
	Status         string // "active", "idle", "error", "stopped"
	ProcessID      string
	StartTime      time.Time
	LastActivity   time.Time
	LinesExchanged int
	CurrentTask    string
	ErrorMessage   string
}

// AddAICoderSessionMsg is a message that adds a new AI coder session to the view
type AddAICoderSessionMsg struct {
	Session *AICoderSessionInfo
}

// ComponentReadyMsg is sent when a component completes initialization
type ComponentReadyMsg struct {
	ComponentID string    `json:"component_id"`
	Time        time.Time `json:"time"`
}

// QuitRequestedMsg is sent when a component requests application quit
type QuitRequestedMsg struct {
	Source string    `json:"source"`
	Time   time.Time `json:"time"`
}

// SystemNotificationMsg represents a system notification message
type SystemNotificationMsg struct {
	Title   string    `json:"title"`
	Message string    `json:"message"`
	Level   NotificationLevel `json:"level"`
	Time    time.Time `json:"time"`
}

// ErrorMessage represents an error message
type ErrorMessage struct {
	Source  string    `json:"source"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// ErrorMsg represents a generic error message
type ErrorMsg struct {
	Error   error
	Context string
}

// Clear command messages
type ClearErrorsMsg struct{}

type ClearLogsMsg struct{}

type ClearWebMsg struct{}

type ShowClearConfirmationMsg struct {
	Tab string // "errors", "logs", "web"
}

type ClearConfirmationResultMsg struct {
	Tab     string // "errors", "logs", "web"
	Confirmed bool
}

// Scroll messages for logs view
type ScrollLogsUpMsg struct{}

type ScrollLogsDownMsg struct{}

type ToggleAutoScrollMsg struct{}

// Log filter messages
type SetShowPatternMsg struct {
	Pattern string
}

type SetHidePatternMsg struct {
	Pattern string
}

// Web view navigation messages
type NavigateWebUpMsg struct{}

type NavigateWebDownMsg struct{}

type ToggleWebDetailMsg struct{}

// SendToAICoderMsg is a message that requests sending content to an AI Coder session
type SendToAICoderMsg struct {
	ContentType string // "error", "proxy_request", "log", "custom"
	Content     string // The formatted content to send
	Title       string // Optional title/summary for the content
	SessionID   string // Optional specific session ID, empty = active session
}

// UpdateProcessesMsg is a message to update the list of processes in the state.
type UpdateProcessesMsg struct {
	Processes []*processes.Process
}

// SelectProcessMsg is a message to select a process in the processes view.
type SelectProcessMsg struct {
	ProcessID string
}

// Msg is a union type for all messages
type Msg interface{}