package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandInputComponent provides command palette and input functionality
type CommandInputComponent struct {
	BaseComponent
	
	// Input state
	textInput      textinput.Model
	mode           CommandMode
	prompt         string
	placeholder    string
	
	// Command history
	history        []string
	historyIndex   int
	
	// Suggestions
	suggestions    []CommandSuggestion
	selectedIndex  int
	showSuggestions bool
	
	// Command results
	lastCommand    string
	lastResult     string
	
	// Styles
	promptStyle    lipgloss.Style
	inputStyle     lipgloss.Style
	suggestionStyle lipgloss.Style
	selectedStyle  lipgloss.Style
	resultStyle    lipgloss.Style
	errorStyle     lipgloss.Style
}

// CommandMode and CommandSuggestion types are defined in custom_messages.go

// NewCommandInputComponent creates a new command input component
func NewCommandInputComponent() *CommandInputComponent {
	c := &CommandInputComponent{
		mode:            CommandModeNormal,
		prompt:          ">",
		placeholder:     "Type a command or ? for help",
		history:         []string{},
		historyIndex:    -1,
		suggestions:     []CommandSuggestion{},
		showSuggestions: false,
	}
	
	// Initialize styles
	c.initStyles()
	
	// Initialize text input
	c.textInput = textinput.New()
	c.textInput.Placeholder = c.placeholder
	c.textInput.CharLimit = 256
	c.textInput.Width = 80
	
	// Load default suggestions
	c.loadDefaultSuggestions()
	
	// Set visible by default
	c.visible = true
	
	return c
}

// initStyles initializes the component styles
func (c *CommandInputComponent) initStyles() {
	c.promptStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	
	c.inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))
	
	c.suggestionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	c.selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)
	
	c.resultStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))
	
	c.errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
}

// loadDefaultSuggestions loads default command suggestions
func (c *CommandInputComponent) loadDefaultSuggestions() {
	c.suggestions = []CommandSuggestion{
		// Process commands
		{Command: "start", Description: "Start a script", Category: "Process", Shortcut: "s"},
		{Command: "stop", Description: "Stop a process", Category: "Process", Shortcut: "x"},
		{Command: "restart", Description: "Restart a process", Category: "Process", Shortcut: "r"},
		{Command: "kill", Description: "Force kill a process", Category: "Process", Shortcut: "k"},
		
		// Navigation commands
		{Command: "goto", Description: "Go to view", Category: "Navigation", Shortcut: "g"},
		{Command: "processes", Description: "View processes", Category: "Navigation", Shortcut: "1"},
		{Command: "logs", Description: "View logs", Category: "Navigation", Shortcut: "2"},
		{Command: "errors", Description: "View errors", Category: "Navigation", Shortcut: "3"},
		{Command: "urls", Description: "View URLs", Category: "Navigation", Shortcut: "4"},
		{Command: "web", Description: "View web", Category: "Navigation", Shortcut: "5"},
		{Command: "settings", Description: "View settings", Category: "Navigation", Shortcut: "6"},
		
		// Search commands
		{Command: "search", Description: "Search logs", Category: "Search", Shortcut: "/"},
		{Command: "filter", Description: "Filter current view", Category: "Search", Shortcut: "f"},
		{Command: "find", Description: "Find in page", Category: "Search", Shortcut: "ctrl+f"},
		
		// AI commands
		{Command: "ai", Description: "AI assistant", Category: "AI", Shortcut: "a"},
		{Command: "explain", Description: "Explain error", Category: "AI", Shortcut: "e"},
		{Command: "suggest", Description: "Get suggestions", Category: "AI", Shortcut: "?"},
		
		// System commands
		{Command: "reload", Description: "Reload configuration", Category: "System", Shortcut: "R"},
		{Command: "clear", Description: "Clear current view", Category: "System", Shortcut: "c"},
		{Command: "help", Description: "Show help", Category: "System", Shortcut: "h"},
		{Command: "quit", Description: "Quit application", Category: "System", Shortcut: "q"},
	}
}

