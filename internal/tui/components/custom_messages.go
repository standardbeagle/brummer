package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

// Process-specific messages

// ProcessStartMsg requests starting a process
type ProcessStartMsg struct {
	ScriptName string
	Args       []string
	Env        []string
}

// ProcessStopMsg requests stopping a process
type ProcessStopMsg struct {
	ProcessID string
	Force     bool
}

// ProcessRestartMsg requests restarting a process
type ProcessRestartMsg struct {
	ProcessID string
}

// ProcessUpdateMsg carries process state updates
type ProcessUpdateMsg struct {
	Process    *process.Process
	UpdateType ProcessUpdateType
}

type ProcessUpdateType int

const (
	ProcessStarted ProcessUpdateType = iota
	ProcessStopped
	ProcessRestarted
	ProcessOutput
	ProcessError
	ProcessStateChange
)

// ProcessListUpdateMsg updates the entire process list
type ProcessListUpdateMsg struct {
	Processes []*process.Process
}

// ProcessSelectMsg indicates a process was selected
type ProcessSelectMsg struct {
	ProcessID string
	Process   *process.Process
}

// ProcessOutputMsg carries process output
type ProcessOutputMsg struct {
	ProcessID string
	Output    string
	IsError   bool
	Timestamp time.Time
}

// Logs-specific messages

// LogsFilterMsg sets the log filter
type LogsFilterMsg struct {
	Filter    string
	ProcessID string // Optional: filter by process
	Level     string // Optional: filter by level (error, warn, info)
}

// LogsUpdateMsg carries new log entries
type LogsUpdateMsg struct {
	Entries   []*logs.LogEntry
	ProcessID string // Optional: logs for specific process
	Append    bool   // Whether to append or replace
}

// LogsClearMsg clears logs
type LogsClearMsg struct {
	ProcessID string // Optional: clear for specific process only
}

// LogsScrollMsg handles log scrolling
type LogsScrollMsg struct {
	Direction ScrollDirection
	Amount    int
}

// LogsFollowMsg toggles following logs
type LogsFollowMsg struct {
	Follow bool
}

// Errors-specific messages

// ErrorOccurredMsg indicates a new error
type ErrorOccurredMsg struct {
	ProcessID   string
	ProcessName string
	Error       string
	Context     []string // Surrounding log lines
	Timestamp   time.Time
	Type        string
}

// ErrorClearMsg clears errors
type ErrorClearMsg struct {
	ProcessID string // Optional: clear for specific process
	ErrorID   string // Optional: clear specific error
}

// ErrorsUpdateMsg updates the error list
type ErrorsUpdateMsg struct {
	Errors []ErrorInfo
}

type ErrorInfo struct {
	ID          string
	ProcessID   string
	ProcessName string
	Message     string
	Context     []string
	Timestamp   time.Time
	Type        string
	Count       int // Number of occurrences
}

// ErrorSelectMsg indicates an error was selected
type ErrorSelectMsg struct {
	ErrorID string
	Error   ErrorInfo
}

// URLs-specific messages

// URLDetectedMsg indicates a new URL was detected
type URLDetectedMsg struct {
	URL       string
	ProcessID string
	Type      URLType
	Timestamp time.Time
}

type URLType int

const (
	URLTypeHTTP URLType = iota
	URLTypeHTTPS
	URLTypeWebSocket
	URLTypeFile
	URLTypeCustom
)

// URLsUpdateMsg updates the URL list
type URLsUpdateMsg struct {
	URLs []URLInfo
}

type URLInfo struct {
	URL       string
	ProcessID string
	Type      URLType
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int
	Active    bool
}

// URLOpenMsg requests opening a URL
type URLOpenMsg struct {
	URL      string
	External bool // Open in external browser
}

// URLCopyMsg requests copying a URL to clipboard
type URLCopyMsg struct {
	URL string
}

// Web/Proxy-specific messages

// ProxyRequestMsg represents an HTTP request through the proxy
type ProxyRequestMsg struct {
	ID        string
	Method    string
	URL       string
	Headers   map[string][]string
	Body      []byte
	Timestamp time.Time
	ProcessID string
}

// ProxyResponseMsg represents an HTTP response
type ProxyResponseMsg struct {
	RequestID  string
	StatusCode int
	Headers    map[string][]string
	Body       []byte
	Duration   time.Duration
	Error      error
}

// ProxyRequestsUpdateMsg updates the proxy request list
type ProxyRequestsUpdateMsg struct {
	Requests []ProxyRequestInfo
}

type ProxyRequestInfo struct {
	ID         string
	Method     string
	URL        string
	StatusCode int
	Duration   time.Duration
	Size       int64
	Timestamp  time.Time
	ProcessID  string
	Error      string // Error message if request failed
}

// Web browser messages
type WebNavigateMsg struct {
	URL string
}

type WebLoadCompleteMsg struct {
	URL     string
	Content string
}

type WebBackMsg struct{}

type WebForwardMsg struct{}

type WebRefreshMsg struct{}

type WebViewModeMsg struct {
	Mode WebViewMode
}

type WebViewMode int

const (
	WebViewModeBrowser WebViewMode = iota
	WebViewModeProxy
	WebViewModeInspector
)

// AI Coder-specific messages

// AICoderStartMsg starts a new AI coder session
type AICoderStartMsg struct {
	Name         string
	Model        string
	SystemPrompt string
	WorkingDir   string
}

// AICoderStopMsg stops an AI coder session
type AICoderStopMsg struct {
	SessionID string
	Save      bool // Save session history
}

