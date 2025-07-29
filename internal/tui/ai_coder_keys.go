package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// AICoderKeyMap defines keyboard shortcuts for AI coder operations
type AICoderKeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Focus management
	SwitchFocus   key.Binding
	ToggleDetails key.Binding

	// AI Coder operations
	NewCoder      key.Binding
	DeleteCoder   key.Binding
	StartCoder    key.Binding
	PauseCoder    key.Binding
	ResumeCoder   key.Binding
	StopCoder     key.Binding
	RestartCoder  key.Binding
	SendCommand   key.Binding
	ViewWorkspace key.Binding
	ViewLogs      key.Binding
	ClearOutput   key.Binding

	// Selection and interaction
	Select       key.Binding
	Filter       key.Binding
	ClearFilter  key.Binding
	Refresh      key.Binding
	CopyID       key.Binding
	CopyWorkPath key.Binding

	// Help and navigation
	Help   key.Binding
	Escape key.Binding
	Quit   key.Binding
}

// NewAICoderKeyMap returns default key bindings for AI coder operations
func NewAICoderKeyMap() AICoderKeyMap {
	return AICoderKeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup/ctrl+u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn/ctrl+d", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),

		// Focus management
		SwitchFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch focus"),
		),
		ToggleDetails: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle details panel"),
		),

		// AI Coder operations
		NewCoder: key.NewBinding(
			key.WithKeys("n", "ctrl+n"),
			key.WithHelp("n/ctrl+n", "new AI coder"),
		),
		DeleteCoder: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d/del", "delete AI coder"),
		),
		StartCoder: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start AI coder"),
		),
		PauseCoder: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pause AI coder"),
		),
		ResumeCoder: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "resume AI coder"),
		),
		StopCoder: key.NewBinding(
			key.WithKeys("x", "ctrl+c"),
			key.WithHelp("x/ctrl+c", "stop AI coder"),
		),
		RestartCoder: key.NewBinding(
			key.WithKeys("R", "ctrl+r"),
			key.WithHelp("R/ctrl+r", "restart AI coder"),
		),
		SendCommand: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "send command"),
		),
		ViewWorkspace: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "view workspace"),
		),
		ViewLogs: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "view logs"),
		),
		ClearOutput: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "clear output"),
		),

		// Selection and interaction
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter list"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear filter"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+f", "f5"),
			key.WithHelp("ctrl+f/F5", "refresh"),
		),
		CopyID: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy coder ID"),
		),
		CopyWorkPath: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "copy workspace path"),
		),

		// Help and navigation
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+q"),
			key.WithHelp("q/ctrl+q", "quit"),
		),
	}
}

// ShortHelp returns a slice of key bindings for the short help view
func (k AICoderKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NewCoder,
		k.StartCoder,
		k.PauseCoder,
		k.DeleteCoder,
		k.SendCommand,
		k.Help,
	}
}

// FullHelp returns a slice of key bindings for the full help view
func (k AICoderKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Navigation
		{
			k.Up,
			k.Down,
			k.PageUp,
			k.PageDown,
			k.Home,
			k.End,
		},
		// AI Coder operations
		{
			k.NewCoder,
			k.StartCoder,
			k.PauseCoder,
			k.ResumeCoder,
			k.StopCoder,
			k.DeleteCoder,
		},
		// Interaction
		{
			k.Select,
			k.SendCommand,
			k.ViewWorkspace,
			k.ViewLogs,
			k.ClearOutput,
			k.Refresh,
		},
		// Focus and view
		{
			k.SwitchFocus,
			k.ToggleDetails,
			k.Filter,
			k.ClearFilter,
			k.Help,
			k.Escape,
		},
	}
}

// HandleKeyPress processes key events for AI coder operations
func HandleAICoderKey(msg tea.KeyMsg, keyMap AICoderKeyMap, view *AICoderView) tea.Cmd {
	switch {
	// Navigation keys are handled by list/viewport components

	// AI Coder operations - removed NewCoder key (use /ai command instead)

	case key.Matches(msg, keyMap.DeleteCoder):
		if view.focusMode == FocusList && view.selectedCoder != nil {
			return view.deleteCoder(view.selectedCoder.ID)
		}

	case key.Matches(msg, keyMap.StartCoder):
		if view.focusMode == FocusList && view.selectedCoder != nil {
			return view.startCoder(view.selectedCoder.ID)
		}

	case key.Matches(msg, keyMap.PauseCoder):
		if view.focusMode == FocusList && view.selectedCoder != nil {
			return view.pauseCoder(view.selectedCoder.ID)
		}

	case key.Matches(msg, keyMap.ResumeCoder):
		if view.focusMode == FocusList && view.selectedCoder != nil {
			return view.resumeCoder(view.selectedCoder.ID)
		}

	case key.Matches(msg, keyMap.SendCommand):
		if view.focusMode == FocusList && view.selectedCoder != nil {
			view.commandMode = true
			view.commandInput.SetValue("")
			view.commandInput.Placeholder = "Enter command for AI coder..."
			view.commandInput.Focus()
			return textinput.Blink
		}

	case key.Matches(msg, keyMap.SwitchFocus):
		if view.showDetails {
			view.focusMode = (view.focusMode + 1) % 2 // Toggle between list and detail
		}

	case key.Matches(msg, keyMap.ToggleDetails):
		view.showDetails = !view.showDetails
		if !view.showDetails {
			view.focusMode = FocusList
		}
		view.updateLayout()

	case key.Matches(msg, keyMap.Refresh):
		return view.refreshCoders()

	case key.Matches(msg, keyMap.Escape):
		if view.commandMode {
			view.commandMode = false
			view.commandInput.Blur()
			view.commandInput.SetValue("")
		}
	}

	return nil
}

// aiCoderKeyMap is the global key map instance for AI coder operations
var aiCoderKeyMap = NewAICoderKeyMap()