// Init implements tea.Model interface
func (c *CommandInputComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *CommandInputComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle component-specific messages
	switch msg := msg.(type) {
	case CommandModeMsg:
		// Change command mode
		c.mode = msg.Mode
		c.prompt = c.getPromptForMode(msg.Mode)
		c.placeholder = c.getPlaceholderForMode(msg.Mode)
		c.textInput.Placeholder = c.placeholder
		c.textInput.SetValue("")
		c.textInput.Focus()
		c.showSuggestions = true
		c.updateSuggestions()
		return c, textinput.Blink
		
	case CommandExecuteMsg:
		// Execute command
		result := c.executeCommand(msg.Command)
		c.lastCommand = msg.Command
		c.lastResult = result
		
		// Add to history
		c.addToHistory(msg.Command)
		
		// Clear input
		c.textInput.SetValue("")
		c.showSuggestions = false
		
		// Send notification
		if strings.HasPrefix(result, "Error:") {
			return c, SendSystemNotification(NotificationError,
				"Command failed", result)
		}
		return c, SendSystemNotification(NotificationSuccess,
			"Command executed", result)
		
	case CommandSuggestMsg:
		// Update suggestions based on input
		c.updateSuggestions()
		return c, nil
		
	case CommandHistoryMsg:
		// Navigate history
		if msg.Direction == "up" {
			c.navigateHistoryUp()
		} else {
			c.navigateHistoryDown()
		}
		return c, nil
		
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.textInput.Width = min(msg.Width-len(c.prompt)-4, 100)
		
	case tea.KeyMsg:
		// Handle key presses
		switch msg.String() {
		case "enter":
			// Execute command
			cmd := c.textInput.Value()
			if cmd != "" {
				return c, func() tea.Msg {
					return CommandExecuteMsg{Command: cmd}
				}
			}
			
		case "tab":
			// Autocomplete or cycle suggestions
			if c.showSuggestions && len(c.suggestions) > 0 {
				c.selectedIndex = (c.selectedIndex + 1) % len(c.suggestions)
				c.textInput.SetValue(c.suggestions[c.selectedIndex].Command)
			}
			return c, nil
			
		case "shift+tab":
			// Cycle suggestions backwards
			if c.showSuggestions && len(c.suggestions) > 0 {
				c.selectedIndex--
				if c.selectedIndex < 0 {
					c.selectedIndex = len(c.suggestions) - 1
				}
				c.textInput.SetValue(c.suggestions[c.selectedIndex].Command)
			}
			return c, nil
			
		case "up":
			// Navigate history up
			if !c.showSuggestions {
				c.navigateHistoryUp()
				return c, nil
			}
			// Navigate suggestions up
			if c.selectedIndex > 0 {
				c.selectedIndex--
			}
			return c, nil
			
		case "down":
			// Navigate history down
			if !c.showSuggestions {
				c.navigateHistoryDown()
				return c, nil
			}
			// Navigate suggestions down
			if c.selectedIndex < len(c.suggestions)-1 {
				c.selectedIndex++
			}
			return c, nil
			
		case "esc":
			// Cancel input
			c.textInput.SetValue("")
			c.showSuggestions = false
			c.textInput.Blur()
			return c, nil
			
		case "ctrl+c":
			// Clear input
			c.textInput.SetValue("")
			return c, nil
			
		case "?":
			// Show help
			c.showSuggestions = true
			c.updateSuggestions()
			return c, nil
		}
		
		// Update text input
		var cmd tea.Cmd
		c.textInput, cmd = c.textInput.Update(msg)
		
		// Update suggestions on input change
		c.updateSuggestions()
		
		return c, cmd
	}
	
	return c, tea.Batch(cmds...)
}

