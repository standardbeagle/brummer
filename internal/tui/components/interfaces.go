package components

import tea "github.com/charmbracelet/bubbletea"

// SuggestionProvider provides command suggestions based on prefix matching
type SuggestionProvider interface {
	// Suggest returns command suggestions matching the given prefix
	Suggest(prefix string) []CommandSuggestion
}

// CommandExecutor executes commands and returns BubbleTea commands
type CommandExecutor interface {
	// Execute runs the command and returns a tea.Cmd for state updates
	Execute(command string) tea.Cmd
}

// CommandSuggestion is defined in custom_messages.go
