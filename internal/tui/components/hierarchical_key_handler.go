package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// KeyHandlerPriority defines the priority levels for key handling
type KeyHandlerPriority int

const (
	// PriorityHighest - System critical keys (quit, force quit, etc.)
	PriorityHighest KeyHandlerPriority = iota
	// PriorityOverlay - Modal overlays, dialogs, popups
	PriorityOverlay
	// PriorityFocused - Currently focused component (text input, list, etc.)
	PriorityFocused
	// PriorityView - Current view's specific handlers
	PriorityView
	// PriorityGlobal - Global shortcuts (view switching, help, etc.)
	PriorityGlobal
	// PriorityLowest - Catch-all handlers
	PriorityLowest
)

// KeyHandler defines the interface for key handlers
type KeyHandler interface {
	// CanHandle returns true if this handler can process the given key
	CanHandle(msg tea.KeyMsg) bool
	// Handle processes the key and returns the updated model and command
	Handle(msg tea.KeyMsg) (tea.Model, tea.Cmd)
	// Priority returns the handler's priority level
	Priority() KeyHandlerPriority
	// Name returns a human-readable name for debugging
	Name() string
}

// KeyHandlerContext provides context for key handling
type KeyHandlerContext struct {
	// Current focused component (if any)
	FocusedComponent KeyHandler
	// Active overlays (higher priority than normal components)
	ActiveOverlays []KeyHandler
	// Current view handler
	ViewHandler KeyHandler
	// Global handlers
	GlobalHandlers []KeyHandler
	// Catch-all handler
	CatchAllHandler KeyHandler
}

// HierarchicalKeyHandler manages key handling with proper priority ordering
type HierarchicalKeyHandler struct {
	handlers []KeyHandler
	context  *KeyHandlerContext
}

// NewHierarchicalKeyHandler creates a new hierarchical key handler
func NewHierarchicalKeyHandler() *HierarchicalKeyHandler {
	return &HierarchicalKeyHandler{
		handlers: make([]KeyHandler, 0),
		context: &KeyHandlerContext{
			ActiveOverlays: make([]KeyHandler, 0),
			GlobalHandlers: make([]KeyHandler, 0),
		},
	}
}

// SetContext updates the key handling context
func (h *HierarchicalKeyHandler) SetContext(ctx *KeyHandlerContext) {
	h.context = ctx
	h.rebuildHandlers()
}

// AddGlobalHandler adds a global key handler
func (h *HierarchicalKeyHandler) AddGlobalHandler(handler KeyHandler) {
	h.context.GlobalHandlers = append(h.context.GlobalHandlers, handler)
	h.rebuildHandlers()
}

// AddOverlay adds an overlay handler (higher priority)
func (h *HierarchicalKeyHandler) AddOverlay(handler KeyHandler) {
	h.context.ActiveOverlays = append(h.context.ActiveOverlays, handler)
	h.rebuildHandlers()
}

// RemoveOverlay removes an overlay handler
func (h *HierarchicalKeyHandler) RemoveOverlay(handler KeyHandler) {
	for i, existing := range h.context.ActiveOverlays {
		if existing == handler {
			h.context.ActiveOverlays = append(h.context.ActiveOverlays[:i], h.context.ActiveOverlays[i+1:]...)
			break
		}
	}
	h.rebuildHandlers()
}

// SetFocusedComponent sets the currently focused component
func (h *HierarchicalKeyHandler) SetFocusedComponent(handler KeyHandler) {
	h.context.FocusedComponent = handler
	h.rebuildHandlers()
}

// SetViewHandler sets the current view handler
func (h *HierarchicalKeyHandler) SetViewHandler(handler KeyHandler) {
	h.context.ViewHandler = handler
	h.rebuildHandlers()
}

// SetCatchAllHandler sets the catch-all handler
func (h *HierarchicalKeyHandler) SetCatchAllHandler(handler KeyHandler) {
	h.context.CatchAllHandler = handler
	h.rebuildHandlers()
}

// rebuildHandlers rebuilds the handlers array in priority order
func (h *HierarchicalKeyHandler) rebuildHandlers() {
	h.handlers = make([]KeyHandler, 0)

	// Add handlers in priority order
	if h.context.CatchAllHandler != nil {
		h.handlers = append(h.handlers, h.context.CatchAllHandler)
	}

	for _, overlay := range h.context.ActiveOverlays {
		h.handlers = append(h.handlers, overlay)
	}

	if h.context.FocusedComponent != nil {
		h.handlers = append(h.handlers, h.context.FocusedComponent)
	}

	if h.context.ViewHandler != nil {
		h.handlers = append(h.handlers, h.context.ViewHandler)
	}

	for _, global := range h.context.GlobalHandlers {
		h.handlers = append(h.handlers, global)
	}

	// Sort by priority (higher priority first, smaller numbers first)
	for i := 0; i < len(h.handlers); i++ {
		for j := i + 1; j < len(h.handlers); j++ {
			if h.handlers[i].Priority() > h.handlers[j].Priority() {
				h.handlers[i], h.handlers[j] = h.handlers[j], h.handlers[i]
			}
		}
	}
}

