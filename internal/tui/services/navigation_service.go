package services

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Sentinel errors
var (
	ErrInvalidView      = errors.New("INVALID_VIEW")
	ErrTransitionFailed = errors.New("TRANSITION_FAILED")
	ErrNoActiveView     = errors.New("NO_ACTIVE_VIEW")
	ErrCycleFailed      = errors.New("CYCLE_FAILED")
	ErrInvalidNumber    = errors.New("INVALID_NUMBER")
)

// ViewTransitionCallback represents a callback function for view transitions
type ViewTransitionCallback func(fromView, toView messages.ViewType) error

// NavigationService manages view navigation state and transitions
//
// ⚠️  SPECIAL CASE: This service has LIMITED internal state using lock-free patterns
//
// WHY THIS IS OK:
// - Uses atomic.Value for currentView (lock-free reads/writes)
// - Uses sync.Map for callbacks (lock-free concurrent access)
// - State is NOT duplicated from MasterData (navigation is service-specific)
// - No cached data from Model (would violate Redux/Elm pattern)
//
// ⚠️  WARNING: DO NOT add cached data from MasterData!
// ❌ WRONG:  cachedProcesses []*Process  // DON'T cache Model data!
// ❌ WRONG:  lastViewState    ViewState  // DON'T duplicate MasterData!
// WHY: Creates multiple sources of truth, requires synchronization, breaks Redux/Elm
// ✅ CORRECT: Only store service-specific state (navigation, callbacks, config)
//
// Note: The mu RWMutex protects viewOrder/numberKeyMappings (config only, rarely changed)
type NavigationService struct {
	// Current view state (atomic for lock-free access)
	currentView atomic.Value // messages.ViewType

	// View configuration
	viewOrder         []messages.ViewType
	numberKeyMappings map[int]messages.ViewType

	// Callback management
	transitionCallbacks sync.Map // map[string]*callbackEntry

	// Event system
	eventBus *events.EventBus

	// State management (ONLY for config protection, NOT for data access)
	mu sync.RWMutex
}

// callbackEntry stores callback information
type callbackEntry struct {
	id       string
	callback ViewTransitionCallback
	active   atomic.Bool
}

// NewNavigationService creates a new navigation service with default configuration
func NewNavigationService(eventBus *events.EventBus) *NavigationService {
	service := &NavigationService{
		eventBus: eventBus,
		viewOrder: []messages.ViewType{
			messages.ProcessesView,
			messages.LogsView,
			messages.ErrorsView,
			messages.URLsView,
			messages.WebView,
			messages.AICodersView,
			messages.SettingsView,
		},
		numberKeyMappings: make(map[int]messages.ViewType),
	}

	// Set default number key mappings (1-7)
	service.numberKeyMappings[1] = messages.ProcessesView
	service.numberKeyMappings[2] = messages.LogsView
	service.numberKeyMappings[3] = messages.ErrorsView
	service.numberKeyMappings[4] = messages.URLsView
	service.numberKeyMappings[5] = messages.WebView
	service.numberKeyMappings[6] = messages.AICodersView
	service.numberKeyMappings[7] = messages.SettingsView

	// Initialize with first view
	service.currentView.Store(messages.ProcessesView)

	return service
}

// SwitchToView switches to the specified view with validation
func (ns *NavigationService) SwitchToView(targetView messages.ViewType, trigger string) error {
	if !ns.isValidView(targetView) {
		return ErrInvalidView
	}

	// Get current view
	currentView, err := ns.GetCurrentView()
	if err != nil {
		// If no current view, set the target as current
		ns.currentView.Store(targetView)
		ns.publishViewChangedEvent(messages.ViewType(""), targetView, trigger)
		return nil
	}

	// No transition needed if already on target view
	if currentView == targetView {
		return nil
	}

	// Execute transition callbacks
	if err := ns.executeTransitionCallbacks(currentView, targetView); err != nil {
		ns.publishViewTransitionCallbackEvent(currentView, targetView, "", false, err.Error())
		return fmt.Errorf("%w: %v", ErrTransitionFailed, err)
	}

	// Update current view
	ns.currentView.Store(targetView)

	// Publish view changed event
	ns.publishViewChangedEvent(currentView, targetView, trigger)

	return nil
}

// GetCurrentView returns the current active view
func (ns *NavigationService) GetCurrentView() (messages.ViewType, error) {
	view := ns.currentView.Load()
	if view == nil {
		return "", ErrNoActiveView
	}
	return view.(messages.ViewType), nil
}

