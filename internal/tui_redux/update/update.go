package update

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/standardbeagle/brummer/internal/tui_redux/messages"
	"github.com/standardbeagle/brummer/internal/tui_redux/model"
)

// Update is the pure reducer function at the heart of the Redux/Elm architecture
// It receives a message and returns the updated model and optional commands
// CRITICAL: This function must be pure - same input must always produce same output
func Update(m model.Model, msg tea.Msg) (model.Model, tea.Cmd) {
	// Update performance metrics
	m.UpdateMetrics()

	// Route message to appropriate handler
	switch msg := msg.(type) {
	// View messages
	case messages.ViewSwitchedMsg:
		return handleViewSwitched(m, msg)
	case messages.ViewHistoryBackMsg:
		return handleViewHistoryBack(m)
	case messages.ViewHistoryForwardMsg:
		return handleViewHistoryForward(m)
	case messages.WindowResizedMsg:
		return handleWindowResized(m, msg)
	case messages.FullscreenToggleMsg:
		return handleFullscreenToggle(m)

	// Process messages
	case messages.ProcessStartedMsg:
		return handleProcessStarted(m, msg)
	case messages.ProcessStoppedMsg:
		return handleProcessStopped(m, msg)
	case messages.ProcessOutputMsg:
		return handleProcessOutput(m, msg)
	case messages.ProcessErrorMsg:
		return handleProcessError(m, msg)
	case messages.ProcessSelectedMsg:
		return handleProcessSelected(m, msg)
	case messages.ProcessRestartedMsg:
		return handleProcessRestarted(m, msg)

	// Log messages
	case messages.LogFilterChangedMsg:
		return handleLogFilterChanged(m, msg)
	case messages.LogScrollMsg:
		return handleLogScroll(m, msg)
	case messages.LogLoadMsg:
		return handleLogLoad(m, msg)
	case messages.LogSaveMsg:
		return handleLogSave(m, msg)
	case messages.LogClearMsg:
		return handleLogClear(m, msg)
	case messages.LogStreamToggleMsg:
		return handleLogStreamToggle(m, msg)

	// Error messages
	case messages.ErrorDetectedMsg:
		return handleErrorDetected(m, msg)
	case messages.ErrorDismissMsg:
		return handleErrorDismiss(m, msg)
	case messages.ErrorClearMsg:
		return handleErrorClear(m)
	case messages.ErrorSelectedMsg:
		return handleErrorSelected(m, msg)

	// URL messages
	case messages.URLDiscoveredMsg:
		return handleURLDiscovered(m, msg)
	case messages.URLOpenRequestMsg:
		return handleURLOpenRequest(m, msg)
	case messages.URLCopyMsg:
		return handleURLCopy(m, msg)
	case messages.URLTestMsg:
		return handleURLTest(m, msg)
	case messages.URLSelectedMsg:
		return handleURLSelected(m, msg)

	// AI Coder messages
	case messages.AISessionStartedMsg:
		return handleAISessionStarted(m, msg)
	case messages.AISessionEndedMsg:
		return handleAISessionEnded(m, msg)
	case messages.AISessionSelectedMsg:
		return handleAISessionSelected(m, msg)
	case messages.AISendMsg:
		return handleAISend(m, msg)
	case messages.AIResponseMsg:
		return handleAIResponse(m, msg)
	case messages.AIErrorMsg:
		return handleAIError(m, msg)
	case messages.AIClearMsg:
		return handleAIClear(m, msg)

	// Settings messages
	case messages.SettingsUpdatedMsg:
		return handleSettingsUpdated(m, msg)
	case messages.SettingsLoadMsg:
		return handleSettingsLoad(m)
	case messages.SettingsSaveMsg:
		return handleSettingsSave(m, msg)
	case messages.SettingsResetMsg:
		return handleSettingsReset(m)
	case messages.ThemeChangedMsg:
		return handleThemeChanged(m, msg)

	// Command palette messages
	case messages.PaletteOpenMsg:
		return handlePaletteOpen(m)
	case messages.PaletteCloseMsg:
		return handlePaletteClose(m)
	case messages.PaletteQueryMsg:
		return handlePaletteQuery(m, msg)
	case messages.PaletteSelectMsg:
		return handlePaletteSelect(m, msg)
	case messages.PaletteExecuteMsg:
		return handlePaletteExecute(m, msg)

	// Keyboard messages
	case messages.KeyBindToggleMsg:
		return handleKeyBindToggle(m, msg)
	case messages.KeyBindReloadMsg:
		return handleKeyBindReload(m)

	// System messages
	case messages.TickMsg:
		return handleTick(m, msg)
	case messages.QuitMsg:
		return handleQuit(m)
	case messages.ForceQuitMsg:
		return handleForceQuit(m)
	case messages.RefreshMsg:
		return handleRefresh(m)
	case messages.StatusUpdateMsg:
		return handleStatusUpdate(m, msg)

	// Debug messages
	case messages.DebugToggleMsg:
		return handleDebugToggle(m, msg)
	case messages.DebugLogMsg:
		return handleDebugLog(m, msg)

	// Performance messages
	case messages.PerformanceMetricsMsg:
		return handlePerformanceMetrics(m, msg)

	// Help messages
	case messages.HelpToggleMsg:
		return handleHelpToggle(m)
	case messages.HelpShowMsg:
		return handleHelpShow(m, msg)
	case messages.HelpHideMsg:
		return handleHelpHide(m)

	// BubbleTea framework messages
	case tea.KeyMsg:
		return handleKeyMsg(m, msg)
	case tea.MouseMsg:
		return handleMouseMsg(m, msg)
	case tea.WindowSizeMsg:
		return handleWindowSizeMsg(m, msg)

	// Unknown message type - return model unchanged
	default:
		return m, nil
	}
}

