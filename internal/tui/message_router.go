package tui

import (
	"fmt"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
)

// MessageHandler defines the interface for handling specific message types
type MessageHandler interface {
	HandleMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd)
	CanHandle(msg tea.Msg) bool
}

// MessageRouter routes messages to appropriate handlers
type MessageRouter struct {
	handlers []MessageHandler
}

// NewMessageRouter creates a new message router with all handlers
// Note: This is called during Model creation before keyRegistry exists,
// so keyboard handler registration happens later via RegisterKeyboardHandler
func NewMessageRouter() *MessageRouter {
	router := &MessageRouter{
		handlers: make([]MessageHandler, 0),
	}

	// Register handlers in order of priority
	// Priority 0: Notifications (observes all messages, never blocks)
	router.RegisterHandler(NewNotificationsMessageHandler())

	// Priority 1: Dialogs (highest blocking priority)
	router.RegisterHandler(NewDialogMessageHandler())

	// Priority 2: System messages
	router.RegisterHandler(NewSystemMessageHandler())

	// Priority 3: Domain-specific handlers
	router.RegisterHandler(NewProcessMessageHandler())
	router.RegisterHandler(NewLogMessageHandler())
	router.RegisterHandler(NewMCPMessageHandler())
	router.RegisterHandler(NewAICoderMessageHandler())
	router.RegisterHandler(NewSettingsMessageHandler())
	router.RegisterHandler(NewNavigationMessageHandler())

	// Priority 4: View-specific handlers (low priority)
	router.RegisterHandler(NewViewSpecificHandler())
	router.RegisterHandler(NewViewMessageHandler())

	// Note: KeyboardMessageHandler is registered after keyRegistry creation
	// via RegisterKeyboardHandler() method

	return router
}

// RegisterKeyboardHandler adds the keyboard handler to the router
// This must be called after keyRegistry is created
func (r *MessageRouter) RegisterKeyboardHandler(keyRegistry *KeyHandlerRegistry) {
	// Add keyboard handler at the end - it should process key messages that
	// haven't been handled by more specific handlers like dialogs
	keyboardHandler := NewKeyboardMessageHandler(keyRegistry)
	r.handlers = append(r.handlers, keyboardHandler)
}

// RegisterHandler adds a message handler to the router
func (r *MessageRouter) RegisterHandler(handler MessageHandler) {
	r.handlers = append(r.handlers, handler)
}

// Route dispatches a message to the appropriate handler
func (r *MessageRouter) Route(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Try each handler that can handle the message
	// Multiple handlers may process the same message (e.g., for cross-cutting concerns)
	for _, handler := range r.handlers {
		if handler.CanHandle(msg) {
			var cmd tea.Cmd
			var newModel tea.Model
			var err error

			// Safe handler execution with error checking
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("handler panic: %v", r)
						if model.logStore != nil {
							model.logStore.Add("error", "MessageRouter", fmt.Sprintf("🚨 Handler panic: %v", r), true)
						}
					}
				}()
				newModel, cmd = handler.HandleMessage(msg, model)
			}()

			if err != nil {
				if model.logStore != nil {
					model.logStore.Add("error", "MessageRouter", fmt.Sprintf("❌ Handler error: %v", err), true)
				}
				continue // Skip this handler and continue with others
			}

			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if newModel != nil {
				model = newModel.(*Model)
			}
		}
	}

	// Return the updated model and any commands
	return model, tea.Batch(cmds...)
}

// GetMessageType returns the reflect.Type of a message for routing decisions
func GetMessageType(msg tea.Msg) reflect.Type {
	return reflect.TypeOf(msg)
}
