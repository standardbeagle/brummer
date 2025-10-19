package state

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// MCPConnectionItem represents an MCP client connection
type MCPConnectionItem struct {
	ClientID       string
	ClientName     string
	ConnectedAt    time.Time
	LastActivity   time.Time
	RequestCount   int
	IsConnected    bool
	ConnectionType string // "SSE", "HTTP", "HTTP+SSE", "Legacy"
	Method         string // "GET", "POST"
}

// MCPActivity represents an activity log entry for an MCP connection
type MCPActivity struct {
	Timestamp time.Time
	Method    string
	Params    string
	Response  string
	Error     string
	Duration  time.Duration
}

// MCPDebugState manages all MCP debug-related state
// This is a sub-model that encapsulates MCP connection debugging according to Elm/Redux patterns
type MCPDebugState struct {
	// Connection tracking
	connections    map[string]*MCPConnectionItem // sessionId -> connection
	activities     map[string][]MCPActivity      // sessionId -> activities
	selectedClient string                        // Selected MCP client ID

	// Memory management settings
	maxConnections       int
	maxActivitiesPerConn int
	cleanupInterval      time.Duration
	lastCleanup          time.Time

	// Display settings
	showDetails     bool
	autoRefresh     bool
	refreshInterval time.Duration

	// UI state
	loading         bool
	needsRefresh    bool
	cacheTimestamp  time.Time
	lastError       string

	// Version for atomic updates and state tracking
	version uint64
	mu      sync.RWMutex
}

// NewMCPDebugState creates a new MCP debug sub-model with initial state
func NewMCPDebugState() *MCPDebugState {
	return &MCPDebugState{
		connections:           make(map[string]*MCPConnectionItem),
		activities:            make(map[string][]MCPActivity),
		selectedClient:        "",
		maxConnections:        100,         // Keep up to 100 connections
		maxActivitiesPerConn:  100,         // Keep up to 100 activities per connection
		cleanupInterval:       time.Minute, // Cleanup every minute
		lastCleanup:           time.Now(),
		showDetails:           true,
		autoRefresh:           true,
		refreshInterval:       30 * time.Second,
		loading:               false,
		needsRefresh:          true,
		cacheTimestamp:        time.Time{},
		lastError:             "",
		version:               0,
	}
}

// AddConnection adds or updates a connection (immutable)
func (s *MCPDebugState) AddConnection(sessionId, clientInfo string, connected bool, connectedAt time.Time, connectionType, method string) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()

	if connected {
		// Determine client name based on client info
		clientName := "Unknown Client"
		if strings.Contains(clientInfo, "Claude Code") {
			clientName = "Claude Code"
		} else if strings.Contains(clientInfo, "VS Code") {
			clientName = "VS Code MCP"
		} else if clientInfo != "" {
			if len(clientInfo) > 20 {
				clientName = clientInfo[:17] + "..."
			} else {
				clientName = clientInfo
			}
		}

		newState.connections[sessionId] = &MCPConnectionItem{
			ClientID:       sessionId,
			ClientName:     clientName,
			ConnectedAt:    connectedAt,
			LastActivity:   connectedAt,
			RequestCount:   0,
			IsConnected:    true,
			ConnectionType: connectionType,
			Method:         method,
		}

		// Initialize activities for this connection
		if _, exists := newState.activities[sessionId]; !exists {
			newState.activities[sessionId] = []MCPActivity{}
		}
	} else {
		// Mark as disconnected
		if conn, exists := newState.connections[sessionId]; exists {
			newConn := *conn
			newConn.IsConnected = false
			newState.connections[sessionId] = &newConn
		}
	}

	newState.incrementVersion()
	return newState
}

