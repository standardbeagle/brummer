package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
	"github.com/standardbeagle/brummer/internal/tui/system"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Core interface that all controllers depend on
type CoreModelInterface interface {
	GetWidth() int
	GetHeight() int
	GetUpdateChannel() chan tea.Msg
}

// Navigation interface for view management
type NavigationModelInterface interface {
	CoreModelInterface
	GetCurrentView() View
	SwitchToView(view View)
}

// Process management interface
type ProcessModelInterface interface {
	CoreModelInterface
	GetProcessManager() ProcessManagerInterface
	GetSelectedProcess() string
	SetSelectedProcess(id string)
	UpdateProcessList()
}

// Logging interface
type LogModelInterface interface {
	CoreModelInterface
	GetLogStore() LogStoreInterface
}

// Event handling interface
type EventModelInterface interface {
	CoreModelInterface
	GetEventBus() EventBusInterface
}

// Script management interface
type ScriptModelInterface interface {
	CoreModelInterface
	GetScripts() map[string]string
	IsScriptSelectorVisible() bool
	ShowScriptSelector()
	HideScriptSelector()
}

// Search interface
type SearchModelInterface interface {
	CoreModelInterface
	IsSearchActive() bool
	GetSearchPattern() string
	SetSearchPattern(pattern string)
}

// UI component interfaces
type UIComponentModelInterface interface {
	CoreModelInterface
	// Command window
	IsCommandWindowVisible() bool
	ShowCommandWindow()
	HideCommandWindow()
	// Settings
	IsSettingsVisible() bool
	ShowSettings()
	HideSettings()
}

// AI Coder interface
type AICoderModelInterface interface {
	CoreModelInterface
	HasActiveAICoders() bool
	GetAICoderController() AICoderControllerInterface
}

// Composite interface for controllers that need multiple concerns
type ModelInterface interface {
	NavigationModelInterface
	ProcessModelInterface
	LogModelInterface
	EventModelInterface
	ScriptModelInterface
	SearchModelInterface
	UIComponentModelInterface
	AICoderModelInterface
}

// ProcessManagerInterface defines process management operations
type ProcessManagerInterface interface {
	StartScript(name string) (*process.Process, error)
	StartCommand(name, command string, args []string) (*process.Process, error)
	StopProcess(id string) error
	StopProcessAndWait(id string, timeout time.Duration) error
	GetAllProcesses() []*process.Process
	GetProcess(id string) (*process.Process, bool)
	GetScripts() map[string]string
}

// LogStoreInterface defines log storage operations
type LogStoreInterface interface {
	Add(processID, processName, message string, isError bool) *logs.LogEntry
	GetByProcess(processID string) []logs.LogEntry
	GetAll() []logs.LogEntry
	GetErrors() []logs.LogEntry
	Search(pattern string) []logs.LogEntry
	ClearLogs()
	ClearErrors()
	ClearLogsForProcess(processName string)
}

// EventBusInterface defines event handling operations
type EventBusInterface interface {
	Publish(event events.Event)
	Subscribe(eventType events.EventType, handler events.Handler)
	Shutdown()
}

// NavigationControllerInterface defines navigation operations
type NavigationControllerInterface interface {
	GetCurrentView() View
	SwitchTo(view View)
	NextView()
	PreviousView()
	GetViewName(view View) string
	GetViewIcon(view View) string
}

// LayoutControllerInterface defines layout calculations
type LayoutControllerInterface interface {
	UpdateSizes(width, height int)
	GetMainContentWidth() int
	GetAICoderWidth() int
	GetContentHeight() int
	GetHeaderHeight() int
	GetFooterHeight() int
	UpdateAllViewports()
}

// InputControllerInterface defines keyboard input handling
type InputControllerInterface interface {
	HandleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool)
}

// CommandWindowControllerInterface defines command window operations
type CommandWindowControllerInterface interface {
	IsVisible() bool
	Show()
	Hide()
	GetCommands() []CommandItem
	GetSelectedCommand() int
	SelectNext()
	SelectPrevious()
	ExecuteSelected() tea.Cmd
	SetProcessManager(pm ProcessManagerInterface)
}

// AICoderControllerInterface defines AI coder operations
type AICoderControllerInterface interface {
	HandleAICommand(command string) tea.Cmd
	CreateAICoderSession(provider string)
	HasActiveSessions() bool
	SetHasActiveSessions(active bool)
	GetActiveSessions() []AICoderSession
}

// SystemControllerInterface defines system message operations
type SystemControllerInterface interface {
	GetUnreadIndicators() map[navigation.View]system.UnreadIndicator
}

// NotificationsControllerInterface defines notification operations
type NotificationsControllerInterface interface {
	IsActive() bool
	GetMessage() string
}

// ErrorsViewControllerInterface defines error view operations
type ErrorsViewControllerInterface interface {
	UpdateErrorsList() int
	GetErrors() []ErrorItem
	GetSelectedError() int
	SelectNext()
	SelectPrevious()
	ClearErrors()
	HasUnreadErrors() bool
	MarkAllRead()
}

// Additional helper types for interfaces

// CommandItem represents a command in the command window
type CommandItem struct {
	Name        string
	Description string
	Handler     func() tea.Cmd
}

// AICoderSession represents an active AI coder session
type AICoderSession struct {
	ID       string
	Provider string
	Status   string
	Created  time.Time
}

// SystemMessage represents a system message
type SystemMessage struct {
	Level     string
	Message   string
	Timestamp time.Time
	Read      bool
}

// NotificationLevel represents notification severity
type NotificationLevel int

const (
	NotificationInfo NotificationLevel = iota
	NotificationWarning
	NotificationError
	NotificationSuccess
)

// Notification represents a notification
type Notification struct {
	ID        string
	Title     string
	Message   string
	Level     NotificationLevel
	Timestamp time.Time
	Duration  time.Duration
}

// ErrorItem represents an error in the errors view
type ErrorItem struct {
	ProcessID   string
	ProcessName string
	Message     string
	Timestamp   time.Time
	Count       int
	Read        bool
}
