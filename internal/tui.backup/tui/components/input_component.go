package components

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// InputComponent manages keyboard input routing and key bindings
// This component migrates functionality from the old InputController
// to use the new service-based architecture
type InputComponent struct {
	BaseComponent

	// Service integration
	inputService *services.InputService
	eventBus     *events.EventBus

	// Component state
	context         string
	globalHandlers  []string      // Handler IDs
	contextHandlers []string      // Handler IDs
	keyBindings     []services.KeyBinding

	// Event subscription
	subscriptionCh chan events.Event

	// Input processing state
	lastKeyTime time.Time
	keySequence []string
}

// NewInputComponent creates a new input component
func NewInputComponent(inputService *services.InputService) *InputComponent {
	if inputService == nil {
		inputService = services.NewInputService()
	}

	component := &InputComponent{
		BaseComponent:   NewBaseComponent(),
		inputService:    inputService,
		eventBus:        inputService.EventBus(),
		context:         "global",
		subscriptionCh:  make(chan events.Event, 100),
		globalHandlers:  make([]string, 0),
		contextHandlers: make([]string, 0),
		keyBindings:     make([]services.KeyBinding, 0),
		lastKeyTime:     time.Now(),
		keySequence:     make([]string, 0),
	}

	// Subscribe to input service events
	component.subscribeToEvents()

	return component
}

// Init initializes the component
func (ic *InputComponent) Init() tea.Cmd {
	// Start the input service
	return func() tea.Msg {
		ctx := context.Background()
		if err := ic.inputService.Start(ctx); err != nil {
			return messages.ErrorMessage{
				Source:  "InputComponent",
				Message: "Failed to start input service: " + err.Error(),
				Time:    time.Now(),
			}
		}
		return messages.ComponentReadyMsg{
			ComponentID: "input",
			Time:        time.Now(),
		}
	}
}

// Update handles incoming messages and keyboard input
func (ic *InputComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return ic.handleKeyMsg(msg)

	case messages.InputContextChangedMsg:
		return ic.handleContextChanged(msg), nil

	case messages.KeyBindingRegisterMsg:
		return ic.handleKeyBindingRegister(msg), nil

	case messages.KeyBindingUnregisterMsg:
		return ic.handleKeyBindingUnregister(msg), nil

	case events.Event:
		return ic.handleServiceEvent(msg)

	default:
		return ic, nil
	}
}

// View renders the input component
// Note: This component is primarily for input routing and doesn't have visual output
// The visual feedback is handled by other components based on the routing results
func (ic *InputComponent) View() string {
	// This component doesn't have a visual representation
	// It handles input routing behind the scenes
	return ""
}

// handleKeyMsg processes keyboard input using the service
func (ic *InputComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	now := time.Now()
	ic.lastKeyTime = now

	// Convert key to string for sequence tracking
	keyStr := ic.keyToString(msg)
	ic.keySequence = append(ic.keySequence, keyStr)

	// Keep sequence limited to last 10 keys
	if len(ic.keySequence) > 10 {
		ic.keySequence = ic.keySequence[1:]
	}

	// Route the key through the input service
	handled, cmd := ic.inputService.RouteKeyMessage(msg, ic.context)

	// Create input key pressed message for notification
	keyPressedMsg := messages.InputKeyPressedMsg{
		Key:     msg,
		Context: ic.context,
		Handled: handled,
		Time:    now,
	}

	// If the key wasn't handled by the service, check for built-in behaviors
	if !handled {
		return ic.handleUnhandledKey(msg, keyPressedMsg)
	}

	// Return the command from the handler if one was provided
	if cmd != nil {
		return ic, cmd
	}

	return ic, func() tea.Msg { return keyPressedMsg }
}

// handleContextChanged handles input context changes
func (ic *InputComponent) handleContextChanged(msg messages.InputContextChangedMsg) tea.Model {
	ic.context = msg.NewContext

	// Clear context-specific handlers when context changes
	ic.contextHandlers = make([]string, 0)

	// Re-register handlers for new context if provided
	if msg.Handler != nil {
		handlerID, err := ic.registerContextHandler(msg.NewContext, msg.Handler)
		if err == nil {
			ic.contextHandlers = append(ic.contextHandlers, handlerID)
		}
	}

	return ic
}