// AICoderMessageMsg sends a message to AI coder
type AICoderMessageMsg struct {
	SessionID string
	Message   string
	IsUser    bool
}

// AICoderResponseMsg carries AI response
type AICoderResponseMsg struct {
	SessionID string
	Response  string
	Error     error
	Complete  bool
}

// AICoderSessionUpdateMsg updates AI coder session info
type AICoderSessionUpdateMsg struct {
	Sessions []AICoderSessionInfo
}

type AICoderSessionInfo struct {
	ID           string
	Name         string
	Model        string
	State        string
	MessageCount int
	StartTime    time.Time
	LastActivity time.Time
}

// Settings-specific messages

// SettingUpdateMsg updates a setting
type SettingUpdateMsg struct {
	Key      string
	Value    interface{}
	Category string
}

// SettingsLoadMsg loads settings
type SettingsLoadMsg struct {
	Settings map[string]interface{}
}

// SettingsSaveMsg saves settings
type SettingsSaveMsg struct {
	Settings map[string]interface{}
}

// SettingsResetMsg resets settings to defaults
type SettingsResetMsg struct {
	Category string // Optional: reset specific category
}

// Script Selector messages

// ScriptListMsg updates available scripts
type ScriptListMsg struct {
	Scripts []ScriptInfo
}

type ScriptInfo struct {
	Name        string
	Command     string
	Description string
	Env         map[string]string
	Running     bool
}

// ScriptSelectMsg indicates a script was selected
type ScriptSelectMsg struct {
	Script ScriptInfo
	Action ScriptAction
}

type ScriptAction int

const (
	ScriptActionRun ScriptAction = iota
	ScriptActionStop
	ScriptActionRestart
	ScriptActionEdit
	ScriptActionView
)

// Command/Input messages

// CommandMode represents the input mode
type CommandMode string

const (
	CommandModeNormal CommandMode = "normal"
	CommandModeSearch CommandMode = "search"
	CommandModeFilter CommandMode = "filter"
	CommandModeGoto   CommandMode = "goto"
	CommandModeRun    CommandMode = "run"
	CommandModeAI     CommandMode = "ai"
)

// CommandModeMsg changes the command input mode
type CommandModeMsg struct {
	Mode CommandMode
}

// CommandExecuteMsg executes a command
type CommandExecuteMsg struct {
	Command    string
	Args       []string
	Background bool
}

// CommandHistoryMsg manages command history
type CommandHistoryMsg struct {
	Action    HistoryAction
	Command   string
	Direction string // "up" or "down" for navigation
}

type HistoryAction int

const (
	HistoryAdd HistoryAction = iota
	HistoryPrevious
	HistoryNext
	HistoryClear
)

// CommandSuggestMsg updates command suggestions
type CommandSuggestMsg struct {
	Input string
}

// CommandSuggestionsMsg provides command suggestions
type CommandSuggestionsMsg struct {
	Prefix      string
	Suggestions []CommandSuggestion
}

type CommandSuggestion struct {
	Command     string
	Description string
	Category    string
	Shortcut    string
}

// Notification messages

// SystemNotificationMsg represents a system notification
type SystemNotificationMsg struct {
	ID          string
	Level       NotificationLevel
	Title       string
	Message     string
	Details     string
	Source      string
	Timestamp   time.Time
	Actions     []NotificationAction
	AutoDismiss bool
	Duration    time.Duration
	Count       int // Number of times this notification has occurred
}

// NotificationDismissMsg dismisses a notification
type NotificationDismissMsg struct {
	ID  string
	All bool // Dismiss all notifications
}

// NotificationActionMsg executes a notification action
type NotificationActionMsg struct {
	NotificationID string
	ActionIndex    int
}

// Helper functions for creating messages

// SendProcessStart creates a command to start a process
func SendProcessStart(scriptName string, args ...string) tea.Cmd {
	return func() tea.Msg {
		return ProcessStartMsg{
			ScriptName: scriptName,
			Args:       args,
		}
	}
}

// SendProcessStop creates a command to stop a process
func SendProcessStop(processID string, force bool) tea.Cmd {
	return func() tea.Msg {
		return ProcessStopMsg{
			ProcessID: processID,
			Force:     force,
		}
	}
}

// SendLogsFilter creates a command to filter logs
func SendLogsFilter(filter, processID, level string) tea.Cmd {
	return func() tea.Msg {
		return LogsFilterMsg{
			Filter:    filter,
			ProcessID: processID,
			Level:     level,
		}
	}
}

// SendErrorClear creates a command to clear errors
func SendErrorClear(processID, errorID string) tea.Cmd {
	return func() tea.Msg {
		return ErrorClearMsg{
			ProcessID: processID,
			ErrorID:   errorID,
		}
	}
}

// SendURLOpen creates a command to open a URL
func SendURLOpen(url string, external bool) tea.Cmd {
	return func() tea.Msg {
		return URLOpenMsg{
			URL:      url,
			External: external,
		}
	}
}

// SendSystemNotification creates a system notification
func SendSystemNotification(level NotificationLevel, title, message string) tea.Cmd {
	return func() tea.Msg {
		return SystemNotificationMsg{
			ID:          generateID(),
			Level:       level,
			Title:       title,
			Message:     message,
			Timestamp:   time.Now(),
			AutoDismiss: level != NotificationError,
			Duration:    5 * time.Second,
		}
	}
}

// generateID generates a unique ID for messages
func generateID() string {
	return time.Now().Format("20060102150405.999999")
}

// Note: ViewChangeMsg and SendViewChange are defined in messages.go
