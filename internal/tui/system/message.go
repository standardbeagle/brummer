package system

import (
	"time"
)

// Message represents a system message with metadata
type Message struct {
	Timestamp time.Time
	Level     string // "error", "warning", "info", "success"
	Message   string
	Context   string // Where the message originated (e.g., "Process Control", "Settings")
}

// Level constants for system messages
const (
	LevelError   = "error"
	LevelWarning = "warning"
	LevelInfo    = "info"
	LevelSuccess = "success"
)

// NewMessage creates a new system message
func NewMessage(level, context, message string) Message {
	return Message{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   context,
	}
}

// GetIcon returns the appropriate icon for the message level
func GetIcon(level string) string {
	switch level {
	case LevelError:
		return "❌"
	case LevelWarning:
		return "⚠️"
	case LevelSuccess:
		return "✅"
	case LevelInfo:
		return "ℹ️"
	default:
		return "•"
	}
}
