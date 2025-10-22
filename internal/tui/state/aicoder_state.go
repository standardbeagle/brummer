package state

import (
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// AICoderState manages all AI coder terminal state
// This is a sub-model that encapsulates PTY/VT10x state according to Elm/Redux patterns
type AICoderState struct {
	// === NEW SCREEN BUFFER FIELDS ===

	// Screen buffer for active session only (rendered terminal screen)
	ScreenBuffer [][]rune

	// Cursor position in the screen buffer
	ScreenCursor Position

	// Flag indicating screen needs redraw
	ScreenDirty bool

	// Lightweight raw buffers for all sessions (100K lines max each)
	RawBuffers map[string]*RingBuffer

	// Track where we left off processing for each session
	ProcessedOffset map[string]int64

	// Debouncing state for smart mode
	LastUpdateTime map[string]time.Time
	UpdatePending  map[string]bool

	// === EXISTING FIELDS TO KEEP ===

	// Session metadata for tracking active sessions
	ActiveSessions map[string]SessionInfo

	// Current focused session for UI
	FocusedSession string

	// === FIELDS TO REMOVE (kept temporarily for migration) ===

	// PTY output buffers (session ID → lines of raw output) - DEPRECATED: use RawBuffers
	PTYOutputBuffers map[string][]string

	// VT10x terminal states (session ID → processed terminal state) - DEPRECATED: use ScreenBuffer
	TerminalStates map[string]*messages.TerminalState

	// Polling state management - DEPRECATED: use LastUpdateTime
	LastPollTime    time.Time
	PollingInterval time.Duration

	// Version for atomic updates and state tracking
	version uint64
}

// Position represents cursor position in the terminal screen
type Position struct {
	X int // Column (0-based)
	Y int // Row (0-based)
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
		// New fields
		ScreenBuffer:    make([][]rune, 0),
		ScreenCursor:    Position{X: 0, Y: 0},
		ScreenDirty:     false,
		RawBuffers:      make(map[string]*RingBuffer),
		ProcessedOffset: make(map[string]int64),
		LastUpdateTime:  make(map[string]time.Time),
		UpdatePending:   make(map[string]bool),

		// Existing fields
		ActiveSessions: make(map[string]SessionInfo),
		FocusedSession: "",

		// Deprecated fields (kept for migration)
		PTYOutputBuffers: make(map[string][]string),
		TerminalStates:   make(map[string]*messages.TerminalState),
		PollingInterval:  100 * time.Millisecond,

		version: 0,
	}
}

// === Pure State Transformations (Elm/Redux Pattern) ===

// === NEW SCREEN BUFFER METHODS ===

// UpdateScreenBuffer updates the rendered terminal screen for the active session
func (s *AICoderState) UpdateScreenBuffer(buffer [][]rune, cursor Position) *AICoderState {
	newState := s.copy()
	newState.ScreenBuffer = buffer
	newState.ScreenCursor = cursor
	newState.ScreenDirty = true
	newState.incrementVersion()
	return newState
}

// MarkScreenClean marks the screen as rendered (no longer dirty)
func (s *AICoderState) MarkScreenClean() *AICoderState {
	newState := s.copy()
	newState.ScreenDirty = false
	newState.incrementVersion()
	return newState
}

// AddRawOutput adds raw PTY output to a session's ring buffer
func (s *AICoderState) AddRawOutput(sessionID string, data []byte) *AICoderState {
	newState := s.copy()

	// Get or create ring buffer for session
	buffer, exists := newState.RawBuffers[sessionID]
	if !exists {
		// Create new ring buffer with 8MB capacity (100K lines * ~80 bytes/line)
		buffer = NewRingBuffer(8 * 1024 * 1024)
		newState.RawBuffers[sessionID] = buffer
		newState.ProcessedOffset[sessionID] = 0
	}

	// Append data to ring buffer
	buffer.Write(data)

	// Mark as having pending updates
	newState.UpdatePending[sessionID] = true
	newState.LastUpdateTime[sessionID] = time.Now()

	newState.incrementVersion()
	return newState
}

// SetProcessedOffset updates where we've processed up to for a session
func (s *AICoderState) SetProcessedOffset(sessionID string, offset int64) *AICoderState {
	newState := s.copy()
	newState.ProcessedOffset[sessionID] = offset
	newState.UpdatePending[sessionID] = false
	newState.incrementVersion()
	return newState
}

// SwitchActiveSession handles switching the active session
func (s *AICoderState) SwitchActiveSession(newSessionID string) *AICoderState {
	newState := s.copy()

	// Clear current screen buffer (will be repopulated from new session)
	newState.ScreenBuffer = make([][]rune, 0)
	newState.ScreenCursor = Position{X: 0, Y: 0}
	newState.ScreenDirty = true

	// Update focused session
	newState.FocusedSession = newSessionID

	// Mark that we need to process this session's buffer
	if _, exists := newState.RawBuffers[newSessionID]; exists {
		newState.UpdatePending[newSessionID] = true
	}

	newState.incrementVersion()
	return newState
}

// === DEPRECATED METHODS (kept for migration) ===

// AddPTYOutput adds PTY output for a session (pure function)
// DEPRECATED: Use AddRawOutput instead
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
		// New fields
		ScreenDirty:     s.ScreenDirty,
		ScreenCursor:    s.ScreenCursor,
		RawBuffers:      make(map[string]*RingBuffer, len(s.RawBuffers)),
		ProcessedOffset: make(map[string]int64, len(s.ProcessedOffset)),
		LastUpdateTime:  make(map[string]time.Time, len(s.LastUpdateTime)),
		UpdatePending:   make(map[string]bool, len(s.UpdatePending)),

		// Existing fields
		ActiveSessions: make(map[string]SessionInfo, len(s.ActiveSessions)),
		FocusedSession: s.FocusedSession,

		// Deprecated fields
		PTYOutputBuffers: make(map[string][]string, len(s.PTYOutputBuffers)),
		TerminalStates:   make(map[string]*messages.TerminalState, len(s.TerminalStates)),
		LastPollTime:     s.LastPollTime,
		PollingInterval:  s.PollingInterval,

		version: s.version,
	}

	// Copy screen buffer (2D array of runes)
	if len(s.ScreenBuffer) > 0 {
		newState.ScreenBuffer = make([][]rune, len(s.ScreenBuffer))
		for i, line := range s.ScreenBuffer {
			newState.ScreenBuffer[i] = append([]rune{}, line...)
		}
	}

	// Copy ring buffers (they are immutable, so just copy references)
	for k, v := range s.RawBuffers {
		newState.RawBuffers[k] = v
	}

	// Copy processed offsets
	for k, v := range s.ProcessedOffset {
		newState.ProcessedOffset[k] = v
	}

	// Copy update times
	for k, v := range s.LastUpdateTime {
		newState.LastUpdateTime[k] = v
	}

	// Copy pending flags
	for k, v := range s.UpdatePending {
		newState.UpdatePending[k] = v
	}

	// Copy session info
	for k, v := range s.ActiveSessions {
		newState.ActiveSessions[k] = v
	}

	// Copy deprecated PTY output buffers (for migration)
	for k, v := range s.PTYOutputBuffers {
		newState.PTYOutputBuffers[k] = append([]string{}, v...)
	}

	// Copy deprecated terminal states (they are immutable already)
	for k, v := range s.TerminalStates {
		newState.TerminalStates[k] = v
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