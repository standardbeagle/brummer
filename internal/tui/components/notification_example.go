package components

import (
	"fmt"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
)

// NotificationManager demonstrates how to integrate notifications with the component system
type NotificationManager struct {
	shell *EnhancedShell
}

// NewNotificationManager creates a notification manager for the enhanced shell
func NewNotificationManager(shell *EnhancedShell) *NotificationManager {
	return &NotificationManager{
		shell: shell,
	}
}

// ProcessStarted adds a notification when a process starts
func (nm *NotificationManager) ProcessStarted(processName string) {
	nm.shell.UpdateNotification(ViewProcesses, 1, "success")
}

// ProcessStopped adds a notification when a process stops
func (nm *NotificationManager) ProcessStopped(processName string) {
	nm.shell.UpdateNotification(ViewProcesses, 1, "info")
}

// ProcessError adds an error notification for process issues
func (nm *NotificationManager) ProcessError(processName string, err error) {
	nm.shell.UpdateNotification(ViewProcesses, 1, "error")
	nm.shell.UpdateNotification(ViewErrors, 1, "error")
}

// LogReceived adds a notification for new log entries
func (nm *NotificationManager) LogReceived(processName string, logLevel string) {
	severity := "info"
	if logLevel == "ERROR" {
		severity = "error"
		nm.shell.UpdateNotification(ViewErrors, 1, "error")
	} else if logLevel == "WARN" {
		severity = "warning"
	}
	nm.shell.UpdateNotification(ViewLogs, 1, severity)
}

// URLDiscovered adds a notification when a new URL is found
func (nm *NotificationManager) URLDiscovered(url string) {
	nm.shell.UpdateNotification(ViewURLs, 1, "info")
}

// WebRequestReceived adds a notification for proxy requests
func (nm *NotificationManager) WebRequestReceived(method, path string, statusCode int) {
	severity := "info"
	if statusCode >= 400 && statusCode < 500 {
		severity = "warning"
	} else if statusCode >= 500 {
		severity = "error"
		nm.shell.UpdateNotification(ViewErrors, 1, "error")
	}
	nm.shell.UpdateNotification(ViewWeb, 1, severity)
}

// AICoderEvent adds a notification for AI coder activities
func (nm *NotificationManager) AICoderEvent(event string) {
	nm.shell.UpdateNotification(ViewAICoders, 1, "info")
}

// SettingsChanged adds a notification when settings are modified
func (nm *NotificationManager) SettingsChanged(setting string) {
	nm.shell.UpdateNotification(ViewSettings, 1, "success")
}

// ExampleUsageModel demonstrates how to use notifications in a BubbleTea model
type ExampleUsageModel struct {
	shell       *EnhancedShell
	notifMgr    *NotificationManager
	eventLog    []string
	currentTime time.Time
}

// NewExampleUsageModel creates an example model with notifications
func NewExampleUsageModel() *ExampleUsageModel {
	shell := NewEnhancedShell()
	return &ExampleUsageModel{
		shell:       shell,
		notifMgr:    NewNotificationManager(shell),
		eventLog:    []string{},
		currentTime: time.Now(),
	}
}

// SimulatedEvent represents various events that can occur
type SimulatedEvent struct {
	Type    string
	Message string
	Time    time.Time
}

// Init initializes the example model
func (m *ExampleUsageModel) Init() tea.Cmd {
	return tea.Batch(
		m.shell.Init(),
		m.simulateEvents(),
	)
}

// simulateEvents generates random events to demonstrate notifications
func (m *ExampleUsageModel) simulateEvents() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		// Simulate different types of events
		events := []SimulatedEvent{
			{Type: "process_start", Message: "npm run dev started"},
			{Type: "process_stop", Message: "npm test stopped"},
			{Type: "process_error", Message: "build failed with error"},
			{Type: "log", Message: "Server listening on port 3000"},
			{Type: "log_error", Message: "Connection refused"},
			{Type: "url", Message: "http://localhost:3000 discovered"},
			{Type: "web_request", Message: "GET /api/users 200"},
			{Type: "web_error", Message: "POST /api/login 401"},
			{Type: "ai_event", Message: "AI Coder session started"},
			{Type: "settings", Message: "Theme changed to dark"},
		}
		
		// Pick a random event
		event := events[int(t.Unix())%len(events)]
		event.Time = t
		return event
	})
}

// Update handles messages for the example model
func (m *ExampleUsageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SimulatedEvent:
		// Log the event
		m.eventLog = append(m.eventLog, fmt.Sprintf("[%s] %s", 
			msg.Time.Format("15:04:05"), msg.Message))
		
		// Keep only last 10 events
		if len(m.eventLog) > 10 {
			m.eventLog = m.eventLog[1:]
		}
		
		// Trigger appropriate notification
		switch msg.Type {
		case "process_start":
			m.notifMgr.ProcessStarted("npm run dev")
		case "process_stop":
			m.notifMgr.ProcessStopped("npm test")
		case "process_error":
			m.notifMgr.ProcessError("build", fmt.Errorf("compilation failed"))
		case "log":
			m.notifMgr.LogReceived("server", "INFO")
		case "log_error":
			m.notifMgr.LogReceived("server", "ERROR")
		case "url":
			m.notifMgr.URLDiscovered("http://localhost:3000")
		case "web_request":
			m.notifMgr.WebRequestReceived("GET", "/api/users", 200)
		case "web_error":
			m.notifMgr.WebRequestReceived("POST", "/api/login", 401)
		case "ai_event":
			m.notifMgr.AICoderEvent("session_started")
		case "settings":
			m.notifMgr.SettingsChanged("theme")
		}
		
		// Continue simulating events
		return m, m.simulateEvents()
		
	default:
		// Pass through to shell
		newShell, cmd := m.shell.Update(msg)
		m.shell = newShell.(*EnhancedShell)
		return m, cmd
	}
}

// View renders the example model
func (m *ExampleUsageModel) View() string {
	shellView := m.shell.View()
	
	// Add event log overlay
	if len(m.eventLog) > 0 {
		eventLogStr := "\n\n--- Recent Events ---\n"
		for _, event := range m.eventLog {
			eventLogStr += event + "\n"
		}
		shellView += eventLogStr
	}
	
	return shellView
}

// Integration example with minimal shell
func IntegrateWithMinimalShell(shell *MinimalShell) {
	// Example of adding notifications to MinimalShell
	// This would typically be called from event handlers
	
	// Add error notification
	shell.notifications["errors"] = 5
	
	// Add log notification
	shell.notifications["logs"] = 12
	
	// Add URL notification
	shell.notifications["urls"] = 3
	
	// Clear notifications for current view
	delete(shell.notifications, shell.currentView)
}

// NotificationConfig provides configuration for notification behavior
type NotificationConfig struct {
	// EnableSound plays a sound for critical notifications
	EnableSound bool
	
	// AutoClear automatically clears notifications after a timeout
	AutoClear bool
	
	// AutoClearTimeout duration before clearing notifications
	AutoClearTimeout time.Duration
	
	// MaxNotifications maximum notifications to show per tab
	MaxNotifications int
	
	// PriorityColors custom colors for different severities
	PriorityColors map[string]string
}

// DefaultNotificationConfig returns sensible defaults
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		EnableSound:      false,
		AutoClear:        false,
		AutoClearTimeout: time.Minute * 5,
		MaxNotifications: 99,
		PriorityColors: map[string]string{
			"error":   "196", // Red
			"warning": "214", // Orange
			"info":    "39",  // Blue
			"success": "82",  // Green
		},
	}
}