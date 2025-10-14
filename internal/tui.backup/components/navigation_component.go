package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// NavigationComponent manages view navigation and switching
// This component migrates functionality from the old NavigationController
// to use the new service-based architecture
type NavigationComponent struct {
	BaseComponent

	// Service integration
	navService *services.NavigationService
	eventBus   *events.EventBus

	// Component state
	currentView            messages.ViewType
	viewOrder              []messages.ViewType
	numberKeyMappings      map[int]messages.ViewType
	transitionCallbackIDs  []string

	// Event subscription
	subscriptionCh chan events.Event

	// Performance tracking
	lastNavigationTime time.Time
	navigationCount    int
}

// NewNavigationComponent creates a new navigation component
func NewNavigationComponent(navService *services.NavigationService) *NavigationComponent {
	if navService == nil {
		navService = services.NewNavigationService(events.NewEventBus())
	}

	component := &NavigationComponent{
		BaseComponent:        NewBaseComponent(),
		navService:           navService,
		eventBus:             navService.EventBus(),
		subscriptionCh:       make(chan events.Event, 100),
		numberKeyMappings:    make(map[int]messages.ViewType),
		transitionCallbackIDs: make([]string, 0),
		lastNavigationTime:   time.Now(),
		navigationCount:      0,
	}

	// Initialize component state from service
	component.syncFromService()

	// Subscribe to navigation service events
	component.subscribeToEvents()

	return component
}

// Init initializes the component
func (nc *NavigationComponent) Init() tea.Cmd {
	// Get initial view from service
	initCmd := func() tea.Msg {
		currentView, err := nc.navService.GetCurrentView()
		if err != nil {
			return messages.ErrorMessage{
				Source:  "NavigationComponent",
				Message: "Failed to get current view: " + err.Error(),
				Time:    time.Now(),
			}
		}

		nc.currentView = currentView
		return messages.ComponentReadyMsg{
			ComponentID: "navigation",
			Time:        time.Now(),
		}
	}

	// Start listening for subscription events
	listenCmd := nc.listenForSubscriptionEvents()

	return tea.Batch(initCmd, listenCmd)
}

// listenForSubscriptionEvents listens for events on the subscription channel
func (nc *NavigationComponent) listenForSubscriptionEvents() tea.Cmd {
	return func() tea.Msg {
		for {
			select {
			case event := <-nc.subscriptionCh:
				return event
			}
		}
	}
}

// Update handles incoming messages and navigation requests
func (nc *NavigationComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.ViewChangedMsg:
		return nc.handleViewChanged(msg)

	case messages.ViewSwitchRequestedMsg:
		return nc.handleViewSwitchRequest(msg)

	case messages.ViewOrderChangedMsg:
		return nc.handleViewOrderChanged(msg)

	case messages.NumberKeyMappingChangedMsg:
		return nc.handleNumberKeyMappingChanged(msg)

	case tea.KeyMsg:
		return nc.handleKeyMsg(msg)

	case events.Event:
		return nc.handleServiceEvent(msg)

	default:
		return nc, nil
	}
}

// View renders the navigation component
// Note: This component is primarily for navigation management and doesn't have visual output
// The visual feedback is handled by other components based on the navigation results
func (nc *NavigationComponent) View() string {
	// This component doesn't have a visual representation
	// It handles navigation behind the scenes
	return ""
}

// handleViewChanged handles view changed messages
func (nc *NavigationComponent) handleViewChanged(msg messages.ViewChangedMsg) (tea.Model, tea.Cmd) {
	// Update component state
	nc.currentView = msg.ToView
	nc.lastNavigationTime = msg.Time
	nc.navigationCount++

	return nc, nil
}

