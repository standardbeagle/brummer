package aicoder

import (
	"fmt"
	"strings"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/proxy"
)

// DataInjectionType represents the type of data being injected
type DataInjectionType string

const (
	DataInjectError       DataInjectionType = "error"
	DataInjectLastError   DataInjectionType = "last_error"
	DataInjectLogs        DataInjectionType = "logs"
	DataInjectTestFailure DataInjectionType = "test_failure"
	DataInjectBuildOutput DataInjectionType = "build_output"
	DataInjectProcessInfo DataInjectionType = "process_info"
	DataInjectURLs        DataInjectionType = "urls"
	DataInjectProxyReq    DataInjectionType = "proxy_request"
	DataInjectSystemMsg   DataInjectionType = "system_message"
)

// DataInjector handles formatting and injecting Brummer data into AI sessions
type DataInjector struct {
	// Configuration for data formatting
	maxLines      int
	maxLength     int
	includeColors bool
}

// NewDataInjector creates a new data injector
func NewDataInjector() *DataInjector {
	return &DataInjector{
		maxLines:      20,    // Maximum lines to inject
		maxLength:     2000,  // Maximum character length
		includeColors: false, // Disable colors for now
	}
}

// FormatData formats data based on its type for injection into terminal
func (di *DataInjector) FormatData(dataType DataInjectionType, data interface{}) (string, error) {
	switch dataType {
	case DataInjectError, DataInjectLastError:
		return di.formatError(data)
	case DataInjectLogs:
		return di.formatLogs(data)
	case DataInjectTestFailure:
		return di.formatTestFailure(data)
	case DataInjectBuildOutput:
		return di.formatBuildOutput(data)
	case DataInjectProcessInfo:
		return di.formatProcessInfo(data)
	case DataInjectURLs:
		return di.formatURLs(data)
	case DataInjectProxyReq:
		return di.formatProxyRequest(data)
	case DataInjectSystemMsg:
		return di.formatSystemMessage(data)
	default:
		return fmt.Sprintf("Unknown data type: %v", data), nil
	}
}

// GetDataTypeLabel returns a human-readable label for the data type
func (di *DataInjector) GetDataTypeLabel(dataType DataInjectionType) string {
	switch dataType {
	case DataInjectError:
		return "ERROR"
	case DataInjectLastError:
		return "LAST ERROR"
	case DataInjectLogs:
		return "RECENT LOGS"
	case DataInjectTestFailure:
		return "TEST FAILURE"
	case DataInjectBuildOutput:
		return "BUILD OUTPUT"
	case DataInjectProcessInfo:
		return "PROCESS INFO"
	case DataInjectURLs:
		return "DETECTED URLS"
	case DataInjectProxyReq:
		return "PROXY REQUEST"
	case DataInjectSystemMsg:
		return "SYSTEM MESSAGE"
	default:
		return "DATA"
	}
}