// CycleToNextView cycles to the next view in the order with wrap-around
func (ns *NavigationService) CycleToNextView() error {
	currentView, err := ns.GetCurrentView()
	if err != nil {
		// If no current view, start with first view
		if len(ns.viewOrder) > 0 {
			return ns.SwitchToView(ns.viewOrder[0], "cycle")
		}
		return fmt.Errorf("%w: no views configured", ErrCycleFailed)
	}

	// Find current view index
	currentIndex := -1
	for i, view := range ns.viewOrder {
		if view == currentView {
			currentIndex = i
			break
		}
	}

	// If current view not in order, start with first view
	if currentIndex == -1 {
		if len(ns.viewOrder) > 0 {
			return ns.SwitchToView(ns.viewOrder[0], "cycle")
		}
		return fmt.Errorf("%w: current view not in order", ErrCycleFailed)
	}

	// Calculate next view index with wrap-around
	nextIndex := (currentIndex + 1) % len(ns.viewOrder)
	nextView := ns.viewOrder[nextIndex]

	return ns.SwitchToView(nextView, "cycle")
}

// GetViewForNumberKey returns the view mapped to a number key (1-9)
func (ns *NavigationService) GetViewForNumberKey(number int) (messages.ViewType, error) {
	if number < 1 || number > 9 {
		return "", ErrInvalidNumber
	}

	ns.mu.RLock()
	view, exists := ns.numberKeyMappings[number]
	ns.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("%w: no mapping for number %d", ErrInvalidNumber, number)
	}

	return view, nil
}

// RegisterTransitionCallback registers a callback for view transitions
func (ns *NavigationService) RegisterTransitionCallback(callback ViewTransitionCallback) string {
	callbackID := uuid.New().String()

	entry := &callbackEntry{
		id:       callbackID,
		callback: callback,
	}
	entry.active.Store(true)

	ns.transitionCallbacks.Store(callbackID, entry)

	return callbackID
}

// UnregisterTransitionCallback removes a transition callback
func (ns *NavigationService) UnregisterTransitionCallback(callbackID string) {
	if entry, exists := ns.transitionCallbacks.Load(callbackID); exists {
		if cbEntry := entry.(*callbackEntry); cbEntry != nil {
			cbEntry.active.Store(false)
		}
		ns.transitionCallbacks.Delete(callbackID)
	}
}

// SetViewOrder sets the custom view cycling order
func (ns *NavigationService) SetViewOrder(newOrder []messages.ViewType) error {
	// Validate all views in order
	for _, view := range newOrder {
		if !ns.isValidView(view) {
			return ErrInvalidView
		}
	}

	ns.mu.Lock()
	oldOrder := make([]messages.ViewType, len(ns.viewOrder))
	copy(oldOrder, ns.viewOrder)
	ns.viewOrder = make([]messages.ViewType, len(newOrder))
	copy(ns.viewOrder, newOrder)
	ns.mu.Unlock()

	// Publish view order changed event
	ns.publishViewOrderChangedEvent(oldOrder, newOrder)

	return nil
}

// SetNumberKeyMapping sets a custom number key to view mapping
func (ns *NavigationService) SetNumberKeyMapping(number int, view messages.ViewType) error {
	if number < 1 || number > 9 {
		return ErrInvalidNumber
	}

	if !ns.isValidView(view) {
		return ErrInvalidView
	}

	ns.mu.Lock()
	oldView, exists := ns.numberKeyMappings[number]
	ns.numberKeyMappings[number] = view
	ns.mu.Unlock()

	// Determine action for event
	action := "added"
	if exists {
		if oldView == view {
			action = "unchanged"
		} else {
			action = "changed"
		}
	}

	// Publish number key mapping changed event
	ns.publishNumberKeyMappingChangedEvent(number, view, action)

	return nil
}

// GetViewOrder returns the current view cycling order
func (ns *NavigationService) GetViewOrder() []messages.ViewType {
	ns.mu.RLock()
	order := make([]messages.ViewType, len(ns.viewOrder))
	copy(order, ns.viewOrder)
	ns.mu.RUnlock()
	return order
}

// GetNumberKeyMappings returns all number key mappings
func (ns *NavigationService) GetNumberKeyMappings() map[int]messages.ViewType {
	ns.mu.RLock()
	mappings := make(map[int]messages.ViewType)
	for k, v := range ns.numberKeyMappings {
		mappings[k] = v
	}
	ns.mu.RUnlock()
	return mappings
}

// EventBus returns the event bus for external subscriptions
func (ns *NavigationService) EventBus() *events.EventBus {
	return ns.eventBus
}

// GetServiceName returns the service name for ServiceBridge integration
func (ns *NavigationService) GetServiceName() string {
	return "navigation"
}

