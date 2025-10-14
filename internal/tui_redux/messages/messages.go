package messages

import (
	"time"

	"github.com/standardbeagle/brummer/internal/tui_redux/model"
	tea "github.com/charmbracelet/bubbletea"
)

// View Messages

// ViewSwitchedMsg is sent when the user switches to a different view
type ViewSwitchedMsg struct {
	PreviousView model.View `json:"previous_view"`
	NewView      model.View `json:"new_view"`
	Timestamp    time.Time  `json:"timestamp"`
}

// ViewHistoryBackMsg is sent when the user navigates back in view history
type ViewHistoryBackMsg struct{}

// ViewHistoryForwardMsg is sent when the user navigates forward in view history
type ViewHistoryForwardMsg struct{}

// WindowResizedMsg is sent when the terminal window is resized
type WindowResizedMsg struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// FullscreenToggleMsg is sent when fullscreen mode is toggled
type FullscreenToggleMsg struct{}

// Process Messages

// ProcessStartedMsg is sent when a new process is started
type ProcessStartedMsg struct {
	Process *model.Process `json:"process"`
}

// ProcessStoppedMsg is sent when a process stops
type ProcessStoppedMsg struct {
	ProcessID string `json:"process_id"`
	ExitCode  int    `json:"exit_code"`
	Timestamp time.Time `json:"timestamp"`
}

// ProcessOutputMsg is sent when a process produces output
type ProcessOutputMsg struct {
	ProcessID string    `json:"process_id"`
	Line      string    `json:"line"`
	Stream    string    `json:"stream"` // stdout, stderr
	Timestamp time.Time `json:"timestamp"`
}

// ProcessErrorMsg is sent when a process encounters an error
type ProcessErrorMsg struct {
	ProcessID string    `json:"process_id"`
	Error     error     `json:"error"`
	Context   string    `json:"context"`
	Timestamp time.Time `json:"timestamp"`
}

// ProcessSelectedMsg is sent when a process is selected
type ProcessSelectedMsg struct {
	ProcessID string `json:"process_id"`
}

// ProcessRestartedMsg is sent when a process is restarted
type ProcessRestartedMsg struct {
	ProcessID string `json:"process_id"`
	Process   *model.Process `json:"process"`
}

// Log Messages

// LogFilterChangedMsg is sent when the log filter is changed
type LogFilterChangedMsg struct {
	Filter string `json:"filter"`
}

// LogScrollMsg is sent when the log view should scroll
type LogScrollMsg struct {
	Direction string `json:"direction"` // up, down, top, bottom
	Amount    int    `json:"amount"`
}

// LogLoadMsg is sent when logs should be loaded
type LogLoadMsg struct {
	ProcessID string `json:"process_id"`
}

// LogSaveMsg is sent when logs should be saved
type LogSaveMsg struct {
	ProcessID string `json:"process_id"`
	Path      string `json:"path"`
}

// LogClearMsg is sent when logs should be cleared
type LogClearMsg struct {
	ProcessID string `json:"process_id"`
}

// LogStreamToggleMsg is sent when log streaming is toggled
type LogStreamToggleMsg struct {
	ProcessID string `json:"process_id"`
	Enabled   bool   `json:"enabled"`
}

// Error Messages

// ErrorDetectedMsg is sent when an error is detected
type ErrorDetectedMsg struct {
	ProcessID string    `json:"process_id"`
	Error     error     `json:"error"`
	Context   string    `json:"context"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorDismissMsg is sent when an error is dismissed
type ErrorDismissMsg struct {
	ErrorIndex int `json:"error_index"`
}

// ErrorClearMsg is sent when all errors should be cleared
type ErrorClearMsg struct{}

// ErrorSelectedMsg is sent when an error is selected
type ErrorSelectedMsg struct {
	ErrorIndex int `json:"error_index"`
}

// URL Messages

// URLDiscoveredMsg is sent when a new URL is discovered
type URLDiscoveredMsg struct {
	ProcessID string `json:"process_id"`
	URL       string `json:"url"`
	Label     string `json:"label"`
	Timestamp time.Time `json:"timestamp"`
}

// URLOpenRequestMsg is sent when the user wants to open a URL
type URLOpenRequestMsg struct {
	URL string `json:"url"`
}

// URLCopyMsg is sent when the user wants to copy a URL
type URLCopyMsg struct {
	URL string `json:"url"`
}

// URLTestMsg is sent when the user wants to test a URL
type URLTestMsg struct {
	URL string `json:"url"`
}

// URLSelectedMsg is sent when a URL is selected
type URLSelectedMsg struct {
	URLIndex int `json:"url_index"`
}

// AI Coder Messages

// AISessionStartedMsg is sent when an AI coder session starts
type AISessionStartedMsg struct {
	Session *model.AISession `json:"session"`
}

// AISessionEndedMsg is sent when an AI coder session ends
type AISessionEndedMsg struct {
	SessionID string `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

// AISessionSelectedMsg is sent when an AI session is selected
type AISessionSelectedMsg struct {
	SessionID string `json:"session_id"`
}

// AISendMsg is sent when a message is sent to AI
type AISendMsg struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Timestamp time.Time `json:"timestamp"`
}