// HandleKey processes a key message using the hierarchical handler
func (h *HierarchicalKeyHandler) HandleKey(msg tea.KeyMsg, model tea.Model) (tea.Model, tea.Cmd) {
	// Iterate through handlers in priority order
	for _, handler := range h.handlers {
		if handler.CanHandle(msg) {
			newModel, cmd := handler.Handle(msg)
			// Return the result, even if model is the same reference
			// The key was handled if we get here
			return newModel, cmd
		}
	}

	// No handler could process the key
	return model, nil
}

// GetActiveHandlers returns the current list of active handlers (for debugging)
func (h *HierarchicalKeyHandler) GetActiveHandlers() []KeyHandler {
	result := make([]KeyHandler, len(h.handlers))
	copy(result, h.handlers)
	return result
}

// GetContext returns the current key handler context
func (h *HierarchicalKeyHandler) GetContext() *KeyHandlerContext {
	return h.context
}

// ComponentKeyHandler is a base implementation for component key handlers
type ComponentKeyHandler struct {
	name     string
	priority KeyHandlerPriority
	model    tea.Model
	keys     map[string]func(tea.KeyMsg) (tea.Model, tea.Cmd)
}

// NewComponentKeyHandler creates a new component key handler
func NewComponentKeyHandler(name string, priority KeyHandlerPriority, model tea.Model) *ComponentKeyHandler {
	return &ComponentKeyHandler{
		name:     name,
		priority: priority,
		model:    model,
		keys:     make(map[string]func(tea.KeyMsg) (tea.Model, tea.Cmd)),
	}
}

// Name returns the handler name
func (h *ComponentKeyHandler) Name() string {
	return h.name
}

// Priority returns the handler priority
func (h *ComponentKeyHandler) Priority() KeyHandlerPriority {
	return h.priority
}

// AddKey adds a key binding to the handler
func (h *ComponentKeyHandler) AddKey(keyStr string, handler func(tea.KeyMsg) (tea.Model, tea.Cmd)) {
	h.keys[keyStr] = handler
}

// AddKeys adds multiple key bindings
func (h *ComponentKeyHandler) AddKeys(keys map[string]func(tea.KeyMsg) (tea.Model, tea.Cmd)) {
	for k, v := range keys {
		h.keys[k] = v
	}
}

// CanHandle checks if the handler can process the given key
func (h *ComponentKeyHandler) CanHandle(msg tea.KeyMsg) bool {
	_, exists := h.keys[msg.String()]
	return exists
}

// Handle processes the key
func (h *ComponentKeyHandler) Handle(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handler, exists := h.keys[msg.String()]; exists {
		return handler(msg)
	}
	return h.model, nil
}

// SetModel updates the handler's model reference
func (h *ComponentKeyHandler) SetModel(model tea.Model) {
	h.model = model
}

// BubbleTeaComponentAdapter adapts Bubble Tea components to work with the hierarchical system
type BubbleTeaComponentAdapter struct {
	name      string
	priority  KeyHandlerPriority
	component interface {
		Update(tea.Msg) (tea.Model, tea.Cmd)
	}
	model tea.Model
	// Keys that this component should handle (empty means handle all)
	handledKeys map[string]bool
}

// NewBubbleTeaComponentAdapter creates an adapter for Bubble Tea components
func NewBubbleTeaComponentAdapter(name string, priority KeyHandlerPriority, component interface {
	Update(tea.Msg) (tea.Model, tea.Cmd)
}, model tea.Model) *BubbleTeaComponentAdapter {
	return &BubbleTeaComponentAdapter{
		name:        name,
		priority:    priority,
		component:   component,
		model:       model,
		handledKeys: make(map[string]bool),
	}
}

// SetHandledKeys specifies which keys this component should handle
func (a *BubbleTeaComponentAdapter) SetHandledKeys(keys ...string) {
	a.handledKeys = make(map[string]bool)
	for _, key := range keys {
		a.handledKeys[key] = true
	}
}

// Name returns the adapter name
func (a *BubbleTeaComponentAdapter) Name() string {
	return a.name
}

// Priority returns the adapter priority
func (a *BubbleTeaComponentAdapter) Priority() KeyHandlerPriority {
	return a.priority
}

// CanHandle checks if the adapter can process the given key
func (a *BubbleTeaComponentAdapter) CanHandle(msg tea.KeyMsg) bool {
	// If no specific keys are set, handle all keys
	if len(a.handledKeys) == 0 {
		return true
	}

	// Only handle specified keys
	return a.handledKeys[msg.String()]
}

// Handle processes the key using the Bubble Tea component
func (a *BubbleTeaComponentAdapter) Handle(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel, cmd := a.component.Update(msg)
	// Update the adapter's model reference
	a.model = newModel
	return newModel, cmd
}

// GetModel returns the current model
func (a *BubbleTeaComponentAdapter) GetModel() tea.Model {
	return a.model
}

// SetModel updates the adapter's model reference
func (a *BubbleTeaComponentAdapter) SetModel(model tea.Model) {
	a.model = model
}
