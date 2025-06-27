package mcp

import (
	"fmt"
	"sync"
	"time"
)

// SessionContext tracks the context and state for a client session
type SessionContext struct {
	// Basic info
	ID           string
	ConnectedAt  time.Time
	LastActivity time.Time
	
	// Instance connection
	ConnectedInstance string // ID of connected instance
	
	// Registered items from connected instance
	RegisteredTools     map[string]bool // tool name -> registered
	RegisteredResources map[string]bool // resource URI -> registered
	RegisteredPrompts   map[string]bool // prompt name -> registered
	
	// Activity tracking
	ToolCalls      int
	ResourceReads  int
	PromptRequests int
	
	// Error tracking
	LastError    error
	ErrorCount   int
	
	// Metadata
	ClientInfo   map[string]string
	CustomData   map[string]interface{}
	
	mu sync.RWMutex
}

// SessionManager manages all client sessions and their contexts
type SessionManager struct {
	sessions map[string]*SessionContext
	mu       sync.RWMutex
	
	// Callbacks
	onConnect    func(sessionID, instanceID string)
	onDisconnect func(sessionID, instanceID string)
	onError      func(sessionID string, err error)
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionContext),
	}
}

// CreateSession creates a new session context
func (sm *SessionManager) CreateSession(sessionID string, clientInfo map[string]string) *SessionContext {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session := &SessionContext{
		ID:                  sessionID,
		ConnectedAt:         time.Now(),
		LastActivity:        time.Now(),
		RegisteredTools:     make(map[string]bool),
		RegisteredResources: make(map[string]bool),
		RegisteredPrompts:   make(map[string]bool),
		ClientInfo:          clientInfo,
		CustomData:          make(map[string]interface{}),
	}
	
	sm.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*SessionContext, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	
	return session, nil
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		// Trigger disconnect callback if connected
		if session.ConnectedInstance != "" && sm.onDisconnect != nil {
			sm.onDisconnect(sessionID, session.ConnectedInstance)
		}
		
		delete(sm.sessions, sessionID)
	}
}

// ConnectToInstance connects a session to an instance
func (sm *SessionManager) ConnectToInstance(sessionID, instanceID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}
	
	// Disconnect from previous instance if connected
	if session.ConnectedInstance != "" && session.ConnectedInstance != instanceID {
		if sm.onDisconnect != nil {
			sm.onDisconnect(sessionID, session.ConnectedInstance)
		}
		
		// Clear registered items
		session.RegisteredTools = make(map[string]bool)
		session.RegisteredResources = make(map[string]bool)
		session.RegisteredPrompts = make(map[string]bool)
	}
	
	session.ConnectedInstance = instanceID
	session.LastActivity = time.Now()
	
	// Trigger connect callback
	if sm.onConnect != nil {
		sm.onConnect(sessionID, instanceID)
	}
	
	return nil
}

// DisconnectFromInstance disconnects a session from its instance
func (sm *SessionManager) DisconnectFromInstance(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}
	
	if session.ConnectedInstance != "" {
		// Trigger disconnect callback
		if sm.onDisconnect != nil {
			sm.onDisconnect(sessionID, session.ConnectedInstance)
		}
		
		session.ConnectedInstance = ""
		
		// Clear registered items
		session.RegisteredTools = make(map[string]bool)
		session.RegisteredResources = make(map[string]bool)
		session.RegisteredPrompts = make(map[string]bool)
	}
	
	session.LastActivity = time.Now()
	return nil
}

// RecordToolRegistration records that a tool was registered for a session
func (sm *SessionManager) RecordToolRegistration(sessionID, toolName string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		session.RegisteredTools[toolName] = true
		session.mu.Unlock()
	}
}

// RecordResourceRegistration records that a resource was registered for a session
func (sm *SessionManager) RecordResourceRegistration(sessionID, resourceURI string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		session.RegisteredResources[resourceURI] = true
		session.mu.Unlock()
	}
}

// RecordPromptRegistration records that a prompt was registered for a session
func (sm *SessionManager) RecordPromptRegistration(sessionID, promptName string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		session.RegisteredPrompts[promptName] = true
		session.mu.Unlock()
	}
}

// RecordActivity records activity for a session
func (sm *SessionManager) RecordActivity(sessionID string, activityType string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		defer session.mu.Unlock()
		
		session.LastActivity = time.Now()
		
		switch activityType {
		case "tool_call":
			session.ToolCalls++
		case "resource_read":
			session.ResourceReads++
		case "prompt_request":
			session.PromptRequests++
		}
	}
}

// RecordError records an error for a session
func (sm *SessionManager) RecordError(sessionID string, err error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		session.mu.Lock()
		session.LastError = err
		session.ErrorCount++
		session.LastActivity = time.Now()
		session.mu.Unlock()
		
		// Trigger error callback
		if sm.onError != nil {
			sm.onError(sessionID, err)
		}
	}
}

// GetSessionStats returns statistics for a session
func (sm *SessionManager) GetSessionStats(sessionID string) (map[string]interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	
	session.mu.RLock()
	defer session.mu.RUnlock()
	
	stats := map[string]interface{}{
		"id":                session.ID,
		"connected_at":      session.ConnectedAt,
		"last_activity":     session.LastActivity,
		"connected_instance": session.ConnectedInstance,
		"tool_calls":        session.ToolCalls,
		"resource_reads":    session.ResourceReads,
		"prompt_requests":   session.PromptRequests,
		"error_count":       session.ErrorCount,
		"registered_tools":  len(session.RegisteredTools),
		"registered_resources": len(session.RegisteredResources),
		"registered_prompts": len(session.RegisteredPrompts),
	}
	
	if session.LastError != nil {
		stats["last_error"] = session.LastError.Error()
	}
	
	return stats, nil
}

// GetAllSessions returns all active sessions
func (sm *SessionManager) GetAllSessions() map[string]*SessionContext {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Create a copy to avoid races
	sessions := make(map[string]*SessionContext)
	for id, session := range sm.sessions {
		sessions[id] = session
	}
	
	return sessions
}

// CleanupInactiveSessions removes sessions that have been inactive for too long
func (sm *SessionManager) CleanupInactiveSessions(maxInactivity time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	now := time.Now()
	removed := 0
	
	for id, session := range sm.sessions {
		if now.Sub(session.LastActivity) > maxInactivity {
			// Trigger disconnect callback if connected
			if session.ConnectedInstance != "" && sm.onDisconnect != nil {
				sm.onDisconnect(id, session.ConnectedInstance)
			}
			
			delete(sm.sessions, id)
			removed++
		}
	}
	
	return removed
}

// SetCallbacks sets the callback functions
func (sm *SessionManager) SetCallbacks(
	onConnect func(sessionID, instanceID string),
	onDisconnect func(sessionID, instanceID string),
	onError func(sessionID string, err error),
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sm.onConnect = onConnect
	sm.onDisconnect = onDisconnect
	sm.onError = onError
}

// GetInstanceSessions returns all sessions connected to a specific instance
func (sm *SessionManager) GetInstanceSessions(instanceID string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var sessions []string
	for id, session := range sm.sessions {
		if session.ConnectedInstance == instanceID {
			sessions = append(sessions, id)
		}
	}
	
	return sessions
}

// DisconnectAllFromInstance disconnects all sessions from a specific instance
func (sm *SessionManager) DisconnectAllFromInstance(instanceID string) int {
	sessions := sm.GetInstanceSessions(instanceID)
	
	for _, sessionID := range sessions {
		sm.DisconnectFromInstance(sessionID)
	}
	
	return len(sessions)
}