package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// CommandInputComponent provides a command-line style input with autocomplete
// Activated with "/" key, supports Tab completion, Enter to execute, Esc to cancel
type CommandInputComponent struct {
	BaseComponent

	input           textinput.Model
	registry        *CommandRegistry
	active          bool
	suggestions     []string
	selectedSuggestion int
	showSuggestions bool
	currentCommand  *messages.Command
	argumentIndex   int // Which argument we're currently filling
}

// NewCommandInputComponent creates a new command input with autocomplete
func NewCommandInputComponent(registry *CommandRegistry) *CommandInputComponent {
	input := textinput.New()
	input.Placeholder = "Enter command..."
	input.Prompt = "/"
	input.CharLimit = 200

	return &CommandInputComponent{
		registry:        registry,
		input:           input,
		selectedSuggestion: 0,
	}
}

// IsActive returns whether the command input is currently active
func (c *CommandInputComponent) IsActive() bool {
	return c.active
}

// Open activates the command input
func (c *CommandInputComponent) Open() {
	c.active = true
	c.input.Reset()
	c.input.Focus()
	c.updateSuggestions()
}

// Close deactivates the command input
func (c *CommandInputComponent) Close() {
	c.active = false
	c.input.Blur()
	c.input.Reset()
	c.suggestions = nil
	c.selectedSuggestion = 0
	c.showSuggestions = false
	c.currentCommand = nil
	c.argumentIndex = 0
}

// Init initializes the component
func (c *CommandInputComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *CommandInputComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !c.active {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.Close()
			return c, nil

		case "enter":
			// Execute command
			value := strings.TrimSpace(c.input.Value())
			if value != "" {
				cmd := c.parseCommand(value)
				c.Close()
				return c, func() tea.Msg {
					return cmd
				}
			}
			c.Close()
			return c, nil

		case "tab":
			// Apply selected suggestion
			if c.showSuggestions && len(c.suggestions) > 0 {
				c.applySuggestion()
				c.updateSuggestions()
			}
			return c, nil

		case "up":
			// Navigate suggestions
			if c.showSuggestions && c.selectedSuggestion > 0 {
				c.selectedSuggestion--
			}
			return c, nil

		case "down":
			// Navigate suggestions
			if c.showSuggestions && c.selectedSuggestion < len(c.suggestions)-1 {
				c.selectedSuggestion++
			}
			return c, nil

		default:
			// Update input and refresh suggestions
			var cmd tea.Cmd
			c.input, cmd = c.input.Update(msg)
			c.updateSuggestions()
			return c, cmd
		}
	}

	return c, nil
}

