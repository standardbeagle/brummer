package state

import (
	"sync"
	"time"
)

// MCPState manages all MCP-related UI state
// This is a sub-model that encapsulates MCP server status and statistics according to Elm/Redux patterns
type MCPState struct {
	// Server connection state
	Active         bool
	Port           int
	Address        string
	LastConnected  time.Time
	LastError      string

	// Tools and capabilities
	AvailableTools []string
	ToolsCount     int

	// Request statistics
	RequestCount   int
	SessionCount   int
	LastRequest    time.Time

	// Display settings
	ShowDetails    bool
	AutoRefresh    bool
	RefreshInterval time.Duration

	// UI state
	Loading        bool
	NeedsRefresh   bool
	CacheTimestamp time.Time

	// Version for atomic updates and state tracking
	version uint64
	mu      sync.RWMutex
}

// NewMCPState creates a new MCP sub-model with initial state
func NewMCPState() *MCPState {
	return &MCPState{
		Active:          false,
		Port:            7777,
		Address:         "localhost:7777",
		LastConnected:   time.Time{},
		LastError:       "",
		AvailableTools:  []string{},
		ToolsCount:      0,
		RequestCount:    0,
		SessionCount:    0,
		LastRequest:     time.Time{},
		ShowDetails:     false,
		AutoRefresh:     true,
		RefreshInterval: 30 * time.Second,
		Loading:         false,
		NeedsRefresh:    true,
		CacheTimestamp:  time.Time{},
		version:         0,
	}
}

// SetConnectionStatus updates the server connection status (immutable)
func (s *MCPState) SetConnectionStatus(active bool, address string, port int) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.Active = active
	newState.Address = address
	newState.Port = port
	if active {
		newState.LastConnected = time.Now()
		newState.LastError = ""
	}
	newState.incrementVersion()
	return newState
}

// SetAvailableTools updates the available tools list (immutable)
func (s *MCPState) SetAvailableTools(tools []string) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.AvailableTools = make([]string, len(tools))
	copy(newState.AvailableTools, tools)
	newState.ToolsCount = len(tools)
	newState.CacheTimestamp = time.Now()
	newState.NeedsRefresh = false
	newState.incrementVersion()
	return newState
}

// UpdateStatistics updates request and session statistics (immutable)
func (s *MCPState) UpdateStatistics(requestCount, sessionCount int) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.RequestCount = requestCount
	newState.SessionCount = sessionCount
	newState.LastRequest = time.Now()
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *MCPState) SetError(err error) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
		newState.Active = false
	} else {
		newState.LastError = ""
	}
	newState.incrementVersion()
	return newState
}

// SetShowDetails updates the detail view setting (immutable)
func (s *MCPState) SetShowDetails(show bool) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.ShowDetails = show
	newState.incrementVersion()
	return newState
}

// SetAutoRefresh updates the auto-refresh setting (immutable)
func (s *MCPState) SetAutoRefresh(refresh bool) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.AutoRefresh = refresh
	newState.incrementVersion()
	return newState
}

// SetRefreshInterval updates the refresh interval (immutable)
func (s *MCPState) SetRefreshInterval(interval time.Duration) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.RefreshInterval = interval
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *MCPState) SetLoading(loading bool) *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// MarkForRefresh marks that the MCP data needs refreshing (immutable)
func (s *MCPState) MarkForRefresh() *MCPState {
	s.mu.Lock()
	defer s.mu.Unlock()

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// IsCacheValid checks if the cached MCP data is still valid
func (s *MCPState) IsCacheValid(maxAge time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetToolsByCategory returns tools grouped by category
func (s *MCPState) GetToolsByCategory() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := map[string][]string{
		"Scripts":   []string{},
		"Logs":      []string{},
		"Proxy":     []string{},
		"Telemetry": []string{},
		"Browser":   []string{},
		"REPL":      []string{},
		"Other":     []string{},
	}

	// Map tool prefixes to categories
	categoryMap := map[string]string{
		"scripts":   "Scripts",
		"logs":      "Logs",
		"proxy":     "Proxy",
		"telemetry": "Telemetry",
		"browser":   "Browser",
		"repl":      "REPL",
	}

	for _, tool := range s.AvailableTools {
		if len(tool) == 0 {
			continue
		}

		// Find category by tool name prefix
		foundCategory := false
		for prefix, category := range categoryMap {
			if len(tool) >= len(prefix) && tool[:len(prefix)] == prefix {
				categories[category] = append(categories[category], tool)
				foundCategory = true
				break
			}
		}

		if !foundCategory {
			categories["Other"] = append(categories["Other"], tool)
		}
	}

	return categories
}

// GetStateSummary returns a summary of current state
func (s *MCPState) GetStateSummary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active":          s.Active,
		"address":         s.Address,
		"port":            s.Port,
		"tools_count":     s.ToolsCount,
		"request_count":   s.RequestCount,
		"session_count":   s.SessionCount,
		"last_connected":  s.LastConnected,
		"last_request":    s.LastRequest,
		"auto_refresh":    s.AutoRefresh,
		"needs_refresh":   s.NeedsRefresh,
		"cache_timestamp": s.CacheTimestamp,
	}
}

// Validate checks for state consistency
func (s *MCPState) Validate() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var issues []string

	// Basic validation
	if s.Port <= 0 || s.Port > 65535 {
		issues = append(issues, "invalid port number")
	}

	if s.Address == "" {
		issues = append(issues, "empty address")
	}

	if s.ToolsCount != len(s.AvailableTools) {
		issues = append(issues, "tools count mismatch")
	}

	if s.RequestCount < 0 {
		issues = append(issues, "negative request count")
	}

	if s.SessionCount < 0 {
		issues = append(issues, "negative session count")
	}

	return issues
}

// Getters for current state values
func (s *MCPState) GetActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Active
}

func (s *MCPState) GetPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Port
}

func (s *MCPState) GetAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Address
}

func (s *MCPState) GetAvailableTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]string, len(s.AvailableTools))
	copy(tools, s.AvailableTools)
	return tools
}

func (s *MCPState) GetToolsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ToolsCount
}

func (s *MCPState) GetRequestCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RequestCount
}

func (s *MCPState) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SessionCount
}

func (s *MCPState) GetShowDetails() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShowDetails
}

func (s *MCPState) GetAutoRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AutoRefresh
}

func (s *MCPState) GetRefreshInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RefreshInterval
}

func (s *MCPState) GetLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Loading
}

func (s *MCPState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}

func (s *MCPState) GetLastConnected() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastConnected
}

func (s *MCPState) GetLastRequest() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastRequest
}

func (s *MCPState) GetCacheTimestamp() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CacheTimestamp
}

func (s *MCPState) GetNeedsRefresh() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.NeedsRefresh
}

func (s *MCPState) GetVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Helper methods
func (s *MCPState) copy() *MCPState {
	newState := *s

	// Deep copy slices
	newState.AvailableTools = make([]string, len(s.AvailableTools))
	copy(newState.AvailableTools, s.AvailableTools)

	return &newState
}

func (s *MCPState) incrementVersion() {
	s.version++
}