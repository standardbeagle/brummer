package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/components"
)

// CommandAutocompleteAdapter wraps CommandAutocomplete to implement the component interface
type CommandAutocompleteAdapter struct {
	inner *CommandAutocomplete
}

func (a *CommandAutocompleteAdapter) SetWidth(width int) {
	a.inner.SetWidth(width)
}

func (a *CommandAutocompleteAdapter) Focus() {
	a.inner.Focus()
}

func (a *CommandAutocompleteAdapter) View() string {
	return a.inner.View()
}

func (a *CommandAutocompleteAdapter) RenderDropdown(maxSuggestions int) string {
	return a.inner.RenderDropdown(maxSuggestions)
}

func (a *CommandAutocompleteAdapter) GetErrorMessage() string {
	return a.inner.GetErrorMessage()
}


func (a *CommandAutocompleteAdapter) Value() string {
	return a.inner.Value()
}

func (a *CommandAutocompleteAdapter) GetSuggestions() []string {
	return a.inner.GetSuggestions()
}

func (a *CommandAutocompleteAdapter) Update(msg tea.Msg) (components.CommandAutocompleteInterface, tea.Cmd) {
	updated, cmd := a.inner.Update(msg)
	a.inner = &updated
	return a, cmd
}

func (a *CommandAutocompleteAdapter) ValidateInput() (bool, string) {
	return a.inner.ValidateInput()
}

func (a *CommandAutocompleteAdapter) SetError(errMsg string) {
	a.inner.SetError(errMsg)
}

// NewCommandAutocompleteFactory creates a factory for command autocomplete instances
func NewCommandAutocompleteFactory() func(scripts map[string]string, pm *process.Manager) components.CommandAutocompleteInterface {
	return func(scripts map[string]string, pm *process.Manager) components.CommandAutocompleteInterface {
		ca := NewCommandAutocompleteWithProcessManager(scripts, pm)
		return &CommandAutocompleteAdapter{inner: &ca}
	}
}
