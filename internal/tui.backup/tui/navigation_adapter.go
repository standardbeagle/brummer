package tui

import (
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// NavigationAdapter adapts the navigation.Component to NavigationControllerInterface
type NavigationAdapter struct {
	component *components.NavigationComponent
}

// NewNavigationAdapter creates a new navigation adapter
func NewNavigationAdapter(component *components.NavigationComponent) *NavigationAdapter {
	return &NavigationAdapter{component: component}
}

// GetCurrentView returns the current view
func (na *NavigationAdapter) GetCurrentView() messages.ViewType {
	if na.component != nil {
		return na.component.GetCurrentView()
	}
	return messages.ViewType("processes") // fallback
}

// SwitchTo switches to a specific view
func (na *NavigationAdapter) SwitchTo(view messages.ViewType) {
	if na.component != nil {
		_ = na.component.SwitchToView(view, "adapter")
		// In a real BubbleTea application, this would return the cmd
		// For now, we'll just trigger the state change
	}
}

// NextView switches to the next view
func (na *NavigationAdapter) NextView() {
	if na.component != nil {
		_ = na.component.CycleToNextView()
		// In a real BubbleTea application, this would return the cmd
	}
}

// PreviousView switches to the previous view
func (na *NavigationAdapter) PreviousView() {
	if na.component != nil {
		currentView := na.component.GetCurrentView()
		_ = na.component.SwitchToView(currentView, "shift_tab")
		// In a real BubbleTea application, this would return the cmd
	}
}

// GetViewName returns the name of a view
func (na *NavigationAdapter) GetViewName(view messages.ViewType) string {
	// This would need to be implemented based on your view naming logic
	switch view {
	case messages.ViewType("processes"):
		return "Processes"
	case messages.ViewType("logs"):
		return "Logs"
	case messages.ViewType("errors"):
		return "Errors"
	case messages.ViewType("urls"):
		return "URLs"
	case messages.ViewType("web"):
		return "Web"
	case messages.ViewType("settings"):
		return "Settings"
	case messages.ViewType("mcp_connections"):
		return "MCP Connections"
	case messages.ViewType("aicoders"):
		return "AI Coders"
	default:
		return string(view)
	}
}

// GetViewIcon returns the icon for a view
func (na *NavigationAdapter) GetViewIcon(view messages.ViewType) string {
	// This would need to be implemented based on your view icon logic
	switch view {
	case messages.ViewType("processes"):
		return "📋"
	case messages.ViewType("logs"):
		return "📜"
	case messages.ViewType("errors"):
		return "❌"
	case messages.ViewType("urls"):
		return "🔗"
	case messages.ViewType("web"):
		return "🌐"
	case messages.ViewType("settings"):
		return "⚙️"
	case messages.ViewType("mcp_connections"):
		return "🔌"
	case messages.ViewType("aicoders"):
		return "🤖"
	default:
		return ""
	}
}
