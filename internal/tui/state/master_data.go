package state

import (
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/shared/types"
)

// MasterData represents the single source of truth for all TUI state
// This wraps existing state structures without redefining them
type MasterData struct {
	// Version tracking for atomic updates
	version atomic.Uint64

	// Existing state structures - use composition
	StateManager      *StateManager
	OperationsTracker *OperationsTracker
	UiComponents      *UIComponentsManager

	// AI Coder sub-model for PTY/VT10x terminal state
	AICoderState      *AICoderState

	// Logs sub-model for logs display state
	LogsState         *LogsState

	// Processes sub-model for processes display state
	ProcessesState    *ProcessesState

	// Errors sub-model for errors display state
	ErrorsState       *ErrorsState

	// MCP sub-model for MCP server state
	MCPState          *MCPState

	// MCP Debug sub-model for MCP connection debugging
	MCPDebugState     *MCPDebugState

	// Settings sub-model for settings display state
	SettingsState     *SettingsState

	// URLs sub-model for URLs display state
	URLsState         *URLsState

	// Command Window sub-model for command window state
	CommandWindowState *CommandWindowState

	// Web View sub-model for web view state
	WebViewState       *WebViewState

	// Test View sub-model for test view state
	TestViewState      *TestViewState

	// Last update timestamp for tracking
	LastUpdate atomic.Value // time.Time
}

// NewMasterData creates a new master data object that wraps existing state
func NewMasterData() *MasterData {
	return &MasterData{
		// Initialize version
		version: atomic.Uint64{},

		// Initialize existing state structures
		StateManager:      NewStateManager(),
		OperationsTracker: NewOperationsTracker(),
		UiComponents:      NewUIComponentsManager(1000), // max size

		// Initialize AI Coder sub-model
		AICoderState:      NewAICoderState(),

		// Initialize Logs sub-model
		LogsState:         NewLogsState(),

		// Initialize Processes sub-model
		ProcessesState:    NewProcessesState(),

		// Initialize Errors sub-model
		ErrorsState:       NewErrorsState(),

		// Initialize MCP sub-model
		MCPState:          NewMCPState(),

		// Initialize MCP Debug sub-model
		MCPDebugState:     NewMCPDebugState(),

		// Initialize Settings sub-model
		SettingsState:     NewSettingsState(),

		// Initialize URLs sub-model
		URLsState:         NewURLsState(),

		// Initialize Command Window sub-model
		CommandWindowState: NewCommandWindowState(),

		// Initialize Web View sub-model
		WebViewState:       NewWebViewState(),

		// Initialize Test View sub-model
		TestViewState:      NewTestViewState(),

		// Initialize last update time
		LastUpdate: atomic.Value{},
	}
}

// UpdateVersion increments the version and updates timestamp
func (md *MasterData) UpdateVersion() uint64 {
	newVersion := md.version.Add(1)
	md.LastUpdate.Store(time.Now())
	return newVersion
}

// GetVersion returns the current version
func (md *MasterData) GetVersion() uint64 {
	return md.version.Load()
}

// GetLastUpdate returns the last update time
func (md *MasterData) GetLastUpdate() time.Time {
	if val := md.LastUpdate.Load(); val != nil {
		return val.(time.Time)
	}
	return time.Time{}
}

// GetStateSummary returns a summary of current state
func (md *MasterData) GetStateSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"version":     md.GetVersion(),
		"last_update": md.GetLastUpdate(),
		"operations":  md.OperationsTracker.GetStats(),
		// Note: Add other summaries as methods become available
	}

	// Add AI Coder state summary if available
	if md.AICoderState != nil {
		aicoderSummary := md.AICoderState.GetStateSummary()
		for key, value := range aicoderSummary {
			summary["aicoder_"+key] = value
		}
	}

	// Add MCP state summary if available
	if md.MCPState != nil {
		mcpSummary := md.MCPState.GetStateSummary()
		for key, value := range mcpSummary {
			summary["mcp_"+key] = value
		}
	}

	// Add MCP debug state summary if available
	if md.MCPDebugState != nil {
		mcpDebugSummary := md.MCPDebugState.GetStateSummary()
		for key, value := range mcpDebugSummary {
			summary["mcp_debug_"+key] = value
		}
	}

	// Add Settings state summary if available
	if md.SettingsState != nil {
		settingsSummary := md.SettingsState.GetStateSummary()
		for key, value := range settingsSummary {
			summary["settings_"+key] = value
		}
	}

	// Add URLs state summary if available
	if md.URLsState != nil {
		urlsSummary := md.URLsState.GetStateSummary()
		for key, value := range urlsSummary {
			summary["urls_"+key] = value
		}
	}

	// Add Command Window state summary if available
	if md.CommandWindowState != nil {
		cmdWindowSummary := md.CommandWindowState.GetStateSummary()
		for key, value := range cmdWindowSummary {
			summary["cmdwindow_"+key] = value
		}
	}

	// Add Web View state summary if available
	if md.WebViewState != nil {
		webViewSummary := md.WebViewState.GetStateSummary()
		for key, value := range webViewSummary {
			summary["webview_"+key] = value
		}
	}

	// Add Test View state summary if available
	if md.TestViewState != nil {
		testViewSummary := md.TestViewState.GetStateSummary()
		for key, value := range testViewSummary {
			summary["testview_"+key] = value
		}
	}

	return summary
}