// AddActivity adds an activity to a connection (immutable)
func (s *MCPDebugState) AddActivity(sessionId string, activity MCPActivity) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()

	// Update connection info if exists
	if conn, exists := newState.connections[sessionId]; exists {
		newConn := *conn
		newConn.LastActivity = activity.Timestamp
		newConn.RequestCount++
		newState.connections[sessionId] = &newConn
	} else {
		// Create a connection entry for sessions that only have activity (e.g., POST requests)
		newState.connections[sessionId] = &MCPConnectionItem{
			ClientID:       sessionId,
			ClientName:     "HTTP Client",
			ConnectedAt:    activity.Timestamp,
			LastActivity:   activity.Timestamp,
			RequestCount:   1,
			IsConnected:    false,
			ConnectionType: "HTTP",
			Method:         "POST",
		}
	}

	// Add activity to the list
	activities := newState.activities[sessionId]

	// Limit activities to prevent memory issues
	if len(activities) >= newState.maxActivitiesPerConn {
		activities = activities[1:] // Remove oldest
	}

	activities = append(activities, activity)
	newState.activities[sessionId] = activities

	newState.incrementVersion()
	return newState
}

// SetSelectedClient updates the selected client (immutable)
func (s *MCPDebugState) SetSelectedClient(clientID string) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.selectedClient = clientID
	newState.incrementVersion()
	return newState
}

// SetShowDetails updates the detail view setting (immutable)
func (s *MCPDebugState) SetShowDetails(show bool) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.showDetails = show
	newState.incrementVersion()
	return newState
}

// SetAutoRefresh updates the auto-refresh setting (immutable)
func (s *MCPDebugState) SetAutoRefresh(refresh bool) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.autoRefresh = refresh
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *MCPDebugState) SetLoading(loading bool) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *MCPDebugState) SetError(err error) *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	if err != nil {
		newState.lastError = err.Error()
	} else {
		newState.lastError = ""
	}
	newState.incrementVersion()
	return newState
}

// CleanupOldData removes old connections and activities to manage memory usage (immutable)
func (s *MCPDebugState) CleanupOldData() *MCPDebugState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()

	now := time.Now()

	// Only cleanup if enough time has passed
	if now.Sub(newState.lastCleanup) < newState.cleanupInterval {
		return newState
	}
	newState.lastCleanup = now

	// If we're under the connection limit, no need to cleanup
	if len(newState.connections) <= newState.maxConnections {
		return newState
	}

	// Create a slice of connections with their last activity times for sorting
	type connWithTime struct {
		sessionId    string
		lastActivity time.Time
		isConnected  bool
	}

	var connsWithTime []connWithTime
	for sessionId, conn := range newState.connections {
		connsWithTime = append(connsWithTime, connWithTime{
			sessionId:    sessionId,
			lastActivity: conn.LastActivity,
			isConnected:  conn.IsConnected,
		})
	}

	// Sort by last activity time (oldest first), but prioritize disconnected sessions for removal
	sort.Slice(connsWithTime, func(i, j int) bool {
		// If one is connected and other is not, prioritize removing disconnected
		if connsWithTime[i].isConnected != connsWithTime[j].isConnected {
			return !connsWithTime[i].isConnected && connsWithTime[j].isConnected
		}
		// Otherwise sort by last activity time (oldest first)
		return connsWithTime[i].lastActivity.Before(connsWithTime[j].lastActivity)
	})

	// Remove excess connections (keep the most recent maxConnections)
	connectionsToRemove := len(newState.connections) - newState.maxConnections
	if connectionsToRemove > 0 {
		for i := 0; i < connectionsToRemove; i++ {
			sessionId := connsWithTime[i].sessionId
			delete(newState.connections, sessionId)
			delete(newState.activities, sessionId)

			// If we're removing the selected client, clear the selection
			if newState.selectedClient == sessionId {
				newState.selectedClient = ""
			}
		}
	}

	newState.incrementVersion()
	return newState
}

// GetSortedConnections returns connections sorted by connection time (newest first)
func (s *MCPDebugState) GetSortedConnections() []MCPConnectionItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]MCPConnectionItem, 0, len(s.connections))

	// Convert connections to list items
	for _, conn := range s.connections {
		items = append(items, *conn)
	}

	// Sort by connection time (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].ConnectedAt.After(items[j].ConnectedAt)
	})

	return items
}