// handleKeyMsg processes keyboard input using the Redux/Elm pattern
func handleKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	// Extract character input
	if len(msg.Runes) > 0 {
		switch msg.Runes[0] {
		case '1':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewProcesses))
		case '2':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewLogs))
		case '3':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewErrors))
		case '4':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewURLs))
		case '5':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewWeb))
		case '6':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewAICoders))
		case '7':
			return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), model.ViewSettings))
		case 'q':
			return handleQuit(m)
		case '/':
			return handlePaletteOpen(m)
		}
	}

	// Handle special keys
	switch msg.Type {
	case tea.KeyCtrlC:
		return handleForceQuit(m)
	case tea.KeyTab:
		// Cycle to next view
		nextView := m.CurrentView() + 1
		if nextView > model.ViewSettings {
			nextView = model.ViewProcesses
		}
		return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), nextView))
	case tea.KeyShiftTab:
		// Cycle to previous view
		prevView := m.CurrentView() - 1
		if prevView < model.ViewProcesses {
			prevView = model.ViewSettings
		}
		return handleViewSwitched(m, messages.NewViewSwitchedMsg(m.CurrentView(), prevView))
	case tea.KeyBackspace:
		if m.CurrentView() == model.ViewLogs {
			return handleViewHistoryBack(m)
		}
	}

	// Pass through to current view's specific handlers
	return handleViewSpecificKeyMsg(m, msg)
}

// handleMouseMsg processes mouse input
func handleMouseMsg(m model.Model, msg tea.MouseMsg) (model.Model, tea.Cmd) {
	// For now, just return model unchanged
	// Mouse handling will be implemented per view in Phase 7
	return m, nil
}

// handleWindowSizeMsg processes window size changes from BubbleTea
func handleWindowSizeMsg(m model.Model, msg tea.WindowSizeMsg) (model.Model, tea.Cmd) {
	windowMsg := messages.WindowResizedMsg{
		Width:  msg.Width,
		Height: msg.Height,
	}
	return handleWindowResized(m, windowMsg)
}

// handleViewSpecificKeyMsg routes key messages to view-specific handlers
func handleViewSpecificKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	switch m.CurrentView() {
	case model.ViewProcesses:
		return handleProcessesKeyMsg(m, msg)
	case model.ViewLogs:
		return handleLogsKeyMsg(m, msg)
	case model.ViewErrors:
		return handleErrorsKeyMsg(m, msg)
	case model.ViewURLs:
		return handleURLsKeyMsg(m, msg)
	case model.ViewWeb:
		return handleWebKeyMsg(m, msg)
	case model.ViewAICoders:
		return handleAICodersKeyMsg(m, msg)
	case model.ViewSettings:
		return handleSettingsKeyMsg(m, msg)
	default:
		return m, nil
	}
}

// Placeholder view-specific key handlers (will be implemented in handler files)
func handleProcessesKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}

func handleLogsKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}

func handleErrorsKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}

func handleURLsKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}

func handleWebKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}

func handleAICodersKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}

func handleSettingsKeyMsg(m model.Model, msg tea.KeyMsg) (model.Model, tea.Cmd) {
	return m, nil
}