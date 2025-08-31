package services

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ProcessInfo represents basic process information
type ProcessInfo struct {
	ID        string
	Name      string
	Status    string
	StartTime time.Time
	Command   string
	PID       int
}

// ProcessService manages process lifecycle
type ProcessService interface {
	// Start a script or command
	Start(name string) tea.Cmd

	// Stop a running process
	Stop(processID string) tea.Cmd

	// Restart a process
	Restart(processID string) tea.Cmd

	// Get status of all processes
	GetStatus() []ProcessInfo

	// Get available scripts
	GetScripts() map[string]string
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	ProcessID string
	Level     string // info, warn, error
	Message   string
	Source    string // stdout, stderr
}

// LogsService manages log operations
type LogsService interface {
	// Get logs for a specific process (or all if processID is empty)
	GetLogs(processID string, limit int) []LogEntry

	// Filter logs by level
	FilterByLevel(level string) []LogEntry

	// Search logs by text
	Search(query string) []LogEntry

	// Clear logs for a process (or all if processID is empty)
	Clear(processID string) tea.Cmd

	// Subscribe to log updates
	Subscribe() tea.Cmd
}

// ProxyInfo represents proxy configuration
type ProxyInfo struct {
	Enabled      bool
	Port         int
	TargetURL    string
	RequestCount int
}

// ProxyService manages proxy settings
type ProxyService interface {
	// Enable or disable proxy
	SetEnabled(enabled bool) tea.Cmd

	// Toggle proxy state
	Toggle() tea.Cmd

	// Get current proxy configuration
	GetInfo() ProxyInfo

	// List intercepted requests
	GetRequests(limit int) []ProxyRequest
}

// ProxyRequest represents an intercepted HTTP request
type ProxyRequest struct {
	Timestamp  time.Time
	Method     string
	URL        string
	StatusCode int
	Duration   time.Duration
}

// AISession represents an AI coder session
type AISession struct {
	ID        string
	Provider  string
	Active    bool
	StartTime time.Time
}

// AICoderStartSpec describes how to initialize a UI session for the AI coder
// The component layer maps this to component-specific messages
type AICoderStartSpec struct {
	Name         string
	Model        string
	SystemPrompt string
	WorkingDir   string
}

// AICoderService manages AI coder sessions (UI-agnostic)
type AICoderService interface {
	// Start returns a start specification for a new AI session (non-blocking)
	Start(provider string) (AICoderStartSpec, error)

	// Stop ends an AI session by id
	Stop(sessionID string) error

	// List all sessions (optional for now)
	ListSessions() []AISession

	// Get available providers
	GetProviders() []string
}

// URLInfo represents a discovered URL
type URLInfo struct {
	URL       string
	ProcessID string
	FirstSeen time.Time
	LastSeen  time.Time
	HitCount  int
}

// URLService manages discovered URLs
type URLService interface {
	// Get all discovered URLs
	GetURLs() []URLInfo

	// Open URL in browser
	OpenURL(url string) tea.Cmd

	// Copy URL to clipboard
	CopyURL(url string) tea.Cmd
}
