package tui

import (
	"sync"

	"github.com/standardbeagle/brummer/internal/aicoder"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// TUIDataProvider implements aicoder.BrummerDataProvider interface
// It provides thread-safe access to TUI model data for PTY sessions
type TUIDataProvider struct {
	model *Model
	mu    sync.RWMutex
}

// NewTUIDataProvider creates a new TUI data provider
func NewTUIDataProvider(model *Model) aicoder.BrummerDataProvider {
	return &TUIDataProvider{
		model: model,
	}
}

// GetLastError returns the most recent error context
func (p *TUIDataProvider) GetLastError() *logs.ErrorContext {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.logStore == nil {
		return nil
	}

	contexts := p.model.logStore.GetErrorContexts()
	if len(contexts) > 0 {
		return &contexts[0]
	}
	return nil
}

// GetRecentLogs returns recent log entries
func (p *TUIDataProvider) GetRecentLogs(count int) []logs.LogEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.logStore == nil {
		return []logs.LogEntry{}
	}

	allLogs := p.model.logStore.GetAll()
	if len(allLogs) <= count {
		return allLogs
	}

	// Return the most recent logs
	return allLogs[len(allLogs)-count:]
}

// GetTestFailures returns test failure information
func (p *TUIDataProvider) GetTestFailures() interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.logStore == nil {
		return nil
	}

	// Get test-related errors
	var testFailures []logs.ErrorContext
	contexts := p.model.logStore.GetErrorContexts()
	
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
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.logStore == nil {
		return ""
	}

	// Get logs from build-related processes
	var buildOutput string
	logs := p.model.logStore.GetAll()
	
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
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.processMgr == nil {
		return nil
	}

	processes := p.model.processMgr.GetAllProcesses()
	
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
		info := ProcessInfo{
			ID:       proc.ID,
			Name:     proc.Name,
			Status:   string(proc.Status),
			PID:      0, // Process PID is not exposed
			Started:  proc.StartTime.Format("15:04:05"),
		}
		
		if proc.EndTime != nil {
			info.Duration = proc.EndTime.Sub(proc.StartTime).String()
		} else {
			info.Duration = "Running"
		}
		
		processInfos = append(processInfos, info)
	}

	return processInfos
}

// GetDetectedURLs returns URLs detected in process logs
func (p *TUIDataProvider) GetDetectedURLs() []logs.URLEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.logStore == nil {
		return []logs.URLEntry{}
	}

	return p.model.logStore.GetURLs()
}

// GetRecentProxyRequests returns recent proxy requests
func (p *TUIDataProvider) GetRecentProxyRequests(count int) []*proxy.Request {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.model == nil || p.model.proxyServer == nil {
		return []*proxy.Request{}
	}

	allRequests := p.model.proxyServer.GetRequests()
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