// handleViewSwitchRequest processes view switch requests
func (nc *NavigationComponent) handleViewSwitchRequest(msg messages.ViewSwitchRequestedMsg) (tea.Model, tea.Cmd) {
	err := nc.navService.SwitchToView(msg.TargetView, msg.Trigger)
	if err != nil {
		return nc, func() tea.Msg {
			return messages.ErrorMessage{
				Source:  "NavigationComponent",
				Message: "Failed to switch view: " + err.Error(),
				Time:    time.Now(),
			}
		}
	}

	// Update component state
	nc.currentView = msg.TargetView
	nc.lastNavigationTime = time.Now()
	nc.navigationCount++

	return nc, func() tea.Msg {
		return messages.ViewChangedMsg{
			FromView: nc.currentView, // This will be updated after the service call
			ToView:   msg.TargetView,
			Trigger:  msg.Trigger,
			Time:     time.Now(),
		}
	}
}

// handleViewOrderChanged handles view order changes
func (nc *NavigationComponent) handleViewOrderChanged(msg messages.ViewOrderChangedMsg) (tea.Model, tea.Cmd) {
	nc.viewOrder = msg.NewOrder
	return nc, nil
}

// handleNumberKeyMappingChanged handles number key mapping changes
func (nc *NavigationComponent) handleNumberKeyMappingChanged(msg messages.NumberKeyMappingChangedMsg) (tea.Model, tea.Cmd) {
	nc.numberKeyMappings[msg.Number] = msg.View
	return nc, nil
}

// handleKeyMsg processes keyboard input for navigation
func (nc *NavigationComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab:
		// Cycle to next view
		err := nc.navService.CycleToNextView()
		if err != nil {
			return nc, func() tea.Msg {
				return messages.ErrorMessage{
					Source:  "NavigationComponent",
					Message: "Failed to cycle view: " + err.Error(),
					Time:    time.Now(),
				}
			}
		}

		nc.lastNavigationTime = time.Now()
		nc.navigationCount++

		return nc, func() tea.Msg {
			return messages.ViewSwitchRequestedMsg{
				TargetView: "", // Service will determine the next view
				Trigger:    "tab",
				Requester:  "NavigationComponent",
				Time:       time.Now(),
			}
		}

	default:
		// Check for number keys (1-9)
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			runeVal := msg.Runes[0]
			if runeVal >= '1' && runeVal <= '9' {
				number := int(runeVal - '0')
				return nc.handleNumberKey(number)
			}
		}

		// Check for specific key strings
		keyStr := msg.String()
		if keyStr == "shift+tab" {
			// Previous view (reverse cycle)
			return nc.handlePreviousView()
		}

		return nc, nil
	}
}

// handleNumberKey handles number key presses for direct view switching
func (nc *NavigationComponent) handleNumberKey(number int) (tea.Model, tea.Cmd) {
	view, err := nc.navService.GetViewForNumberKey(number)
	if err != nil {
		// No mapping for this number - ignore
		return nc, nil
	}

	err = nc.navService.SwitchToView(view, "number_key")
	if err != nil {
		return nc, func() tea.Msg {
			return messages.ErrorMessage{
				Source:  "NavigationComponent",
				Message: "Failed to switch to view: " + err.Error(),
				Time:    time.Now(),
			}
		}
	}

	nc.lastNavigationTime = time.Now()
	nc.navigationCount++

	return nc, func() tea.Msg {
		return messages.ViewSwitchRequestedMsg{
			TargetView: view,
			Trigger:    "number_key",
			Requester:  "NavigationComponent",
			Time:       time.Now(),
		}
	}
}

// handlePreviousView handles cycling to the previous view
func (nc *NavigationComponent) handlePreviousView() (tea.Model, tea.Cmd) {
	// Get current view order
	viewOrder := nc.navService.GetViewOrder()
	if len(viewOrder) < 2 {
		return nc, nil
	}

	// Find current view index
	currentView, err := nc.navService.GetCurrentView()
	if err != nil {
		return nc, nil
	}

	currentIndex := -1
	for i, view := range viewOrder {
		if view == currentView {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return nc, nil
	}

	// Calculate previous view index with wrap-around
	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(viewOrder) - 1
	}

	prevView := viewOrder[prevIndex]

	err = nc.navService.SwitchToView(prevView, "shift_tab")
	if err != nil {
		return nc, func() tea.Msg {
			return messages.ErrorMessage{
				Source:  "NavigationComponent",
				Message: "Failed to switch to previous view: " + err.Error(),
				Time:    time.Now(),
			}
		}
	}

	nc.lastNavigationTime = time.Now()
	nc.navigationCount++

	return nc, func() tea.Msg {
		return messages.ViewSwitchRequestedMsg{
			TargetView: prevView,
			Trigger:    "shift_tab",
			Requester:  "NavigationComponent",
			Time:       time.Now(),
		}
	}
}

