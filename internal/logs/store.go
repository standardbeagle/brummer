package logs

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/standardbeagle/brummer/pkg/filters"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
)

type LogEntry struct {
	ID          string
	ProcessID   string
	ProcessName string
	Timestamp   time.Time
	Content     string
	Level       LogLevel
	IsError     bool
	Tags        []string
	Priority    int
}

type Store struct {
	entries       []LogEntry
	byProcess     map[string][]int
	errors        []LogEntry
	errorContexts []ErrorContext
	errorParser   *ErrorParser
	urls          []URLEntry
	urlMap        map[string]*URLEntry // Map URL to its entry for deduplication
	maxEntries    int
	filters       []filters.Filter
	mu            sync.RWMutex
}

type URLEntry struct {
	URL         string
	ProxyURL    string // Proxy URL if using reverse proxy mode
	ProcessID   string
	ProcessName string
	Timestamp   time.Time
	Context     string
}

func NewStore(maxEntries int) *Store {
	return &Store{
		entries:       make([]LogEntry, 0, maxEntries),
		byProcess:     make(map[string][]int),
		errors:        make([]LogEntry, 0, 100),
		errorContexts: make([]ErrorContext, 0, 100),
		errorParser:   NewErrorParser(),
		urls:          make([]URLEntry, 0, 100),
		urlMap:        make(map[string]*URLEntry),
		maxEntries:    maxEntries,
		filters:       []filters.Filter{},
	}
}

func (s *Store) Add(processID, processName, content string, isError bool) *LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := LogEntry{
		ID:          fmt.Sprintf("%s-%d", processID, time.Now().UnixNano()),
		ProcessID:   processID,
		ProcessName: processName,
		Timestamp:   time.Now(),
		Content:     content,
		IsError:     isError,
		Level:       s.detectLogLevel(content, isError),
		Tags:        s.extractTags(content),
		Priority:    s.calculatePriority(content, isError),
	}

	if len(s.entries) >= s.maxEntries {
		s.entries = s.entries[1:]
		for pid, indices := range s.byProcess {
			for i := range indices {
				indices[i]--
			}
			s.byProcess[pid] = indices
		}
	}

	idx := len(s.entries)
	s.entries = append(s.entries, entry)
	s.byProcess[processID] = append(s.byProcess[processID], idx)

	// Track errors with enhanced parsing
	if isError || entry.Level >= LevelError {
		s.errors = append(s.errors, entry)
		if len(s.errors) > 100 {
			s.errors = s.errors[1:]
		}
	}

	// Process through error parser for better error context
	if errorCtx := s.errorParser.ProcessLine(processID, processName, content, entry.Timestamp); errorCtx != nil {
		s.errorContexts = append(s.errorContexts, *errorCtx)
		if len(s.errorContexts) > 100 {
			s.errorContexts = s.errorContexts[1:]
		}
	}

	// Detect and track URLs (with deduplication)
	urls := s.detectURLs(content)
	for _, url := range urls {
		// Check if we already have this URL
		if existing, exists := s.urlMap[url]; exists {
			// Update the existing entry with the most recent occurrence
			existing.Timestamp = entry.Timestamp
			existing.Context = content
			existing.ProcessID = processID
			existing.ProcessName = processName
		} else {
			// New URL, add it
			urlEntry := URLEntry{
				URL:         url,
				ProcessID:   processID,
				ProcessName: processName,
				Timestamp:   entry.Timestamp,
				Context:     content,
			}
			s.urlMap[url] = &urlEntry
			// Rebuild the urls slice from the map
			s.rebuildURLsList()
		}
	}

	return &entry
}

func (s *Store) detectLogLevel(content string, isError bool) LogLevel {
	lower := strings.ToLower(content)

	if isError || strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
		return LevelError
	}
	if strings.Contains(lower, "critical") || strings.Contains(lower, "fatal") {
		return LevelCritical
	}
	if strings.Contains(lower, "warn") || strings.Contains(lower, "warning") {
		return LevelWarn
	}
	if strings.Contains(lower, "debug") {
		return LevelDebug
	}
	return LevelInfo
}

func (s *Store) extractTags(content string) []string {
	tags := []string{}
	lower := strings.ToLower(content)

	if strings.Contains(lower, "build") {
		tags = append(tags, "build")
	}
	if strings.Contains(lower, "test") {
		tags = append(tags, "test")
	}
	if strings.Contains(lower, "lint") {
		tags = append(tags, "lint")
	}
	if strings.Contains(lower, "compile") {
		tags = append(tags, "compile")
	}
	if strings.Contains(lower, "warning") {
		tags = append(tags, "warning")
	}
	if strings.Contains(lower, "error") {
		tags = append(tags, "error")
	}

	return tags
}

func (s *Store) calculatePriority(content string, isError bool) int {
	priority := 0

	if isError {
		priority += 50
	}

	lower := strings.ToLower(content)

	if strings.Contains(lower, "failed") {
		priority += 40
	}
	if strings.Contains(lower, "error") {
		priority += 30
	}
	if strings.Contains(lower, "warning") {
		priority += 20
	}
	if strings.Contains(lower, "build") {
		priority += 10
	}
	if strings.Contains(lower, "test") && (strings.Contains(lower, "fail") || strings.Contains(lower, "pass")) {
		priority += 15
	}

	for _, filter := range s.filters {
		if filter.Matches(content) {
			priority += filter.PriorityBoost
		}
	}

	return priority
}

