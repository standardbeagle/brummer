package messages

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Input Messages handles keyboard input and key binding communication
// between InputService and InputComponent

// InputKeyPressedMsg is sent when a key is pressed and processed by the input service
type InputKeyPressedMsg struct {
	Key      interface{} `json:"key"`
	Context  string      `json:"context"`
	Handled  bool        `json:"handled"`
	Time     time.Time   `json:"time"`
}

// KeyBindingChangedMsg is sent when key bindings are modified
type KeyBindingChangedMsg struct {
	Action   string      `json:"action"` // "added", "removed", "conflict"
	Key      interface{} `json:"key"`
	Binding  interface{} `json:"binding"` // KeyBinding struct from service
	Time     time.Time   `json:"time"`
}

// InputFocusChangedMsg is sent when input focus changes between components
type InputFocusChangedMsg struct {
	FromComponent string    `json:"from_component"`
	ToComponent   string    `json:"to_component"`
	Time          time.Time `json:"time"`
}

// InputHandlerRegisteredMsg is sent when a new input handler is registered
type InputHandlerRegisteredMsg struct {
	HandlerID  string    `json:"handler_id"`
	Component  string    `json:"component"`
	Priority   int       `json:"priority"`
	Time       time.Time `json:"time"`
}

// InputHandlerUnregisteredMsg is sent when an input handler is unregistered
type InputHandlerUnregisteredMsg struct {
	HandlerID string    `json:"handler_id"`
	Component string    `json:"component"`
	Time      time.Time `json:"time"`
}

// InputContextChangedMsg is sent when the input context changes
type InputContextChangedMsg struct {
	OldContext string                    `json:"old_context"`
	NewContext string                    `json:"new_context"`
	Handler    func(tea.KeyMsg) tea.Cmd  `json:"-"` // Not serialized
	Time       time.Time                 `json:"time"`
}

// KeyBindingRegisterMsg is sent to request registration of a key binding
type KeyBindingRegisterMsg struct {
	Key      tea.Key `json:"key"`
	Action   string  `json:"action"`
	Context  string  `json:"context"`
	Priority int     `json:"priority"`
}

// KeyBindingUnregisterMsg is sent to request removal of a key binding
type KeyBindingUnregisterMsg struct {
	Key tea.Key `json:"key"`
}

