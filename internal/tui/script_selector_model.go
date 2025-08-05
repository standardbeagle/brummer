package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ScriptSelectorModel is a Bubble Tea model for the script selector
// It implements the tea.Model interface for proper Bubble Tea integration
type ScriptSelectorModel struct {
	// Input field
	input textinput.Model
	
	// Available scripts
	scripts []string
	
	// Filtered suggestions
	suggestions []string
	
	// Selected suggestion index
	selected int
	
	// Visual state
	width  int
	height int
	
	// Callback for when a script is selected
	onSelect func(script string)
	
	// Callback for cancel
	onCancel func()
}

// NewScriptSelectorModel creates a new script selector model
func NewScriptSelectorModel(scripts []string, onSelect func(string), onCancel func()) ScriptSelectorModel {
	input := textinput.New()
	input.Placeholder = "Type to search scripts..."
	input.Focus()
	input.CharLimit = 50
	
	m := ScriptSelectorModel{
		input:       input,
		scripts:     scripts,
		suggestions: scripts, // Show all initially
		selected:    0,
		onSelect:    onSelect,
		onCancel:    onCancel,
	}
	
	return m
}

// Init initializes the model
func (m ScriptSelectorModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m ScriptSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateInputWidth()
		
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.onCancel != nil {
				m.onCancel()
			}
			return m, nil
			
		case "enter":
			if m.selected >= 0 && m.selected < len(m.suggestions) {
				if m.onSelect != nil {
					m.onSelect(m.suggestions[m.selected])
				}
			}
			return m, nil
			
		case "up", "ctrl+p":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
			
		case "down", "ctrl+n":
			if m.selected < len(m.suggestions)-1 {
				m.selected++
			}
			return m, nil
			
		case "tab":
			// Autocomplete with selected suggestion
			if m.selected >= 0 && m.selected < len(m.suggestions) {
				m.input.SetValue(m.suggestions[m.selected])
			}
			return m, nil
			
		default:
			// Handle text input
			prevValue := m.input.Value()
			m.input, cmd = m.input.Update(msg)
			
			// Update suggestions if value changed
			if m.input.Value() != prevValue {
				m.updateSuggestions()
			}
			return m, cmd
		}
	}
	
	return m, nil
}

// View renders the model
func (m ScriptSelectorModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	
	// Create styles
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Width(m.width).
		Align(lipgloss.Center).
		MarginBottom(1)
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(1)
	
	// Build sections
	title := titleStyle.Render("🐝 Select a Script to Run")
	content := m.renderContent()
	help := helpStyle.Render("↑/↓ Navigate • Enter Run • Tab Complete • Esc Cancel")
	
	// Combine sections
	fullContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		content,
		help,
	)
	
	return containerStyle.Render(fullContent)
}

// renderContent renders the input and dropdown
func (m ScriptSelectorModel) renderContent() string {
	contentWidth := m.width - ScriptSelectorMargin
	if contentWidth < ScriptSelectorMinWidth {
		contentWidth = ScriptSelectorMinWidth
	}
	if contentWidth > ScriptSelectorMaxWidth {
		contentWidth = ScriptSelectorMaxWidth
	}
	
	// Container styles
	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Left)
	
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center)
	
	// Build sections
	sections := []string{
		contentStyle.Render(m.input.View()),
	}
	
	// Add dropdown
	dropdown := m.renderDropdown()
	if dropdown != "" {
		sections = append(sections, contentStyle.Render(dropdown))
	}
	
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return containerStyle.Render(content)
}

// renderDropdown renders the suggestion dropdown
func (m ScriptSelectorModel) renderDropdown() string {
	if len(m.suggestions) == 0 {
		return ""
	}
	
	maxSuggestions := MaxDropdownSuggestions
	if m.height > 0 && m.height < SmallTerminalHeightThreshold {
		maxSuggestions = MaxDropdownSuggestionsSmall
	}
	
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("237"))
	
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	var lines []string
	count := len(m.suggestions)
	if count > maxSuggestions {
		count = maxSuggestions
	}
	
	for i := 0; i < count; i++ {
		line := "  " + m.suggestions[i]
		if i == m.selected {
			line = "▶ " + m.suggestions[i]
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, normalStyle.Render(line))
		}
	}
	
	// Add "more" indicator
	if len(m.suggestions) > maxSuggestions {
		moreStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		moreCount := len(m.suggestions) - maxSuggestions
		lines = append(lines, moreStyle.Render(fmt.Sprintf("  ... and %d more", moreCount)))
	}
	
	return strings.Join(lines, "\n")
}

// updateSuggestions filters scripts based on input
func (m *ScriptSelectorModel) updateSuggestions() {
	input := strings.ToLower(m.input.Value())
	
	if input == "" {
		m.suggestions = m.scripts
	} else {
		m.suggestions = nil
		for _, script := range m.scripts {
			if strings.Contains(strings.ToLower(script), input) {
				m.suggestions = append(m.suggestions, script)
			}
		}
	}
	
	// Reset selection
	m.selected = 0
}

// updateInputWidth updates the input field width based on terminal size
func (m *ScriptSelectorModel) updateInputWidth() {
	width := m.width - ScriptSelectorMargin
	if width < ScriptSelectorMinWidth {
		width = ScriptSelectorMinWidth
	}
	if width > ScriptSelectorMaxWidth {
		width = ScriptSelectorMaxWidth
	}
	m.input.Width = width - 4 // Account for borders/padding
}