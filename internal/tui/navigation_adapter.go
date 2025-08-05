package tui

import (
	"github.com/standardbeagle/brummer/internal/tui/navigation"
)

// NavigationAdapter adapts the navigation.Controller to NavigationControllerInterface
type NavigationAdapter struct {
	controller *navigation.Controller
}

// NewNavigationAdapter creates a new navigation adapter
func NewNavigationAdapter(controller *navigation.Controller) NavigationControllerInterface {
	return &NavigationAdapter{controller: controller}
}

// GetCurrentView returns the current view
func (na *NavigationAdapter) GetCurrentView() View {
	return View(na.controller.CurrentView())
}

// SwitchTo switches to a specific view
func (na *NavigationAdapter) SwitchTo(view View) {
	na.controller.SwitchTo(navigation.View(view))
}

// NextView switches to the next view
func (na *NavigationAdapter) NextView() {
	na.controller.CycleNext()
}

// PreviousView switches to the previous view
func (na *NavigationAdapter) PreviousView() {
	na.controller.CyclePrevious()
}

// GetViewName returns the name of a view
func (na *NavigationAdapter) GetViewName(view View) string {
	// This would need to be implemented based on your view naming logic
	switch view {
	case ViewProcesses:
		return "Processes"
	case ViewLogs:
		return "Logs"
	case ViewErrors:
		return "Errors"
	case ViewURLs:
		return "URLs"
	case ViewWeb:
		return "Web"
	case ViewSettings:
		return "Settings"
	case ViewMCPConnections:
		return "MCP Connections"
	case ViewAICoders:
		return "AI Coders"
	default:
		return string(view)
	}
}

// GetViewIcon returns the icon for a view
func (na *NavigationAdapter) GetViewIcon(view View) string {
	// This would need to be implemented based on your view icon logic
	switch view {
	case ViewProcesses:
		return "📋"
	case ViewLogs:
		return "📜"
	case ViewErrors:
		return "❌"
	case ViewURLs:
		return "🔗"
	case ViewWeb:
		return "🌐"
	case ViewSettings:
		return "⚙️"
	case ViewMCPConnections:
		return "🔌"
	case ViewAICoders:
		return "🤖"
	default:
		return ""
	}
}