// GetServiceStatus returns the current status of the navigation service
func (ns *NavigationService) GetServiceStatus() ServiceStatus {
	status := "active"
	currentView, _ := ns.GetCurrentView()
	details := fmt.Sprintf("current: %s, views: %d", currentView, len(ns.viewOrder))

	return ServiceStatus{
		Service: ns.GetServiceName(),
		Status:  status,
		Details: details,
	}
}

// SubscribeToEvents subscribes the navigation service to relevant system events
func (ns *NavigationService) SubscribeToEvents() {
	if ns.eventBus == nil {
		return
	}

	// Navigation service doesn't need to subscribe to external events
	// It only publishes events based on navigation actions
}

// Stop gracefully shuts down the navigation service
func (ns *NavigationService) Stop() error {
	// Clear all transition callbacks
	ns.transitionCallbacks.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*callbackEntry); ok {
			entry.active.Store(false)
		}
		ns.transitionCallbacks.Delete(key)
		return true
	})

	return nil
}

// Helper methods

// isValidView checks if a view type is valid
func (ns *NavigationService) isValidView(view messages.ViewType) bool {
	switch view {
	case messages.ProcessesView, messages.LogsView, messages.ErrorsView,
		messages.URLsView, messages.WebView, messages.AICodersView, messages.SettingsView:
		return true
	default:
		return false
	}
}

