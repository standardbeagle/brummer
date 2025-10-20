package fixtures

import (
	"fmt"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/state"
)

// Test state fixtures for state objects

// CreateTestErrorsState creates an ErrorsState with n test errors
func CreateTestErrorsState(n int) *state.ErrorsState {
	errorsState := state.NewErrorsState()

	var contexts []logs.ErrorContext
	for i := 0; i < n; i++ {
		context := logs.ErrorContext{
			ID:          fmt.Sprintf("error-%d", i),
			ProcessID:   fmt.Sprintf("proc-%d", 1000+i),
			ProcessName: fmt.Sprintf("test-process-%d", i%3),
			Timestamp:   time.Now().Add(-time.Duration(i) * time.Minute),
			Type:        "Error",
			Message:     fmt.Sprintf("Error: Test error %d", i),
			Stack: []string{
				fmt.Sprintf("  at function%d (file%d.js:%d:10)", i, i, i*10),
				fmt.Sprintf("  at module%d (file%d.js:%d:5)", i, i, i*5),
			},
			Context:  []string{fmt.Sprintf("Additional context %d", i)},
			Severity: "error",
			Language: "js",
		}
		contexts = append(contexts, context)
	}

	return errorsState.UpdateErrorContexts(contexts)
}

// CreateTestLogEntries creates log entries with specific counts for each level
func CreateTestLogEntries(info, warn, errorCount, debug int) []logs.LogEntry {
	var logEntries []logs.LogEntry
	now := time.Now()

	// Add info logs
	for i := 0; i < info; i++ {
		logEntries = append(logEntries, logs.LogEntry{
			ID:          fmt.Sprintf("log-info-%d", i),
			Timestamp:   now.Add(-time.Duration(i) * time.Second),
			ProcessID:   fmt.Sprintf("proc-%d", 1000+i),
			ProcessName: fmt.Sprintf("test-process-%d", i%3),
			Content:     fmt.Sprintf("Info message %d", i),
			Level:       logs.LevelInfo,
			IsError:     false,
			Tags:        []string{"info"},
			Priority:    1,
		})
	}

	// Add warning logs
	for i := 0; i < warn; i++ {
		logEntries = append(logEntries, logs.LogEntry{
			ID:          fmt.Sprintf("log-warn-%d", i),
			Timestamp:   now.Add(-time.Duration(i+info) * time.Second),
			ProcessID:   fmt.Sprintf("proc-%d", 1000+i),
			ProcessName: fmt.Sprintf("test-process-%d", i%3),
			Content:     fmt.Sprintf("Warning message %d", i),
			Level:       logs.LevelWarn,
			IsError:     false,
			Tags:        []string{"warning"},
			Priority:    2,
		})
	}

	// Add error logs
	for i := 0; i < errorCount; i++ {
		logEntries = append(logEntries, logs.LogEntry{
			ID:          fmt.Sprintf("log-error-%d", i),
			Timestamp:   now.Add(-time.Duration(i+info+warn) * time.Second),
			ProcessID:   fmt.Sprintf("proc-%d", 1000+i),
			ProcessName: fmt.Sprintf("test-process-%d", i%3),
			Content:     fmt.Sprintf("Error message %d", i),
			Level:       logs.LevelError,
			IsError:     true,
			Tags:        []string{"error"},
			Priority:    3,
		})
	}

	// Add debug logs
	for i := 0; i < debug; i++ {
		logEntries = append(logEntries, logs.LogEntry{
			ID:          fmt.Sprintf("log-debug-%d", i),
			Timestamp:   now.Add(-time.Duration(i+info+warn+errorCount) * time.Second),
			ProcessID:   fmt.Sprintf("proc-%d", 1000+i),
			ProcessName: fmt.Sprintf("test-process-%d", i%3),
			Content:     fmt.Sprintf("Debug message %d", i),
			Level:       logs.LevelDebug,
			IsError:     false,
			Tags:        []string{"debug"},
			Priority:    0,
		})
	}

	return logEntries
}

// CreateTestLogsState creates a LogsState with n test log entries
func CreateTestLogsState(n int) *state.LogsState {
	logsState := state.NewLogsState()

	// For now, we'll just return the basic state
	// The actual log caching would be done through UpdateCachedLogs method
	return logsState
}

// CreateTestWebViewState creates a WebViewState with n test requests
func CreateTestWebViewState(n int) *state.WebViewState {
	webViewState := state.NewWebViewState()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	statusCodes := []int{200, 201, 400, 404, 500}

	for i := 0; i < n; i++ {
		request := state.RequestInfo{
			ID:          fmt.Sprintf("event-%d", i),
			Timestamp:   time.Now().Add(-time.Duration(i) * time.Minute),
			Method:      methods[i%len(methods)],
			URL:         fmt.Sprintf("https://api.example.com/resource/%d", i),
			Path:        fmt.Sprintf("/resource/%d", i),
			StatusCode:  statusCodes[i%len(statusCodes)],
			Duration:    time.Duration(i%1000) * time.Millisecond,
			ProcessName: fmt.Sprintf("test-process-%d", i%3),
			Size:        int64(100 + i*10),
		}
		webViewState = webViewState.AddRequest(request)
	}

	return webViewState
}


// CreateTestMasterData creates a complete MasterData with all states populated
func CreateTestMasterData() *state.MasterData {
	masterData := state.NewMasterData()

	// Populate available states with test data
	masterData.ErrorsState = CreateTestErrorsState(10)
	masterData.WebViewState = CreateTestWebViewState(20)
	// Other states will be initialized with defaults

	return masterData
}

// CreateEmptyMasterData creates an empty MasterData
func CreateEmptyMasterData() *state.MasterData {
	return state.NewMasterData()
}

// CreateLargeTestMasterData creates a MasterData with large datasets for performance testing
func CreateLargeTestMasterData() *state.MasterData {
	masterData := state.NewMasterData()

	// Large datasets
	masterData.ErrorsState = CreateTestErrorsState(1000)
	masterData.WebViewState = CreateTestWebViewState(1000)

	return masterData
}