package aicoder

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PTYManager manages multiple PTY sessions for AI coders
type PTYManager struct {
	sessions       map[string]*PTYSession
	activeSessions []string // Ordered list of session IDs
	currentSession string   // Currently focused session
	mu             sync.RWMutex
	
	// Brummer integration
	dataProvider BrummerDataProvider
	eventBus     EventBus
}

// NewPTYManager creates a new PTY session manager
func NewPTYManager(dataProvider BrummerDataProvider, eventBus EventBus) *PTYManager {
	return &PTYManager{
		sessions:     make(map[string]*PTYSession),
		dataProvider: dataProvider,
		eventBus:     eventBus,
	}
}

// CreateSession creates a new PTY session
func (pm *PTYManager) CreateSession(name, command string, args []string) (*PTYSession, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	sessionID := uuid.New().String()
	
	session, err := NewPTYSession(sessionID, name, command, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY session: %w", err)
	}
	
	pm.sessions[sessionID] = session
	pm.activeSessions = append(pm.activeSessions, sessionID)
	
	// Set as current session if it's the first one
	if pm.currentSession == "" {
		pm.currentSession = sessionID
	}
	
	// Start monitoring the session
	go pm.monitorSession(session)
	
	// Emit session created event
	if pm.eventBus != nil {
		pm.eventBus.Emit("pty_session_created", map[string]interface{}{
			"session_id": sessionID,
			"name":       name,
			"command":    command,
		})
	}
	
	return session, nil
}

// GetSession retrieves a PTY session by ID
func (pm *PTYManager) GetSession(sessionID string) (*PTYSession, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	session, exists := pm.sessions[sessionID]
	return session, exists
}

// GetCurrentSession returns the currently focused session
func (pm *PTYManager) GetCurrentSession() (*PTYSession, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if pm.currentSession == "" {
		return nil, false
	}
	
	session, exists := pm.sessions[pm.currentSession]
	return session, exists
}

// SetCurrentSession sets the currently focused session
func (pm *PTYManager) SetCurrentSession(sessionID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if _, exists := pm.sessions[sessionID]; !exists {
		return fmt.Errorf("session %s does not exist", sessionID)
	}
	
	pm.currentSession = sessionID
	
	// Emit session focus event
	if pm.eventBus != nil {
		pm.eventBus.Emit("pty_session_focused", map[string]interface{}{
			"session_id": sessionID,
		})
	}
	
	return nil
}

// NextSession switches to the next session in the list
func (pm *PTYManager) NextSession() (*PTYSession, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if len(pm.activeSessions) == 0 {
		return nil, fmt.Errorf("no active sessions")
	}
	
	// Find current session index
	currentIndex := -1
	for i, sessionID := range pm.activeSessions {
		if sessionID == pm.currentSession {
			currentIndex = i
			break
		}
	}
	
	// Move to next session (wrap around)
	nextIndex := (currentIndex + 1) % len(pm.activeSessions)
	pm.currentSession = pm.activeSessions[nextIndex]
	
	session := pm.sessions[pm.currentSession]
	
	// Emit session switch event
	if pm.eventBus != nil {
		pm.eventBus.Emit("pty_session_switched", map[string]interface{}{
			"session_id": pm.currentSession,
			"direction":  "next",
		})
	}
	
	return session, nil
}

// PreviousSession switches to the previous session in the list
func (pm *PTYManager) PreviousSession() (*PTYSession, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if len(pm.activeSessions) == 0 {
		return nil, fmt.Errorf("no active sessions")
	}
	
	// Find current session index
	currentIndex := -1
	for i, sessionID := range pm.activeSessions {
		if sessionID == pm.currentSession {
			currentIndex = i
			break
		}
	}
	
	// Move to previous session (wrap around)
	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(pm.activeSessions) - 1
	}
	pm.currentSession = pm.activeSessions[prevIndex]
	
	session := pm.sessions[pm.currentSession]
	
	// Emit session switch event
	if pm.eventBus != nil {
		pm.eventBus.Emit("pty_session_switched", map[string]interface{}{
			"session_id": pm.currentSession,
			"direction":  "previous",
		})
	}
	
	return session, nil
}

// CloseSession closes and removes a PTY session
func (pm *PTYManager) CloseSession(sessionID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	session, exists := pm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionID)
	}
	
	// Close the session
	if err := session.Close(); err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}
	
	// Remove from maps and lists
	delete(pm.sessions, sessionID)
	
	// Remove from active sessions list
	for i, id := range pm.activeSessions {
		if id == sessionID {
			pm.activeSessions = append(pm.activeSessions[:i], pm.activeSessions[i+1:]...)
			break
		}
	}
	
	// If this was the current session, switch to another one
	if pm.currentSession == sessionID {
		if len(pm.activeSessions) > 0 {
			pm.currentSession = pm.activeSessions[0]
		} else {
			pm.currentSession = ""
		}
	}
	
	// Emit session closed event
	if pm.eventBus != nil {
		pm.eventBus.Emit("pty_session_closed", map[string]interface{}{
			"session_id": sessionID,
		})
	}
	
	return nil
}