// executeTransitionCallbacks executes all registered transition callbacks
func (ns *NavigationService) executeTransitionCallbacks(fromView, toView messages.ViewType) error {
	var errors []string

	ns.transitionCallbacks.Range(func(key, value interface{}) bool {
		entry := value.(*callbackEntry)

		// Skip inactive callbacks
		if !entry.active.Load() {
			return true
		}

		// Execute callback
		if err := entry.callback(fromView, toView); err != nil {
			errors = append(errors, fmt.Sprintf("callback %s: %v", entry.id, err))

			// Publish callback error event
			ns.publishViewTransitionCallbackEvent(fromView, toView, entry.id, false, err.Error())
		} else {
			// Publish callback success event
			ns.publishViewTransitionCallbackEvent(fromView, toView, entry.id, true, "")
		}

		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("transition callbacks failed: %v", errors)
	}

	return nil
}

// Event publishing methods

// publishViewChangedEvent publishes a view changed event
func (ns *NavigationService) publishViewChangedEvent(fromView, toView messages.ViewType, trigger string) {
	if ns.eventBus != nil {
		msg := messages.ViewChangedMsg{
			FromView: fromView,
			ToView:   toView,
			Trigger:  trigger,
			Time:     time.Now(),
		}

		ns.eventBus.Publish(events.Event{
			Type: "navigation.view.changed",
			Data: map[string]interface{}{
				"message": msg,
			},
		})
	}
}

// publishViewOrderChangedEvent publishes a view order changed event
func (ns *NavigationService) publishViewOrderChangedEvent(oldOrder, newOrder []messages.ViewType) {
	if ns.eventBus != nil {
		msg := messages.ViewOrderChangedMsg{
			NewOrder: newOrder,
			OldOrder: oldOrder,
			Time:     time.Now(),
		}

		ns.eventBus.Publish(events.Event{
			Type: "navigation.view_order.changed",
			Data: map[string]interface{}{
				"message": msg,
			},
		})
	}
}

// publishNumberKeyMappingChangedEvent publishes a number key mapping changed event
func (ns *NavigationService) publishNumberKeyMappingChangedEvent(number int, view messages.ViewType, action string) {
	if ns.eventBus != nil {
		msg := messages.NumberKeyMappingChangedMsg{
			Number: number,
			View:   view,
			Action: action,
			Time:   time.Now(),
		}

		ns.eventBus.Publish(events.Event{
			Type: "navigation.number_key_mapping.changed",
			Data: map[string]interface{}{
				"message": msg,
			},
		})
	}
}

// publishViewTransitionCallbackEvent publishes a view transition callback event
func (ns *NavigationService) publishViewTransitionCallbackEvent(fromView, toView messages.ViewType, callbackID string, success bool, errorMsg string) {
	if ns.eventBus != nil {
		msg := messages.ViewTransitionCallbackMsg{
			FromView:   fromView,
			ToView:     toView,
			CallbackID: callbackID,
			Success:    success,
			Error:      errorMsg,
			Time:       time.Now(),
		}

		ns.eventBus.Publish(events.Event{
			Type: "navigation.view_transition.callback",
			Data: map[string]interface{}{
				"message": msg,
			},
		})
	}
}

// PTY Testing Support Methods
// These methods provide enhanced functionality for PTY-based testing scenarios

// SetCurrentViewForTesting sets the current view directly for testing purposes
// This method bypasses normal validation and callbacks for test scenarios
func (ns *NavigationService) SetCurrentViewForTesting(view messages.ViewType) {
	ns.currentView.Store(view)
}

// ForceSwitchToViewForTesting forces a view switch for testing, bypassing validation
// This is useful for testing specific scenarios where normal validation would prevent the switch
func (ns *NavigationService) ForceSwitchToViewForTesting(targetView, trigger string) error {
	// Get current view for event publishing
	currentView, _ := ns.GetCurrentView()

	// Convert string to ViewType for testing
	targetViewType := messages.ViewType(targetView)

	// Skip validation and callbacks for testing
	ns.currentView.Store(targetViewType)

	// Publish view changed event
	ns.publishViewChangedEvent(currentView, targetViewType, trigger)

	return nil
}

// GetNavigationStateForTesting returns the current navigation state for testing validation
func (ns *NavigationService) GetNavigationStateForTesting() NavigationState {
	currentView, _ := ns.GetCurrentView()

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	state := NavigationState{
		CurrentView:        currentView,
		ViewOrder:          make([]messages.ViewType, len(ns.viewOrder)),
		NumberKeyMappings:  make(map[int]messages.ViewType),
		RegisteredCallbacks: ns.getCallbackCount(),
	}

	// Copy view order
	copy(state.ViewOrder, ns.viewOrder)

	// Copy number key mappings
	for k, v := range ns.numberKeyMappings {
		state.NumberKeyMappings[k] = v
	}

	return state
}

// ValidateStateForTesting performs comprehensive state validation for testing
func (ns *NavigationService) ValidateStateForTesting() error {
	// Validate current view
	currentView, err := ns.GetCurrentView()
	if err != nil {
		return fmt.Errorf("current view validation failed: %w", err)
	}

	if !ns.isValidView(currentView) {
		return fmt.Errorf("current view %s is invalid", currentView)
	}

	// Validate view order
	ns.mu.RLock()
	viewOrder := make([]messages.ViewType, len(ns.viewOrder))
	copy(viewOrder, ns.viewOrder)
	ns.mu.RUnlock()

	if len(viewOrder) == 0 {
		return fmt.Errorf("view order is empty")
	}

	for _, view := range viewOrder {
		if !ns.isValidView(view) {
			return fmt.Errorf("view %s in view order is invalid", view)
		}
	}

	// Validate number key mappings
	mappings := ns.GetNumberKeyMappings()
	for num, view := range mappings {
		if num < 1 || num > 9 {
			return fmt.Errorf("number key %d is out of range", num)
		}
		if !ns.isValidView(view) {
			return fmt.Errorf("number key %d maps to invalid view %s", num, view)
		}
	}

	return nil
}

// ResetToInitialState resets the navigation service to initial state for testing
func (ns *NavigationService) ResetToInitialState() {
	// Reset to default view
	ns.currentView.Store(messages.ProcessesView)

	// Clear all transition callbacks
	ns.transitionCallbacks.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*callbackEntry); ok {
			entry.active.Store(false)
		}
		ns.transitionCallbacks.Delete(key)
		return true
	})

	// Reset to default view order and mappings
	ns.mu.Lock()
	ns.viewOrder = []messages.ViewType{
		messages.ProcessesView,
		messages.LogsView,
		messages.ErrorsView,
		messages.URLsView,
		messages.WebView,
		messages.AICodersView,
		messages.SettingsView,
	}
	ns.numberKeyMappings = map[int]messages.ViewType{
		1: messages.ProcessesView,
		2: messages.LogsView,
		3: messages.ErrorsView,
		4: messages.URLsView,
		5: messages.WebView,
		6: messages.AICodersView,
		7: messages.SettingsView,
	}
	ns.mu.Unlock()
}

// GetTransitionCountForTesting returns the number of active transition callbacks for testing
func (ns *NavigationService) GetTransitionCountForTesting() int {
	count := 0
	ns.transitionCallbacks.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*callbackEntry); ok {
			if entry.active.Load() {
				count++
			}
		}
		return true
	})
	return count
}

// getCallbackCount returns the number of registered callbacks
func (ns *NavigationService) getCallbackCount() int {
	count := 0
	ns.transitionCallbacks.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// NavigationState represents the navigation service state for testing
type NavigationState struct {
	CurrentView        messages.ViewType
	ViewOrder          []messages.ViewType
	NumberKeyMappings  map[int]messages.ViewType
	RegisteredCallbacks int
}