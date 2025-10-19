package state

import (
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// AICoderState manages all AI coder terminal state
// This is a sub-model that encapsulates PTY/VT10x state according to Elm/Redux patterns
type AICoderState struct {
	// PTY output buffers (session ID → lines of raw output)
	PTYOutputBuffers map[string][]string

	// VT10x terminal states (session ID → processed terminal state)
	TerminalStates map[string]*messages.TerminalState

	// Session metadata for tracking active sessions
	ActiveSessions map[string]SessionInfo

	// Current focused session for UI
	FocusedSession string

	// Polling state management
	LastPollTime    time.Time
	PollingInterval time.Duration

	// Version for atomic updates and state tracking
	version uint64
}

// SessionInfo contains metadata about an AI coder session
type SessionInfo struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"` // "active", "idle", "ended"
	Created  time.Time `json:"created"`
	LastSeen time.Time `json:"lastSeen"`
}

// NewAICoderState creates a new AI coder sub-model with initial state
func NewAICoderState() *AICoderState {
	return &AICoderState{
		PTYOutputBuffers: make(map[string][]string),
		TerminalStates:   make(map[string]*messages.TerminalState),
		ActiveSessions:   make(map[string]SessionInfo),
		FocusedSession:   "",
		PollingInterval:  100 * time.Millisecond,
		version:          0,
	}
}

// === Pure State Transformations (Elm/Redux Pattern) ===

// AddPTYOutput adds PTY output for a session (pure function)
// Returns new state without modifying original (immutable update)
func (s *AICoderState) AddPTYOutput(sessionID string, output string) *AICoderState {
	// Copy on write for immutability
	newState := s.copy()

	// Split output into lines
	lines := strings.Split(output, "\n")

	// Get existing buffer or create new one
	existingBuffer := newState.PTYOutputBuffers[sessionID]
	buffer := make([]string, 0, len(existingBuffer)+len(lines))
	buffer = append(buffer, existingBuffer...)

	// Add new lines, filtering out empty strings at the end
	for _, line := range lines {
		if line != "" || len(buffer) == 0 || buffer[len(buffer)-1] != "" {
			buffer = append(buffer, line)
		}
	}

	// Limit buffer size to prevent memory issues
	if len(buffer) > 1000 {
		buffer = buffer[len(buffer)-1000:]
	}

	newState.PTYOutputBuffers[sessionID] = buffer
	newState.updateSessionActivity(sessionID)
	newState.incrementVersion()

	return newState
}

// UpdateTerminalState updates VT10x terminal state for a session
func (s *AICoderState) UpdateTerminalState(sessionID string, state *messages.TerminalState) *AICoderState {
	newState := s.copy()
	newState.TerminalStates[sessionID] = state
	newState.updateSessionActivity(sessionID)
	newState.incrementVersion()
	return newState
}

// SetFocusedSession changes the focused session
func (s *AICoderState) SetFocusedSession(sessionID string) *AICoderState {
	newState := s.copy()
	newState.FocusedSession = sessionID
	newState.incrementVersion()
	return newState
}

// AddSession adds a new session to the active sessions
func (s *AICoderState) AddSession(sessionID, sessionName string) *AICoderState {
	newState := s.copy()
	newState.ActiveSessions[sessionID] = SessionInfo{
		ID:       sessionID,
		Name:     sessionName,
		Status:   "active",
		Created:  time.Now(),
		LastSeen: time.Now(),
	}

	// Initialize buffers for new session
	if _, exists := newState.PTYOutputBuffers[sessionID]; !exists {
		newState.PTYOutputBuffers[sessionID] = []string{}
	}

	newState.incrementVersion()
	return newState
}

// UpdateSessionStatus updates the status of a session
func (s *AICoderState) UpdateSessionStatus(sessionID, status string) *AICoderState {
	newState := s.copy()
	if session, exists := newState.ActiveSessions[sessionID]; exists {
		session.Status = status
		session.LastSeen = time.Now()
		newState.ActiveSessions[sessionID] = session
		newState.incrementVersion()
	}
	return newState
}

// RemoveSession removes a session and cleans up its data
func (s *AICoderState) RemoveSession(sessionID string) *AICoderState {
	newState := s.copy()
	delete(newState.ActiveSessions, sessionID)
	delete(newState.PTYOutputBuffers, sessionID)
	delete(newState.TerminalStates, sessionID)

	// Clear focused session if it was the removed one
	if newState.FocusedSession == sessionID {
		newState.FocusedSession = ""
	}

	newState.incrementVersion()
	return newState
}

