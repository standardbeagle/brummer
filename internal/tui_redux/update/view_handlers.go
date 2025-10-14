package update

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui_redux/messages"
	"github.com/standardbeagle/brummer/internal/tui_redux/model"
)

// handleViewSwitched handles view switching messages
func handleViewSwitched(m model.Model, msg messages.ViewSwitchedMsg) (model.Model, tea.Cmd) {
	// Update the view in the model
	m.SetCurrentView(msg.NewView)

	// Add to view history
	m.AddToViewHistory(msg.NewView)

	return m, nil
}

// handleViewHistoryBack handles navigation back in view history
func handleViewHistoryBack(m model.Model) (model.Model, tea.Cmd) {
	// This would implement back navigation
	// For now, just return model unchanged
	return m, nil
}

// handleViewHistoryForward handles navigation forward in view history
func handleViewHistoryForward(m model.Model) (model.Model, tea.Cmd) {
	// This would implement forward navigation
	// For now, just return model unchanged
	return m, nil
}

// handleWindowResized handles window resize messages
func handleWindowResized(m model.Model, msg messages.WindowResizedMsg) (model.Model, tea.Cmd) {
	m.SetWindowSize(msg.Width, msg.Height)
	return m, nil
}

// handleFullscreenToggle handles fullscreen toggle messages
func handleFullscreenToggle(m model.Model) (model.Model, tea.Cmd) {
	m.SetFullscreen(!m.IsFullscreen())
	return m, nil
}