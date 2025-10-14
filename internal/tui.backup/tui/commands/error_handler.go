package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
)

// ErrorContext provides context for error handling
type ErrorContext struct {
	Operation   string            // What operation was attempted
	Component   string            // Which component had the error
	ProcessName string            // Process name if applicable
	LogStore    LogStoreInterface // Where to log the error
	UpdateChan  chan<- tea.Msg    // Channel for UI updates
}

// LogStoreInterface defines log storage operations for commands package
type LogStoreInterface interface {
	Add(processID, processName, message string, isError bool) *logs.LogEntry
}

// StandardErrorHandler provides consistent error handling across the TUI
type StandardErrorHandler struct {
	defaultLogStore   LogStoreInterface
	defaultUpdateChan chan<- tea.Msg
}

// NewStandardErrorHandler creates a new error handler
func NewStandardErrorHandler(logStore LogStoreInterface, updateChan chan<- tea.Msg) *StandardErrorHandler {
	return &StandardErrorHandler{
		defaultLogStore:   logStore,
		defaultUpdateChan: updateChan,
	}
}

// HandleError handles an error with consistent formatting and logging
func (h *StandardErrorHandler) HandleError(err error, ctx ErrorContext) {
	if err == nil {
		return
	}

	// Use defaults if not provided
	logStore := ctx.LogStore
	if logStore == nil {
		logStore = h.defaultLogStore
	}
	updateChan := ctx.UpdateChan
	if updateChan == nil {
		updateChan = h.defaultUpdateChan
	}

	// Format error message consistently
	var message string
	if ctx.ProcessName != "" {
		message = fmt.Sprintf("Error %s for process '%s': %v", ctx.Operation, ctx.ProcessName, err)
	} else {
		message = fmt.Sprintf("Error %s: %v", ctx.Operation, err)
	}

	// Log the error
	if logStore != nil {
		logStore.Add("system", ctx.Component, message, true)
	}

	// Trigger UI update
	if updateChan != nil {
		updateChan <- logUpdateMsg{}
	}
}

// HandleSuccess handles successful operations with consistent messaging
func (h *StandardErrorHandler) HandleSuccess(message string, ctx ErrorContext) {
	// Use defaults if not provided
	logStore := ctx.LogStore
	if logStore == nil {
		logStore = h.defaultLogStore
	}
	updateChan := ctx.UpdateChan
	if updateChan == nil {
		updateChan = h.defaultUpdateChan
	}

	// Log the success
	if logStore != nil {
		logStore.Add("system", ctx.Component, message, false)
	}

	// Trigger UI update
	if updateChan != nil {
		updateChan <- processUpdateMsg{}
	}
}

// Common error contexts for frequent operations

// ScriptStartContext creates an error context for script starting
func ScriptStartContext(scriptName, component string, logStore LogStoreInterface, updateChan chan<- tea.Msg) ErrorContext {
	return ErrorContext{
		Operation:   "starting script",
		Component:   component,
		ProcessName: scriptName,
		LogStore:    logStore,
		UpdateChan:  updateChan,
	}
}

// ProcessStopContext creates an error context for process stopping
func ProcessStopContext(processName, component string, logStore LogStoreInterface, updateChan chan<- tea.Msg) ErrorContext {
	return ErrorContext{
		Operation:   "stopping process",
		Component:   component,
		ProcessName: processName,
		LogStore:    logStore,
		UpdateChan:  updateChan,
	}
}

// ProcessRestartContext creates an error context for process restarting
func ProcessRestartContext(processName, component string, logStore LogStoreInterface, updateChan chan<- tea.Msg) ErrorContext {
	return ErrorContext{
		Operation:   "restarting process",
		Component:   component,
		ProcessName: processName,
		LogStore:    logStore,
		UpdateChan:  updateChan,
	}
}
