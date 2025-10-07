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

// ProcessUpdateMsg is sent when process list needs to be refreshed
type ProcessUpdateMsg struct{}

// MCPActivityMsg represents MCP server activity
type MCPActivityMsg struct {
	ClientID    string
	Activity    string
	Timestamp   int64
	Details     map[string]interface{}
}

// mcpConnectionMsg is sent when an MCP connection status changes
type mcpConnectionMsg struct {
	sessionId      string
	clientInfo     string
	connected      bool
	connectedAt    int64
	connectionType string
	method         string
}

// restartProcessMsg is sent to restart a process
type restartProcessMsg struct {
	processID string
	clearLogs bool
	message   string
	isError   bool
}

// restartAllMsg is sent to restart all processes
type restartAllMsg struct {
	clearLogs bool
	message   string
	isError   bool
}

// errorUpdateMsg is sent when errors need to be refreshed
type errorUpdateMsg struct{}

// mcpActivityMsg is the lowercase version for internal use
type mcpActivityMsg struct {
	data interface{}
}