package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/pkg/events"
)

// AICoderDebugForwarder handles automatic event forwarding to debug-enabled AI coder sessions
type AICoderDebugForwarder struct {
	model         *Model
	lastErrorTime time.Time
	lastTestTime  time.Time
	throttleDelay time.Duration
}

// NewAICoderDebugForwarder creates a new debug forwarder
func NewAICoderDebugForwarder(model *Model) *AICoderDebugForwarder {
	return &AICoderDebugForwarder{
		model:         model,
		throttleDelay: 5 * time.Second, // Throttle events to avoid overwhelming
	}
}

// HandleBrummerEvent processes Brummer events and forwards to debug-enabled sessions
func (f *AICoderDebugForwarder) HandleBrummerEvent(event interface{}) tea.Cmd {
	// Check if we have an active PTY session with debug mode
	if f.model.aiCoderPTYView == nil || f.model.aiCoderPTYView.currentSession == nil {
		return nil
	}

	if !f.model.aiCoderPTYView.currentSession.IsDebugModeEnabled() {
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
	case systemMessageMsg:
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
		if f.model.ptyManager != nil {
			err := f.model.ptyManager.InjectDataToCurrent(aicoder.DataInjectError)
			if err == nil {
				// Log the injection
				f.model.logStore.Add("ai-coder", "AI Debug",
					"[AUTO] Injected error context", false)
			}
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
		if f.model.ptyManager != nil {
			err := f.model.ptyManager.InjectDataToCurrent(aicoder.DataInjectTestFailure)
			if err == nil {
				// Log the injection
				f.model.logStore.Add("ai-coder", "AI Debug",
					"[AUTO] Injected test failure", false)
			}
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
		if f.model.ptyManager != nil {
			err := f.model.ptyManager.InjectDataToCurrent(aicoder.DataInjectBuildOutput)
			if err == nil {
				// Log the injection
				f.model.logStore.Add("ai-coder", "AI Debug",
					"[AUTO] Injected build failure", false)
			}
		}
		return nil
	}
}

// handleSystemMessage forwards critical system messages to AI coder
func (f *AICoderDebugForwarder) handleSystemMessage(msg systemMessageMsg) tea.Cmd {
	// Only forward errors and warnings
	if msg.level != "error" && msg.level != "warn" {
		return nil
	}

	// Don't forward AI coder's own messages
	if strings.Contains(msg.context, "AI") {
		return nil
	}

	return func() tea.Msg {
		// Inject system message into current session
		if f.model.ptyManager != nil {
			err := f.model.ptyManager.InjectDataToCurrent(aicoder.DataInjectSystemMsg)
			if err == nil {
				// Log the injection
				f.model.logStore.Add("ai-coder", "AI Debug",
					"[AUTO] Injected system message", false)
			}
		}
		return nil
	}
}

// GetDebugStatus returns the current debug mode status
func (f *AICoderDebugForwarder) GetDebugStatus() (enabled bool, sessionName string) {
	if f.model.aiCoderPTYView == nil || f.model.aiCoderPTYView.currentSession == nil {
		return false, ""
	}

	session := f.model.aiCoderPTYView.currentSession
	return session.IsDebugModeEnabled(), session.Name
}

// ToggleDebugMode toggles debug mode for the current session
func (f *AICoderDebugForwarder) ToggleDebugMode() tea.Cmd {
	if f.model.aiCoderPTYView == nil || f.model.aiCoderPTYView.currentSession == nil {
		return nil
	}

	session := f.model.aiCoderPTYView.currentSession
	newState := !session.IsDebugModeEnabled()
	session.SetDebugMode(newState)

	// Log the change
	status := "disabled"
	if newState {
		status = "enabled"
	}

	return func() tea.Msg {
		f.model.logStore.Add("ai-coder", "AI Debug",
			"Debug mode "+status+" for "+session.Name, false)
		return nil
	}
}