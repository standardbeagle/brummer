package tui

import (
	"sync/atomic"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// TUIDataProvider implements aicoder.BrummerDataProvider interface
// It provides thread-safe access to TUI model data for PTY sessions
type TUIDataProvider struct {
	model atomic.Value // stores *Model
}

// NewTUIDataProvider creates a new TUI data provider
func NewTUIDataProvider(model *Model) aicoder.BrummerDataProvider {
	p := &TUIDataProvider{}
	p.model.Store(model)
	return p
}

// SetModel sets the model reference for the data provider
func (p *TUIDataProvider) SetModel(model *Model) {
	p.model.Store(model)
}

// GetLastError returns the most recent error context
func (p *TUIDataProvider) GetLastError() *logs.ErrorContext {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.logStore == nil {
		return nil
	}

	contexts := model.logStore.GetErrorContexts()
	if len(contexts) > 0 {
		return &contexts[0]
	}
	return nil
}

// GetRecentLogs returns recent log entries
func (p *TUIDataProvider) GetRecentLogs(count int) []logs.LogEntry {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.logStore == nil {
		return []logs.LogEntry{}
	}

	allLogs := model.logStore.GetAll()
	if len(allLogs) <= count {
		return allLogs
	}

	// Return the most recent logs
	return allLogs[len(allLogs)-count:]
}

// GetTestFailures returns test failure information
func (p *TUIDataProvider) GetTestFailures() interface{} {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.logStore == nil {
		return nil
	}

	// Get test-related errors
	var testFailures []logs.ErrorContext
	contexts := model.logStore.GetErrorContexts()

	// Get last 10 test-related errors
	count := 0
	for i := len(contexts) - 1; i >= 0 && count < 10; i-- {
		ctx := contexts[i]
		if ctx.Type == "test_failure" || ctx.Type == "test_error" {
			testFailures = append(testFailures, ctx)
			count++
		}
	}

	return testFailures
}

// GetBuildOutput returns recent build output
func (p *TUIDataProvider) GetBuildOutput() string {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.logStore == nil {
		return ""
	}

	// Get logs from build-related processes
	var buildOutput string
	logs := model.logStore.GetAll()

	// Look for build-related logs in the last 50 entries
	start := len(logs) - 50
	if start < 0 {
		start = 0
	}

	for i := start; i < len(logs); i++ {
		log := logs[i]
		// Check if this is a build-related log
		if log.ProcessName == "build" || log.ProcessName == "compile" ||
			log.ProcessName == "webpack" || log.ProcessName == "vite" ||
			log.ProcessName == "go build" || log.ProcessName == "make" {
			buildOutput += log.Content + "\n"
		}
	}

	return buildOutput
}

// GetProcessInfo returns information about running processes
func (p *TUIDataProvider) GetProcessInfo() interface{} {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.processMgr == nil {
		return nil
	}

	processes := model.processMgr.GetAllProcesses()

	// Create a simplified process info structure
	type ProcessInfo struct {
		ID       string
		Name     string
		Status   string
		PID      int
		Started  string
		Duration string
	}

	var processInfos []ProcessInfo
	for _, proc := range processes {
		// Use ProcessState for atomic access to multiple fields
		state := proc.GetStateAtomic()

		info := ProcessInfo{
			ID:      state.ID,
			Name:    state.Name,
			Status:  string(state.Status),
			PID:     0, // Process PID is not exposed
			Started: state.StartTime.Format("15:04:05"),
		}

		if state.EndTime != nil {
			info.Duration = state.EndTime.Sub(state.StartTime).String()
		} else {
			info.Duration = "Running"
		}

		processInfos = append(processInfos, info)
	}

	return processInfos
}

// GetDetectedURLs returns URLs detected in process logs
func (p *TUIDataProvider) GetDetectedURLs() []logs.URLEntry {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.logStore == nil {
		return []logs.URLEntry{}
	}

	return model.logStore.GetURLs()
}

// GetRecentProxyRequests returns recent proxy requests
func (p *TUIDataProvider) GetRecentProxyRequests(count int) []*proxy.Request {
	model, ok := p.model.Load().(*Model)
	if !ok || model == nil || model.proxyServer == nil {
		return []*proxy.Request{}
	}

	allRequests := model.proxyServer.GetRequests()
	if len(allRequests) <= count {
		// Convert to pointer slice
		result := make([]*proxy.Request, len(allRequests))
		for i := range allRequests {
			result[i] = &allRequests[i]
		}
		return result
	}

	// Return the most recent requests as pointers
	start := len(allRequests) - count
	result := make([]*proxy.Request, count)
	for i := 0; i < count; i++ {
		result[i] = &allRequests[start+i]
	}
	return result
}
