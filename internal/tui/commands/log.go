package commands

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
)

// LogCommand represents a log operation command
type LogCommand struct {
	*BaseCommand
	operation    func() (interface{}, error)
	logStore     *logs.Store
	operationType LogOperationType
	processID    string
	pattern      string
	highPriority bool
}

// LogOperationType represents the type of log operation
type LogOperationType int

const (
	LogGetAll LogOperationType = iota
	LogGetByProcess
	LogSearch
	LogGetHighPriority
	LogGetErrors
	LogGetURLs
	LogClear
	LogFilter
)

// LogResultMsg represents the result of a log command
type LogResultMsg struct {
	ID           string               `json:"id"`
	Type         CommandType          `json:"type"`
	Result       interface{}          `json:"result,omitempty"`
	Error        error                `json:"error,omitempty"`
	Context      OperationContext     `json:"context"`
	Duration     time.Duration        `json:"duration"`
	OperationType LogOperationType    `json:"operationType"`
	ProcessID    string               `json:"processID,omitempty"`
	Pattern      string               `json:"pattern,omitempty"`
}

// NewLogCommand creates a new log command
func NewLogCommand(id string, logStore *logs.Store, opType LogOperationType, operation func() (interface{}, error), context OperationContext) *LogCommand {
	return &LogCommand{
		BaseCommand:   NewBaseCommand(id, CommandTypeSystem, PriorityMedium, context),
		operation:     operation,
		logStore:      logStore,
		operationType: opType,
	}
}

// Run executes the log command
func (c *LogCommand) Run() tea.Msg {
	start := time.Now()

	// Create command context for timeout handling
	cmdCtx := NewCommandContext(c.Context())
	defer cmdCtx.Cleanup()

	var result interface{}
	var err error

	// Execute operation with timeout
	result, err = cmdCtx.ExecuteWithTimeout(c.operation)
	duration := time.Since(start)

	// Check for timeout
	if cmdCtx.IsTimedOut() {
		err = fmt.Errorf("log operation timed out after %v", duration)
	}

	return LogResultMsg{
		ID:            c.ID(),
		Type:          c.Type(),
		Result:        result,
		Error:         err,
		Context:       c.Context(),
		Duration:      duration,
		OperationType: c.operationType,
		ProcessID:     c.processID,
		Pattern:       c.pattern,
	}
}

// NewGetAllLogsCommand creates a command to get all logs
func NewGetAllLogsCommand(id string, logStore *logs.Store, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		entries := logStore.GetAllCollapsed()
		return LogData{
			Entries: entries,
			Count:   len(entries),
			Type:    "all",
		}, nil
	}

	return NewLogCommand(id, logStore, LogGetAll, operation, context)
}

// NewGetProcessLogsCommand creates a command to get logs for a specific process
func NewGetProcessLogsCommand(id string, logStore *logs.Store, processID string, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		entries := logStore.GetByProcessCollapsed(processID)
		return LogData{
			Entries:  entries,
			Count:    len(entries),
			Type:     "process",
			ProcessID: processID,
		}, nil
	}

	cmd := NewLogCommand(id, logStore, LogGetByProcess, operation, context)
	cmd.processID = processID
	return cmd
}

// NewSearchLogsCommand creates a command to search logs
func NewSearchLogsCommand(id string, logStore *logs.Store, query string, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		// Convert to collapsed entries for consistency
		collapsedEntries := logStore.GetAllCollapsed()
		var filteredCollapsed []logs.CollapsedLogEntry

		queryLower := strings.ToLower(query)
		for _, entry := range collapsedEntries {
			if strings.Contains(strings.ToLower(entry.Content), queryLower) ||
			   strings.Contains(strings.ToLower(entry.ProcessName), queryLower) {
				filteredCollapsed = append(filteredCollapsed, entry)
			}
		}

		return LogData{
			Entries: filteredCollapsed,
			Count:   len(filteredCollapsed),
			Type:    "search",
			Query:   query,
		}, nil
	}

	cmd := NewLogCommand(id, logStore, LogSearch, operation, context)
	cmd.pattern = query
	return cmd
}

