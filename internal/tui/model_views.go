package tui

import "github.com/standardbeagle/brummer/internal/tui/navigation"


// ViewConfig holds configuration for each view
type ViewConfig struct {
	Title       string
	Description string
	KeyBinding  string
	Icon        string
}

// GetViewConfigs returns the configuration for all views
func GetViewConfigs() map[navigation.View]ViewConfig {
	return map[navigation.View]ViewConfig{
		navigation.ViewProcesses: {
			Title:       "Processes",
			Description: "Process management",
			KeyBinding:  "1",
			Icon:        "🏃",
		},
		navigation.ViewLogs: {
			Title:       "Logs",
			Description: "Process logs",
			KeyBinding:  "2",
			Icon:        "📄",
		},
		navigation.ViewErrors: {
			Title:       "Errors",
			Description: "Error tracking",
			KeyBinding:  "3",
			Icon:        "❌",
		},
		navigation.ViewURLs: {
			Title:       "URLs",
			Description: "Detected URLs",
			KeyBinding:  "4",
			Icon:        "🔗",
		},
		navigation.ViewWeb: {
			Title:       "Web",
			Description: "Proxy requests",
			KeyBinding:  "5",
			Icon:        "🌐",
		},
		navigation.ViewAICoders: {
			Title:       "AI Coders",
			Description: "Manage and monitor agentic AI coding assistants",
			KeyBinding:  "6",
			Icon:        "🤖",
		},
		navigation.ViewSettings: {
			Title:       "Settings",
			Description: "Configuration",
			KeyBinding:  "7",
			Icon:        "⚙️",
		},
		navigation.ViewMCPConnections: {
			Title:       "MCP",
			Description: "MCP Connections",
			KeyBinding:  "8",
			Icon:        "🔌",
		},
	}
}

// GetOrderedViews returns the views in the order they should appear in tabs
func GetOrderedViews(debugMode bool) []navigation.View {
	views := []navigation.View{navigation.ViewProcesses, navigation.ViewLogs, navigation.ViewErrors, navigation.ViewURLs, navigation.ViewWeb, navigation.ViewAICoders, navigation.ViewSettings}
	if debugMode {
		views = append(views, navigation.ViewMCPConnections)
	}
	return views
}

// Constants are defined in constants.go