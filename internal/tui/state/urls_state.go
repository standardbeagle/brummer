package state

import (
	"sort"
	"strings"
	"time"
)

// URLInfo represents information about a discovered URL
type URLInfo struct {
	URL         string    `json:"url"`
	Label       string    `json:"label"`
	ProcessName string    `json:"process_name"`
	Timestamp   time.Time `json:"timestamp"`
	VisitCount  int       `json:"visit_count"`
}

// URLsState manages all URLs-related UI state
// This is a sub-model that encapsulates URLs display according to Elm/Redux patterns
//
// IMMUTABLE: This struct is immutable. All update methods return a NEW instance.
// NO MUTEXES: Thread-safety is achieved through immutability and atomic operations.
type URLsState struct {
	// URL data cache
	URLs           []URLInfo
	CacheTimestamp time.Time
	NeedsRefresh   bool

	// Selection state
	SelectedURL    *URLInfo
	SelectedURLID  string

	// Display settings
	ShowDetails    bool
	GroupByProcess bool
	SortBy        string // "timestamp", "label", "url"
	FilterText    string
	AutoRefresh    bool
	RefreshInterval time.Duration

	// UI state
	Loading        bool
	LastError      string

	// Statistics
	TotalURLs      int
	UniqueDomains  map[string]int
	LastActivity   time.Time

	// Version for state tracking
	version        uint64
}

// NewURLsState creates a new URLs sub-model with initial state
func NewURLsState() *URLsState {
	return &URLsState{
		URLs:           []URLInfo{},
		CacheTimestamp: time.Time{},
		NeedsRefresh:   true,
		SelectedURL:    nil,
		SelectedURLID:  "",
		ShowDetails:    true,
		GroupByProcess: true,
		SortBy:         "timestamp",
		FilterText:     "",
		AutoRefresh:    false,
		RefreshInterval: 30 * time.Second,
		Loading:        false,
		LastError:      "",
		TotalURLs:      0,
		UniqueDomains:  make(map[string]int),
		LastActivity:   time.Time{},
		version:        0,
	}
}

// UpdateURLs updates the cached URLs list (immutable)
func (s *URLsState) UpdateURLs(urls []URLInfo) *URLsState {

	newState := s.copy()
	newState.URLs = make([]URLInfo, len(urls))
	copy(newState.URLs, urls)
	newState.CacheTimestamp = time.Now()
	newState.NeedsRefresh = false
	newState.TotalURLs = len(urls)
	newState.updateStatistics()
	newState.incrementVersion()
	return newState
}

// AddURL adds a new URL to the list (immutable)
func (s *URLsState) AddURL(url URLInfo) *URLsState {

	newState := s.copy()
	newState.URLs = append(newState.URLs, url)
	newState.CacheTimestamp = time.Now()
	newState.TotalURLs = len(newState.URLs)
	newState.updateStatistics()
	newState.incrementVersion()
	return newState
}

// SetSelectedURL updates the selected URL (immutable)
func (s *URLsState) SetSelectedURL(urlID string) *URLsState {

	newState := s.copy()
	newState.SelectedURLID = urlID

	// Find the URL in the cached list
	newState.SelectedURL = nil
	for _, url := range newState.URLs {
		if url.URL == urlID {
			newState.SelectedURL = &url
			break
		}
	}

	newState.incrementVersion()
	return newState
}

// SetShowDetails updates the detail view setting (immutable)
func (s *URLsState) SetShowDetails(show bool) *URLsState {

	newState := s.copy()
	newState.ShowDetails = show
	newState.incrementVersion()
	return newState
}

// SetGroupByProcess updates the grouping setting (immutable)
func (s *URLsState) SetGroupByProcess(group bool) *URLsState {

	newState := s.copy()
	newState.GroupByProcess = group
	newState.incrementVersion()
	return newState
}

