package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/aicoder"
)

// PTY event messages for BubbleTea

// ptyOutputMsg indicates terminal output from a PTY session
type ptyOutputMsg struct {
	sessionID string
	data      []byte
	timestamp time.Time
}

// ptySessionClosedMsg indicates a PTY session has closed
type ptySessionClosedMsg struct {
	sessionID string
	reason    string
	timestamp time.Time
}

// ptySessionCreatedMsg indicates a new PTY session was created
type ptySessionCreatedMsg struct {
	sessionID string
	name      string
	timestamp time.Time
}

// ptyDataInjectedMsg indicates data was injected into a PTY session
type ptyDataInjectedMsg struct {
	sessionID string
	dataType  aicoder.DataInjectionType
	timestamp time.Time
}

// ptyResizeMsg indicates a PTY session was resized
type ptyResizeMsg struct {
	sessionID string
	width     int
	height    int
	timestamp time.Time
}

// convertPTYEvent converts aicoder.PTYEvent to appropriate tea.Msg
func (m *Model) convertPTYEvent(event aicoder.PTYEvent) tea.Msg {
	switch event.Type {
	case aicoder.PTYEventOutput:
		if data, ok := event.Data.([]byte); ok {
			return ptyOutputMsg{
				sessionID: event.SessionID,
				data:      data,
				timestamp: event.Timestamp,
			}
		}
		
	case aicoder.PTYEventClose:
		reason := "Session closed"
		if str, ok := event.Data.(string); ok {
			reason = str
		}
		return ptySessionClosedMsg{
			sessionID: event.SessionID,
			reason:    reason,
			timestamp: event.Timestamp,
		}
		
	case aicoder.PTYEventDataInject:
		if injectionData, ok := event.Data.(map[string]interface{}); ok {
			if dataType, ok := injectionData["type"].(aicoder.DataInjectionType); ok {
				return ptyDataInjectedMsg{
					sessionID: event.SessionID,
					dataType:  dataType,
					timestamp: event.Timestamp,
				}
			}
		}
		
	case aicoder.PTYEventResize:
		if resizeData, ok := event.Data.(map[string]int); ok {
			return ptyResizeMsg{
				sessionID: event.SessionID,
				width:     resizeData["width"],
				height:    resizeData["height"],
				timestamp: event.Timestamp,
			}
		}
	}
	
	// Return nil for unhandled events
	return nil
}

// listenPTYEvents listens for PTY events and converts them to tea.Msg
func (m *Model) listenPTYEvents() tea.Cmd {
	return func() tea.Msg {
		if m.ptyEventSub == nil {
			return nil
		}
		
		// This will block until an event is received
		event := <-m.ptyEventSub
		return m.convertPTYEvent(event)
	}
}

// subscribeToActivePTY subscribes to output from the active PTY session
func (m *Model) subscribeToActivePTY() tea.Cmd {
	if m.aiCoderPTYView == nil || m.aiCoderPTYView.currentSession == nil {
		return nil
	}
	
	return func() tea.Msg {
		// This will block until output is received
		data := <-m.aiCoderPTYView.currentSession.OutputChan
		return PTYOutputMsg{
			SessionID: m.aiCoderPTYView.currentSession.ID,
			Data:      data,
		}
	}
}

// handlePTYEventMsg handles PTY event messages in the Update method
func (m *Model) handlePTYEventMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case ptyOutputMsg:
		// Forward to PTY view
		if m.aiCoderPTYView != nil {
			m.aiCoderPTYView.Update(PTYOutputMsg{
				SessionID: msg.sessionID,
				Data:      msg.data,
			})
		}
		// Re-subscribe for more output
		return m.subscribeToActivePTY()
		
	case ptySessionClosedMsg:
		// Forward to PTY view
		if m.aiCoderPTYView != nil {
			m.aiCoderPTYView.Update(PTYEventMsg{
				Event: aicoder.PTYEvent{
					Type:      aicoder.PTYEventClose,
					SessionID: msg.sessionID,
					Data:      msg.reason,
					Timestamp: msg.timestamp,
				},
			})
		}
		return nil
		
	case ptySessionCreatedMsg:
		// Auto-switch to AI Coder view
		m.currentView = ViewAICoders
		// Attach to the new session
		if m.aiCoderPTYView != nil {
			m.aiCoderPTYView.AttachToSession(msg.sessionID)
		}
		// Start subscribing to output
		return m.subscribeToActivePTY()
		
	case ptyDataInjectedMsg:
		// Just for logging/feedback, PTY view handles the actual display
		m.logStore.Add("ai-coder", "AI Coder", 
			"Injected data: "+string(msg.dataType), false)
		return nil
		
	case ptyResizeMsg:
		// PTY view handles resize internally
		return nil
	}
	
	return nil
}