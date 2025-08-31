package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PaletteAdapter adapts the new interfaces to the existing CommandHandler interface
type PaletteAdapter struct {
	provider SuggestionProvider
	executor CommandExecutor
}

// NewPaletteAdapter creates an adapter for CommandPalette
func NewPaletteAdapter(provider SuggestionProvider, executor CommandExecutor) *PaletteAdapter {
	return &PaletteAdapter{
		provider: provider,
		executor: executor,
	}
}

// ExecuteCommand implements CommandHandler interface
func (a *PaletteAdapter) ExecuteCommand(command string) tea.Cmd {
	if a.executor == nil {
		return nil
	}
	return a.executor.Execute(command)
}

// GetSuggestions implements CommandHandler interface
func (a *PaletteAdapter) GetSuggestions(prefix string) []string {
	if a.provider == nil {
		return []string{}
	}
	
	suggestions := a.provider.Suggest(prefix)
	result := make([]string, 0, len(suggestions))
	
	for _, s := range suggestions {
		// Include command and description for display
		if s.Description != "" {
			result = append(result, s.Command+" - "+s.Description)
		} else {
			result = append(result, s.Command)
		}
	}
	
	return result
}