// AIResponseMsg is sent when AI responds
type AIResponseMsg struct {
	SessionID string `json:"session_id"`
	Response  string `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// AIErrorMsg is sent when AI encounters an error
type AIErrorMsg struct {
	SessionID string `json:"session_id"`
	Error     error  `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// AIClearMsg is sent when AI conversation should be cleared
type AIClearMsg struct {
	SessionID string `json:"session_id"`
}

// Settings Messages

// SettingsUpdatedMsg is sent when settings are updated
type SettingsUpdatedMsg struct {
	Settings model.Settings `json:"settings"`
}

// SettingsLoadMsg is sent when settings should be loaded
type SettingsLoadMsg struct{}

// SettingsSaveMsg is sent when settings should be saved
type SettingsSaveMsg struct {
	Settings model.Settings `json:"settings"`
}

// SettingsResetMsg is sent when settings should be reset to defaults
type SettingsResetMsg struct{}

// ThemeChangedMsg is sent when the theme is changed
type ThemeChangedMsg struct {
	Theme model.Theme `json:"theme"`
}

// Command Palette Messages

// PaletteOpenMsg is sent when the command palette should open
type PaletteOpenMsg struct{}

// PaletteCloseMsg is sent when the command palette should close
type PaletteCloseMsg struct{}

// PaletteQueryMsg is sent when the palette query changes
type PaletteQueryMsg struct {
	Query string `json:"query"`
}

// PaletteSelectMsg is sent when a palette command is selected
type PaletteSelectMsg struct {
	Command model.Command `json:"command"`
}

// PaletteExecuteMsg is sent when a palette command should be executed
type PaletteExecuteMsg struct {
	Command model.Command `json:"command"`
}

// Keyboard Messages

// KeyBindToggleMsg is sent when key bindings should be toggled
type KeyBindToggleMsg struct {
	Enabled bool `json:"enabled"`
}

// KeyBindReloadMsg is sent when key bindings should be reloaded
type KeyBindReloadMsg struct{}

// System Messages

// TickMsg is sent periodically for time-based updates
type TickMsg struct {
	Time time.Time `json:"time"`
}

// QuitMsg is sent when the application should quit
type QuitMsg struct{}

// ForceQuitMsg is sent when the application should quit immediately
type ForceQuitMsg struct{}

// RefreshMsg is sent when the UI should be refreshed
type RefreshMsg struct{}

// StatusUpdateMsg is sent when the status should be updated
type StatusUpdateMsg struct {
	Message string `json:"message"`
	Level   string `json:"level"` // info, warn, error
}

// Debug Messages

// DebugToggleMsg is sent when debug mode is toggled
type DebugToggleMsg struct {
	Enabled bool `json:"enabled"`
}

// DebugLogMsg is sent for debug logging
type DebugLogMsg struct {
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
	Context   string    `json:"context"`
}

// Performance Messages

// PerformanceMetricsMsg is sent with performance metrics
type PerformanceMetricsMsg struct {
	UpdateTime    time.Duration `json:"update_time"`
	RenderTime    time.Duration `json:"render_time"`
	MemoryUsage   int64         `json:"memory_usage"`
	GoroutineCount int          `json:"goroutine_count"`
	Timestamp     time.Time     `json:"timestamp"`
}

// Help Messages

// HelpToggleMsg is sent when help should be toggled
type HelpToggleMsg struct{}

// HelpShowMsg is sent when help should be shown
type HelpShowMsg struct {
	Topic string `json:"topic"`
}

// HelpHideMsg is sent when help should be hidden
type HelpHideMsg struct{}

// Utility functions for creating common messages

// NewViewSwitchedMsg creates a new ViewSwitchedMsg
func NewViewSwitchedMsg(previous, new model.View) ViewSwitchedMsg {
	return ViewSwitchedMsg{
		PreviousView: previous,
		NewView:      new,
		Timestamp:    time.Now(),
	}
}

// NewProcessStartedMsg creates a new ProcessStartedMsg
func NewProcessStartedMsg(process *model.Process) ProcessStartedMsg {
	return ProcessStartedMsg{
		Process: process,
	}
}

// NewProcessStoppedMsg creates a new ProcessStoppedMsg
func NewProcessStoppedMsg(processID string, exitCode int) ProcessStoppedMsg {
	return ProcessStoppedMsg{
		ProcessID: processID,
		ExitCode:  exitCode,
		Timestamp: time.Now(),
	}
}

// NewProcessOutputMsg creates a new ProcessOutputMsg
func NewProcessOutputMsg(processID, line, stream string) ProcessOutputMsg {
	return ProcessOutputMsg{
		ProcessID: processID,
		Line:      line,
		Stream:    stream,
		Timestamp: time.Now(),
	}
}

// NewErrorDetectedMsg creates a new ErrorDetectedMsg
func NewErrorDetectedMsg(processID string, err error, context string) ErrorDetectedMsg {
	return ErrorDetectedMsg{
		ProcessID: processID,
		Error:     err,
		Context:   context,
		Timestamp: time.Now(),
	}
}

// NewURLDiscoveredMsg creates a new URLDiscoveredMsg
func NewURLDiscoveredMsg(processID, url, label string) URLDiscoveredMsg {
	return URLDiscoveredMsg{
		ProcessID: processID,
		URL:       url,
		Label:     label,
		Timestamp: time.Now(),
	}
}

// NewTickMsg creates a new TickMsg
func NewTickMsg() TickMsg {
	return TickMsg{
		Time: time.Now(),
	}
}

// Message validation functions

// IsValidMessage checks if a message is a valid TUI message
func IsValidMessage(msg tea.Msg) bool {
	switch msg.(type) {
	// View messages
	case ViewSwitchedMsg, ViewHistoryBackMsg, ViewHistoryForwardMsg,
		WindowResizedMsg, FullscreenToggleMsg:
		return true

	// Process messages
	case ProcessStartedMsg, ProcessStoppedMsg, ProcessOutputMsg,
		ProcessErrorMsg, ProcessSelectedMsg, ProcessRestartedMsg:
		return true

	// Log messages
	case LogFilterChangedMsg, LogScrollMsg, LogLoadMsg, LogSaveMsg,
		LogClearMsg, LogStreamToggleMsg:
		return true

	// Error messages
	case ErrorDetectedMsg, ErrorDismissMsg, ErrorClearMsg, ErrorSelectedMsg:
		return true

	// URL messages
	case URLDiscoveredMsg, URLOpenRequestMsg, URLCopyMsg, URLTestMsg, URLSelectedMsg:
		return true

	// AI coder messages
	case AISessionStartedMsg, AISessionEndedMsg, AISessionSelectedMsg,
		AISendMsg, AIResponseMsg, AIErrorMsg, AIClearMsg:
		return true

	// Settings messages
	case SettingsUpdatedMsg, SettingsLoadMsg, SettingsSaveMsg,
		SettingsResetMsg, ThemeChangedMsg:
		return true

	// Command palette messages
	case PaletteOpenMsg, PaletteCloseMsg, PaletteQueryMsg,
		PaletteSelectMsg, PaletteExecuteMsg:
		return true

	// Keyboard messages
	case KeyBindToggleMsg, KeyBindReloadMsg:
		return true

	// System messages
	case TickMsg, QuitMsg, ForceQuitMsg, RefreshMsg, StatusUpdateMsg:
		return true

	// Debug messages
	case DebugToggleMsg, DebugLogMsg:
		return true

	// Performance messages
	case PerformanceMetricsMsg:
		return true

	// Help messages
	case HelpToggleMsg, HelpShowMsg, HelpHideMsg:
		return true

	// BubbleTea messages
	case tea.KeyMsg, tea.MouseMsg, tea.WindowSizeMsg:
		return true

	// Unknown message type
	default:
		return false
	}
}