// NewGetHighPriorityLogsCommand creates a command to get high priority logs
func NewGetHighPriorityLogsCommand(id string, logStore *logs.Store, threshold int, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		// Convert to collapsed entries for consistency
		allCollapsed := logStore.GetAllCollapsed()
		var highPriorityCollapsed []logs.CollapsedLogEntry

		for _, entry := range allCollapsed {
			if entry.Priority >= threshold {
				highPriorityCollapsed = append(highPriorityCollapsed, entry)
			}
		}

		return LogData{
			Entries:     highPriorityCollapsed,
			Count:       len(highPriorityCollapsed),
			Type:        "high_priority",
			Threshold:   threshold,
		}, nil
	}

	cmd := NewLogCommand(id, logStore, LogGetHighPriority, operation, context)
	cmd.highPriority = true
	return cmd
}

// NewGetErrorsCommand creates a command to get error logs
func NewGetErrorsCommand(id string, logStore *logs.Store, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		entries := logStore.GetErrors()
		return ErrorData{
			Entries: entries,
			Count:   len(entries),
			Type:    "errors",
		}, nil
	}

	return NewLogCommand(id, logStore, LogGetErrors, operation, context)
}

// NewGetURLsCommand creates a command to get URLs from logs
func NewGetURLsCommand(id string, logStore *logs.Store, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		entries := logStore.GetURLs()
		return URLData{
			Entries: entries,
			Count:   len(entries),
			Type:    "urls",
		}, nil
	}

	return NewLogCommand(id, logStore, LogGetURLs, operation, context)
}

// NewClearLogsCommand creates a command to clear logs
func NewClearLogsCommand(id string, logStore *logs.Store, processName string, context OperationContext) *LogCommand {
	operation := func() (interface{}, error) {
		if processName != "" {
			logStore.ClearLogsForProcess(processName)
			return ClearResult{
				Type:        "process",
				ProcessName: processName,
				Status:      "cleared",
			}, nil
		} else {
			logStore.ClearLogs()
			return ClearResult{
				Type:   "all",
				Status: "cleared",
			}, nil
		}
	}

	cmd := NewLogCommand(id, logStore, LogClear, operation, context)
	cmd.processID = processName
	return cmd
}

// Data structures for log operations

type LogData struct {
	Entries  []logs.CollapsedLogEntry `json:"entries"`
	Count    int                      `json:"count"`
	Type     string                   `json:"type"`
	ProcessID string                  `json:"processID,omitempty"`
	Query    string                   `json:"query,omitempty"`
	Threshold int                     `json:"threshold,omitempty"`
}

type ErrorData struct {
	Entries []logs.LogEntry `json:"entries"`
	Count   int             `json:"count"`
	Type    string          `json:"type"`
}

type URLData struct {
	Entries []logs.URLEntry `json:"entries"`
	Count   int             `json:"count"`
	Type    string          `json:"type"`
}

type ClearResult struct {
	Type        string `json:"type"`
	ProcessName string `json:"processName,omitempty"`
	Status      string `json:"status"`
}

// Message interface implementation

func (m LogResultMsg) Message() tea.Msg {
	return m
}

// IsSuccess returns true if the log operation completed successfully
func (m LogResultMsg) IsSuccess() bool {
	return m.Error == nil
}

// IsFailure returns true if the log operation failed
func (m LogResultMsg) IsFailure() bool {
	return m.Error != nil
}

// String returns a string representation of the log result
func (m LogResultMsg) String() string {
	if m.IsSuccess() {
		return fmt.Sprintf("LogResult{id=%s, type=%v, operation=%v, duration=%v}",
			m.ID, m.Type, m.OperationType, m.Duration)
	}
	return fmt.Sprintf("LogResult{id=%s, type=%v, operation=%v, error=%v, duration=%v}",
		m.ID, m.Type, m.OperationType, m.Error, m.Duration)
}