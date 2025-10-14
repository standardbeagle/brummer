package messages

import (
	"time"
)

// System Messages handles system message display and panel management communication
// between SystemService and SystemComponent

// SystemMessageAddedMsg is sent when a new system message is added
type SystemMessageAddedMsg struct {
	MessageID   string      `json:"message_id"`
	Level       LogLevel    `json:"level"`
	Content     string      `json:"content"`
	Source      string      `json:"source"`
	UnreadCount int         `json:"unread_count"`
	Time        time.Time   `json:"time"`
}

// SystemPanelToggledMsg is sent when the system panel expanded state changes
type SystemPanelToggledMsg struct {
	Expanded    bool       `json:"expanded"`
	Trigger     string     `json:"trigger"` // "keyboard", "mouse", "auto"
	Time        time.Time  `json:"time"`
}

// SystemMessagesReadMsg is sent when messages are marked as read
type SystemMessagesReadMsg struct {
	UpToTime      time.Time `json:"up_to_time"`
	ReadCount     int       `json:"read_count"`
	UnreadCount   int       `json:"unread_count"`
	Time          time.Time `json:"time"`
}

// SystemMessageBufferClearedMsg is sent when the message buffer is cleared
type SystemMessageBufferClearedMsg struct {
	ClearedCount  int       `json:"cleared_count"`
	Trigger       string    `json:"trigger"`
	Time          time.Time `json:"time"`
}

// SystemMessageLevel represents the severity level of system messages
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)