// handleKeyBindingRegister handles key binding registration
func (ic *InputComponent) handleKeyBindingRegister(msg messages.KeyBindingRegisterMsg) tea.Model {
	binding := services.KeyBinding{
		Key:      msg.Key,
		Action:   msg.Action,
		Context:  msg.Context,
		Priority: msg.Priority,
	}

	ctx := context.Background()
	if err := ic.inputService.RegisterKeyBinding(ctx, binding); err != nil {
		// Publish error event
		ic.eventBus.Publish(events.Event{
			Type: "input.binding.error",
			Data: map[string]interface{}{
				"error":   err.Error(),
				"binding": binding,
			},
			Timestamp: time.Now(),
		})
	} else {
		ic.keyBindings = append(ic.keyBindings, binding)
	}

	return ic
}

// handleKeyBindingUnregister handles key binding removal
func (ic *InputComponent) handleKeyBindingUnregister(msg messages.KeyBindingUnregisterMsg) tea.Model {
	ctx := context.Background()
	if err := ic.inputService.UnregisterKeyBinding(ctx, msg.Key); err != nil {
		// Publish error event
		ic.eventBus.Publish(events.Event{
			Type: "input.binding.error",
			Data: map[string]interface{}{
				"error": err.Error(),
				"key":   msg.Key,
			},
			Timestamp: time.Now(),
		})
	} else {
		// Remove from local cache
		ic.removeKeyBinding(msg.Key)
	}

	return ic
}

// handleServiceEvent handles events from the input service
func (ic *InputComponent) handleServiceEvent(event events.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case "input.key.pressed":
		return ic.handleInputKeyPressedEvent(event)
	case "input.binding.changed":
		return ic.handleBindingChangedEvent(event)
	case "input.handler.registered":
		return ic.handleHandlerRegisteredEvent(event)
	default:
		return ic, nil
	}
}

// handleUnhandledKey processes keys that weren't handled by the service
func (ic *InputComponent) handleUnhandledKey(msg tea.KeyMsg, keyPressedMsg messages.InputKeyPressedMsg) (tea.Model, tea.Cmd) {
	// Check for built-in component behaviors
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		// Handle quit functionality with double-press protection
		return ic.handleQuitKey(keyPressedMsg)
	default:
		// Return the unhandled key press message
		return ic, func() tea.Msg { return keyPressedMsg }
	}
}

// handleQuitKey handles quit functionality with double-press protection
func (ic *InputComponent) handleQuitKey(keyPressedMsg messages.InputKeyPressedMsg) (tea.Model, tea.Cmd) {
	// Check for double-press within 500ms
	if len(ic.keySequence) >= 2 {
		lastKey := ic.keySequence[len(ic.keySequence)-2]
		currentKey := ic.keySequence[len(ic.keySequence)-1]

		// Check if last two keys were quit keys within time window
		if (lastKey == "ctrl+c" || lastKey == "esc") &&
		   (currentKey == "ctrl+c" || currentKey == "esc") &&
		   ic.lastKeyTime.Sub(ic.lastKeyTime) < 500*time.Millisecond {

			// Double quit detected - send quit message
			return ic, func() tea.Msg {
				return messages.QuitRequestedMsg{
					Source: "InputComponent",
					Time:   time.Now(),
				}
			}
		}
	}

	// First quit press - show notification
	return ic, func() tea.Msg {
		return messages.SystemNotificationMsg{
			Title:   "Quit",
			Message: "Press quit key again within 500ms to exit",
			Level:   messages.InfoNotification,
			Time:    time.Now(),
		}
	}
}