// SetSortBy updates the sort setting (immutable)
func (s *URLsState) SetSortBy(sortBy string) *URLsState {

	newState := s.copy()
	newState.SortBy = sortBy
	newState.incrementVersion()
	return newState
}

// SetFilterText updates the filter text (immutable)
func (s *URLsState) SetFilterText(filter string) *URLsState {

	newState := s.copy()
	newState.FilterText = strings.ToLower(filter)
	newState.incrementVersion()
	return newState
}

// SetAutoRefresh updates the auto-refresh setting (immutable)
func (s *URLsState) SetAutoRefresh(refresh bool) *URLsState {

	newState := s.copy()
	newState.AutoRefresh = refresh
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *URLsState) SetLoading(loading bool) *URLsState {

	newState := s.copy()
	newState.Loading = loading
	if loading {
		newState.NeedsRefresh = true
	}
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *URLsState) SetError(err error) *URLsState {

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

// MarkForRefresh marks that the URLs data needs refreshing (immutable)
func (s *URLsState) MarkForRefresh() *URLsState {

	newState := s.copy()
	newState.NeedsRefresh = true
	newState.incrementVersion()
	return newState
}

// ClearURLs clears all URLs (immutable)
func (s *URLsState) ClearURLs() *URLsState {

	newState := s.copy()
	newState.URLs = []URLInfo{}
	newState.CacheTimestamp = time.Time{}
	newState.NeedsRefresh = true
	newState.TotalURLs = 0
	newState.UniqueDomains = make(map[string]int)
	newState.SelectedURL = nil
	newState.SelectedURLID = ""
	newState.incrementVersion()
	return newState
}

// updateStatistics updates statistics based on current URLs
func (s *URLsState) updateStatistics() {
	s.UniqueDomains = make(map[string]int)

	for _, url := range s.URLs {
		// Extract domain from URL
		if strings.HasPrefix(url.URL, "http://") {
			domain := strings.TrimPrefix(url.URL, "http://")
			if slashIndex := strings.Index(domain, "/"); slashIndex != -1 {
				domain = domain[:slashIndex]
			}
			s.UniqueDomains[domain]++
		} else if strings.HasPrefix(url.URL, "https://") {
			domain := strings.TrimPrefix(url.URL, "https://")
			if slashIndex := strings.Index(domain, "/"); slashIndex != -1 {
				domain = domain[:slashIndex]
			}
			s.UniqueDomains[domain]++
		}
	}
}

// IsCacheValid checks if the cached URLs are still valid
func (s *URLsState) IsCacheValid(maxAge time.Duration) bool {

	if s.CacheTimestamp.IsZero() || s.NeedsRefresh {
		return false
	}

	return time.Since(s.CacheTimestamp) < maxAge
}

// GetFilteredURLs returns URLs filtered by current settings
func (s *URLsState) GetFilteredURLs() []URLInfo {

	// Filter by text
	var filtered []URLInfo
	if s.FilterText != "" {
		for _, url := range s.URLs {
			if strings.Contains(strings.ToLower(url.URL), s.FilterText) ||
				strings.Contains(strings.ToLower(url.Label), s.FilterText) ||
				strings.Contains(strings.ToLower(url.ProcessName), s.FilterText) {
				filtered = append(filtered, url)
			}
		}
	} else {
		filtered = make([]URLInfo, len(s.URLs))
		copy(filtered, s.URLs)
	}

	// Sort URLs
	s.sortURLs(filtered)

	return filtered
}

// sortURLs sorts URLs based on current sort setting
func (s *URLsState) sortURLs(urls []URLInfo) {
	switch s.SortBy {
	case "timestamp":
		sort.Slice(urls, func(i, j int) bool {
			return urls[i].Timestamp.After(urls[j].Timestamp)
		})
	case "label":
		sort.Slice(urls, func(i, j int) bool {
			return urls[i].Label < urls[j].Label
		})
	case "url":
		sort.Slice(urls, func(i, j int) bool {
			return urls[i].URL < urls[j].URL
		})
	default:
		// Default to timestamp
		sort.Slice(urls, func(i, j int) bool {
			return urls[i].Timestamp.After(urls[j].Timestamp)
		})
	}
}

// GetURLsByProcess returns URLs grouped by process name
func (s *URLsState) GetURLsByProcess() map[string][]URLInfo {

	grouped := make(map[string][]URLInfo)
	for _, url := range s.URLs {
		grouped[url.ProcessName] = append(grouped[url.ProcessName], url)
	}

	// Sort each group by timestamp
	for _, urls := range grouped {
		sort.Slice(urls, func(i, j int) bool {
			return urls[i].Timestamp.After(urls[j].Timestamp)
		})
	}

	return grouped
}

// GetStateSummary returns a summary of current state
func (s *URLsState) GetStateSummary() map[string]interface{} {

	return map[string]interface{}{
		"total_urls":       s.TotalURLs,
		"unique_domains":   len(s.UniqueDomains),
		"selected_url_id":   s.SelectedURLID,
		"show_details":      s.ShowDetails,
		"group_by_process":  s.GroupByProcess,
		"sort_by":           s.SortBy,
		"filter_text":       s.FilterText,
		"auto_refresh":      s.AutoRefresh,
		"refresh_interval":  s.RefreshInterval,
		"loading":           s.Loading,
		"last_error":        s.LastError,
		"last_activity":     s.LastActivity,
		"cache_timestamp":   s.CacheTimestamp,
		"needs_refresh":     s.NeedsRefresh,
	}
}

// Validate checks for state consistency
func (s *URLsState) Validate() []string {

	var issues []string

	// Basic validation
	if s.SortBy == "" {
		issues = append(issues, "invalid sort by setting")
	}

	if s.RefreshInterval <= 0 {
		issues = append(issues, "invalid refresh interval")
	}

	// Validate selected URL exists
	if s.SelectedURLID != "" {
		found := false
		for _, url := range s.URLs {
			if url.URL == s.SelectedURLID {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, "selected URL does not exist")
		}
	}

	return issues
}

// Getters for current state values
func (s *URLsState) GetURLs() []URLInfo {

	urls := make([]URLInfo, len(s.URLs))
	copy(urls, s.URLs)
	return urls
}

func (s *URLsState) GetSelectedURL() *URLInfo {
	return s.SelectedURL
}

func (s *URLsState) GetSelectedURLID() string {
	return s.SelectedURLID
}

func (s *URLsState) GetShowDetails() bool {
	return s.ShowDetails
}

func (s *URLsState) GetGroupByProcess() bool {
	return s.GroupByProcess
}

func (s *URLsState) GetSortBy() string {
	return s.SortBy
}

func (s *URLsState) GetFilterText() string {
	return s.FilterText
}

func (s *URLsState) GetAutoRefresh() bool {
	return s.AutoRefresh
}

func (s *URLsState) GetLoading() bool {
	return s.Loading
}

func (s *URLsState) GetError() string {
	return s.LastError
}

func (s *URLsState) GetTotalURLs() int {
	return s.TotalURLs
}

func (s *URLsState) GetUniqueDomainCount() int {
	return len(s.UniqueDomains)
}

func (s *URLsState) GetLastActivity() time.Time {
	return s.LastActivity
}

func (s *URLsState) GetVersion() uint64 {
	return s.version
}

// Helper methods
func (s *URLsState) copy() *URLsState {
	newState := *s

	// Deep copy slices
	newState.URLs = make([]URLInfo, len(s.URLs))
	copy(newState.URLs, s.URLs)

	// Deep copy maps
	newState.UniqueDomains = make(map[string]int)
	for k, v := range s.UniqueDomains {
		newState.UniqueDomains[k] = v
	}

	return &newState
}

func (s *URLsState) incrementVersion() {
	s.version++
}