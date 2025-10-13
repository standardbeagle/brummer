package tui

// ViewConfig holds configuration for each view
type ViewConfig struct {
	Title       string
	Description string
	KeyBinding  string
	Icon        string
}

// GetViewConfigs returns the configuration for all views
func GetViewConfigs() map[View]ViewConfig {
	return map[View]ViewConfig{
		ViewProcesses: {
			Title:       "Processes",
			Description: "Process management",
			KeyBinding:  "1",
			Icon:        "🏃",
		},
		ViewLogs: {
			Title:       "Logs",
			Description: "Process logs",
			KeyBinding:  "2",
			Icon:        "📄",
		},
		ViewErrors: {
			Title:       "Errors",
			Description: "Error tracking",
			KeyBinding:  "3",
			Icon:        "❌",
		},
		ViewURLs: {
			Title:       "URLs",
			Description: "Detected URLs",
			KeyBinding:  "4",
			Icon:        "🔗",
		},
		ViewWeb: {
			Title:       "Web",
			Description: "Proxy requests",
			KeyBinding:  "5",
			Icon:        "🌐",
		},
		ViewAICoders: {
			Title:       "AI Coders",
			Description: "Manage and monitor agentic AI coding assistants",
			KeyBinding:  "6",
			Icon:        "🤖",
		},
		ViewSettings: {
			Title:       "Settings",
			Description: "Configuration",
			KeyBinding:  "7",
			Icon:        "⚙️",
		},
		ViewMCPConnections: {
			Title:       "MCP",
			Description: "MCP Connections",
			KeyBinding:  "8",
			Icon:        "🔌",
		},
	}
}

// GetOrderedViews returns the views in the order they should appear in tabs
func GetOrderedViews(debugMode bool) []View {
	views := []View{ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewAICoders, ViewSettings}
	if debugMode {
		views = append(views, ViewMCPConnections)
	}
	return views
}

// Constants are defined in constants.go
