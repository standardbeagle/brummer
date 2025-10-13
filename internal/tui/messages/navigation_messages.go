package messages

import (
	"time"
)

// Navigation Messages handles view switching and navigation state communication
// between NavigationService and NavigationComponent

// ViewChangedMsg is sent when the current view changes
type ViewChangedMsg struct {
	FromView  ViewType   `json:"from_view"`
	ToView    ViewType   `json:"to_view"`
	Trigger   string     `json:"trigger"` // "number_key", "tab", "command", etc.
	Time      time.Time  `json:"time"`
}

// ViewSwitchRequestedMsg is sent when a view switch is requested
type ViewSwitchRequestedMsg struct {
	TargetView ViewType   `json:"target_view"`
	Trigger    string     `json:"trigger"`
	Requester  string     `json:"requester"` // Which component initiated the request
	Time       time.Time  `json:"time"`
}

// ViewOrderChangedMsg is sent when the view cycling order is modified
type ViewOrderChangedMsg struct {
	NewOrder  []ViewType `json:"new_order"`
	OldOrder  []ViewType `json:"old_order"`
	Time      time.Time  `json:"time"`
}

// NumberKeyMappingChangedMsg is sent when number key mappings are updated
type NumberKeyMappingChangedMsg struct {
	Number    int        `json:"number"`
	View      ViewType   `json:"view"`
	Action    string     `json:"action"` // "added", "removed", "changed"
	Time      time.Time  `json:"time"`
}

// ViewTransitionCallbackMsg is sent when view transition callbacks are executed
type ViewTransitionCallbackMsg struct {
	FromView    ViewType   `json:"from_view"`
	ToView      ViewType   `json:"to_view"`
	CallbackID  string     `json:"callback_id"`
	Success     bool       `json:"success"`
	Error       string     `json:"error,omitempty"`
	Time        time.Time  `json:"time"`
}

// ViewType represents the different views in the application
type ViewType string

const (
	ProcessesView ViewType = "processes"
	LogsView      ViewType = "logs"
	ErrorsView    ViewType = "errors"
	AICodersView  ViewType = "aicoders"
	SettingsView  ViewType = "settings"
)