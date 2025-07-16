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

// CollapsedLogEntry represents a log entry that may contain multiple identical consecutive logs
type CollapsedLogEntry struct {
	LogEntry
	Count       int       // Number of times this exact log appeared consecutively
	FirstSeen   time.Time // Timestamp of the first occurrence
	LastSeen    time.Time // Timestamp of the last occurrence
	IsCollapsed bool      // Whether this entry represents collapsed logs
}

type Store struct {
	entries       []LogEntry
	byProcess     map[string][]int
	errors        []LogEntry
	errorContexts []ErrorContext
	errorParser     *ErrorParser
	timeBasedParser *TimeBasedErrorParser
	urls          []URLEntry
	urlMap        map[string]*URLEntry // Map URL to its entry for deduplication
	maxEntries    int
	filters       []filters.Filter
	mu            sync.RWMutex

	// Channel-based async operations
	addChan   chan *addLogRequest
	closeChan chan struct{}
	wg        sync.WaitGroup
}

type URLEntry struct {
	URL         string
	ProxyURL    string // Proxy URL if using reverse proxy mode
	ProcessID   string
	ProcessName string
	Timestamp   time.Time
	Context     string
}

type addLogRequest struct {
	processID   string
	processName string
	content     string
	isError     bool
	result      chan *LogEntry
}

func NewStore(maxEntries int) *Store {
	s := &Store{
		entries:       make([]LogEntry, 0, maxEntries),
		byProcess:     make(map[string][]int),
		errors:        make([]LogEntry, 0, 100),
		errorContexts: make([]ErrorContext, 0, 100),
		errorParser:     NewErrorParser(),
		timeBasedParser: NewTimeBasedErrorParser(),
		urls:          make([]URLEntry, 0, 100),
		urlMap:        make(map[string]*URLEntry),
		maxEntries:    maxEntries,
		filters:       []filters.Filter{},
		addChan:       make(chan *addLogRequest, 1000),
		closeChan:     make(chan struct{}),
	}

	// Start async worker
	s.wg.Add(1)
	go s.processAddRequests()

	return s
}

func (s *Store) Add(processID, processName, content string, isError bool) *LogEntry {
	// For high-frequency operations, try pure async first
	req := &addLogRequest{
		processID:   processID,
		processName: processName,
		content:     content,
		isError:     isError,
		result:      nil, // No result channel for fire-and-forget
	}

	// Non-blocking send to channel
	select {
	case s.addChan <- req:
		// Return a dummy entry for async operation (fire-and-forget)
		return &LogEntry{
			ID:          fmt.Sprintf("%s-%d", processID, time.Now().UnixNano()),
			ProcessID:   processID,
			ProcessName: processName,
			Timestamp:   time.Now(),
			Content:     content,
			IsError:     isError,
		}
	default:
		// Channel full, immediate fallback to sync
		return s.addSync(processID, processName, content, isError)
	}
}

func (s *Store) processAddRequests() {
	defer s.wg.Done()

	for {
		select {
		case req := <-s.addChan:
			entry := s.addSync(req.processID, req.processName, req.content, req.isError)
			if req.result != nil {
				select {
				case req.result <- entry:
				default:
				}
			}
		case <-s.closeChan:
			// Drain remaining requests
			for {
				select {
				case req := <-s.addChan:
					entry := s.addSync(req.processID, req.processName, req.content, req.isError)
					if req.result != nil {
						select {
						case req.result <- entry:
						default:
						}
					}
				default:
					return
				}
			}
		}
	}
}

