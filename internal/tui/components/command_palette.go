package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandHandler handles command execution
type CommandHandler interface {
	ExecuteCommand(command string) tea.Cmd
	GetSuggestions(prefix string) []string
}

// CommandPalette is an overlay for command input
type CommandPalette struct {
	Window

	// Input field
	input textinput.Model

	// Command handling
	handler CommandHandler

	// Suggestions
	suggestions []string
	selectedIdx int

	// Styling
	inputStyle      lipgloss.Style
	suggestionStyle lipgloss.Style
	selectedStyle   lipgloss.Style
}

// NewCommandPalette creates a new command palette
func NewCommandPalette(handler CommandHandler) *CommandPalette {
	input := textinput.New()
	input.Placeholder = "Type a command..."
	input.CharLimit = 256
	input.Width = 50

	// Create content wrapper
	palette := &CommandPalette{
		Window:          *NewWindow("Command", nil), // We'll render content directly
		input:           input,
		handler:         handler,
		suggestions:     []string{},
		selectedIdx:     0,
		inputStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		suggestionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		selectedStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
	}

	// Configure window
	palette.SetModal(true) // Modal to capture input
	palette.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86"))

	// Set a reasonable size
	palette.SetSize(60, 20)

	return palette
}

// Show shows the command palette and focuses input
func (c *CommandPalette) Show() {
	c.Window.Show()
	c.input.Focus()
	// Initialize with slash to indicate slash-command mode and populate suggestions
	c.input.SetValue("/")
	if c.handler != nil {
		c.suggestions = c.handler.GetSuggestions(c.input.Value())
	} else {
		c.suggestions = []string{}
	}
	c.selectedIdx = 0
}

// Hide hides the palette and clears input
func (c *CommandPalette) Hide() {
	c.Window.Hide()
	c.input.Blur()
	c.input.SetValue("")
	c.suggestions = []string{}
}

// Init implements tea.Model interface
func (c *CommandPalette) Init() tea.Cmd {
	return nil
}

// Update handles command palette updates
func (c *CommandPalette) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.Hide()
			return c, nil

		case "enter":
			// Execute command
			if c.input.Value() != "" {
				if c.handler != nil {
					cmd := c.handler.ExecuteCommand(c.input.Value())
					c.Hide()
					return c, cmd
				}
			}
			return c, nil

		case "tab":
			// Accept current suggestion (do not advance)
			if len(c.suggestions) > 0 {
				val := parseSuggestionCommand(c.suggestions[c.selectedIdx])
				if val != "" {
					c.input.SetValue(val)
				} else {
					c.input.SetValue(c.suggestions[c.selectedIdx])
				}
			}
			return c, nil

		case "down":
			// Next suggestion (preview)
			if len(c.suggestions) > 0 {
				c.selectedIdx = (c.selectedIdx + 1) % len(c.suggestions)
				val := parseSuggestionCommand(c.suggestions[c.selectedIdx])
				if val != "" {
					c.input.SetValue(val)
				} else {
					c.input.SetValue(c.suggestions[c.selectedIdx])
				}
			}
			return c, nil

		case "shift+tab", "up":
			// Previous suggestion (preview)
			if len(c.suggestions) > 0 {
				c.selectedIdx--
				if c.selectedIdx < 0 {
					c.selectedIdx = len(c.suggestions) - 1
				}
				val := parseSuggestionCommand(c.suggestions[c.selectedIdx])
				if val != "" {
					c.input.SetValue(val)
				} else {
					c.input.SetValue(c.suggestions[c.selectedIdx])
				}
			}
			return c, nil
		}
	}

	// Update input
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	cmds = append(cmds, cmd)

	// Update suggestions based on input
	if c.handler != nil && c.input.Value() != "" {
		c.suggestions = c.handler.GetSuggestions(c.input.Value())
		if c.selectedIdx >= len(c.suggestions) {
			c.selectedIdx = 0
		}
		// Proactive inline completion when exactly one unambiguous suggestion
		// Do not auto-expand if the current input ends with a space (e.g., "/ai ")
		if len(c.suggestions) == 1 {
			val := c.input.Value()
			if !strings.HasSuffix(val, " ") {
				cand := parseSuggestionCommand(c.suggestions[0])
				if cand != "" && strings.HasPrefix(cand, val) && cand != val {
					c.input.SetValue(cand)
				}
			}
		}
	} else {
		c.suggestions = []string{}
	}

	return c, tea.Batch(cmds...)
}

// View renders the command palette
func (c *CommandPalette) View() string {
	if !c.visible {
		return ""
	}

	var content strings.Builder

	// Render input field
	content.WriteString(c.inputStyle.Render("> "))
	content.WriteString(c.input.View())
	content.WriteString("\n")

	// Render suggestions
	if len(c.suggestions) > 0 {
		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Suggestions:"))
		content.WriteString("\n")

		for i, suggestion := range c.suggestions {
			if i >= 10 {
				break // Limit to 10 suggestions
			}

			if i == c.selectedIdx {
				content.WriteString(c.selectedStyle.Render("  → " + suggestion))
			} else {
				content.WriteString(c.suggestionStyle.Render("    " + suggestion))
			}
			content.WriteString("\n")
		}
	}

	// Apply window styling
	styled := c.borderStyle.
		Width(c.width).
		Padding(1, 2).
		Render(content.String())

	// Center in screen
	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Center,
		styled,
	)
}

// parseSuggestionCommand extracts the raw command part from a suggestion line like
// "/ai terminal - Start AI Coder with terminal" -> "/ai terminal"
func parseSuggestionCommand(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, " - ", 2)
	return strings.TrimSpace(parts[0])
}

// Focus focuses the input
func (c *CommandPalette) Focus() tea.Cmd {
	c.focused = true
	return c.input.Focus()
}

// Blur blurs the input
func (c *CommandPalette) Blur() tea.Cmd {
	c.focused = false
	c.input.Blur()
	return nil
}
