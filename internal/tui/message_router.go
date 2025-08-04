package tui

import (
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
func NewMessageRouter() *MessageRouter {
	router := &MessageRouter{
		handlers: make([]MessageHandler, 0),
	}

	// Register handlers in order of priority
	router.RegisterHandler(NewDialogMessageHandler()) // Handle dialogs first (highest priority)
	router.RegisterHandler(NewSystemMessageHandler())
	router.RegisterHandler(NewProcessMessageHandler())
	router.RegisterHandler(NewLogMessageHandler())
	router.RegisterHandler(NewMCPMessageHandler())
	router.RegisterHandler(NewAICoderMessageHandler())
	router.RegisterHandler(NewViewSpecificHandler()) // View-specific keyboard interactions
	router.RegisterHandler(NewViewMessageHandler())  // General view messages (lowest priority)

	return router
}

// RegisterHandler adds a message handler to the router
func (r *MessageRouter) RegisterHandler(handler MessageHandler) {
	r.handlers = append(r.handlers, handler)
}

// Route dispatches a message to the appropriate handler
func (r *MessageRouter) Route(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Try each handler until one can handle the message
	for _, handler := range r.handlers {
		if handler.CanHandle(msg) {
			newModel, cmd := handler.HandleMessage(msg, model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			model = newModel.(*Model)
		}
	}

	// Return the updated model and any commands
	return model, tea.Batch(cmds...)
}

// GetMessageType returns the reflect.Type of a message for routing decisions
func GetMessageType(msg tea.Msg) reflect.Type {
	return reflect.TypeOf(msg)
}
