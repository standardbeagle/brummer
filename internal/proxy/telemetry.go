package proxy

import (
	"sync"
	"time"
)

// TelemetryEventType represents the type of telemetry event
type TelemetryEventType string

const (
	TelemetryPageLoad         TelemetryEventType = "page_load"
	TelemetryDOMState         TelemetryEventType = "dom_state"
	TelemetryVisibilityChange TelemetryEventType = "visibility_change"
	TelemetryPerformance      TelemetryEventType = "performance_metrics"
	TelemetryMemoryUsage      TelemetryEventType = "memory_usage"
	TelemetryConsoleOutput    TelemetryEventType = "console_output"
	TelemetryJSError          TelemetryEventType = "javascript_error"
	TelemetryUnhandledReject  TelemetryEventType = "unhandled_rejection"
	TelemetryUserInteraction  TelemetryEventType = "user_interaction"
	TelemetryResourceTiming   TelemetryEventType = "resource_timing"
	TelemetryMonitorInit      TelemetryEventType = "monitor_initialized"
)

// TelemetryEvent represents a single telemetry event from the browser
type TelemetryEvent struct {
	Type      TelemetryEventType     `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	SessionID string                 `json:"sessionId"`
	URL       string                 `json:"url"`
	Data      map[string]interface{} `json:"data"`
}

// TelemetryBatch represents a batch of telemetry events
type TelemetryBatch struct {
	SessionID string           `json:"sessionId"`
	Events    []TelemetryEvent `json:"events"`
	Received  time.Time        `json:"received"`
}

// PageSession represents monitoring data for a single page session
type PageSession struct {
	SessionID    string
	URL          string
	ProcessName  string
	StartTime    time.Time
	LastActivity time.Time
	Events       []TelemetryEvent
	
	// Aggregated metrics
	PerformanceMetrics map[string]interface{}
	MemorySnapshots    []map[string]interface{}
	ErrorCount         int
	InteractionCount   int
	ConsoleLogCount    map[string]int // Count by level (log, warn, error, etc)
}

// TelemetryStore manages telemetry data
type TelemetryStore struct {
	mu       sync.RWMutex
	sessions map[string]*PageSession
	
	// Configuration
	maxSessionsPerProcess int
	maxEventsPerSession   int
	sessionTimeout        time.Duration
}

// NewTelemetryStore creates a new telemetry store
func NewTelemetryStore() *TelemetryStore {
	return &TelemetryStore{
		sessions:              make(map[string]*PageSession),
		maxSessionsPerProcess: 100,
		maxEventsPerSession:   1000,
		sessionTimeout:        30 * time.Minute,
	}
}

// AddBatch adds a batch of telemetry events
func (ts *TelemetryStore) AddBatch(batch TelemetryBatch, processName string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	// Get or create session
	session, exists := ts.sessions[batch.SessionID]
	if !exists {
		session = &PageSession{
			SessionID:       batch.SessionID,
			ProcessName:     processName,
			StartTime:       time.Now(),
			Events:          make([]TelemetryEvent, 0, ts.maxEventsPerSession),
			ConsoleLogCount: make(map[string]int),
		}
		ts.sessions[batch.SessionID] = session
	}
	
	// Update session
	session.LastActivity = time.Now()
	
	// Add events
	for _, event := range batch.Events {
		// Update URL if provided
		if event.URL != "" {
			session.URL = event.URL
		}
		
		// Process event based on type
		ts.processEvent(session, event)
		
		// Add to event list (with size limit)
		if len(session.Events) < ts.maxEventsPerSession {
			session.Events = append(session.Events, event)
		}
	}
	
	// Clean up old sessions
	ts.cleanupOldSessions()
}

// processEvent processes a telemetry event and updates aggregated metrics
func (ts *TelemetryStore) processEvent(session *PageSession, event TelemetryEvent) {
	switch event.Type {
	case TelemetryPerformance:
		if session.PerformanceMetrics == nil {
			session.PerformanceMetrics = event.Data
		}
		
	case TelemetryMemoryUsage:
		if session.MemorySnapshots == nil {
			session.MemorySnapshots = make([]map[string]interface{}, 0)
		}
		session.MemorySnapshots = append(session.MemorySnapshots, event.Data)
		
		// Keep only last 20 snapshots
		if len(session.MemorySnapshots) > 20 {
			session.MemorySnapshots = session.MemorySnapshots[len(session.MemorySnapshots)-20:]
		}
		
	case TelemetryJSError, TelemetryUnhandledReject:
		session.ErrorCount++
		
	case TelemetryUserInteraction:
		session.InteractionCount++
		
	case TelemetryConsoleOutput:
		if level, ok := event.Data["level"].(string); ok {
			session.ConsoleLogCount[level]++
		}
	}
}

// GetSession returns a specific session
func (ts *TelemetryStore) GetSession(sessionID string) (*PageSession, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	session, exists := ts.sessions[sessionID]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid race conditions
	sessionCopy := *session
	sessionCopy.Events = make([]TelemetryEvent, len(session.Events))
	copy(sessionCopy.Events, session.Events)
	
	return &sessionCopy, true
}

// GetSessionsForProcess returns all sessions for a specific process
func (ts *TelemetryStore) GetSessionsForProcess(processName string) []*PageSession {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	var sessions []*PageSession
	for _, session := range ts.sessions {
		if session.ProcessName == processName {
			// Create a copy
			sessionCopy := *session
			sessionCopy.Events = make([]TelemetryEvent, len(session.Events))
			copy(sessionCopy.Events, session.Events)
			sessions = append(sessions, &sessionCopy)
		}
	}
	
	// Sort by start time (newest first)
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].StartTime.After(sessions[i].StartTime) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}
	
	return sessions
}

// GetSessionsForURL returns all sessions for a specific URL
func (ts *TelemetryStore) GetSessionsForURL(url string) []*PageSession {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	var sessions []*PageSession
	for _, session := range ts.sessions {
		if session.URL == url {
			// Create a copy
			sessionCopy := *session
			sessionCopy.Events = make([]TelemetryEvent, len(session.Events))
			copy(sessionCopy.Events, session.Events)
			sessions = append(sessions, &sessionCopy)
		}
	}
	
	// Sort by last activity (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].LastActivity.After(sessions[i].LastActivity) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}
	
	return sessions
}

// GetAllSessions returns all active sessions
func (ts *TelemetryStore) GetAllSessions() []*PageSession {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	sessions := make([]*PageSession, 0, len(ts.sessions))
	for _, session := range ts.sessions {
		// Create a copy
		sessionCopy := *session
		sessionCopy.Events = make([]TelemetryEvent, len(session.Events))
		copy(sessionCopy.Events, session.Events)
		sessions = append(sessions, &sessionCopy)
	}
	
	// Sort by last activity (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].LastActivity.After(sessions[i].LastActivity) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}
	
	return sessions
}

// cleanupOldSessions removes sessions that have been inactive
func (ts *TelemetryStore) cleanupOldSessions() {
	now := time.Now()
	for sessionID, session := range ts.sessions {
		if now.Sub(session.LastActivity) > ts.sessionTimeout {
			delete(ts.sessions, sessionID)
		}
	}
	
	// Also limit sessions per process
	processCount := make(map[string]int)
	for _, session := range ts.sessions {
		processCount[session.ProcessName]++
	}
	
	// If any process has too many sessions, remove oldest
	for processName, count := range processCount {
		if count > ts.maxSessionsPerProcess {
			// Find sessions for this process
			var processSessions []*PageSession
			for _, session := range ts.sessions {
				if session.ProcessName == processName {
					processSessions = append(processSessions, session)
				}
			}
			
			// Sort by last activity (oldest first)
			for i := 0; i < len(processSessions)-1; i++ {
				for j := i + 1; j < len(processSessions); j++ {
					if processSessions[i].LastActivity.After(processSessions[j].LastActivity) {
						processSessions[i], processSessions[j] = processSessions[j], processSessions[i]
					}
				}
			}
			
			// Remove oldest sessions
			toRemove := count - ts.maxSessionsPerProcess
			for i := 0; i < toRemove && i < len(processSessions); i++ {
				delete(ts.sessions, processSessions[i].SessionID)
			}
		}
	}
}

// ClearSessionsForProcess removes all sessions for a specific process
func (ts *TelemetryStore) ClearSessionsForProcess(processName string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	for sessionID, session := range ts.sessions {
		if session.ProcessName == processName {
			delete(ts.sessions, sessionID)
		}
	}
}

// GetMetricsSummary returns a summary of metrics for a session
func (session *PageSession) GetMetricsSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"sessionId":        session.SessionID,
		"url":              session.URL,
		"processName":      session.ProcessName,
		"startTime":        session.StartTime,
		"lastActivity":     session.LastActivity,
		"duration":         session.LastActivity.Sub(session.StartTime).Seconds(),
		"eventCount":       len(session.Events),
		"errorCount":       session.ErrorCount,
		"interactionCount": session.InteractionCount,
		"consoleLogCount":  session.ConsoleLogCount,
	}
	
	// Add performance metrics if available
	if session.PerformanceMetrics != nil {
		summary["performance"] = session.PerformanceMetrics
	}
	
	// Add latest memory snapshot if available
	if len(session.MemorySnapshots) > 0 {
		summary["latestMemory"] = session.MemorySnapshots[len(session.MemorySnapshots)-1]
	}
	
	return summary
}