// handleServiceEvent handles events from the navigation service
func (nc *NavigationComponent) handleServiceEvent(event events.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case "navigation.view.changed":
		return nc.handleViewChangedEvent(event)
	case "navigation.view_order.changed":
		return nc.handleViewOrderChangedEvent(event)
	case "navigation.number_key_mapping.changed":
		return nc.handleNumberKeyMappingChangedEvent(event)
	default:
		return nc, nil
	}
}

// handleViewChangedEvent handles view changed events from service
func (nc *NavigationComponent) handleViewChangedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data
	if messageData, ok := data["message"].(map[string]interface{}); ok {
		if toView, ok := messageData["to_view"].(string); ok {
			nc.currentView = messages.ViewType(toView)
		}
	}

	return nc, func() tea.Msg {
		// Convert event data back to message
		if messageData, ok := data["message"].(map[string]interface{}); ok {
			if fromView, ok := messageData["from_view"].(string); ok {
				if toView, ok := messageData["to_view"].(string); ok {
					if trigger, ok := messageData["trigger"].(string); ok {
						return messages.ViewChangedMsg{
							FromView: messages.ViewType(fromView),
							ToView:   messages.ViewType(toView),
							Trigger:  trigger,
							Time:     event.Timestamp,
						}
					}
				}
			}
		}
		return nil
	}
}

// handleViewOrderChangedEvent handles view order changed events from service
func (nc *NavigationComponent) handleViewOrderChangedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data
	if messageData, ok := data["message"].(map[string]interface{}); ok {
		if newOrderData, ok := messageData["new_order"].([]interface{}); ok {
			newOrder := make([]messages.ViewType, len(newOrderData))
			for i, view := range newOrderData {
				if viewStr, ok := view.(string); ok {
					newOrder[i] = messages.ViewType(viewStr)
				}
			}
			nc.viewOrder = newOrder
		}
	}

	return nc, func() tea.Msg {
		// Convert event data back to message
		if _, ok := data["message"].(map[string]interface{}); ok {
			return messages.ViewOrderChangedMsg{
				Time: event.Timestamp,
			}
		}
		return nil
	}
}

// handleNumberKeyMappingChangedEvent handles number key mapping changed events from service
func (nc *NavigationComponent) handleNumberKeyMappingChangedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data
	if messageData, ok := data["message"].(map[string]interface{}); ok {
		if number, ok := messageData["number"].(float64); ok {
			if view, ok := messageData["view"].(string); ok {
				nc.numberKeyMappings[int(number)] = messages.ViewType(view)
			}
		}
	}

	return nc, func() tea.Msg {
		// Convert event data back to message
		if messageData, ok := data["message"].(map[string]interface{}); ok {
			if number, ok := messageData["number"].(float64); ok {
				if view, ok := messageData["view"].(string); ok {
					if action, ok := messageData["action"].(string); ok {
						return messages.NumberKeyMappingChangedMsg{
							Number: int(number),
							View:   messages.ViewType(view),
							Action: action,
							Time:   event.Timestamp,
						}
					}
				}
			}
		}
		return nil
	}
}

// syncFromService synchronizes component state from the service
func (nc *NavigationComponent) syncFromService() {
	// Get current view
	if view, err := nc.navService.GetCurrentView(); err == nil {
		nc.currentView = view
	}

	// Get view order
	nc.viewOrder = nc.navService.GetViewOrder()

	// Get number key mappings
	nc.numberKeyMappings = nc.navService.GetNumberKeyMappings()
}