// ListSessions returns all active sessions
func (pm *PTYManager) ListSessions() []*PTYSession {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	sessions := make([]*PTYSession, 0, len(pm.activeSessions))
	for _, sessionID := range pm.activeSessions {
		if session, exists := pm.sessions[sessionID]; exists {
			sessions = append(sessions, session)
		}
	}
	
	return sessions
}

// InjectDataToCurrent injects data into the current session
func (pm *PTYManager) InjectDataToCurrent(dataType DataInjectionType) error {
	session, exists := pm.GetCurrentSession()
	if !exists {
		return fmt.Errorf("no current session")
	}
	
	data, err := pm.getDataForInjection(dataType)
	if err != nil {
		return fmt.Errorf("failed to get data for injection: %w", err)
	}
	
	return session.InjectData(dataType, data)
}

// InjectDataToSession injects data into a specific session
func (pm *PTYManager) InjectDataToSession(sessionID string, dataType DataInjectionType) error {
	session, exists := pm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("session %s does not exist", sessionID)
	}
	
	data, err := pm.getDataForInjection(dataType)
	if err != nil {
		return fmt.Errorf("failed to get data for injection: %w", err)
	}
	
	return session.InjectData(dataType, data)
}

// getDataForInjection retrieves data from Brummer based on injection type
func (pm *PTYManager) getDataForInjection(dataType DataInjectionType) (interface{}, error) {
	if pm.dataProvider == nil {
		return nil, fmt.Errorf("no data provider available")
	}
	
	switch dataType {
	case DataInjectError, DataInjectLastError:
		return pm.dataProvider.GetLastError(), nil
	case DataInjectLogs:
		return pm.dataProvider.GetRecentLogs(10), nil
	case DataInjectTestFailure:
		return pm.dataProvider.GetTestFailures(), nil
	case DataInjectBuildOutput:
		return pm.dataProvider.GetBuildOutput(), nil
	case DataInjectProcessInfo:
		return pm.dataProvider.GetProcessInfo(), nil
	case DataInjectURLs:
		return pm.dataProvider.GetDetectedURLs(), nil
	case DataInjectProxyReq:
		return pm.dataProvider.GetRecentProxyRequests(5), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// monitorSession monitors a PTY session for events
func (pm *PTYManager) monitorSession(session *PTYSession) {
	for event := range session.EventChan {
		// Handle session events
		switch event.Type {
		case PTYEventClose:
			// Auto-cleanup closed sessions
			pm.CloseSession(session.ID)
			
		case PTYEventOutput:
			// In debug mode, analyze output for triggers
			if session.IsDebugModeEnabled() {
				pm.analyzeOutputForAutoInjection(session, event.Data)
			}
		}
		
		// Forward events to main event bus
		if pm.eventBus != nil {
			pm.eventBus.Emit(string(event.Type), event)
		}
	}
}

// analyzeOutputForAutoInjection analyzes output for automatic data injection triggers
func (pm *PTYManager) analyzeOutputForAutoInjection(session *PTYSession, data interface{}) {
	if outputBytes, ok := data.([]byte); ok {
		output := string(outputBytes)
		
		// Look for error patterns and automatically inject relevant data
		if containsErrorPattern(output) {
			// Auto-inject last error
			go func() {
				time.Sleep(100 * time.Millisecond) // Small delay to avoid race conditions
				session.InjectData(DataInjectLastError, pm.dataProvider.GetLastError())
			}()
		}
		
		// Look for test failure patterns
		if containsTestFailurePattern(output) {
			go func() {
				time.Sleep(100 * time.Millisecond)
				session.InjectData(DataInjectTestFailure, pm.dataProvider.GetTestFailures())
			}()
		}
		
		// Look for build failure patterns
		if containsBuildFailurePattern(output) {
			go func() {
				time.Sleep(100 * time.Millisecond)
				session.InjectData(DataInjectBuildOutput, pm.dataProvider.GetBuildOutput())
			}()
		}
	}
}

// Helper functions for pattern matching
func containsErrorPattern(output string) bool {
	errorPatterns := []string{
		"error:",
		"Error:",
		"ERROR:",
		"failed",
		"Failed",
		"FAILED",
		"exception",
		"Exception",
	}
	
	for _, pattern := range errorPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}

func containsTestFailurePattern(output string) bool {
	testPatterns := []string{
		"test failed",
		"Test failed",
		"TEST FAILED",
		"FAIL:",
		"✗",
		"❌",
	}
	
	for _, pattern := range testPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}

func containsBuildFailurePattern(output string) bool {
	buildPatterns := []string{
		"build failed",
		"Build failed",
		"BUILD FAILED",
		"compilation error",
		"Compilation error",
		"compile error",
	}
	
	for _, pattern := range buildPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}
	return false
}

// GetSessionCount returns the number of active sessions
func (pm *PTYManager) GetSessionCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.activeSessions)
}

// CloseAllSessions closes all active sessions
func (pm *PTYManager) CloseAllSessions() error {
	pm.mu.Lock()
	sessionIDs := make([]string, len(pm.activeSessions))
	copy(sessionIDs, pm.activeSessions)
	pm.mu.Unlock()
	
	var lastError error
	for _, sessionID := range sessionIDs {
		if err := pm.CloseSession(sessionID); err != nil {
			lastError = err
		}
	}
	
	return lastError
}