package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/system"
	"github.com/standardbeagle/brummer/pkg/events"
)

// EventController manages event subscriptions and message handling
type EventController struct {
	// Dependencies
	model      *Model
	eventBus   *events.EventBus
	updateChan chan tea.Msg
	debugMode  bool
}

// NewEventController creates a new event controller
func NewEventController(model *Model, eventBus *events.EventBus, updateChan chan tea.Msg, debugMode bool) *EventController {
	return &EventController{
		model:      model,
		eventBus:   eventBus,
		updateChan: updateChan,
		debugMode:  debugMode,
	}
}

// SetupEventSubscriptions sets up all event subscriptions
func (ec *EventController) SetupEventSubscriptions() {
	// Process events
	ec.eventBus.Subscribe(events.ProcessStarted, func(e events.Event) {
		ec.updateChan <- processUpdateMsg{}
	})

	ec.eventBus.Subscribe(events.ProcessExited, func(e events.Event) {
		// Clean up URLs from the exited process
		if e.ProcessID != "" {
			ec.model.logStore.RemoveURLsForProcess(e.ProcessID)
		}

		// Check if process failed and add system message
		if exitCode, ok := e.Data["exitCode"].(int); ok && exitCode != 0 && ec.model.currentView() != ViewProcesses {
			// Add error message via system controller
			if ec.model.systemController != nil {
				ec.model.systemController.AddMessage("error", "Process", fmt.Sprintf("Process failed with exit code %d", exitCode))
			}
		}

		ec.updateChan <- processUpdateMsg{}
	})

	// Log events
	ec.eventBus.Subscribe(events.LogLine, func(e events.Event) {
		ec.updateChan <- logUpdateMsg{}
	})

	ec.eventBus.Subscribe(events.ErrorDetected, func(e events.Event) {
		ec.updateChan <- errorUpdateMsg{}
	})

	// Proxy events
	ec.eventBus.Subscribe(events.EventType("proxy.request"), func(e events.Event) {
		// Web view updates will be handled by the controller during rendering
	})

	// Telemetry events
	ec.eventBus.Subscribe(events.EventType("telemetry.received"), func(e events.Event) {
		// Web view updates will be handled by the controller during rendering
	})

	// System messages
	ec.eventBus.Subscribe(events.EventType("system.message"), func(e events.Event) {
		level, _ := e.Data["level"].(string)
		context, _ := e.Data["context"].(string)
		message, _ := e.Data["message"].(string)
		if message != "" {
			// Send the message data through the update channel
			SafeGoroutineNoError(
				"send system message",
				func() {
					ec.updateChan <- system.SystemMessageMsg{
						Level:   level,
						Context: context,
						Message: message,
					}
				},
				func(err error) {
					// Log critical error (cannot use update channel here to avoid recursion)
					// This is a fallback for critical system message delivery failures
				},
			)
		}
	})

	// MCP events (debug mode only)
	if ec.debugMode {
		ec.setupMCPEventSubscriptions()
	}
}

// setupMCPEventSubscriptions sets up MCP-specific event subscriptions
func (ec *EventController) setupMCPEventSubscriptions() {
	ec.eventBus.Subscribe(events.MCPConnected, func(e events.Event) {
		sessionId, _ := e.Data["sessionId"].(string)
		clientInfo, _ := e.Data["clientInfo"].(string)
		connectedAt, _ := e.Data["connectedAt"].(time.Time)
		connectionType, _ := e.Data["connectionType"].(string)
		method, _ := e.Data["method"].(string)

		ec.updateChan <- mcpConnectionMsg{
			sessionId:      sessionId,
			clientInfo:     clientInfo,
			connected:      true,
			connectedAt:    connectedAt,
			connectionType: connectionType,
			method:         method,
		}
	})

	ec.eventBus.Subscribe(events.MCPDisconnected, func(e events.Event) {
		sessionId, _ := e.Data["sessionId"].(string)

		ec.updateChan <- mcpConnectionMsg{
			sessionId: sessionId,
			connected: false,
		}
	})

	ec.eventBus.Subscribe(events.MCPActivity, func(e events.Event) {
		sessionId, _ := e.Data["sessionId"].(string)
		method, _ := e.Data["method"].(string)
		params, _ := e.Data["params"].(string)
		response, _ := e.Data["response"].(string)
		errMsg, _ := e.Data["error"].(string)
		duration, _ := e.Data["duration"].(time.Duration)

		activity := MCPActivity{
			Timestamp: time.Now(),
			Method:    method,
			Params:    params,
			Response:  response,
			Error:     errMsg,
			Duration:  duration,
		}

		ec.updateChan <- mcpActivityMsg{
			sessionId: sessionId,
			activity:  activity,
		}
	})
}

// WaitForUpdates returns a command that waits for update messages
func (ec *EventController) WaitForUpdates() tea.Cmd {
	return func() tea.Msg {
		return <-ec.updateChan
	}
}

// TickCmd returns a tick command for periodic updates
func (ec *EventController) TickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// SendStartupMessage sends the initial startup message
func (ec *EventController) SendStartupMessage() {
	SafeGoroutineNoError(
		"send startup message",
		func() {
			ec.updateChan <- system.SystemMessageMsg{
				Level:   "info",
				Context: "System",
				Message: "🚀 Brummer started - initializing services...",
			}
		},
		func(err error) {
			// Critical error in startup message delivery - cannot use update channel
			// System will continue but startup message may be lost
		},
	)
}
