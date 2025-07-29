package aicoder

import (
	"strings"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// BrummerDataProviderImpl implements the BrummerDataProvider interface
type BrummerDataProviderImpl struct {
	logStore    *logs.Store
	proxyServer *proxy.Server
}

// NewBrummerDataProvider creates a new Brummer data provider
func NewBrummerDataProvider(logStore *logs.Store, proxyServer *proxy.Server) *BrummerDataProviderImpl {
	return &BrummerDataProviderImpl{
		logStore:    logStore,
		proxyServer: proxyServer,
	}
}

// GetLastError returns the most recent error from the log store
func (bdp *BrummerDataProviderImpl) GetLastError() *logs.ErrorContext {
	if bdp.logStore == nil {
		return nil
	}
	
	errorContexts := bdp.logStore.GetErrorContexts()
	if len(errorContexts) == 0 {
		return nil
	}
	
	// Return the most recent error
	return &errorContexts[len(errorContexts)-1]
}

// GetRecentLogs returns recent log entries
func (bdp *BrummerDataProviderImpl) GetRecentLogs(count int) []logs.LogEntry {
	if bdp.logStore == nil {
		return nil
	}
	
	allLogs := bdp.logStore.GetAll()
	if len(allLogs) == 0 {
		return nil
	}
	
	// Return the most recent logs
	start := len(allLogs) - count
	if start < 0 {
		start = 0
	}
	
	return allLogs[start:]
}

// GetTestFailures returns test failure information
func (bdp *BrummerDataProviderImpl) GetTestFailures() interface{} {
	// This would be implemented to extract test failure information
	// from logs or other sources
	if bdp.logStore == nil {
		return "No test failure data available"
	}
	
	// Look for test-related errors
	errorContexts := bdp.logStore.GetErrorContexts()
	var testErrors []logs.ErrorContext
	
	for _, err := range errorContexts {
		if isTestRelatedError(&err) {
			testErrors = append(testErrors, err)
		}
	}
	
	if len(testErrors) == 0 {
		return "No test failures found"
	}
	
	return testErrors
}

// GetBuildOutput returns recent build output
func (bdp *BrummerDataProviderImpl) GetBuildOutput() string {
	if bdp.logStore == nil {
		return "No build output available"
	}
	
	// Look for build-related logs
	allLogs := bdp.logStore.GetAll()
	var buildLogs []string
	
	for i := len(allLogs) - 1; i >= 0 && len(buildLogs) < 20; i-- {
		log := allLogs[i]
		if isBuildRelatedLog(log) {
			buildLogs = append([]string{log.Content}, buildLogs...) // Prepend to maintain order
		}
	}
	
	if len(buildLogs) == 0 {
		return "No recent build output found"
	}
	
	result := "Recent build output:\n"
	for _, line := range buildLogs {
		result += line + "\n"
	}
	
	return result
}

// GetProcessInfo returns information about running processes
func (bdp *BrummerDataProviderImpl) GetProcessInfo() interface{} {
	// This would be implemented to return process information
	// For now, return a placeholder
	return "Process information would be available here"
}

// GetDetectedURLs returns detected URLs from the log store
func (bdp *BrummerDataProviderImpl) GetDetectedURLs() []logs.URLEntry {
	if bdp.logStore == nil {
		return nil
	}
	
	return bdp.logStore.GetURLs()
}

// GetRecentProxyRequests returns recent proxy requests
func (bdp *BrummerDataProviderImpl) GetRecentProxyRequests(count int) []*proxy.Request {
	if bdp.proxyServer == nil {
		return nil
	}
	
	allRequests := bdp.proxyServer.GetRequests()
	if len(allRequests) == 0 {
		return nil
	}
	
	// Return the most recent requests
	start := len(allRequests) - count
	if start < 0 {
		start = 0
	}
	
	// Convert []proxy.Request to []*proxy.Request
	result := make([]*proxy.Request, len(allRequests)-start)
	for i, req := range allRequests[start:] {
		result[i] = &req
	}
	
	return result
}

// Helper functions

// isTestRelatedError checks if an error is related to testing
func isTestRelatedError(err *logs.ErrorContext) bool {
	if err == nil {
		return false
	}
	
	testKeywords := []string{
		"test",
		"Test",
		"TEST",
		"spec",
		"Spec",
		"SPEC",
		"jest",
		"mocha",
		"pytest",
		"go test",
		"npm test",
		"yarn test",
		"pnpm test",
	}
	
	for _, keyword := range testKeywords {
		if containsString(err.Message, keyword) || 
		   containsString(err.ProcessName, keyword) {
			return true
		}
		
		// Check context lines for test-related patterns
		for _, contextLine := range err.Context {
			if containsString(contextLine, keyword) {
				return true
			}
		}
	}
	
	return false
}

// isBuildRelatedLog checks if a log entry is related to building
func isBuildRelatedLog(log logs.LogEntry) bool {
	buildKeywords := []string{
		"build",
		"Build",
		"BUILD",
		"compile",
		"Compile",
		"COMPILE",
		"webpack",
		"vite",
		"rollup",
		"tsc",
		"go build",
		"npm run build",
		"yarn build",
		"pnpm build",
		"make",
		"cmake",
	}
	
	for _, keyword := range buildKeywords {
		if containsString(log.Content, keyword) || 
		   containsString(log.ProcessName, keyword) {
			return true
		}
	}
	
	return false
}

// containsString checks if a string contains a substring (case-insensitive helper)
func containsString(text, substr string) bool {
	return len(text) >= len(substr) && 
		   (text == substr || 
		    strings.Contains(strings.ToLower(text), strings.ToLower(substr)))
}