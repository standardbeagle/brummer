package messages

import (
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

// Process lifecycle events
// These messages represent state changes in the process manager

type ProcessStartedMsg struct {
	ProcessID string
	Name      string
	Script    string
	StartTime time.Time
}

type ProcessStoppedMsg struct {
	ProcessID string
	Name      string
	ExitCode  int
	EndTime   time.Time
}

type ProcessOutputMsg struct {
	ProcessID string
	Line      string
	IsError   bool
	Timestamp time.Time
}

type ProcessStateChangedMsg struct {
	ProcessID string
	OldState  process.ProcessStatus
	NewState  process.ProcessStatus
}

// Log events
// These messages represent changes to the log store

type LogAddedMsg struct {
	Entry *logs.LogEntry
}

type LogsClearedMsg struct {
	ProcessID string // Empty string = all cleared
}

// URL detection events
// These messages represent URLs detected in process output

type URLDetectedMsg struct {
	URL        string
	ProcessID  string
	Type       string // "http", "https", "ws", "wss"
	Timestamp  time.Time
	HasProxy   bool
	ProxyPort  int
	ProxyURL   string
	ProxyLabel string
}

// Error events
// These messages represent errors from various services

type ErrorOccurredMsg struct {
	ProcessID   string
	ProcessName string
	Error       string
	Timestamp   time.Time
}

// Service health events
// These messages represent health status of core services

type ServiceHealthMsg struct {
	Service string // "process", "proxy", "mcp"
	Healthy bool
	Message string
}

// ServiceBridge lifecycle events
// These messages coordinate ServiceBridge initialization

// StartServiceBridgeMsg triggers ServiceBridge initialization.
// Sent by IntegratedShell.Init() to start observers after program is running.
type StartServiceBridgeMsg struct{}

// ServiceBridgeStartedMsg confirms ServiceBridge is active.
// Sent after observers are registered and event flow is established.
type ServiceBridgeStartedMsg struct{}

// Proxy request events
// These messages represent HTTP requests captured by the proxy server

type ProxyRequestMsg struct {
	RequestID   string
	Method      string
	URL         string
	Path        string
	StatusCode  int
	Duration    time.Duration
	ProcessName string
	Timestamp   time.Time
}
