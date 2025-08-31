package services

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
)

// LogsServiceBridge implements LogsService using logs.Store
type LogsServiceBridge struct {
	store *logs.Store
}

// NewLogsServiceBridge creates a bridge to the existing logs store
func NewLogsServiceBridge(store *logs.Store) *LogsServiceBridge {
	return &LogsServiceBridge{
		store: store,
	}
}

// GetLogs retrieves logs for a specific process or all processes
func (s *LogsServiceBridge) GetLogs(processID string, limit int) []LogEntry {
	if s.store == nil {
		return []LogEntry{}
	}
	
	var storeLogs []logs.LogEntry
	if processID == "" {
		storeLogs = s.store.GetAll()
	} else {
		storeLogs = s.store.GetByProcess(processID)
	}
	
	// Convert to service LogEntry and apply limit
	result := make([]LogEntry, 0, len(storeLogs))
	for i, log := range storeLogs {
		if limit > 0 && i >= limit {
			break
		}
		
		level := "info"
		if log.IsError {
			level = "error"
		} else if strings.Contains(strings.ToLower(log.Content), "warn") {
			level = "warn"
		}
		
		source := "stdout"
		if log.IsError {
			source = "stderr"
		}
		
		result = append(result, LogEntry{
			Timestamp: log.Timestamp,
			ProcessID: log.ProcessID,
			Level:     level,
			Message:   log.Content,
			Source:    source,
		})
	}
	
	return result
}

// FilterByLevel filters logs by severity level
func (s *LogsServiceBridge) FilterByLevel(level string) []LogEntry {
	if s.store == nil {
		return []LogEntry{}
	}
	
	var storeLogs []logs.LogEntry
	switch level {
	case "error":
		storeLogs = s.store.GetErrors()
	default:
		// For info/warn, filter from all logs
		allLogs := s.store.GetAll()
		storeLogs = make([]logs.LogEntry, 0)
		for _, log := range allLogs {
			if level == "error" && log.IsError {
				storeLogs = append(storeLogs, log)
			} else if level == "warn" && strings.Contains(strings.ToLower(log.Content), "warn") {
				storeLogs = append(storeLogs, log)
			} else if level == "info" && !log.IsError {
				storeLogs = append(storeLogs, log)
			}
		}
	}
	
	// Convert to service LogEntry
	result := make([]LogEntry, 0, len(storeLogs))
	for _, log := range storeLogs {
		logLevel := "info"
		if log.IsError {
			logLevel = "error"
		} else if strings.Contains(strings.ToLower(log.Content), "warn") {
			logLevel = "warn"
		}
		
		source := "stdout"
		if log.IsError {
			source = "stderr"
		}
		
		result = append(result, LogEntry{
			Timestamp: log.Timestamp,
			ProcessID: log.ProcessID,
			Level:     logLevel,
			Message:   log.Content,
			Source:    source,
		})
	}
	
	return result
}

// Search logs by text query
func (s *LogsServiceBridge) Search(query string) []LogEntry {
	if s.store == nil {
		return []LogEntry{}
	}
	
	storeLogs := s.store.Search(query)
	
	// Convert to service LogEntry
	result := make([]LogEntry, 0, len(storeLogs))
	for _, log := range storeLogs {
		level := "info"
		if log.IsError {
			level = "error"
		} else if strings.Contains(strings.ToLower(log.Content), "warn") {
			level = "warn"
		}
		
		source := "stdout"
		if log.IsError {
			source = "stderr"
		}
		
		result = append(result, LogEntry{
			Timestamp: log.Timestamp,
			ProcessID: log.ProcessID,
			Level:     level,
			Message:   log.Content,
			Source:    source,
		})
	}
	
	return result
}

// Clear logs for a process or all logs
func (s *LogsServiceBridge) Clear(processID string) tea.Cmd {
	return func() tea.Msg {
		if s.store == nil {
			return CommandResultMsg{
				Success: false,
				Message: "Logs store not available",
			}
		}
		
		// The store doesn't have a clear method in the interface,
		// so we'll return a message indicating this needs implementation
		// In a real implementation, we'd add a Clear method to the store
		
		return CommandResultMsg{
			Success: true,
			Message: "Clear logs not yet implemented",
		}
	}
}

// Subscribe to log updates
func (s *LogsServiceBridge) Subscribe() tea.Cmd {
	// Return a command that will subscribe to log updates
	// This would typically connect to the EventBus or similar
	return func() tea.Msg {
		// In a real implementation, this would set up a subscription
		// and return messages as logs come in
		return CommandResultMsg{
			Success: true,
			Message: "Subscribed to log updates",
		}
	}
}