// GetActivitiesForClient returns activities for a specific client
func (s *MCPDebugState) GetActivitiesForClient(clientID string) []MCPActivity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activities, exists := s.activities[clientID]
	if !exists {
		return []MCPActivity{}
	}

	// Return a copy to prevent mutation
	result := make([]MCPActivity, len(activities))
	copy(result, activities)
	return result
}

// GetConnectionByID returns a specific connection by ID
func (s *MCPDebugState) GetConnectionByID(clientID string) *MCPConnectionItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if conn, exists := s.connections[clientID]; exists {
		// Return a copy to prevent mutation
		connCopy := *conn
		return &connCopy
	}
	return nil
}

// GetMemoryStats returns current memory usage statistics
func (s *MCPDebugState) GetMemoryStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalActivities := 0
	for _, activities := range s.activities {
		totalActivities += len(activities)
	}

	return map[string]interface{}{
		"connections":          len(s.connections),
		"maxConnections":       s.maxConnections,
		"totalActivities":      totalActivities,
		"maxActivitiesPerConn": s.maxActivitiesPerConn,
		"lastCleanup":          s.lastCleanup,
		"cleanupInterval":      s.cleanupInterval,
		"memoryUsagePercent":   float64(len(s.connections)) / float64(s.maxConnections) * 100,
	}
}

// GetStateSummary returns a summary of current state
func (s *MCPDebugState) GetStateSummary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeConnections := 0
	for _, conn := range s.connections {
		if conn.IsConnected {
			activeConnections++
		}
	}

	return map[string]interface{}{
		"total_connections":    len(s.connections),
		"active_connections":   activeConnections,
		"selected_client":      s.selectedClient,
		"show_details":         s.showDetails,
		"auto_refresh":         s.autoRefresh,
		"loading":              s.loading,
		"last_error":           s.lastError,
		"cache_timestamp":      s.cacheTimestamp,
		"needs_refresh":        s.needsRefresh,
	}
}

// Validate checks for state consistency
func (s *MCPDebugState) Validate() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var issues []string

	// Basic validation
	if s.maxConnections <= 0 {
		issues = append(issues, "invalid maxConnections setting")
	}

	if s.maxActivitiesPerConn <= 0 {
		issues = append(issues, "invalid maxActivitiesPerConn setting")
	}

	if s.cleanupInterval <= 0 {
		issues = append(issues, "invalid cleanupInterval setting")
	}

	// Check if selected client exists
	if s.selectedClient != "" {
		if _, exists := s.connections[s.selectedClient]; !exists {
			issues = append(issues, "selected client does not exist")
		}
	}

	return issues
}

// Getters for current state values
func (s *MCPDebugState) GetSelectedClient() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedClient
}

func (s *MCPDebugState) GetShowDetails() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.showDetails
}

func (s *MCPDebugState) GetAutoRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autoRefresh
}

func (s *MCPDebugState) GetLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loading
}

func (s *MCPDebugState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *MCPDebugState) GetConnectionsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

func (s *MCPDebugState) GetActiveConnectionsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, conn := range s.connections {
		if conn.IsConnected {
			count++
		}
	}
	return count
}

func (s *MCPDebugState) GetVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Helper methods
func (s *MCPDebugState) copy() *MCPDebugState {
	newState := *s

	// Deep copy connections
	newState.connections = make(map[string]*MCPConnectionItem)
	for id, conn := range s.connections {
		connCopy := *conn
		newState.connections[id] = &connCopy
	}

	// Deep copy activities
	newState.activities = make(map[string][]MCPActivity)
	for id, activities := range s.activities {
		activitiesCopy := make([]MCPActivity, len(activities))
		copy(activitiesCopy, activities)
		newState.activities[id] = activitiesCopy
	}

	return &newState
}

func (s *MCPDebugState) incrementVersion() {
	s.version++
}