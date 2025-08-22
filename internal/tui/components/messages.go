package components

import (
	"fmt"
	
	tea "github.com/charmbracelet/bubbletea"
)

// Component Communication Messages
// These messages enable components to communicate without tight coupling

// FocusMsg requests focus for a component
type FocusMsg struct {
	ComponentID string
}

// BlurMsg removes focus from a component
type BlurMsg struct {
	ComponentID string
}

// ShowMsg shows a component (typically for overlays/windows)
type ShowMsg struct {
	ComponentID string
}

// HideMsg hides a component
type HideMsg struct {
	ComponentID string
}

// ToggleMsg toggles component visibility
type ToggleMsg struct {
	ComponentID string
}

// ResizeMsg notifies components of size changes
// (Separate from tea.WindowSizeMsg for component-specific sizing)
type ResizeMsg struct {
	Width  int
	Height int
	ComponentID string // Optional: target specific component
}

// ViewChangeMsg requests switching to a different view
type ViewChangeMsg struct {
	ViewID string
	Data   interface{} // Optional data to pass to the view
}

// OverlayMsg manages overlay components
type OverlayMsg struct {
	Action  OverlayAction
	Overlay Component
	ID      string
}

type OverlayAction int

const (
	OverlayShow OverlayAction = iota
	OverlayHide
	OverlayToggle
	OverlayRemove
)

// DataUpdateMsg carries data updates between components
type DataUpdateMsg struct {
	Type   string      // Type of data update
	Data   interface{} // The actual data
	Source string      // Component that sent the update
	Target string      // Optional target component
}

// Common data update types
const (
	DataUpdateProcess   = "process"
	DataUpdateLogs      = "logs"
	DataUpdateErrors    = "errors"
	DataUpdateURLs      = "urls"
	DataUpdateSettings  = "settings"
	DataUpdateSelection = "selection"
)

// CommandMsg represents a command to execute
type CommandMsg struct {
	Command string
	Args    []string
	Source  string // Component that issued the command
}

// NotificationMsg displays a notification
type NotificationMsg struct {
	Level   NotificationLevel
	Title   string
	Message string
	Actions []NotificationAction
}

type NotificationLevel int

const (
	NotificationInfo NotificationLevel = iota
	NotificationWarning
	NotificationError
	NotificationSuccess
)

type NotificationAction struct {
	Label   string
	Command tea.Cmd
}

// SelectionMsg notifies of selection changes
type SelectionMsg struct {
	ComponentID string
	Selected    interface{} // The selected item
	Index       int         // Index of selection
}

// ScrollMsg requests scrolling
type ScrollMsg struct {
	ComponentID string
	Direction   ScrollDirection
	Amount      int // Lines or pages to scroll
}

type ScrollDirection int

const (
	ScrollUp ScrollDirection = iota
	ScrollDown
	ScrollLeft
	ScrollRight
	ScrollPageUp
	ScrollPageDown
	ScrollHome
	ScrollEnd
)

// ModalMsg manages modal state
type ModalMsg struct {
	ComponentID string
	Modal       bool
}

// StateChangeMsg notifies of component state changes
type StateChangeMsg struct {
	ComponentID string
	OldState    interface{}
	NewState    interface{}
}

// RequestDataMsg requests data from a source
type RequestDataMsg struct {
	DataType string
	Source   string // Component requesting
	Filters  map[string]interface{}
}

// ResponseDataMsg responds with requested data
type ResponseDataMsg struct {
	RequestID string
	DataType  string
	Data      interface{}
	Error     error
}

// ProgressMsg updates progress indicators
type ProgressMsg struct {
	ComponentID string
	Percent     float64
	Message     string
	Complete    bool
}

// TabNavigationMsg handles tab navigation between components
type TabNavigationMsg struct {
	Direction TabDirection
}

type TabDirection int

const (
	TabForward TabDirection = iota
	TabBackward
)

// ComponentEventMsg wraps component-specific events
type ComponentEventMsg struct {
	ComponentID string
	Event       interface{}
}

// LifecycleMsg notifies of component lifecycle events
type LifecycleMsg struct {
	ComponentID string
	Event       LifecycleEvent
}

type LifecycleEvent int

const (
	LifecycleCreated LifecycleEvent = iota
	LifecycleInitialized
	LifecycleShown
	LifecycleHidden
	LifecycleDestroyed
)

// ThemeChangeMsg notifies of theme changes
type ThemeChangeMsg struct {
	Theme string
}

// ContextMenuMsg manages context menus
type ContextMenuMsg struct {
	ComponentID string
	X, Y        int
	Items       []ContextMenuItem
}

type ContextMenuItem struct {
	Label    string
	Command  tea.Cmd
	Disabled bool
	Submenu  []ContextMenuItem
}

// DragDropMsg handles drag and drop operations
type DragDropMsg struct {
	Action   DragDropAction
	Source   string
	Target   string
	Data     interface{}
	X, Y     int
}

type DragDropAction int

const (
	DragStart DragDropAction = iota
	DragOver
	DragDrop
	DragEnd
	DragCancel
)

