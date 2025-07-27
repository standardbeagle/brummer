package mcp

import (
	"context"
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

	// Context lifecycle management
	ctx    context.Context
	cancel context.CancelFunc

	// Instance connection
	ConnectedInstance string // ID of connected instance

	// Connection contexts per instance
	connectionContexts map[string]*ConnectionLifecycleContext
	connMu             sync.RWMutex

	// Registered items from connected instance
	RegisteredTools     map[string]bool // tool name -> registered
	RegisteredResources map[string]bool // resource URI -> registered
	RegisteredPrompts   map[string]bool // prompt name -> registered

	// Activity tracking
	ToolCalls      int
	ResourceReads  int
	PromptRequests int

	// Error tracking
	LastError  error
	ErrorCount int

	// Metadata
	ClientInfo map[string]string
	CustomData map[string]interface{}

	mu sync.RWMutex
}

// ConnectionLifecycleContext manages the context lifecycle for a specific connection
type ConnectionLifecycleContext struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Operation contexts
	operations map[string]*OperationLifecycleContext
	opMu       sync.RWMutex

	// Connection metadata
	instanceID string
	createdAt  time.Time
	lastUsed   time.Time
}

// OperationLifecycleContext manages context for specific operations
type OperationLifecycleContext struct {
	ctx       context.Context
	cancel    context.CancelFunc
	operation string
	createdAt time.Time
	timeout   time.Duration
}

// SessionManager manages all client sessions and their contexts
type SessionManager struct {
	sessions map[string]*SessionContext
	mu       sync.RWMutex

	// Context management
	defaultTimeout   time.Duration
	operationTimeout time.Duration
	cleanupInterval  time.Duration

	// Lifecycle
	stopCh chan struct{}
	wg     sync.WaitGroup

	// Callbacks
	onConnect    func(sessionID, instanceID string)
	onDisconnect func(sessionID, instanceID string)
	onError      func(sessionID string, err error)
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:         make(map[string]*SessionContext),
		defaultTimeout:   24 * time.Hour,   // Long-lived sessions
		operationTimeout: 30 * time.Second, // Individual operations
		cleanupInterval:  5 * time.Minute,
		stopCh:           make(chan struct{}),
	}
}

// Start begins the session manager lifecycle
func (sm *SessionManager) Start() {
	sm.wg.Add(1)
	go sm.cleanupLoop()
}

// Stop gracefully shuts down the session manager
func (sm *SessionManager) Stop() {
	close(sm.stopCh)

	// Cancel all active sessions
	sm.mu.Lock()
	for _, session := range sm.sessions {
		if session.cancel != nil {
			session.cancel()
		}
	}
	sm.mu.Unlock()

	sm.wg.Wait()
}

// CreateSession creates a new session context
func (sm *SessionManager) CreateSession(sessionID string, clientInfo map[string]string) *SessionContext {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Cancel existing session if it exists
	if existing, exists := sm.sessions[sessionID]; exists && existing.cancel != nil {
		existing.cancel()
	}

	// Create session context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), sm.defaultTimeout)

	session := &SessionContext{
		ID:                  sessionID,
		ConnectedAt:         time.Now(),
		LastActivity:        time.Now(),
		ctx:                 ctx,
		cancel:              cancel,
		connectionContexts:  make(map[string]*ConnectionLifecycleContext),
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

		// Cancel session context and all connection contexts
		if session.cancel != nil {
			session.cancel()
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
		"id":                   session.ID,
		"connected_at":         session.ConnectedAt,
		"last_activity":        session.LastActivity,
		"connected_instance":   session.ConnectedInstance,
		"tool_calls":           session.ToolCalls,
		"resource_reads":       session.ResourceReads,
		"prompt_requests":      session.PromptRequests,
		"error_count":          session.ErrorCount,
		"registered_tools":     len(session.RegisteredTools),
		"registered_resources": len(session.RegisteredResources),
		"registered_prompts":   len(session.RegisteredPrompts),
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

// Context Lifecycle Management Methods

// GetSessionContext returns the context for a session
func (sm *SessionManager) GetSessionContext(sessionID string) (context.Context, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session.ctx, nil
}

// CreateConnectionContext creates a connection context within a session
func (sc *SessionContext) CreateConnectionContext(instanceID string) *ConnectionLifecycleContext {
	sc.connMu.Lock()
	defer sc.connMu.Unlock()

	// Cancel existing connection context if it exists
	if existing, exists := sc.connectionContexts[instanceID]; exists && existing.cancel != nil {
		existing.cancel()
	}

	// Create connection context derived from session context
	ctx, cancel := context.WithCancel(sc.ctx)

	connCtx := &ConnectionLifecycleContext{
		ctx:        ctx,
		cancel:     cancel,
		operations: make(map[string]*OperationLifecycleContext),
		instanceID: instanceID,
		createdAt:  time.Now(),
		lastUsed:   time.Now(),
	}

	sc.connectionContexts[instanceID] = connCtx
	return connCtx
}

// GetConnectionContext retrieves a connection context
func (sc *SessionContext) GetConnectionContext(instanceID string) (*ConnectionLifecycleContext, bool) {
	sc.connMu.RLock()
	defer sc.connMu.RUnlock()

	connCtx, exists := sc.connectionContexts[instanceID]
	if exists {
		connCtx.lastUsed = time.Now()
	}
	return connCtx, exists
}

// GetOrCreateConnectionContext gets existing or creates new connection context
func (sc *SessionContext) GetOrCreateConnectionContext(instanceID string) *ConnectionLifecycleContext {
	if connCtx, exists := sc.GetConnectionContext(instanceID); exists {
		return connCtx
	}
	return sc.CreateConnectionContext(instanceID)
}

// RemoveConnectionContext removes a connection context
func (sc *SessionContext) RemoveConnectionContext(instanceID string) {
	sc.connMu.Lock()
	defer sc.connMu.Unlock()

	if connCtx, exists := sc.connectionContexts[instanceID]; exists && connCtx.cancel != nil {
		connCtx.cancel()
		delete(sc.connectionContexts, instanceID)
	}
}

// Context returns the session context
func (sc *SessionContext) Context() context.Context {
	return sc.ctx
}

// CreateOperationContext creates a context for a specific operation
func (cc *ConnectionLifecycleContext) CreateOperationContext(operation string, timeout time.Duration) *OperationLifecycleContext {
	cc.opMu.Lock()
	defer cc.opMu.Unlock()

	// Cancel existing operation context if it exists
	if existing, exists := cc.operations[operation]; exists && existing.cancel != nil {
		existing.cancel()
	}

	// Create operation context with timeout
	ctx, cancel := context.WithTimeout(cc.ctx, timeout)

	opCtx := &OperationLifecycleContext{
		ctx:       ctx,
		cancel:    cancel,
		operation: operation,
		createdAt: time.Now(),
		timeout:   timeout,
	}

	cc.operations[operation] = opCtx
	return opCtx
}

// GetOperationContext retrieves an operation context
func (cc *ConnectionLifecycleContext) GetOperationContext(operation string) (*OperationLifecycleContext, bool) {
	cc.opMu.RLock()
	defer cc.opMu.RUnlock()

	opCtx, exists := cc.operations[operation]
	return opCtx, exists
}

// RemoveOperationContext removes an operation context
func (cc *ConnectionLifecycleContext) RemoveOperationContext(operation string) {
	cc.opMu.Lock()
	defer cc.opMu.Unlock()

	if opCtx, exists := cc.operations[operation]; exists && opCtx.cancel != nil {
		opCtx.cancel()
		delete(cc.operations, operation)
	}
}

// Context returns the connection context
func (cc *ConnectionLifecycleContext) Context() context.Context {
	return cc.ctx
}

// Cancel cancels the connection context
func (cc *ConnectionLifecycleContext) Cancel() {
	if cc.cancel != nil {
		cc.cancel()
	}
}

// Context returns the operation context
func (oc *OperationLifecycleContext) Context() context.Context {
	return oc.ctx
}

// Cancel cancels the operation context
func (oc *OperationLifecycleContext) Cancel() {
	if oc.cancel != nil {
		oc.cancel()
	}
}

// cleanupLoop periodically cleans up expired contexts
func (sm *SessionManager) cleanupLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupExpiredContexts()
		case <-sm.stopCh:
			return
		}
	}
}