// formatError formats error context for injection
func (di *DataInjector) formatError(data interface{}) (string, error) {
	switch v := data.(type) {
	case *logs.ErrorContext:
		var result strings.Builder

		result.WriteString(fmt.Sprintf("Type: %s\n", v.Type))
		result.WriteString(fmt.Sprintf("Severity: %s\n", v.Severity))
		result.WriteString(fmt.Sprintf("Process: %s\n", v.ProcessName))
		result.WriteString(fmt.Sprintf("Time: %s\n", v.Timestamp.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("Message: %s\n", v.Message))

		// ErrorContext doesn't have File/Line fields directly
		// Check if we can extract file info from context or stack
		if len(v.Stack) > 0 {
			result.WriteString("Stack trace:\n")
			for i, stackLine := range v.Stack {
				if i >= 5 { // Limit stack trace lines
					result.WriteString("  ... (truncated)\n")
					break
				}
				result.WriteString(fmt.Sprintf("  %s\n", stackLine))
			}
		}

		if len(v.Context) > 0 {
			result.WriteString("Context:\n")
			for i, line := range v.Context {
				if i >= di.maxLines {
					result.WriteString("... (truncated)\n")
					break
				}
				result.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}

		return di.truncate(result.String()), nil

	case string:
		return di.truncate(v), nil

	default:
		return di.truncate(fmt.Sprintf("%v", data)), nil
	}
}

// formatLogs formats log entries for injection
func (di *DataInjector) formatLogs(data interface{}) (string, error) {
	switch v := data.(type) {
	case []logs.LogEntry:
		var result strings.Builder

		for i, entry := range v {
			if i >= di.maxLines {
				result.WriteString("... (truncated)\n")
				break
			}

			result.WriteString(fmt.Sprintf("[%s] %s: %s\n",
				entry.Timestamp.Format("15:04:05"),
				entry.ProcessName,
				entry.Content))
		}

		return di.truncate(result.String()), nil

	case string:
		return di.truncate(v), nil

	default:
		return di.truncate(fmt.Sprintf("%v", data)), nil
	}
}

// formatTestFailure formats test failure information
func (di *DataInjector) formatTestFailure(data interface{}) (string, error) {
	// This would be expanded based on actual test failure data structure
	return di.truncate(fmt.Sprintf("Test Failure Details:\n%v", data)), nil
}

// formatBuildOutput formats build output
func (di *DataInjector) formatBuildOutput(data interface{}) (string, error) {
	switch v := data.(type) {
	case string:
		lines := strings.Split(v, "\n")
		if len(lines) > di.maxLines {
			lines = lines[len(lines)-di.maxLines:]
			lines[0] = "... (showing last " + fmt.Sprintf("%d", di.maxLines) + " lines)"
		}
		return di.truncate(strings.Join(lines, "\n")), nil

	default:
		return di.truncate(fmt.Sprintf("%v", data)), nil
	}
}

// formatProcessInfo formats process information
func (di *DataInjector) formatProcessInfo(data interface{}) (string, error) {
	// This would format process status, resource usage, etc.
	return di.truncate(fmt.Sprintf("Process Information:\n%v", data)), nil
}

// formatURLs formats detected URLs
func (di *DataInjector) formatURLs(data interface{}) (string, error) {
	switch v := data.(type) {
	case []logs.URLEntry:
		var result strings.Builder

		for i, url := range v {
			if i >= di.maxLines {
				result.WriteString("... (truncated)\n")
				break
			}

			result.WriteString(fmt.Sprintf("%s", url.URL))
			if url.ProxyURL != "" {
				result.WriteString(fmt.Sprintf(" → %s", url.ProxyURL))
			}
			result.WriteString(fmt.Sprintf(" (%s)\n", url.ProcessName))
		}

		return di.truncate(result.String()), nil

	case string:
		return di.truncate(v), nil

	default:
		return di.truncate(fmt.Sprintf("%v", data)), nil
	}
}

// formatProxyRequest formats proxy request information
func (di *DataInjector) formatProxyRequest(data interface{}) (string, error) {
	switch v := data.(type) {
	case *proxy.Request:
		var result strings.Builder

		result.WriteString(fmt.Sprintf("Method: %s\n", v.Method))
		result.WriteString(fmt.Sprintf("URL: %s\n", v.URL))
		result.WriteString(fmt.Sprintf("Status: %d\n", v.StatusCode))
		result.WriteString(fmt.Sprintf("Duration: %s\n", v.Duration))
		result.WriteString(fmt.Sprintf("Time: %s\n", v.StartTime.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("Host: %s\n", v.Host))
		result.WriteString(fmt.Sprintf("Path: %s\n", v.Path))

		if v.Error != "" {
			result.WriteString(fmt.Sprintf("Error: %s\n", v.Error))
		}

		if v.ProcessName != "" {
			result.WriteString(fmt.Sprintf("Process: %s\n", v.ProcessName))
		}

		return di.truncate(result.String()), nil

	default:
		return di.truncate(fmt.Sprintf("%v", data)), nil
	}
}

// formatSystemMessage formats system messages
func (di *DataInjector) formatSystemMessage(data interface{}) (string, error) {
	return di.truncate(fmt.Sprintf("%v", data)), nil
}

// truncate truncates text to maximum length
func (di *DataInjector) truncate(text string) string {
	if len(text) <= di.maxLength {
		return text
	}

	return text[:di.maxLength-3] + "..."
}

// KeyBinding represents a key combination for data injection
type KeyBinding struct {
	Key         string
	Description string
	DataType    DataInjectionType
}

// GetDefaultKeyBindings returns the default key bindings for data injection
func GetDefaultKeyBindings() []KeyBinding {
	return []KeyBinding{
		{
			Key:         "ctrl+e",
			Description: "Inject last error message",
			DataType:    DataInjectLastError,
		},
		{
			Key:         "ctrl+l",
			Description: "Inject recent log lines",
			DataType:    DataInjectLogs,
		},
		{
			Key:         "ctrl+t",
			Description: "Inject test failure details",
			DataType:    DataInjectTestFailure,
		},
		{
			Key:         "ctrl+b",
			Description: "Inject build output",
			DataType:    DataInjectBuildOutput,
		},
		{
			Key:         "ctrl+p",
			Description: "Inject process information",
			DataType:    DataInjectProcessInfo,
		},
		{
			Key:         "ctrl+u",
			Description: "Inject detected URLs",
			DataType:    DataInjectURLs,
		},
		{
			Key:         "ctrl+r",
			Description: "Inject proxy request details",
			DataType:    DataInjectProxyReq,
		},
	}
}

// BrummerDataProvider interface for getting data from Brummer
type BrummerDataProvider interface {
	GetLastError() *logs.ErrorContext
	GetRecentLogs(count int) []logs.LogEntry
	GetTestFailures() interface{}
	GetBuildOutput() string
	GetProcessInfo() interface{}
	GetDetectedURLs() []logs.URLEntry
	GetRecentProxyRequests(count int) []*proxy.Request
}
