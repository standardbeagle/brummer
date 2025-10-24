package state

import (
	"time"
)

// RequestInfo holds display information for a proxy request
type RequestInfo struct {
	ID          string
	Method      string
	URL         string
	Path        string
	StatusCode  int
	Duration    time.Duration
	ProcessName string
	Timestamp   time.Time
	Size        int64
	Error       string

	// Authentication
	HasAuth   bool
	AuthType  string
	JWTError  string
	JWTClaims map[string]interface{}

	// Telemetry
	HasTelemetry bool
	SessionID    string
	PerformanceMetrics map[string]interface{} // Browser performance metrics (TTFB, DOM, etc.)
	// Note: Full telemetry data would require more complex types for this component

	// Request/Response data
	RequestHeaders  map[string]string
	ResponseHeaders map[string]string
	RequestBody     string
	ResponseBody    string
}

// WebViewState manages all web view UI state
// This is a sub-model that encapsulates web view display according to Elm/Redux patterns
type WebViewState struct {
	// Request data
	Requests         []RequestInfo
	SelectedIndex    int
	SelectedRequest  *RequestInfo
	ShowDetail       bool
	Filter           string // "all", "2xx", "3xx", "4xx", "5xx", "error"
	AutoScroll       bool

	// Viewport state
	ViewportOffset   int
	ViewportHeight   int

	// Layout
	Width            int
	Height           int

	// UI state
	Loading          bool
	LastError        string
	LastUpdated      time.Time
	MaxRequests      int // Maximum requests to keep in memory

	// Statistics
	TotalRequests    int
	RequestsByStatus map[string]int
	RequestsByMethod map[string]int

	// Version for atomic updates and state tracking
	version          uint64
}

// NewWebViewState creates a new web view sub-model with initial state
func NewWebViewState() *WebViewState {
	return &WebViewState{
		Requests:         []RequestInfo{},
		SelectedIndex:    -1,
		SelectedRequest:  nil,
		ShowDetail:       false,
		Filter:           "all",
		AutoScroll:       true,
		ViewportOffset:   0,
		ViewportHeight:   24,
		Width:            80,
		Height:           24,
		Loading:          false,
		LastError:        "",
		LastUpdated:      time.Time{},
		MaxRequests:      1000, // Keep last 1000 requests
		TotalRequests:    0,
		RequestsByStatus: make(map[string]int),
		RequestsByMethod: make(map[string]int),
		version:          0,
	}
}

// AddRequest adds a new request to the list (immutable)
func (s *WebViewState) AddRequest(request RequestInfo) *WebViewState {

	newState := s.copy()

	// Add request to the beginning (newest first)
	newState.Requests = append([]RequestInfo{request}, newState.Requests...)

	// Limit the number of requests kept in memory
	if len(newState.Requests) > newState.MaxRequests {
		newState.Requests = newState.Requests[:newState.MaxRequests]
	}

	// Update statistics
	newState.TotalRequests++
	statusCode := s.getStatusCodeCategory(request.StatusCode)
	newState.RequestsByStatus[statusCode]++
	newState.RequestsByMethod[request.Method]++

	// Always update selection to first request when new requests are added
	// This ensures there's always a visible selection for navigation
	filteredRequests := newState.getFilteredRequests()
	if len(filteredRequests) > 0 {
		// Select the newest request (index 0 since requests are added to beginning)
		newState.SelectedIndex = 0
		newState.SelectedRequest = &filteredRequests[0]
	}

	newState.LastUpdated = time.Now()
	newState.incrementVersion()
	return newState
}

// SetSelectedIndex updates the selected request index (immutable)
func (s *WebViewState) SetSelectedIndex(index int) *WebViewState {

	newState := s.copy()
	filteredRequests := newState.getFilteredRequests()

	if index >= 0 && index < len(filteredRequests) {
		newState.SelectedIndex = index
		newState.SelectedRequest = &filteredRequests[index]
	} else {
		newState.SelectedIndex = -1
		newState.SelectedRequest = nil
	}

	newState.incrementVersion()
	return newState
}

// ToggleDetailView toggles the detail view (immutable)
func (s *WebViewState) ToggleDetailView() *WebViewState {

	newState := s.copy()
	newState.ShowDetail = !newState.ShowDetail
	newState.incrementVersion()
	return newState
}

// SetFilter updates the filter setting (immutable)
func (s *WebViewState) SetFilter(filter string) *WebViewState {

	newState := s.copy()
	newState.Filter = filter

	// Reset selection to first filtered request
	filteredRequests := newState.getFilteredRequests()
	if len(filteredRequests) > 0 {
		newState.SelectedIndex = 0
		newState.SelectedRequest = &filteredRequests[0]
	} else {
		newState.SelectedIndex = -1
		newState.SelectedRequest = nil
	}

	newState.incrementVersion()
	return newState
}

// ToggleAutoScroll toggles auto-scroll setting (immutable)
func (s *WebViewState) ToggleAutoScroll() *WebViewState {

	newState := s.copy()
	newState.AutoScroll = !newState.AutoScroll
	newState.incrementVersion()
	return newState
}

// SetViewportOffset updates the viewport offset (immutable)
func (s *WebViewState) SetViewportOffset(offset int) *WebViewState {

	newState := s.copy()
	newState.ViewportOffset = offset
	newState.incrementVersion()
	return newState
}

// SetSize updates the component dimensions (immutable)
func (s *WebViewState) SetSize(width, height int) *WebViewState {

	newState := s.copy()
	newState.Width = width
	newState.Height = height
	newState.ViewportHeight = height
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *WebViewState) SetLoading(loading bool) *WebViewState {

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *WebViewState) SetError(err error) *WebViewState {

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.Loading = false
	newState.incrementVersion()
	return newState
}