// ValidateState checks for state consistency across all components
func (md *MasterData) ValidateState() []string {
	var issues []string

	// Check version consistency
	if md.GetVersion() == 0 {
		issues = append(issues, "version is 0 - state not initialized")
	}

	// Basic validation - check for nil pointers
	if md.StateManager == nil {
		issues = append(issues, "StateManager is nil")
	}
	if md.OperationsTracker == nil {
		issues = append(issues, "OperationsTracker is nil")
	}
	if md.UiComponents == nil {
		issues = append(issues, "UiComponents is nil")
	}
	if md.AICoderState == nil {
		issues = append(issues, "AICoderState is nil")
	} else {
		// Validate AI Coder sub-model state
		aicoderIssues := md.AICoderState.Validate()
		issues = append(issues, aicoderIssues...)
	}

	if md.MCPState == nil {
		issues = append(issues, "MCPState is nil")
	} else {
		// Validate MCP sub-model state
		mcpIssues := md.MCPState.Validate()
		issues = append(issues, mcpIssues...)
	}

	if md.MCPDebugState == nil {
		issues = append(issues, "MCPDebugState is nil")
	} else {
		// Validate MCP Debug sub-model state
		mcpDebugIssues := md.MCPDebugState.Validate()
		issues = append(issues, mcpDebugIssues...)
	}

	if md.SettingsState == nil {
		issues = append(issues, "SettingsState is nil")
	} else {
		// Validate Settings sub-model state
		settingsIssues := md.SettingsState.Validate()
		issues = append(issues, settingsIssues...)
	}

	if md.URLsState == nil {
		issues = append(issues, "URLsState is nil")
	} else {
		// Validate URLs sub-model state
		urlsIssues := md.URLsState.Validate()
		issues = append(issues, urlsIssues...)
	}

	if md.CommandWindowState == nil {
		issues = append(issues, "CommandWindowState is nil")
	} else {
		// Validate Command Window sub-model state
		cmdWindowIssues := md.CommandWindowState.Validate()
		issues = append(issues, cmdWindowIssues...)
	}

	if md.WebViewState == nil {
		issues = append(issues, "WebViewState is nil")
	} else {
		// Validate Web View sub-model state
		webViewIssues := md.WebViewState.Validate()
		issues = append(issues, webViewIssues...)
	}

	if md.TestViewState == nil {
		issues = append(issues, "TestViewState is nil")
	} else {
		// Validate Test View sub-model state
		testViewIssues := md.TestViewState.Validate()
		issues = append(issues, testViewIssues...)
	}

	return issues
}

// State update methods for common operations

// UpdateCurrentView updates the current view in the application state
func (md *MasterData) UpdateCurrentView(view messages.ViewType) {
	// TODO: Implement once we understand the StateManager structure
	md.UpdateVersion()
}

// AddOperation adds an operation to the operations tracker
func (md *MasterData) AddOperation(id string, commandType types.CommandType, context types.OperationContext, timeout time.Duration) {
	if md.OperationsTracker != nil {
		md.OperationsTracker.AddOperation(id, commandType, context, timeout)
	}
	md.UpdateVersion()
}

// UpdateCache updates the cache state
func (md *MasterData) UpdateCache(key string, value interface{}) {
	// TODO: Implement once we understand the Cache structure
	md.UpdateVersion()
}

// Cleanup performs cleanup operations across all state components
func (md *MasterData) Cleanup() {
	// TODO: Implement cleanup operations
	md.UpdateVersion()
}