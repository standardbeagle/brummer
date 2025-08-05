package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/navigation"
)

// ModelAdapter implements ModelInterface for the concrete Model type
// This allows us to gradually migrate controllers to use the interface
type ModelAdapter struct {
	model *Model
}

// NewModelAdapter creates a new adapter for the given model
func NewModelAdapter(model *Model) ModelInterface {
	return &ModelAdapter{model: model}
}

// Core dimensions
func (ma *ModelAdapter) GetWidth() int {
	return ma.model.width
}

func (ma *ModelAdapter) GetHeight() int {
	return ma.model.height
}

// Navigation
func (ma *ModelAdapter) GetCurrentView() View {
	return View(ma.model.navController.CurrentView())
}

func (ma *ModelAdapter) SwitchToView(view View) {
	ma.model.navController.SwitchTo(navigation.View(view))
}

// Process management
func (ma *ModelAdapter) GetProcessManager() ProcessManagerInterface {
	return ma.model.processMgr
}

func (ma *ModelAdapter) GetSelectedProcess() string {
	return ma.model.selectedProcess
}

func (ma *ModelAdapter) SetSelectedProcess(id string) {
	ma.model.selectedProcess = id
}

func (ma *ModelAdapter) UpdateProcessList() {
	ma.model.updateProcessList()
}

// Logging
func (ma *ModelAdapter) GetLogStore() LogStoreInterface {
	return ma.model.logStore
}

// Events
func (ma *ModelAdapter) GetEventBus() EventBusInterface {
	return ma.model.eventBus
}

func (ma *ModelAdapter) GetUpdateChannel() chan tea.Msg {
	return ma.model.updateChan
}

// Script management
func (ma *ModelAdapter) GetScripts() map[string]string {
	return ma.model.processMgr.GetScripts()
}

func (ma *ModelAdapter) IsScriptSelectorVisible() bool {
	return ma.model.scriptSelectorController != nil && ma.model.scriptSelectorController.IsVisible()
}

func (ma *ModelAdapter) ShowScriptSelector() {
	if ma.model.scriptSelectorController != nil {
		ma.model.scriptSelectorController.Show(false)
	}
}

func (ma *ModelAdapter) HideScriptSelector() {
	if ma.model.scriptSelectorController != nil {
		ma.model.scriptSelectorController.Hide()
	}
}

// Search
func (ma *ModelAdapter) IsSearchActive() bool {
	return ma.model.logsViewController != nil &&
		(ma.model.logsViewController.GetShowPattern() != "" ||
			ma.model.logsViewController.GetHidePattern() != "")
}

func (ma *ModelAdapter) GetSearchPattern() string {
	if ma.model.logsViewController != nil {
		return ma.model.logsViewController.GetShowPattern()
	}
	return ""
}

func (ma *ModelAdapter) SetSearchPattern(pattern string) {
	if ma.model.logsViewController != nil {
		ma.model.logsViewController.SetShowPattern(pattern)
	}
}

// Command window
func (ma *ModelAdapter) IsCommandWindowVisible() bool {
	return ma.model.commandWindowController.IsShowingCommandWindow()
}

func (ma *ModelAdapter) ShowCommandWindow() {
	// Get scripts and AI providers
	scripts := ma.model.processMgr.GetScripts()
	var aiProviders []string
	if ma.model.aiCoderController != nil {
		aiProviders = ma.model.aiCoderController.GetProviders()
	}
	ma.model.commandWindowController.ShowCommandWindow(scripts, aiProviders)
}

func (ma *ModelAdapter) HideCommandWindow() {
	ma.model.commandWindowController.HideCommandWindow()
}

// Settings
func (ma *ModelAdapter) IsSettingsVisible() bool {
	return View(ma.model.navController.CurrentView()) == ViewSettings
}

func (ma *ModelAdapter) ShowSettings() {
	ma.model.navController.SwitchTo(navigation.ViewSettings)
}

func (ma *ModelAdapter) HideSettings() {
	// Switch away from settings view
	ma.model.navController.SwitchTo(navigation.ViewProcesses)
}

// AI Coder
func (ma *ModelAdapter) HasActiveAICoders() bool {
	if ma.model.aiCoderController == nil {
		return false
	}
	sessions := ma.model.aiCoderController.ListSessions()
	return len(sessions) > 0
}

func (ma *ModelAdapter) GetAICoderController() AICoderControllerInterface {
	// Need to wrap AICoderController to implement the interface
	return &aiCoderControllerWrapper{controller: ma.model.aiCoderController}
}

// Removed ProcessManagerAdapter - process.Manager already implements ProcessManagerInterface

// Removed LogStoreAdapter since logs.Store already implements LogStoreInterface

// aiCoderControllerWrapper wraps AICoderController to implement AICoderControllerInterface
type aiCoderControllerWrapper struct {
	controller *AICoderController
}

func (w *aiCoderControllerWrapper) HandleAICommand(command string) tea.Cmd {
	w.controller.HandleAICommand(command)
	return nil
}

func (w *aiCoderControllerWrapper) CreateAICoderSession(provider string) {
	w.controller.HandleAICommand(provider)
}

func (w *aiCoderControllerWrapper) HasActiveSessions() bool {
	if w.controller == nil {
		return false
	}
	sessions := w.controller.ListSessions()
	return len(sessions) > 0
}

func (w *aiCoderControllerWrapper) SetHasActiveSessions(active bool) {
	// This is a no-op for the real controller since it manages its own state
}

func (w *aiCoderControllerWrapper) GetActiveSessions() []AICoderSession {
	if w.controller == nil {
		return nil
	}

	sessions := w.controller.ListSessions()
	result := make([]AICoderSession, 0, len(sessions))
	for _, s := range sessions {
		status := "inactive"
		if s.IsActive {
			status = "active"
		}
		result = append(result, AICoderSession{
			ID:       s.ID,
			Provider: "terminal", // Default provider for now
			Status:   status,
			Created:  time.Now(), // We don't track creation time in PTYSession
		})
	}
	return result
}
