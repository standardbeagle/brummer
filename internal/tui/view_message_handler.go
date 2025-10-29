package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/services"
)

// ViewMessageHandler handles view-related messages and keyboard input
type ViewMessageHandler struct{}

// NewViewMessageHandler creates a new view message handler
func NewViewMessageHandler() MessageHandler {
	return &ViewMessageHandler{}
}

// CanHandle checks if this handler can process the message
func (h *ViewMessageHandler) CanHandle(msg tea.Msg) bool {
	switch msg.(type) {
	// Removed tea.KeyMsg to prevent duplicate keyboard processing
	// Keyboard is now handled by ViewSpecificHandler for view-specific keys
	case tea.WindowSizeMsg, tea.MouseMsg, urlDetectedMsg, refreshMsg:
		return true
	default:
		return false
	}
}

// HandleMessage processes view-related messages
func (h *ViewMessageHandler) HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	// Keyboard handling removed - now handled by ViewSpecificHandler

	case tea.WindowSizeMsg:
		// Update model dimensions
		model.width = m.Width
		model.height = m.Height

		// ===== NEW: Calculate layout regions ONCE (centralized) =====
		opts := components.DefaultLayoutOptions(m.Width, m.Height)
		// TODO: Check if system panel is open once method is available
		// opts.SystemPanelOpen = model.systemComponent.IsOpen()
		model.layoutRegions = components.CalculateLayoutRegions(opts)

		// ===== NEW: Pass layout regions to views that support it =====
		if model.aicodersView != nil {
			model.aicodersView.SetLayoutRegions(model.layoutRegions)

			// Resize active PTY session to match new terminal size
			if model.aiCoderService != nil && model.masterData != nil && model.masterData.AICoderState != nil {
				focusedSessionID := model.masterData.AICoderState.FocusedSession
				if focusedSessionID != "" {
					viewportWidth, viewportHeight := model.layoutRegions.GetViewportArea()
					if viewportWidth > 0 && viewportHeight > 0 {
						// Calculate PTY dimensions (account for header/footer/padding)
						ptyWidth := viewportWidth - 4
						ptyHeight := viewportHeight - 5

						// Ensure minimum size
						if ptyWidth < 80 {
							ptyWidth = 80
						}
						if ptyHeight < 24 {
							ptyHeight = 24
						}

						// Resize the PTY
						if svc, ok := model.aiCoderService.(*services.AICoderService); ok {
							_ = svc.ResizePTY(focusedSessionID, ptyWidth, ptyHeight)
						}
					}
				}
			}
		}

		// ===== OLD: Legacy view sizing (keep for backward compatibility) =====
		// Layout calculations now use components.LayoutManager (T021)
		// Header and footer heights are constants
		headerHeight := 3
		footerHeight := 1

		// Calculate content height once for all controllers
		contentHeight := model.calculateContentHeight()

		// Update all view controllers with new size including content height
		if model.processesView != nil {
			model.processesView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}
		if model.logsView != nil {
			model.logsView.SetSize(m.Width, contentHeight)
		}
		if model.webView != nil {
			model.webView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}
		if model.settingsView != nil {
			model.settingsView.UpdateSize(m.Width, m.Height, headerHeight, footerHeight, contentHeight)
		}

		// Update overlay controllers
		if model.commandWindowView != nil {
			// Sizing is handled by component itself in stateless pattern
			// model.commandWindowView.UpdateSize(m.Width, m.Height)
		}
		// Note: ScriptSelectorController gets size directly via getTerminalSize()
		// to ensure proper sizing for overlay rendering

	case tea.MouseMsg:
		// Handle mouse events - Note: AI Coder mouse wheel events are handled by AICoderMouseMessageHandler
		// This handler only deals with mouse events for other views or non-wheel events
		if model.currentView() != ViewAICoders {
			// For non-AI Coder views, we could add mouse handling here in the future
			// Currently no mouse handling for other views
		}
		// AI Coder mouse wheel events are now handled by AICoderMouseMessageHandler with higher priority

	case urlDetectedMsg:
		// Handle URL detection - update web view
		if model.webView != nil {
			// URL detection is handled internally by the web view component during updates
		}

	case refreshMsg:
		// Handle refresh request
		// Could trigger data refreshes in various controllers
		if model.currentView() == ViewProcesses && model.processesView != nil {
			// Refresh process list
			model.updateProcessList()
			cmds = append(cmds, model.waitForUpdates())
		}
	}

	return model, tea.Batch(cmds...)
}
