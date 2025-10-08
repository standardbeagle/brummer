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

func (a *ScriptSelectorAutocompleteAdapter) IsShowingDropdown() bool {
	return a.autocomplete.IsShowingDropdown()
}

func (a *ScriptSelectorAutocompleteAdapter) SetShowDropdown(show bool) {
	a.autocomplete.SetShowDropdown(show)
}

func (a *ScriptSelectorAutocompleteAdapter) GetErrorMessage() string {
	return a.autocomplete.GetErrorMessage()
}

func (a *ScriptSelectorAutocompleteAdapter) SetErrorMessage(msg string) {
	a.autocomplete.SetErrorMessage(msg)
}

func (a *ScriptSelectorAutocompleteAdapter) IsArbitraryMode() bool {
	return a.autocomplete.IsArbitraryMode()
}

func (a *ScriptSelectorAutocompleteAdapter) SetArbitraryMode(mode bool) {
	a.autocomplete.SetArbitraryMode(mode)
}

func (a *ScriptSelectorAutocompleteAdapter) UpdateSuggestions() {
	a.autocomplete.UpdateSuggestions()
}

func (a *ScriptSelectorAutocompleteAdapter) GetAvailableScripts() map[string]string {
	return a.autocomplete.GetAvailableScripts()
}
