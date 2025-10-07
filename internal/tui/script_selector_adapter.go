package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/components"
)

// ScriptSelectorAutocompleteAdapter adapts CommandAutocomplete to ScriptSelectorAutocompleteInterface
type ScriptSelectorAutocompleteAdapter struct {
	autocomplete *CommandAutocomplete
}

// NewScriptSelectorAutocompleteAdapter creates a new adapter
func NewScriptSelectorAutocompleteAdapter(autocomplete CommandAutocomplete) components.ScriptSelectorAutocompleteInterface {
	return &ScriptSelectorAutocompleteAdapter{autocomplete: &autocomplete}
}

func (a *ScriptSelectorAutocompleteAdapter) Focus() {
	a.autocomplete.Focus()
}

func (a *ScriptSelectorAutocompleteAdapter) SetWidth(width int) {
	a.autocomplete.SetWidth(width)
}

func (a *ScriptSelectorAutocompleteAdapter) View() string {
	return a.autocomplete.View()
}

func (a *ScriptSelectorAutocompleteAdapter) RenderDropdown(maxSuggestions int) string {
	return a.autocomplete.RenderDropdown(maxSuggestions)
}

func (a *ScriptSelectorAutocompleteAdapter) Update(msg tea.Msg) (components.ScriptSelectorAutocompleteInterface, tea.Cmd) {
	updated, cmd := a.autocomplete.Update(msg)
	a.autocomplete = &updated
	return a, cmd
}

func (a *ScriptSelectorAutocompleteAdapter) Value() string {
	return a.autocomplete.Value()
}

func (a *ScriptSelectorAutocompleteAdapter) SetValue(value string) {
	a.autocomplete.SetValue(value)
}

func (a *ScriptSelectorAutocompleteAdapter) SetPlaceholder(placeholder string) {
	a.autocomplete.SetPlaceholder(placeholder)
}

func (a *ScriptSelectorAutocompleteAdapter) GetSelected() int {
	return a.autocomplete.GetSelected()
}

func (a *ScriptSelectorAutocompleteAdapter) SetSelected(index int) {
	a.autocomplete.SetSelected(index)
}

func (a *ScriptSelectorAutocompleteAdapter) GetSuggestions() []string {
	return a.autocomplete.GetSuggestions()
}

func (a *ScriptSelectorAutocompleteAdapter) SetSuggestions(suggestions []string) {
	a.autocomplete.SetSuggestions(suggestions)
}