func (s *Store) GetByProcess(processID string) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indices, exists := s.byProcess[processID]
	if !exists {
		return []LogEntry{}
	}

	result := make([]LogEntry, 0, len(indices))
	for _, idx := range indices {
		if idx >= 0 && idx < len(s.entries) {
			result = append(result, s.entries[idx])
		}
	}

	return result
}

func (s *Store) GetAll() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]LogEntry, len(s.entries))
	copy(result, s.entries)
	return result
}

func (s *Store) Search(query string) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	result := []LogEntry{}

	for _, entry := range s.entries {
		if strings.Contains(strings.ToLower(entry.Content), query) ||
			strings.Contains(strings.ToLower(entry.ProcessName), query) {
			result = append(result, entry)
		}
	}

	return result
}

func (s *Store) GetHighPriority(threshold int) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []LogEntry{}
	for _, entry := range s.entries {
		if entry.Priority >= threshold {
			result = append(result, entry)
		}
	}

	return result
}

func (s *Store) AddFilter(filter filters.Filter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filters = append(s.filters, filter)
}

func (s *Store) RemoveFilter(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newFilters := []filters.Filter{}
	for _, f := range s.filters {
		if f.Name != name {
			newFilters = append(newFilters, f)
		}
	}
	s.filters = newFilters
}

func (s *Store) GetFilters() []filters.Filter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]filters.Filter, len(s.filters))
	copy(result, s.filters)
	return result
}

var urlRegex = regexp.MustCompile(`https?://[^\s<>"{}|\\^\[\]` + "`" + `]+`)

func (s *Store) detectURLs(content string) []string {
	matches := urlRegex.FindAllString(content, -1)
	// Remove trailing punctuation
	for i, url := range matches {
		url = strings.TrimRight(url, ".,;:!?)")
		matches[i] = url
	}
	return matches
}

func (s *Store) GetErrors() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]LogEntry, len(s.errors))
	copy(result, s.errors)
	return result
}

func (s *Store) GetURLs() []URLEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]URLEntry, len(s.urls))
	copy(result, s.urls)
	return result
}

func (s *Store) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make([]LogEntry, 0, s.maxEntries)
	s.byProcess = make(map[string][]int)
}

func (s *Store) ClearErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.errors = make([]LogEntry, 0, 100)
	s.errorContexts = make([]ErrorContext, 0, 100)
	s.errorParser.ClearErrors()
}

// ClearLogsForProcess clears all logs for a specific process
func (s *Store) ClearLogsForProcess(processName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create new entries slice without logs from the specified process
	newEntries := make([]LogEntry, 0, s.maxEntries)
	for _, entry := range s.entries {
		if entry.ProcessName != processName {
			newEntries = append(newEntries, entry)
		}
	}
	s.entries = newEntries

	// Rebuild the byProcess index
	s.byProcess = make(map[string][]int)
	for i, entry := range s.entries {
		s.byProcess[entry.ProcessID] = append(s.byProcess[entry.ProcessID], i)
	}

	// Also clear errors from this process
	newErrors := make([]LogEntry, 0, 100)
	for _, err := range s.errors {
		if err.ProcessName != processName {
			newErrors = append(newErrors, err)
		}
	}
	s.errors = newErrors

	// Clear error contexts from this process
	newErrorContexts := make([]ErrorContext, 0, 100)
	for _, ctx := range s.errorContexts {
		if ctx.ProcessName != processName {
			newErrorContexts = append(newErrorContexts, ctx)
		}
	}
	s.errorContexts = newErrorContexts
}

// GetErrorContexts returns parsed error contexts with full details
func (s *Store) GetErrorContexts() []ErrorContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ErrorContext, len(s.errorContexts))
	copy(result, s.errorContexts)
	return result
}

// rebuildURLsList rebuilds the urls slice from the urlMap
func (s *Store) rebuildURLsList() {
	s.urls = make([]URLEntry, 0, len(s.urlMap))
	for _, urlEntry := range s.urlMap {
		s.urls = append(s.urls, *urlEntry)
	}

	// Sort by timestamp (most recent first)
	for i := 0; i < len(s.urls)-1; i++ {
		for j := i + 1; j < len(s.urls); j++ {
			if s.urls[j].Timestamp.After(s.urls[i].Timestamp) {
				s.urls[i], s.urls[j] = s.urls[j], s.urls[i]
			}
		}
	}

	// Keep only the most recent 100 URLs
	if len(s.urls) > 100 {
		s.urls = s.urls[:100]
		// Remove the oldest entries from the map
		for url := range s.urlMap {
			found := false
			for i := 0; i < 100; i++ {
				if s.urls[i].URL == url {
					found = true
					break
				}
			}
			if !found {
				delete(s.urlMap, url)
			}
		}
	}
}

// RemoveURLsForProcess removes all URLs associated with a specific process
func (s *Store) RemoveURLsForProcess(processID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove URLs from the map that belong to this process
	for url, entry := range s.urlMap {
		if entry.ProcessID == processID {
			delete(s.urlMap, url)
		}
	}

	// Rebuild the urls list
	s.rebuildURLsList()
}

// UpdateProxyURL updates the proxy URL for a given URL
func (s *Store) UpdateProxyURL(originalURL, proxyURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, exists := s.urlMap[originalURL]; exists {
		entry.ProxyURL = proxyURL
		// Update the urls list
		for i := range s.urls {
			if s.urls[i].URL == originalURL {
				s.urls[i].ProxyURL = proxyURL
				break
			}
		}
	}
}
