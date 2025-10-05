package messages

import "time"

// CommandResultMsg is a message that indicates the result of a command.
type CommandResultMsg struct {
	Success bool
	Message string
}

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
