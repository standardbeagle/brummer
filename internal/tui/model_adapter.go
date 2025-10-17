package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/services"
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
	return ma.model.currentView()
}

func (ma *ModelAdapter) SwitchToView(view View) {
	ma.model.switchToView(view)
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
	return ma.model.scriptSelector != nil && ma.model.scriptSelector.IsVisible()
}

func (ma *ModelAdapter) ShowScriptSelector() {
	if ma.model.scriptSelector != nil {
		ma.model.scriptSelector.Show(false)
	}
}

func (ma *ModelAdapter) HideScriptSelector() {
	if ma.model.scriptSelector != nil {
		ma.model.scriptSelector.Hide()
	}
}

// Search
func (ma *ModelAdapter) IsSearchActive() bool {
	return ma.model.logsView != nil &&
		(ma.model.logsView.GetShowPattern() != "" ||
			ma.model.logsView.GetHidePattern() != "")
}

func (ma *ModelAdapter) GetSearchPattern() string {
	if ma.model.logsView != nil {
		return ma.model.logsView.GetShowPattern()
	}
	return ""
}

func (ma *ModelAdapter) SetSearchPattern(pattern string) {
	if ma.model.logsView != nil {
		ma.model.logsView.SetShowPattern(pattern)
	}
}

// Command window
func (ma *ModelAdapter) IsCommandWindowVisible() bool {
	return ma.model.commandWindowView.IsShowingCommandWindow()
}

func (ma *ModelAdapter) ShowCommandWindow() {
	// Get scripts
	scripts := ma.model.processMgr.GetScripts()
	ma.model.commandWindowView.ShowCommandWindow(scripts)
}

func (ma *ModelAdapter) HideCommandWindow() {
	ma.model.commandWindowView.HideCommandWindow()
}

// Settings
func (ma *ModelAdapter) IsSettingsVisible() bool {
	return ma.model.currentView() == ViewSettings
}

func (ma *ModelAdapter) ShowSettings() {
	ma.model.switchToView(ViewSettings)
}

func (ma *ModelAdapter) HideSettings() {
	// Switch away from settings view
	ma.model.switchToView(ViewProcesses)
}

// AI Coder
func (ma *ModelAdapter) HasActiveAICoders() bool {
	if ma.model.aiCoderService == nil {
		return false
	}
	svc := ma.model.aiCoderService.(*services.AICoderService)
	sessions := svc.ListSessions()
	return len(sessions) > 0
}

// Removed ProcessManagerAdapter - process.Manager already implements ProcessManagerInterface

// Removed LogStoreAdapter since logs.Store already implements LogStoreInterface
