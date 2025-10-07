package tui

// Message types for BubbleTea communications

// RestartProcessMsg is sent when a process needs to be restarted
type RestartProcessMsg struct {
	processID  string
	clearLogs  bool
	message    string
	isError    bool
}

// RestartAllMsg is sent when all processes need to be restarted
type RestartAllMsg struct {
	clearLogs bool
	message   string
	isError   bool
}

// ErrorUpdateMsg is sent when the error list needs to be updated
type ErrorUpdateMsg struct{}

// TickMsg is sent periodically for real-time updates
type TickMsg struct{}

// SwitchToAICodersMsg is sent to switch to the AI Coders view
type SwitchToAICodersMsg struct{}

// LogUpdateMsg is sent when logs need to be refreshed
type LogUpdateMsg struct{}

// MCPActivityMsg represents MCP server activity
type MCPActivityMsg struct {
	ClientID    string
	Activity    string
	Timestamp   int64
	Details     map[string]interface{}
}