package navigation

import (
	tea "github.com/charmbracelet/bubbletea"
)

// View represents different views in the application
type View string

const (
	ViewScripts        View = "scripts"
	ViewProcesses      View = "processes"
	ViewLogs           View = "logs"
	ViewErrors         View = "errors"
	ViewURLs           View = "urls"
	ViewWeb            View = "web"
	ViewSettings       View = "settings"
	ViewMCPConnections View = "mcp-connections"
	ViewSearch         View = "search"
	ViewFilters        View = "filters"
	ViewScriptSelector View = "script-selector"
	ViewAICoders       View = "ai-coders"
)

// ViewOrder defines the order of views for cycling
var ViewOrder = []View{
	ViewProcesses,
	ViewLogs,
	ViewErrors,
	ViewURLs,
	ViewWeb,
	ViewAICoders,
	ViewSettings,
	ViewMCPConnections,
}

// Controller manages view navigation and switching
type Controller struct {
	currentView  View
	debugMode    bool
	onViewChange func(from View, to View)

	// Callbacks for view-specific operations
	onClearUnreadIndicator func(view View)
	onUpdateLogsView       func()
	onUpdateMCPConnections func()
}

// NewController creates a new navigation controller
func NewController(initialView View, debugMode bool) *Controller {
	return &Controller{
		currentView: initialView,
		debugMode:   debugMode,
	}
}

// SetOnViewChange sets the callback for view changes
func (c *Controller) SetOnViewChange(fn func(from View, to View)) {
	c.onViewChange = fn
}

// SetOnClearUnreadIndicator sets the callback for clearing unread indicators
func (c *Controller) SetOnClearUnreadIndicator(fn func(view View)) {
	c.onClearUnreadIndicator = fn
}

// SetOnUpdateLogsView sets the callback for updating logs view
func (c *Controller) SetOnUpdateLogsView(fn func()) {
	c.onUpdateLogsView = fn
}

// SetOnUpdateMCPConnections sets the callback for updating MCP connections
func (c *Controller) SetOnUpdateMCPConnections(fn func()) {
	c.onUpdateMCPConnections = fn
}

// CurrentView returns the current view
func (c *Controller) CurrentView() View {
	return c.currentView
}

// GetCurrentView returns the current view (alias for interface compatibility)
func (c *Controller) GetCurrentView() View {
	return c.currentView
}

// SwitchTo switches to a specific view
func (c *Controller) SwitchTo(view View) tea.Cmd {
	if c.currentView == view {
		return nil
	}

	oldView := c.currentView
	c.currentView = view

	if c.onViewChange != nil {
		c.onViewChange(oldView, view)
	}

	return nil
}

// SwitchToView changes the current view and performs any necessary setup
func (c *Controller) SwitchToView(view View) tea.Cmd {
	c.SwitchTo(view)

	// Clear unread indicator for this view
	if c.onClearUnreadIndicator != nil {
		c.onClearUnreadIndicator(view)
	}

	// Perform view-specific initialization if needed
	switch view {
	case ViewLogs:
		// Ensure logs are updated
		if c.onUpdateLogsView != nil {
			c.onUpdateLogsView()
		}
	case ViewMCPConnections:
		// Update MCP connections when in debug mode
		if c.debugMode && c.onUpdateMCPConnections != nil {
			c.onUpdateMCPConnections()
		}
	}

	return nil
}

// CycleView cycles to the next view with view-specific setup
func (c *Controller) CycleView() tea.Cmd {
	c.CycleNext()
	// Update MCP connections list when switching to that view
	if c.currentView == ViewMCPConnections && c.debugMode && c.onUpdateMCPConnections != nil {
		c.onUpdateMCPConnections()
	}
	return nil
}

// CyclePreviousView cycles to the previous view with view-specific setup
func (c *Controller) CyclePreviousView() tea.Cmd {
	c.CyclePrevious()
	// Update MCP connections list when switching to that view
	if c.currentView == ViewMCPConnections && c.debugMode && c.onUpdateMCPConnections != nil {
		c.onUpdateMCPConnections()
	}
	return nil
}

// CycleNext cycles to the next view
func (c *Controller) CycleNext() tea.Cmd {
	order := c.getViewOrder()
	currentIndex := -1
	for i, v := range order {
		if v == c.currentView {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		// Current view not in order, switch to first
		return c.SwitchTo(order[0])
	}

	nextIndex := (currentIndex + 1) % len(order)
	return c.SwitchTo(order[nextIndex])
}

// CyclePrevious cycles to the previous view
func (c *Controller) CyclePrevious() tea.Cmd {
	order := c.getViewOrder()
	currentIndex := -1
	for i, v := range order {
		if v == c.currentView {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		// Current view not in order, switch to last
		return c.SwitchTo(order[len(order)-1])
	}

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(order) - 1
	}
	return c.SwitchTo(order[prevIndex])
}

// getViewOrder returns the view order, excluding MCP view if not in debug mode
func (c *Controller) getViewOrder() []View {
	if c.debugMode {
		return ViewOrder
	}

	// Filter out MCP view if not in debug mode
	filtered := make([]View, 0, len(ViewOrder)-1)
	for _, v := range ViewOrder {
		if v != ViewMCPConnections {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// IsValidView checks if a view is valid for the current mode
func (c *Controller) IsValidView(view View) bool {
	if !c.debugMode && view == ViewMCPConnections {
		return false
	}
	return true
}

// GetViewForNumber returns the view for a number key (1-9)
func GetViewForNumber(num int) (View, bool) {
	switch num {
	case 1:
		return ViewProcesses, true
	case 2:
		return ViewLogs, true
	case 3:
		return ViewErrors, true
	case 4:
		return ViewURLs, true
	case 5:
		return ViewWeb, true
	case 6:
		return ViewAICoders, true
	case 7:
		return ViewSettings, true
	case 8:
		return ViewMCPConnections, true
	default:
		return "", false
	}
}
