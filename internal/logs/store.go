package logs

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/beagle/beagle-run/pkg/filters"
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
	ID         string
	ProcessID  string
	ProcessName string
	Timestamp  time.Time
	Content    string
	Level      LogLevel
	IsError    bool
	Tags       []string
	Priority   int
}

type Store struct {
	entries      []LogEntry
	byProcess    map[string][]int
	errors       []LogEntry
	urls         []URLEntry
	maxEntries   int
	filters      []filters.Filter
	mu           sync.RWMutex
}

type URLEntry struct {
	URL        string
	ProcessID  string
	ProcessName string
	Timestamp  time.Time
	Context    string
}

func NewStore(maxEntries int) *Store {
	return &Store{
		entries:    make([]LogEntry, 0, maxEntries),
		byProcess:  make(map[string][]int),
		errors:     make([]LogEntry, 0, 100),
		urls:       make([]URLEntry, 0, 100),
		maxEntries: maxEntries,
		filters:    []filters.Filter{},
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

	// Track errors
	if isError || entry.Level >= LevelError {
		s.errors = append(s.errors, entry)
		if len(s.errors) > 100 {
			s.errors = s.errors[1:]
		}
	}

	// Detect and track URLs
	urls := s.detectURLs(content)
	for _, url := range urls {
		urlEntry := URLEntry{
			URL:         url,
			ProcessID:   processID,
			ProcessName: processName,
			Timestamp:   entry.Timestamp,
			Context:     content,
		}
		s.urls = append(s.urls, urlEntry)
		if len(s.urls) > 100 {
			s.urls = s.urls[1:]
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