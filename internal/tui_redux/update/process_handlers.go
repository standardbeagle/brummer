package update

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui_redux/messages"
	"github.com/standardbeagle/brummer/internal/tui_redux/model"
)

// Process message handlers

func handleProcessStarted(m model.Model, msg messages.ProcessStartedMsg) (model.Model, tea.Cmd) {
	// TODO: Implementation
	return m, nil
}

func handleProcessStopped(m model.Model, msg messages.ProcessStoppedMsg) (model.Model, tea.Cmd) {
	// TODO: Implementation
	return m, nil
}

func handleProcessOutput(m model.Model, msg messages.ProcessOutputMsg) (model.Model, tea.Cmd) {
	// TODO: Implementation
	return m, nil
}

func handleProcessError(m model.Model, msg messages.ProcessErrorMsg) (model.Model, tea.Cmd) {
	// TODO: Implementation
	return m, nil
}

func handleProcessSelected(m model.Model, msg messages.ProcessSelectedMsg) (model.Model, tea.Cmd) {
	// TODO: Implementation
	return m, nil
}

func handleProcessRestarted(m model.Model, msg messages.ProcessRestartedMsg) (model.Model, tea.Cmd) {
	// TODO: Implementation
	return m, nil
}