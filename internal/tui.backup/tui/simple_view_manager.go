package tui

import (
	"sync/atomic"
	tea "github.com/charmbracelet/bubbletea"
)

// SimpleViewManager manages view state with minimal complexity
// This replaces the NavigationService + NavigationComponent architecture
type SimpleViewManager struct {
	currentView atomic.Value // View (string)
}

// NewSimpleViewManager creates a new simplified view manager
func NewSimpleViewManager(initialView View) *SimpleViewManager {
	vm := &SimpleViewManager{}
	vm.currentView.Store(initialView)
	return vm
}

// GetCurrentView returns the current view atomically
func (vm *SimpleViewManager) GetCurrentView() View {
	if v := vm.currentView.Load(); v != nil {
		return v.(View)
	}
	return ViewProcesses // fallback
}

// SetCurrentView updates the current view atomically
func (vm *SimpleViewManager) SetCurrentView(view View) {
	vm.currentView.Store(view)
}

// SwitchToView switches to a new view and returns a command
func (vm *SimpleViewManager) SwitchToView(view View) tea.Cmd {
	oldView := vm.GetCurrentView()
	vm.SetCurrentView(view)

	// Return a message that can trigger view update
	return func() tea.Msg {
		return ViewSwitchedMsg{
			FromView: oldView,
			ToView:   view,
		}
	}
}

// ViewSwitchedMsg is sent when view switches successfully
type ViewSwitchedMsg struct {
	FromView View
	ToView   View
}

// IsValidView checks if a view is valid
func IsValidView(view View) bool {
	switch view {
	case ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewAICoders, ViewSettings,
		ViewMCPConnections, ViewSearch, ViewFilters, ViewScriptSelector:
		return true
	default:
		return false
	}
}

// GetSimpleOrderedViews returns views in tab order for the simplified view manager
func GetSimpleOrderedViews(debugMode bool) []View {
	views := []View{ViewProcesses, ViewLogs, ViewErrors, ViewURLs, ViewWeb, ViewAICoders, ViewSettings}
	if debugMode {
		views = append(views, ViewMCPConnections)
	}
	return views
}

// GetViewForNumberKey maps number keys to views
func GetViewForNumberKey(num rune) (View, bool) {
	viewMap := map[rune]View{
		'1': ViewProcesses,
		'2': ViewLogs,
		'3': ViewErrors,
		'4': ViewURLs,
		'5': ViewWeb,
		'6': ViewAICoders,
		'7': ViewSettings,
	}
	view, ok := viewMap[num]
	return view, ok
}

// CycleToNextView returns the next view in order
func CycleToNextView(currentView View, debugMode bool) View {
	views := GetSimpleOrderedViews(debugMode)

	for i, v := range views {
		if v == currentView {
			// Return next view with wrap-around
			return views[(i+1)%len(views)]
		}
	}

	// If current view not in list, return first view
	if len(views) > 0 {
		return views[0]
	}
	return ViewProcesses
}

// CycleToPrevView returns the previous view in order
func CycleToPrevView(currentView View, debugMode bool) View {
	views := GetSimpleOrderedViews(debugMode)

	for i, v := range views {
		if v == currentView {
			// Return previous view with wrap-around
			prevIdx := i - 1
			if prevIdx < 0 {
				prevIdx = len(views) - 1
			}
			return views[prevIdx]
		}
	}

	// If current view not in list, return first view
	if len(views) > 0 {
		return views[0]
	}
	return ViewProcesses
}