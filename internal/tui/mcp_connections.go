package tui

import (
	"fmt"
	"time"
)

// mcpConnectionItem represents an MCP client connection
type mcpConnectionItem struct {
	clientID       string
	clientName     string
	connectedAt    time.Time
	lastActivity   time.Time
	requestCount   int
	isConnected    bool
	connectionType string // "SSE", "HTTP", "HTTP+SSE", "Legacy"
	method         string // "GET", "POST"
}

func (i mcpConnectionItem) Title() string {
	status := "🟢"
	if !i.isConnected {
		status = "🔴"
	}

	// Add connection type icon
	typeIcon := ""
	switch i.connectionType {
	case "SSE":
		typeIcon = "📡" // Streaming connection
	case "HTTP":
		typeIcon = "🌐" // HTTP request
	case "HTTP+SSE":
		typeIcon = "🔄" // HTTP with SSE response
	default:
		typeIcon = "❓" // Legacy or unknown
	}

	return fmt.Sprintf("%s %s %s", status, typeIcon, i.clientName)
}

func (i mcpConnectionItem) Description() string {
	duration := time.Since(i.connectedAt).Round(time.Second)
	return fmt.Sprintf("ID: %s | %s %s | Connected: %s | Requests: %d",
		i.clientID, i.connectionType, i.method, duration, i.requestCount)
}

func (i mcpConnectionItem) FilterValue() string {
	return i.clientName + " " + i.clientID
}

// MCPActivity represents an activity log entry for an MCP connection
type MCPActivity struct {
	Timestamp time.Time
	Method    string
	Params    string
	Response  string
	Error     string
	Duration  time.Duration
}