// ValidationMsg carries validation results
type ValidationMsg struct {
	ComponentID string
	Valid       bool
	Errors      []string
	Warnings    []string
}

// Message Helpers

// SendFocus creates a command to send a focus message
func SendFocus(componentID string) tea.Cmd {
	return func() tea.Msg {
		return FocusMsg{ComponentID: componentID}
	}
}

// SendBlur creates a command to send a blur message
func SendBlur(componentID string) tea.Cmd {
	return func() tea.Msg {
		return BlurMsg{ComponentID: componentID}
	}
}

// SendShow creates a command to show a component
func SendShow(componentID string) tea.Cmd {
	return func() tea.Msg {
		return ShowMsg{ComponentID: componentID}
	}
}

// SendHide creates a command to hide a component
func SendHide(componentID string) tea.Cmd {
	return func() tea.Msg {
		return HideMsg{ComponentID: componentID}
	}
}

// SendDataUpdate creates a command to send a data update
func SendDataUpdate(dataType string, data interface{}, source string) tea.Cmd {
	return func() tea.Msg {
		return DataUpdateMsg{
			Type:   dataType,
			Data:   data,
			Source: source,
		}
	}
}

// SendNotification creates a command to send a notification
func SendNotification(level NotificationLevel, title, message string) tea.Cmd {
	return func() tea.Msg {
		return NotificationMsg{
			Level:   level,
			Title:   title,
			Message: message,
		}
	}
}

// SendProgress creates a command to update progress
func SendProgress(componentID string, percent float64, message string) tea.Cmd {
	return func() tea.Msg {
		return ProgressMsg{
			ComponentID: componentID,
			Percent:     percent,
			Message:     message,
			Complete:    percent >= 1.0,
		}
	}
}

// SendCommand creates a command to execute
func SendCommand(command string, args ...string) tea.Cmd {
	return func() tea.Msg {
		return CommandMsg{
			Command: command,
			Args:    args,
		}
	}
}

// SendViewChange creates a command to change views
func SendViewChange(viewID string, data interface{}) tea.Cmd {
	return func() tea.Msg {
		return ViewChangeMsg{
			ViewID: viewID,
			Data:   data,
		}
	}
}

// BatchCommands combines multiple commands
func BatchCommands(cmds ...tea.Cmd) tea.Cmd {
	return tea.Batch(cmds...)
}

// Message Router Interface

// MessageRouter routes messages between components
type MessageRouter interface {
	// Route sends a message to a specific component
	Route(componentID string, msg tea.Msg) tea.Cmd
	
	// Broadcast sends a message to all components
	Broadcast(msg tea.Msg) tea.Cmd
	
	// Subscribe registers interest in specific message types
	Subscribe(componentID string, msgType interface{})
	
	// Unsubscribe removes interest in message types
	Unsubscribe(componentID string, msgType interface{})
}

// SimpleMessageRouter provides basic message routing
type SimpleMessageRouter struct {
	components   map[string]Component
	subscriptions map[string][]interface{}
}

// NewMessageRouter creates a new message router
func NewMessageRouter() *SimpleMessageRouter {
	return &SimpleMessageRouter{
		components:    make(map[string]Component),
		subscriptions: make(map[string][]interface{}),
	}
}

// Register registers a component with the router
func (r *SimpleMessageRouter) Register(id string, component Component) {
	r.components[id] = component
}

// Route sends a message to a specific component
func (r *SimpleMessageRouter) Route(componentID string, msg tea.Msg) tea.Cmd {
	if component, ok := r.components[componentID]; ok {
		_, cmd := component.Update(msg)
		return cmd
	}
	return nil
}

// Broadcast sends a message to all components
func (r *SimpleMessageRouter) Broadcast(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for _, component := range r.components {
		_, cmd := component.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Subscribe registers interest in message types
func (r *SimpleMessageRouter) Subscribe(componentID string, msgType interface{}) {
	r.subscriptions[componentID] = append(r.subscriptions[componentID], msgType)
}

// Unsubscribe removes interest in message types
func (r *SimpleMessageRouter) Unsubscribe(componentID string, msgType interface{}) {
	// Implementation would filter out the msgType from subscriptions
}

// Message Filters

// FilterByComponent filters messages for a specific component
func FilterByComponent(componentID string) func(tea.Msg) bool {
	return func(msg tea.Msg) bool {
		switch m := msg.(type) {
		case FocusMsg:
			return m.ComponentID == componentID
		case BlurMsg:
			return m.ComponentID == componentID
		case ShowMsg:
			return m.ComponentID == componentID
		case HideMsg:
			return m.ComponentID == componentID
		case ResizeMsg:
			return m.ComponentID == "" || m.ComponentID == componentID
		case DataUpdateMsg:
			return m.Target == "" || m.Target == componentID
		default:
			return true // Let other messages through
		}
	}
}

// FilterByType filters messages by type
func FilterByType(msgTypes ...interface{}) func(tea.Msg) bool {
	return func(msg tea.Msg) bool {
		msgType := fmt.Sprintf("%T", msg)
		for _, t := range msgTypes {
			if fmt.Sprintf("%T", t) == msgType {
				return true
			}
		}
		return false
	}
}