// View renders the component
func (c *CommandInputComponent) View() string {
	if !c.visible {
		return ""
	}
	
	var b strings.Builder
	
	// Show prompt and input
	b.WriteString(c.promptStyle.Render(c.prompt) + " ")
	b.WriteString(c.textInput.View())
	b.WriteString("\n")
	
	// Show suggestions if enabled
	if c.showSuggestions && len(c.suggestions) > 0 {
		b.WriteString("\n")
		maxShow := min(5, len(c.suggestions))
		for i := 0; i < maxShow; i++ {
			suggestion := c.suggestions[i]
			
			// Format suggestion
			line := fmt.Sprintf("  %-15s %s",
				suggestion.Command,
				suggestion.Description)
			
			if suggestion.Shortcut != "" {
				line += fmt.Sprintf(" [%s]", suggestion.Shortcut)
			}
			
			// Apply style
			if i == c.selectedIndex {
				b.WriteString(c.selectedStyle.Render("▶ " + line))
			} else {
				b.WriteString(c.suggestionStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
		
		if len(c.suggestions) > maxShow {
			b.WriteString(c.suggestionStyle.Render(
				fmt.Sprintf("  ... and %d more", len(c.suggestions)-maxShow)))
			b.WriteString("\n")
		}
	}
	
	// Show last command result
	if c.lastResult != "" {
		b.WriteString("\n")
		if strings.HasPrefix(c.lastResult, "Error:") {
			b.WriteString(c.errorStyle.Render(c.lastResult))
		} else {
			b.WriteString(c.resultStyle.Render(c.lastResult))
		}
		b.WriteString("\n")
	}
	
	return b.String()
}

// Helper methods

func (c *CommandInputComponent) getPromptForMode(mode CommandMode) string {
	switch mode {
	case CommandModeSearch:
		return "/"
	case CommandModeFilter:
		return "filter>"
	case CommandModeGoto:
		return "goto>"
	case CommandModeRun:
		return "run>"
	case CommandModeAI:
		return "ai>"
	default:
		return ">"
	}
}

func (c *CommandInputComponent) getPlaceholderForMode(mode CommandMode) string {
	switch mode {
	case CommandModeSearch:
		return "Search pattern..."
	case CommandModeFilter:
		return "Filter expression..."
	case CommandModeGoto:
		return "View name or number..."
	case CommandModeRun:
		return "Script or command to run..."
	case CommandModeAI:
		return "Ask AI assistant..."
	default:
		return "Type a command or ? for help"
	}
}

func (c *CommandInputComponent) updateSuggestions() {
	input := strings.ToLower(c.textInput.Value())
	if input == "" {
		c.selectedIndex = 0
		return
	}
	
	// Filter suggestions based on input
	var filtered []CommandSuggestion
	for _, s := range c.suggestions {
		if strings.HasPrefix(strings.ToLower(s.Command), input) ||
			strings.Contains(strings.ToLower(s.Description), input) {
			filtered = append(filtered, s)
		}
	}
	
	c.suggestions = filtered
	c.selectedIndex = 0
}

func (c *CommandInputComponent) executeCommand(cmd string) string {
	// Parse command
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ""
	}
	
	command := parts[0]
	args := parts[1:]
	
	// Execute based on command
	switch command {
	case "start":
		if len(args) > 0 {
			return fmt.Sprintf("Starting %s...", args[0])
		}
		return "Error: Specify script to start"
		
	case "stop":
		if len(args) > 0 {
			return fmt.Sprintf("Stopping %s...", args[0])
		}
		return "Error: Specify process to stop"
		
	case "goto", "g":
		if len(args) > 0 {
			return fmt.Sprintf("Switching to %s view", args[0])
		}
		return "Error: Specify view name"
		
	case "search", "/":
		if len(args) > 0 {
			return fmt.Sprintf("Searching for: %s", strings.Join(args, " "))
		}
		return "Error: Specify search pattern"
		
	case "filter", "f":
		if len(args) > 0 {
			return fmt.Sprintf("Filtering: %s", strings.Join(args, " "))
		}
		return "Error: Specify filter expression"
		
	case "help", "?":
		return "Available commands: start, stop, goto, search, filter, help, quit"
		
	case "quit", "q":
		return "Quitting application..."
		
	default:
		return fmt.Sprintf("Unknown command: %s", command)
	}
}

func (c *CommandInputComponent) addToHistory(cmd string) {
	// Don't add empty or duplicate commands
	if cmd == "" {
		return
	}
	
	// Remove if already in history
	for i, h := range c.history {
		if h == cmd {
			c.history = append(c.history[:i], c.history[i+1:]...)
			break
		}
	}
	
	// Add to front of history
	c.history = append([]string{cmd}, c.history...)
	
	// Limit history size
	if len(c.history) > 100 {
		c.history = c.history[:100]
	}
	
	c.historyIndex = -1
}

func (c *CommandInputComponent) navigateHistoryUp() {
	if len(c.history) == 0 {
		return
	}
	
	c.historyIndex++
	if c.historyIndex >= len(c.history) {
		c.historyIndex = len(c.history) - 1
	}
	
	c.textInput.SetValue(c.history[c.historyIndex])
}

func (c *CommandInputComponent) navigateHistoryDown() {
	if c.historyIndex < 0 {
		return
	}
	
	c.historyIndex--
	if c.historyIndex < 0 {
		c.textInput.SetValue("")
		c.historyIndex = -1
	} else {
		c.textInput.SetValue(c.history[c.historyIndex])
	}
}

// Focus sets focus to this component
func (c *CommandInputComponent) Focus() tea.Cmd {
	c.focused = true
	c.textInput.Focus()
	return textinput.Blink
}

// Blur removes focus from this component
func (c *CommandInputComponent) Blur() tea.Cmd {
	c.focused = false
	c.textInput.Blur()
	c.showSuggestions = false
	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}