// RegisterGlobalHandler registers a global keyboard handler
func (ic *InputComponent) RegisterGlobalHandler(handler func(tea.KeyMsg) tea.Cmd) (string, error) {
	ctx := context.Background()
	err := ic.inputService.SubscribeToGlobalKeys(ctx, handler)
	if err != nil {
		return "", err
	}

	// Generate handler ID (simplified - in real implementation, service would return this)
	handlerID := "global_" + time.Now().Format("20060102150405")
	ic.globalHandlers = append(ic.globalHandlers, handlerID)
	return handlerID, nil
}

// registerContextHandler registers a context-specific keyboard handler
func (ic *InputComponent) registerContextHandler(contextName string, handler func(tea.KeyMsg) tea.Cmd) (string, error) {
	ctx := context.Background()
	err := ic.inputService.SubscribeWithContextKeys(ctx, contextName, handler)
	if err != nil {
		return "", err
	}

	// Generate handler ID
	handlerID := "context_" + contextName + "_" + time.Now().Format("20060102150405")
	ic.contextHandlers = append(ic.contextHandlers, handlerID)
	return handlerID, nil
}

// subscribeToEvents subscribes to input service events
func (ic *InputComponent) subscribeToEvents() {
	// Subscribe to input service events through the event bus
	// This would typically use the event bus subscription mechanism
	// For now, we'll simulate this with direct channel subscription
}

// Helper methods

func (ic *InputComponent) keyToString(msg tea.KeyMsg) string {
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		return string(msg.Runes)
	}
	return msg.Type.String()
}

func (ic *InputComponent) removeKeyBinding(key tea.Key) {
	for i, binding := range ic.keyBindings {
		if binding.Key.Type == key.Type &&
		   len(binding.Key.Runes) == len(key.Runes) {
			// Simple key comparison - could be enhanced
			ic.keyBindings = append(ic.keyBindings[:i], ic.keyBindings[i+1:]...)
			break
		}
	}
}

func (ic *InputComponent) handleInputKeyPressedEvent(event events.Event) (tea.Model, tea.Cmd) {
	// Convert service event to component message
	data := event.Data
	if handled, ok := data["handled"].(bool); ok && handled {
		// Key was handled by service
		return ic, nil
	}
	return ic, nil
}

func (ic *InputComponent) handleBindingChangedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data
	action, _ := data["action"].(string)
	key, _ := data["key"].(string)

	bindingMsg := messages.KeyBindingChangedMsg{
		Action: action,
		Key:    key,
		Time:   event.Timestamp,
	}

	return ic, func() tea.Msg { return bindingMsg }
}

func (ic *InputComponent) handleHandlerRegisteredEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data
	handlerID, _ := data["handler_id"].(string)
	component, _ := data["component"].(string)
	priority, _ := data["priority"].(int)

	handlerMsg := messages.InputHandlerRegisteredMsg{
		HandlerID: handlerID,
		Component: component,
		Priority:  priority,
		Time:      event.Timestamp,
	}

	return ic, func() tea.Msg { return handlerMsg }
}

// Focus gives keyboard focus to the input component
func (ic *InputComponent) Focus() tea.Cmd {
	ic.BaseComponent.Focus()

	// Publish focus change event
	ic.eventBus.Publish(events.Event{
		Type: "input.focus.changed",
		Data: map[string]interface{}{
			"component": "InputComponent",
			"focused":   true,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// Blur removes keyboard focus from the input component
func (ic *InputComponent) Blur() tea.Cmd {
	ic.BaseComponent.Blur()

	// Publish focus change event
	ic.eventBus.Publish(events.Event{
		Type: "input.focus.changed",
		Data: map[string]interface{}{
			"component": "InputComponent",
			"focused":   false,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// GetContext returns the current input context
func (ic *InputComponent) GetContext() string {
	return ic.context
}

// SetContext changes the current input context
func (ic *InputComponent) SetContext(context string) {
	ic.context = context
}

// GetKeyBindings returns all registered key bindings
func (ic *InputComponent) GetKeyBindings() []services.KeyBinding {
	return ic.keyBindings
}

// Cleanup performs cleanup when component is destroyed
func (ic *InputComponent) Cleanup() error {
	ctx := context.Background()
	return ic.inputService.Shutdown(ctx)
}