func (s *Store) addSync(processID, processName, content string, isError bool) *LogEntry {
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

	// Process through time-based error parser for better error clustering
	if cluster := s.timeBasedParser.ProcessLogEntry(entry, processName, isError); cluster != nil {
		// Convert cluster to ErrorContext for compatibility
		errorCtx := cluster.ToErrorContext()
		s.errorContexts = append(s.errorContexts, errorCtx)
		if len(s.errorContexts) > 100 {
			s.errorContexts = s.errorContexts[1:]
		}
	}

	// Detect and track URLs (with deduplication)
	urls := detectURLs(content)
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
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// isValidURL checks if a URL is valid and complete
func (s *Store) isValidURL(urlStr string) bool {
	// Must have protocol
	if !strings.Contains(urlStr, "://") {
		return false
	}

	// Split by protocol
	parts := strings.SplitN(urlStr, "://", 2)
	if len(parts) != 2 {
		return false
	}

	// Host part must not be empty
	hostPart := parts[1]
	if hostPart == "" {
		return false
	}

	// If there's a colon, it should be followed by a port number or path
	if idx := strings.Index(hostPart, ":"); idx != -1 {
		afterColon := hostPart[idx+1:]
		// Should have something after the colon (port or path)
		if afterColon == "" || afterColon == "/" {
			return false
		}
	}

	return true
}

func detectURLs(content string) []string {
	// Strip ANSI escape codes before detecting URLs
	cleanContent := ansiRegex.ReplaceAllString(content, "")

	matches := urlRegex.FindAllString(cleanContent, -1)
	validURLs := []string{}

	for _, url := range matches {
		// Remove trailing punctuation
		url = strings.TrimRight(url, ".,;!?)")

		// Handle trailing colons - these could be incomplete ports or punctuation
		if strings.HasSuffix(url, ":") {
			// Count colons after the protocol
			protocolEnd := strings.Index(url, "://")
			if protocolEnd >= 0 {
				afterProtocol := url[protocolEnd+3:]
				colonCount := strings.Count(afterProtocol, ":")

				if colonCount == 1 && strings.HasSuffix(afterProtocol, ":") {
					// This is like "http://localhost:" - incomplete port, skip it
					continue
				} else if colonCount > 1 {
					// Multiple colons like "http://localhost:3000:" - remove trailing
					url = strings.TrimSuffix(url, ":")
				}
			}
		}

		// Validate the URL has protocol and host
		if strings.Contains(url, "://") {
			parts := strings.SplitN(url, "://", 2)
			if len(parts) == 2 && len(parts[1]) > 0 {
				// Additional validation: check if URL ends with bare colon after hostname
				if strings.HasSuffix(url, ":") && !strings.HasSuffix(url, "://") {
					// Check if there's anything after the last colon
					lastColon := strings.LastIndex(url, ":")
					protocolEnd := strings.Index(url, "://")
					if lastColon > protocolEnd+2 { // Colon is after the protocol
						continue // Skip incomplete URLs like http://localhost:
					}
				}
				validURLs = append(validURLs, url)
			}
		}
	}

	return validURLs
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

// Close shuts down the async worker
func (s *Store) Close() {
	// Finalize any remaining error clusters before closing
	s.mu.Lock()
	if remainingClusters := s.timeBasedParser.ForceCompleteAll(); len(remainingClusters) > 0 {
		for _, cluster := range remainingClusters {
			errorCtx := cluster.ToErrorContext()
			s.errorContexts = append(s.errorContexts, errorCtx)
		}
	}
	s.mu.Unlock()
	
	close(s.closeChan)
	s.wg.Wait()
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
	for url, urlEntry := range s.urlMap {
		// Validate URL before including it
		if s.isValidURL(url) {
			s.urls = append(s.urls, *urlEntry)
		} else {
			// Remove invalid URLs from the map
			delete(s.urlMap, url)
		}
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

// DetectURLsInContent detects URLs in the given content without storing them
func (s *Store) DetectURLsInContent(content string) []string {
	return detectURLs(content)
}

// GetAllCollapsed returns all log entries with consecutive duplicates collapsed
func (s *Store) GetAllCollapsed() []CollapsedLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.collapseConsecutiveDuplicates(s.entries)
}

// GetByProcessCollapsed returns collapsed log entries for a specific process
func (s *Store) GetByProcessCollapsed(processID string) []CollapsedLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indices, exists := s.byProcess[processID]
	if !exists {
		return []CollapsedLogEntry{}
	}

	// Build the log entries for this process
	entries := make([]LogEntry, 0, len(indices))
	for _, idx := range indices {
		if idx >= 0 && idx < len(s.entries) {
			entries = append(entries, s.entries[idx])
		}
	}

	return s.collapseConsecutiveDuplicates(entries)
}

// collapseConsecutiveDuplicates takes a slice of LogEntry and returns collapsed entries
func (s *Store) collapseConsecutiveDuplicates(entries []LogEntry) []CollapsedLogEntry {
	if len(entries) == 0 {
		return []CollapsedLogEntry{}
	}

	result := make([]CollapsedLogEntry, 0, len(entries))

	// Start with the first entry
	current := CollapsedLogEntry{
		LogEntry:    entries[0],
		Count:       1,
		FirstSeen:   entries[0].Timestamp,
		LastSeen:    entries[0].Timestamp,
		IsCollapsed: false,
	}

	for i := 1; i < len(entries); i++ {
		entry := entries[i]

		// Check if this entry is identical to the current one (same process and content)
		if s.areLogsIdentical(current.LogEntry, entry) {
			// Increment count and update last seen timestamp
			current.Count++
			current.LastSeen = entry.Timestamp
			current.IsCollapsed = current.Count > 1
		} else {
			// Different log entry, save the current one and start a new one
			result = append(result, current)
			current = CollapsedLogEntry{
				LogEntry:    entry,
				Count:       1,
				FirstSeen:   entry.Timestamp,
				LastSeen:    entry.Timestamp,
				IsCollapsed: false,
			}
		}
	}

	// Add the last entry
	result = append(result, current)

	return result
}

// areLogsIdentical checks if two log entries are identical for collapsing purposes
func (s *Store) areLogsIdentical(a, b LogEntry) bool {
	// Consider logs identical if they have the same process and content
	// We ignore timestamp and ID since those will naturally be different
	return a.ProcessID == b.ProcessID &&
		a.ProcessName == b.ProcessName &&
		a.Content == b.Content &&
		a.Level == b.Level &&
		a.IsError == b.IsError
}