// View renders the command input
func (c *CommandInputComponent) View() string {
	if !c.active {
		return ""
	}

	// Calculate dimensions for centering
	inputWidth := 60
	if c.width > 0 && c.width < inputWidth+10 {
		inputWidth = c.width - 10
	}

	// Input box
	inputStyle := lipgloss.NewStyle().
		Width(inputWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	inputBox := inputStyle.Render(c.input.View())

	// Suggestions dropdown (if any)
	var suggestionsBox string
	if c.showSuggestions && len(c.suggestions) > 0 {
		var lines []string
		maxSuggestions := 8
		for i, suggestion := range c.suggestions {
			if i >= maxSuggestions {
				break
			}

			prefix := "  "
			if i == c.selectedSuggestion {
				prefix = "► "
			}

			lines = append(lines, prefix+suggestion)
		}

		suggestionsStyle := lipgloss.NewStyle().
			Width(inputWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

		suggestionsBox = "\n" + suggestionsStyle.Render(strings.Join(lines, "\n"))
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(inputWidth).
		Align(lipgloss.Center)

	helpText := helpStyle.Render("Tab: autocomplete | Enter: execute | Esc: cancel")

	// Combine all parts
	content := inputBox + suggestionsBox + "\n" + helpText

	// Center on screen
	containerStyle := lipgloss.NewStyle().
		Width(c.width).
		Height(c.height).
		Align(lipgloss.Center, lipgloss.Center)

	return containerStyle.Render(content)
}

// parseCommand parses the input into a CommandExecuteMsg
func (c *CommandInputComponent) parseCommand(input string) messages.CommandExecuteMsg {
	// Remove leading slash if present
	input = strings.TrimPrefix(input, "/")

	// Split into command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return messages.CommandExecuteMsg{}
	}

	commandName := parts[0]
	arguments := parts[1:]

	return messages.CommandExecuteMsg{
		CommandName: commandName,
		Arguments:   arguments,
	}
}

// updateSuggestions updates the autocomplete suggestions based on current input
func (c *CommandInputComponent) updateSuggestions() {
	value := strings.TrimSpace(c.input.Value())

	// Remove leading slash for parsing
	value = strings.TrimPrefix(value, "/")

	parts := strings.Fields(value)

	if len(parts) == 0 {
		// Show all commands
		c.suggestions = c.registry.GetNames()
		c.showSuggestions = true
		c.selectedSuggestion = 0
		c.currentCommand = nil
		c.argumentIndex = 0
		return
	}

	commandName := parts[0]

	// If we're still typing the command name (no space after)
	if !strings.HasSuffix(c.input.Value(), " ") && len(parts) == 1 {
		// Filter command names
		c.suggestions = c.filterCommands(commandName)
		c.showSuggestions = len(c.suggestions) > 0
		c.selectedSuggestion = 0
		c.currentCommand = nil
		c.argumentIndex = 0
		return
	}

	// We have a complete command name, now suggest arguments
	if cmd, ok := c.registry.Get(commandName); ok {
		c.currentCommand = &cmd

		// Calculate which argument we're currently filling
		// If input ends with space, we're ready for next argument
		// Otherwise, we're still typing the current argument
		if strings.HasSuffix(c.input.Value(), " ") {
			c.argumentIndex = len(parts) - 1
		} else {
			c.argumentIndex = len(parts) - 2
			if c.argumentIndex < 0 {
				c.argumentIndex = 0
			}
		}

		// Get suggestions for current argument
		if c.argumentIndex < len(cmd.Arguments) {
			argSuggestions := cmd.Arguments[c.argumentIndex].Suggestions
			if len(argSuggestions) > 0 {
				// Filter suggestions based on current partial input
				currentArg := ""
				if len(parts) > 1 && !strings.HasSuffix(c.input.Value(), " ") {
					currentArg = parts[len(parts)-1]
				}

				c.suggestions = c.filterSuggestions(argSuggestions, currentArg)
				c.showSuggestions = len(c.suggestions) > 0
				c.selectedSuggestion = 0
				return
			}
		}
	}

	// No suggestions available
	c.suggestions = nil
	c.showSuggestions = false
	c.selectedSuggestion = 0
}

// filterCommands returns command names matching the prefix
func (c *CommandInputComponent) filterCommands(prefix string) []string {
	allCommands := c.registry.GetNames()
	if prefix == "" {
		return allCommands
	}

	var filtered []string
	lowerPrefix := strings.ToLower(prefix)
	for _, cmd := range allCommands {
		if strings.HasPrefix(strings.ToLower(cmd), lowerPrefix) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// filterSuggestions returns suggestions matching the prefix
func (c *CommandInputComponent) filterSuggestions(suggestions []string, prefix string) []string {
	if prefix == "" {
		return suggestions
	}

	var filtered []string
	lowerPrefix := strings.ToLower(prefix)
	for _, suggestion := range suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion), lowerPrefix) {
			filtered = append(filtered, suggestion)
		}
	}
	return filtered
}

// applySuggestion applies the currently selected suggestion
func (c *CommandInputComponent) applySuggestion() {
	if len(c.suggestions) == 0 || c.selectedSuggestion >= len(c.suggestions) {
		return
	}

	suggestion := c.suggestions[c.selectedSuggestion]
	value := strings.TrimSpace(c.input.Value())
	value = strings.TrimPrefix(value, "/")
	parts := strings.Fields(value)

	if c.currentCommand == nil {
		// Completing command name
		c.input.SetValue(suggestion + " ")
	} else {
		// Completing argument
		if len(parts) > 0 {
			// Replace last part with suggestion
			parts[len(parts)-1] = suggestion
			c.input.SetValue(strings.Join(parts, " ") + " ")
		}
	}

	// Move cursor to end
	c.input.CursorEnd()
}