// ClearRequests clears all requests (immutable)
func (s *WebViewState) ClearRequests() *WebViewState {

	newState := s.copy()
	newState.Requests = []RequestInfo{}
	newState.SelectedIndex = -1
	newState.SelectedRequest = nil
	newState.TotalRequests = 0
	newState.RequestsByStatus = make(map[string]int)
	newState.RequestsByMethod = make(map[string]int)
	newState.incrementVersion()
	return newState
}

// getFilteredRequests returns requests filtered by current filter setting
func (s *WebViewState) getFilteredRequests() []RequestInfo {
	if s.Filter == "all" {
		return s.Requests
	}

	var filtered []RequestInfo
	for _, req := range s.Requests {
		if s.matchesFilter(req) {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

// matchesFilter checks if a request matches the current filter
func (s *WebViewState) matchesFilter(req RequestInfo) bool {
	switch s.Filter {
	case "2xx", "3xx", "4xx", "5xx":
		return s.getStatusCodeCategory(req.StatusCode) == s.Filter
	case "error":
		return req.Error != "" || req.StatusCode >= 500
	default:
		return true
	}
}

// getStatusCodeCategory returns the status code category (e.g., "2xx" for 200-299)
func (s *WebViewState) getStatusCodeCategory(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// GetStateSummary returns a summary of current state
func (s *WebViewState) GetStateSummary() map[string]interface{} {

	filteredRequests := s.getFilteredRequests()

	return map[string]interface{}{
		"total_requests":       s.TotalRequests,
		"filtered_requests":    len(filteredRequests),
		"selected_index":       s.SelectedIndex,
		"show_detail":          s.ShowDetail,
		"filter":               s.Filter,
		"auto_scroll":          s.AutoScroll,
		"viewport_offset":      s.ViewportOffset,
		"viewport_height":      s.ViewportHeight,
		"width":                s.Width,
		"height":               s.Height,
		"loading":              s.Loading,
		"last_error":           s.LastError,
		"last_updated":         s.LastUpdated,
		"max_requests":         s.MaxRequests,
		"requests_by_status":   s.RequestsByStatus,
		"requests_by_method":   s.RequestsByMethod,
		"version":              s.version,
	}
}

// Validate checks for state consistency
func (s *WebViewState) Validate() []string {

	var issues []string

	// Basic validation
	if s.Width <= 0 || s.Height <= 0 {
		issues = append(issues, "invalid dimensions")
	}

	if s.MaxRequests <= 0 {
		issues = append(issues, "invalid max requests setting")
	}

	// Validate selected index
	filteredRequests := s.getFilteredRequests()
	if s.SelectedIndex < -1 || s.SelectedIndex >= len(filteredRequests) {
		issues = append(issues, "selected index out of bounds")
	}

	// Validate filter
	validFilters := []string{"all", "2xx", "3xx", "4xx", "5xx", "error"}
	filterValid := false
	for _, valid := range validFilters {
		if s.Filter == valid {
			filterValid = true
			break
		}
	}
	if !filterValid {
		issues = append(issues, "invalid filter setting")
	}

	return issues
}

// Getters for current state values
func (s *WebViewState) GetRequests() []RequestInfo {

	requests := make([]RequestInfo, len(s.Requests))
	copy(requests, s.Requests)
	return requests
}

func (s *WebViewState) GetFilteredRequests() []RequestInfo {
	return s.getFilteredRequests()
}

func (s *WebViewState) GetSelectedIndex() int {
	return s.SelectedIndex
}

func (s *WebViewState) GetSelectedRequest() *RequestInfo {
	return s.SelectedRequest
}

func (s *WebViewState) IsShowingDetail() bool {
	return s.ShowDetail
}

func (s *WebViewState) GetFilter() string {
	return s.Filter
}

func (s *WebViewState) IsAutoScroll() bool {
	return s.AutoScroll
}

func (s *WebViewState) GetViewportOffset() int {
	return s.ViewportOffset
}

func (s *WebViewState) GetSize() (int, int) {
	return s.Width, s.Height
}

func (s *WebViewState) GetLoading() bool {
	return s.Loading
}

func (s *WebViewState) GetError() string {
	return s.LastError
}

func (s *WebViewState) GetLastUpdated() time.Time {
	return s.LastUpdated
}

func (s *WebViewState) GetVersion() uint64 {
	return s.version
}

func (s *WebViewState) GetTotalRequests() int {
	return s.TotalRequests
}

func (s *WebViewState) GetRequestsByStatus() map[string]int {

	result := make(map[string]int)
	for k, v := range s.RequestsByStatus {
		result[k] = v
	}
	return result
}

func (s *WebViewState) GetRequestsByMethod() map[string]int {

	result := make(map[string]int)
	for k, v := range s.RequestsByMethod {
		result[k] = v
	}
	return result
}

// Helper methods
func (s *WebViewState) copy() *WebViewState {
	newState := *s

	// Deep copy slices
	newState.Requests = make([]RequestInfo, len(s.Requests))
	copy(newState.Requests, s.Requests)

	// Deep copy maps
	newState.RequestsByStatus = make(map[string]int)
	for k, v := range s.RequestsByStatus {
		newState.RequestsByStatus[k] = v
	}

	newState.RequestsByMethod = make(map[string]int)
	for k, v := range s.RequestsByMethod {
		newState.RequestsByMethod[k] = v
	}

	return &newState
}

func (s *WebViewState) incrementVersion() {
	s.version++
}