// cleanupExpiredContexts removes expired or cancelled sessions and contexts
func (sm *SessionManager) cleanupExpiredContexts() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expiredSessions := []string{}

	for sessionID, session := range sm.sessions {
		// Check if session context is cancelled or expired
		select {
		case <-session.ctx.Done():
			expiredSessions = append(expiredSessions, sessionID)
		default:
			// Check for inactivity (no usage for 2 hours)
			if now.Sub(session.LastActivity) > 2*time.Hour {
				expiredSessions = append(expiredSessions, sessionID)
			} else {
				// Cleanup expired connection contexts within active sessions
				session.cleanupExpiredConnections(now)
			}
		}
	}

	// Remove expired sessions
	for _, sessionID := range expiredSessions {
		if session, exists := sm.sessions[sessionID]; exists {
			if session.cancel != nil {
				session.cancel()
			}
			delete(sm.sessions, sessionID)
		}
	}

	if len(expiredSessions) > 0 {
		debugLog("Cleaned up %d expired sessions", len(expiredSessions))
	}
}

// cleanupExpiredConnections removes expired connection contexts within a session
func (sc *SessionContext) cleanupExpiredConnections(now time.Time) {
	sc.connMu.Lock()
	defer sc.connMu.Unlock()

	expiredConnections := []string{}

	for instanceID, connCtx := range sc.connectionContexts {
		// Check if connection context is cancelled or inactive
		select {
		case <-connCtx.ctx.Done():
			expiredConnections = append(expiredConnections, instanceID)
		default:
			// Check for inactivity (no usage for 30 minutes)
			if now.Sub(connCtx.lastUsed) > 30*time.Minute {
				expiredConnections = append(expiredConnections, instanceID)
			} else {
				// Cleanup expired operations within active connections
				connCtx.cleanupExpiredOperations(now)
			}
		}
	}

	// Remove expired connections
	for _, instanceID := range expiredConnections {
		if connCtx, exists := sc.connectionContexts[instanceID]; exists {
			if connCtx.cancel != nil {
				connCtx.cancel()
			}
			delete(sc.connectionContexts, instanceID)
		}
	}
}

// cleanupExpiredOperations removes expired operation contexts within a connection
func (cc *ConnectionLifecycleContext) cleanupExpiredOperations(now time.Time) {
	cc.opMu.Lock()
	defer cc.opMu.Unlock()

	expiredOperations := []string{}

	for operation, opCtx := range cc.operations {
		// Check if operation context is cancelled or expired
		select {
		case <-opCtx.ctx.Done():
			expiredOperations = append(expiredOperations, operation)
		default:
			// Check if operation has been running too long (more than 2x its timeout)
			if now.Sub(opCtx.createdAt) > opCtx.timeout*2 {
				expiredOperations = append(expiredOperations, operation)
			}
		}
	}

	// Remove expired operations
	for _, operation := range expiredOperations {
		if opCtx, exists := cc.operations[operation]; exists {
			if opCtx.cancel != nil {
				opCtx.cancel()
			}
			delete(cc.operations, operation)
		}
	}
}
