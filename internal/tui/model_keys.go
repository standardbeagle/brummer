package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap holds all keyboard shortcuts for the TUI
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	Back          key.Binding
	Quit          key.Binding
	Tab           key.Binding
	Left          key.Binding
	Right         key.Binding
	Command       key.Binding
	Filter        key.Binding
	Stop          key.Binding
	Restart       key.Binding
	RestartAll    key.Binding
	CopyError     key.Binding
	Priority      key.Binding
	ClearLogs     key.Binding
	ClearErrors   key.Binding
	ClearScreen   key.Binding
	Help          key.Binding
	RunDialog     key.Binding
	AutoScroll    key.Binding
	ToggleError   key.Binding
	ClearMessages key.Binding
	TestView      key.Binding
}

// DefaultKeyMap returns the default keyboard configuration
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch view"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "previous view"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "next view"),
		),
		Command: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "command palette"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filters"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop process"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart process"),
		),
		RestartAll: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "restart all"),
		),
		CopyError: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy recent error"),
		),
		Priority: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "toggle priority"),
		),
		ClearLogs: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "clear logs"),
		),
		ClearErrors: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "clear errors"),
		),
		ClearScreen: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear screen"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		RunDialog: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new process"),
		),
		AutoScroll: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "auto-scroll"),
		),
		ToggleError: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "toggle system messages"),
		),
		ClearMessages: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "clear system messages"),
		),
		TestView: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "test view"),
		),
	}
}

// ShortHelp returns the essential key bindings for help
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Back, k.Help, k.Quit}
}

// FullHelp returns all key bindings organized by category
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Tab, k.Command, k.Filter, k.Priority},
		{k.Stop, k.Restart, k.RestartAll, k.CopyError},
		{k.ClearLogs, k.ClearErrors, k.ToggleError, k.ClearMessages, k.TestView, k.Help, k.Quit},
	}
}