package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
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
	return &EventBusAdapter{eventBus: ma.model.eventBus}
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
	return ma.model.masterData != nil && ma.model.masterData.LogsState != nil &&
		(ma.model.masterData.LogsState.GetShowPattern() != "" ||
			ma.model.masterData.LogsState.GetHidePattern() != "")
}

func (ma *ModelAdapter) GetSearchPattern() string {
	if ma.model.masterData != nil && ma.model.masterData.LogsState != nil {
		return ma.model.masterData.LogsState.GetShowPattern()
	}
	return ""
}

func (ma *ModelAdapter) SetSearchPattern(pattern string) {
	if ma.model.masterData != nil && ma.model.masterData.LogsState != nil {
		ma.model.masterData.LogsState = ma.model.masterData.LogsState.SetShowPattern(pattern)
	}
}

// Command window
func (ma *ModelAdapter) IsCommandWindowVisible() bool {
	return ma.model.masterData.CommandWindowState.IsShowingCommandWindow()
}

func (ma *ModelAdapter) ShowCommandWindow() {
	// Update state directly
	ma.model.masterData.CommandWindowState = ma.model.masterData.CommandWindowState.ShowCommandWindow()
}

func (ma *ModelAdapter) HideCommandWindow() {
	ma.model.masterData.CommandWindowState = ma.model.masterData.CommandWindowState.HideCommandWindow()
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

// Removed ProcessManagerAdapter - processes.Manager already implements ProcessManagerInterface

// Removed LogStoreAdapter since logs.Store already implements LogStoreInterface

// EventBusAdapter wraps events.EventBus to match EventBusInterface
type EventBusAdapter struct {
	eventBus *events.EventBus
}

// Publish forwards to the wrapped EventBus
func (eba *EventBusAdapter) Publish(event events.Event) {
	eba.eventBus.Publish(event)
}

// Subscribe wraps the EventBus Subscribe method to match the interface
// The interface expects no return value, so we discard the unsubscribe function
func (eba *EventBusAdapter) Subscribe(eventType events.EventType, handler events.Handler) {
	// Discard the returned unsubscribe function to match interface
	_ = eba.eventBus.Subscribe(eventType, handler)
}

// Shutdown forwards to the wrapped EventBus
func (eba *EventBusAdapter) Shutdown() {
	eba.eventBus.Shutdown()
}