// UpdatePollingTime updates the last poll time
func (s *AICoderState) UpdatePollingTime() *AICoderState {
	newState := s.copy()
	newState.LastPollTime = time.Now()
	newState.incrementVersion()
	return newState
}

// === Query Methods (Pure Functions) ===

// GetPTYOutput returns the PTY output lines for a session
func (s *AICoderState) GetPTYOutput(sessionID string) []string {
	if buffer, exists := s.PTYOutputBuffers[sessionID]; exists {
		return append([]string{}, buffer...) // Return copy
	}
	return []string{}
}

// GetTerminalState returns the terminal state for a session
func (s *AICoderState) GetTerminalState(sessionID string) *messages.TerminalState {
	if state, exists := s.TerminalStates[sessionID]; exists {
		return state
	}
	return nil
}

// GetActiveSessionIDs returns IDs of all active sessions
func (s *AICoderState) GetActiveSessionIDs() []string {
	ids := make([]string, 0, len(s.ActiveSessions))
	for id := range s.ActiveSessions {
		ids = append(ids, id)
	}
	return ids
}

// GetSessionInfo returns information about a session
func (s *AICoderState) GetSessionInfo(sessionID string) (SessionInfo, bool) {
	info, exists := s.ActiveSessions[sessionID]
	return info, exists
}

// HasActiveSessions returns true if there are active sessions
func (s *AICoderState) HasActiveSessions() bool {
	return len(s.ActiveSessions) > 0
}

// GetVersion returns the current version of the state
func (s *AICoderState) GetVersion() uint64 {
	return s.version
}

// === Internal Helper Methods ===

// copy creates a deep copy of the state for immutability
func (s *AICoderState) copy() *AICoderState {
	newState := &AICoderState{
		PTYOutputBuffers: make(map[string][]string, len(s.PTYOutputBuffers)),
		TerminalStates:   make(map[string]*messages.TerminalState, len(s.TerminalStates)),
		ActiveSessions:   make(map[string]SessionInfo, len(s.ActiveSessions)),
		FocusedSession:   s.FocusedSession,
		LastPollTime:     s.LastPollTime,
		PollingInterval:  s.PollingInterval,
		version:          s.version,
	}

	// Copy PTY output buffers
	for k, v := range s.PTYOutputBuffers {
		newState.PTYOutputBuffers[k] = append([]string{}, v...)
	}

	// Copy terminal states (they are immutable already)
	for k, v := range s.TerminalStates {
		newState.TerminalStates[k] = v
	}

	// Copy session info
	for k, v := range s.ActiveSessions {
		newState.ActiveSessions[k] = v
	}

	return newState
}

// incrementVersion increments the state version
func (s *AICoderState) incrementVersion() {
	s.version++
}

// updateSessionActivity updates the last seen time for a session
func (s *AICoderState) updateSessionActivity(sessionID string) {
	if session, exists := s.ActiveSessions[sessionID]; exists {
		session.LastSeen = time.Now()
		s.ActiveSessions[sessionID] = session
	}
}

// === State Validation ===

// Validate performs basic validation of the state
func (s *AICoderState) Validate() []string {
	var issues []string

	// Check for session data consistency
	for sessionID := range s.PTYOutputBuffers {
		if _, exists := s.ActiveSessions[sessionID]; !exists {
			issues = append(issues, "PTY output buffer exists for unknown session: "+sessionID)
		}
	}

	for sessionID := range s.TerminalStates {
		if _, exists := s.ActiveSessions[sessionID]; !exists {
			issues = append(issues, "Terminal state exists for unknown session: "+sessionID)
		}
	}

	// Check focused session validity
	if s.FocusedSession != "" {
		if _, exists := s.ActiveSessions[s.FocusedSession]; !exists {
			issues = append(issues, "Focused session does not exist: "+s.FocusedSession)
		}
	}

	return issues
}

// GetStateSummary returns a summary of the current state for debugging
func (s *AICoderState) GetStateSummary() map[string]interface{} {
	return map[string]interface{}{
		"active_sessions":    len(s.ActiveSessions),
		"focused_session":    s.FocusedSession,
		"pty_buffers":        len(s.PTYOutputBuffers),
		"terminal_states":    len(s.TerminalStates),
		"version":            s.version,
		"last_poll_time":     s.LastPollTime,
		"polling_interval":   s.PollingInterval.String(),
	}
}