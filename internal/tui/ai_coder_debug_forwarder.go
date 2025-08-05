package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/system"
	"github.com/standardbeagle/brummer/pkg/events"
)

// AICoderDebugForwarder handles automatic event forwarding to debug-enabled AI coder sessions
type AICoderDebugForwarder struct {
	controller    *AICoderController
	lastErrorTime time.Time
	lastTestTime  time.Time
	throttleDelay time.Duration
}

// NewAICoderDebugForwarder creates a new debug forwarder
func NewAICoderDebugForwarder(controller *AICoderController) *AICoderDebugForwarder {
	return &AICoderDebugForwarder{
		controller:    controller,
		throttleDelay: 5 * time.Second, // Throttle events to avoid overwhelming
	}
}

// HandleBrummerEvent processes Brummer events and forwards to debug-enabled sessions
func (f *AICoderDebugForwarder) HandleBrummerEvent(event interface{}) tea.Cmd {
	// Check if we have an active PTY session with debug mode
	if f.controller == nil || f.controller.aiCoderPTYView == nil || f.controller.aiCoderPTYView.currentSession == nil {
		return nil
	}

	if !f.controller.aiCoderPTYView.currentSession.IsDebugModeEnabled() {
		return nil
	}

	// Handle different event types
	switch e := event.(type) {
	case events.Event:
		// Handle events based on event type
		switch e.Type {
		case events.ErrorDetected:
			return f.handleErrorEvent(e)
		case events.TestFailed:
			return f.handleTestEvent(e)
		case events.BuildEvent:
			return f.handleBuildEvent(e)
		}
	case system.SystemMessageMsg:
		return f.handleSystemMessage(e)
	}

	return nil
}

// handleErrorEvent forwards error events to AI coder
func (f *AICoderDebugForwarder) handleErrorEvent(event events.Event) tea.Cmd {
	// Throttle error events
	if time.Since(f.lastErrorTime) < f.throttleDelay {
		return nil
	}
	f.lastErrorTime = time.Now()

	return func() tea.Msg {
		// Inject error data into current session
		if f.controller.ptyManager != nil {
			// TODO: Implement data injection when PTY manager API is available
			// err := f.controller.ptyManager.InjectDataToCurrent(aicoder.DataInjectError)
			// if err == nil {
			//	// Log the injection - need access to logStore
			//	f.controller.logStore.Add("ai-coder", "AI Debug",
			//		"[AUTO] Injected error context", false)
			// }
		}
		return nil
	}
}

// handleTestEvent forwards test failure events to AI coder
func (f *AICoderDebugForwarder) handleTestEvent(event events.Event) tea.Cmd {
	// Test failed events are already filtered by event type

	// Throttle test events
	if time.Since(f.lastTestTime) < f.throttleDelay {
		return nil
	}
	f.lastTestTime = time.Now()

	return func() tea.Msg {
		// Inject test failure data into current session
		if f.controller.ptyManager != nil {
			// TODO: Implement data injection when PTY manager API is available
			// err := f.controller.ptyManager.InjectDataToCurrent(aicoder.DataInjectTestFailure)
			// if err == nil {
			//	// Log the injection
			//	f.controller.logStore.Add("ai-coder", "AI Debug",
			//		"[AUTO] Injected test failure", false)
			// }
		}
		return nil
	}
}

// handleBuildEvent forwards build failure events to AI coder
func (f *AICoderDebugForwarder) handleBuildEvent(event events.Event) tea.Cmd {
	// Check if it's a build failure event
	if status, ok := event.Data["status"].(string); ok && status != "failed" {
		return nil
	}

	return func() tea.Msg {
		// Inject build output into current session
		if f.controller.ptyManager != nil {
			// TODO: Implement data injection when PTY manager API is available
			// err := f.controller.ptyManager.InjectDataToCurrent(aicoder.DataInjectBuildOutput)
			// if err == nil {
			//	// Log the injection
			//	f.controller.logStore.Add("ai-coder", "AI Debug",
			//		"[AUTO] Injected build failure", false)
			// }
		}
		return nil
	}
}

// handleSystemMessage forwards critical system messages to AI coder
func (f *AICoderDebugForwarder) handleSystemMessage(msg system.SystemMessageMsg) tea.Cmd {
	// Only forward errors and warnings
	if msg.Level != "error" && msg.Level != "warn" {
		return nil
	}

	// Don't forward AI coder's own messages
	if strings.Contains(msg.Context, "AI") {
		return nil
	}

	return func() tea.Msg {
		// Inject system message into current session
		if f.controller.ptyManager != nil {
			// TODO: Implement data injection when PTY manager API is available
			// err := f.controller.ptyManager.InjectDataToCurrent(aicoder.DataInjectSystemMsg)
			// if err == nil {
			//	// Log the injection
			//	f.controller.logStore.Add("ai-coder", "AI Debug",
			//		"[AUTO] Injected system message", false)
			// }
		}
		return nil
	}
}

// GetDebugStatus returns the current debug mode status
func (f *AICoderDebugForwarder) GetDebugStatus() (enabled bool, sessionName string) {
	if f.controller.aiCoderPTYView == nil || f.controller.aiCoderPTYView.currentSession == nil {
		return false, ""
	}

	session := f.controller.aiCoderPTYView.currentSession
	return session.IsDebugModeEnabled(), session.Name
}

// ToggleDebugMode toggles debug mode for the current session
func (f *AICoderDebugForwarder) ToggleDebugMode() tea.Cmd {
	if f.controller.aiCoderPTYView == nil || f.controller.aiCoderPTYView.currentSession == nil {
		return nil
	}

	session := f.controller.aiCoderPTYView.currentSession
	newState := !session.IsDebugModeEnabled()
	session.SetDebugMode(newState)

	// Log the change
	status := "disabled"
	if newState {
		status = "enabled"
	}

	return func() tea.Msg {
		f.controller.logStore.Add("ai-coder", "AI Debug",
			"Debug mode "+status+" for "+session.Name, false)
		return nil
	}
}