// subscribeToEvents subscribes to navigation service events
func (nc *NavigationComponent) subscribeToEvents() {
	// Subscribe to navigation service events through the event bus
	if nc.eventBus != nil {
		// Subscribe to view changed events
		nc.eventBus.Subscribe("navigation.view.changed", func(event events.Event) {
			// Send event to component's Update method via channel
			select {
			case nc.subscriptionCh <- event:
			default:
				// Channel is full, drop the event
			}
		})

		// Subscribe to view order changed events
		nc.eventBus.Subscribe("navigation.view_order.changed", func(event events.Event) {
			select {
			case nc.subscriptionCh <- event:
			default:
				// Channel is full, drop the event
			}
		})

		// Subscribe to number key mapping changed events
		nc.eventBus.Subscribe("navigation.number_key_mapping.changed", func(event events.Event) {
			select {
			case nc.subscriptionCh <- event:
			default:
				// Channel is full, drop the event
			}
		})
	}
}

// Public API methods

// GetCurrentView returns the current active view
func (nc *NavigationComponent) GetCurrentView() messages.ViewType {
	return nc.currentView
}

// GetViewOrder returns the current view cycling order
func (nc *NavigationComponent) GetViewOrder() []messages.ViewType {
	return nc.viewOrder
}

// GetNumberKeyMappings returns all number key mappings
func (nc *NavigationComponent) GetNumberKeyMappings() map[int]messages.ViewType {
	return nc.numberKeyMappings
}

// SwitchToView switches to the specified view
func (nc *NavigationComponent) SwitchToView(view messages.ViewType, trigger string) tea.Cmd {
	return func() tea.Msg {
		return messages.ViewSwitchRequestedMsg{
			TargetView: view,
			Trigger:    trigger,
			Requester:  "NavigationComponent",
			Time:       time.Now(),
		}
	}
}

// CycleToNextView cycles to the next view
func (nc *NavigationComponent) CycleToNextView() tea.Cmd {
	return func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyTab}
	}
}

// GetViewForNumberKey returns the view mapped to a number key
func (nc *NavigationComponent) GetViewForNumberKey(number int) (messages.ViewType, error) {
	return nc.navService.GetViewForNumberKey(number)
}

// RegisterTransitionCallback registers a callback for view transitions
func (nc *NavigationComponent) RegisterTransitionCallback(callback func(fromView, toView messages.ViewType) error) string {
	callbackID := nc.navService.RegisterTransitionCallback(callback)
	nc.transitionCallbackIDs = append(nc.transitionCallbackIDs, callbackID)
	return callbackID
}

// UnregisterTransitionCallback removes a transition callback
func (nc *NavigationComponent) UnregisterTransitionCallback(callbackID string) {
	nc.navService.UnregisterTransitionCallback(callbackID)
	// Remove from local list
	for i, id := range nc.transitionCallbackIDs {
		if id == callbackID {
			nc.transitionCallbackIDs = append(nc.transitionCallbackIDs[:i], nc.transitionCallbackIDs[i+1:]...)
			break
		}
	}
}

// GetNavigationStats returns navigation performance statistics
func (nc *NavigationComponent) GetNavigationStats() (int, time.Time) {
	return nc.navigationCount, nc.lastNavigationTime
}

// Focus gives keyboard focus to the navigation component
func (nc *NavigationComponent) Focus() tea.Cmd {
	nc.BaseComponent.Focus()

	// Publish focus change event
	nc.eventBus.Publish(events.Event{
		Type: "navigation.focus.changed",
		Data: map[string]interface{}{
			"component": "NavigationComponent",
			"focused":   true,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// Blur removes keyboard focus from the navigation component
func (nc *NavigationComponent) Blur() tea.Cmd {
	nc.BaseComponent.Blur()

	// Publish focus change event
	nc.eventBus.Publish(events.Event{
		Type: "navigation.focus.changed",
		Data: map[string]interface{}{
			"component": "NavigationComponent",
			"focused":   false,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// Cleanup performs cleanup when component is destroyed
func (nc *NavigationComponent) Cleanup() error {
	// Unregister all transition callbacks
	for _, callbackID := range nc.transitionCallbackIDs {
		nc.navService.UnregisterTransitionCallback(callbackID)
	}
	nc.transitionCallbackIDs = make([]string, 0)

	return nil
}