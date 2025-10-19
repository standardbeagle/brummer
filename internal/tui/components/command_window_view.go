package components

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"golang.org/x/term"
)

const (
	DefaultCommandWindowWidth   = 60
	CommandWindowPadding        = 10
	MinTerminalWidth            = 20
	MinTerminalHeight           = 10
	MaxDropdownSuggestions      = 10
	MaxDropdownSuggestionsSmall = 5
)

var ErrTerminalTooSmall = "Terminal too small"

// CommandAutocompleteInterface defines the interface for command autocomplete
// This avoids import cycle with internal/tui package
type CommandAutocompleteInterface interface {
	SetWidth(width int)
	Focus()
	View() string
	RenderDropdown(maxSuggestions int) string
	GetErrorMessage() string
	Value() string
	GetSuggestions() []string
	Update(msg tea.Msg) (CommandAutocompleteInterface, tea.Cmd)
	ValidateInput() (bool, string)
	SetError(errMsg string)
}

// CommandWindowViewComponentNew is a stateless BubbleTea component for command windows
// Architecture: Stateless rendering with external state (Redux/Elm compliant)
type CommandWindowViewComponentNew struct {
	BaseComponent

	// UI configuration only (no state)
	width  int
	height int

	// Interactive UI components (minimal stateless usage)
	customCommandInput textinput.Model
	commandsList       list.Model

	// Factory function for creating autocomplete instances
	// Injected from parent to avoid import cycle
	autocompleteFactory func(scripts map[string]string, processMgr interface{}) CommandAutocompleteInterface

	// Styles
	windowStyle       lipgloss.Style
	titleStyle        lipgloss.Style
	inputStyle        lipgloss.Style
	helpStyle         lipgloss.Style
	errorStyle        lipgloss.Style
	contentStyle      lipgloss.Style
	overlayStyle      lipgloss.Style
	runDialogStyle    lipgloss.Style
	customDialogStyle lipgloss.Style
}

// NewCommandWindowView creates a new stateless command window view component
func NewCommandWindowViewNew(autocompleteFactory func(scripts map[string]string, processMgr interface{}) CommandAutocompleteInterface) *CommandWindowViewComponentNew {
	// Initialize custom command input
	customCommandInput := textinput.New()
	customCommandInput.Placeholder = "Enter command (e.g., node server.js)"
	customCommandInput.Focus()
	customCommandInput.CharLimit = 200

	// Initialize commands list for run dialog
	commandsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	commandsList.Title = "Available Commands"
	commandsList.SetShowStatusBar(false)
	commandsList.SetFilteringEnabled(true)

	return &CommandWindowViewComponentNew{
		BaseComponent:       NewBaseComponent(),
		width:               80,
		height:              24,
		customCommandInput:  customCommandInput,
		commandsList:        commandsList,
		autocompleteFactory: autocompleteFactory,

		// Initialize styles
		windowStyle: lipgloss.NewStyle().
			Width(DefaultCommandWindowWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("226")).
			Background(lipgloss.Color("235")).
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			MarginBottom(1),

		inputStyle: lipgloss.NewStyle().
			Width(DefaultCommandWindowWidth - 6).
			MarginBottom(1),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			MarginTop(1),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginTop(1),

		contentStyle: lipgloss.NewStyle(),
		overlayStyle: lipgloss.NewStyle(),

		runDialogStyle: lipgloss.NewStyle().
			Width(76).
			Height(20).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Background(lipgloss.Color("235")).
			Padding(1),

		customDialogStyle: lipgloss.NewStyle().
			Width(DefaultCommandWindowWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Background(lipgloss.Color("235")).
			Padding(1, 2),
	}
}

// ViewWithState renders the command window view using external state (Redux/Elm compliant)
func (c *CommandWindowViewComponentNew) ViewWithState(masterData *state.MasterData) string {
	if !c.IsVisible() {
		return ""
	}

	if masterData == nil || masterData.CommandWindowState == nil {
		return c.renderErrorState("Command window state not available")
	}

	cmdWindowState := masterData.CommandWindowState

	// Render the appropriate dialog based on state
	switch {
	case cmdWindowState.IsShowingCommandWindow():
		return c.renderCommandWindow(cmdWindowState)
	case cmdWindowState.IsShowingRunDialog():
		return c.renderRunDialog(cmdWindowState)
	case cmdWindowState.IsShowingCustomCommand():
		return c.renderCustomCommandDialog(cmdWindowState)
	default:
		return ""
	}
}

// renderCommandWindow renders the command palette window
func (c *CommandWindowViewComponentNew) renderCommandWindow(cmdWindowState *state.CommandWindowState) string {
	// Get terminal size directly to avoid dependency on model updates
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		// Fallback to stored values if terminal size can't be determined
		termWidth, termHeight = cmdWindowState.GetTerminalSize()
	}

	// Safety check for minimum dimensions - use fallback for small terminals
	if termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		// Render a minimal command window for small terminals
		return fmt.Sprintf("Command Window (%s)", ErrTerminalTooSmall)
	}

	// Create the command window
	windowWidth := cmdWindowState.GetCommandWindowWidth()
	if termWidth-CommandWindowPadding < windowWidth {
		windowWidth = termWidth - CommandWindowPadding
	}

	windowStyle := c.windowStyle.Width(windowWidth)
	titleStyle := c.titleStyle
	inputStyle := c.inputStyle.Width(windowWidth - 6)

	// Title
	title := titleStyle.Render("Command Palette")

	// Input - this would need to be handled by the autocomplete component
	inputView := inputStyle.Render("> ") // Placeholder for autocomplete view

	// Help text
	helpStyle := c.helpStyle
	helpText := helpStyle.Render("↑↓ Navigate • Tab/Enter Select • Esc Cancel")

	// Error message if any
	errorMsg := cmdWindowState.GetError()
	if errorMsg != "" {
		errorStyle := c.errorStyle
		helpText = errorStyle.Render(errorMsg)
	}

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Left, title, inputView, helpText)
	window := windowStyle.Render(content)

	// Use Lipgloss to center the window in the terminal
	overlayStyle := c.overlayStyle.Width(termWidth).Height(termHeight).Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(window)
}

// renderRunDialog renders the run dialog with available commands
func (c *CommandWindowViewComponentNew) renderRunDialog(cmdWindowState *state.CommandWindowState) string {
	termWidth, termHeight := cmdWindowState.GetTerminalSize()

	windowStyle := c.runDialogStyle.Width(termWidth - 4).Height(termHeight - 4)
	titleStyle := c.titleStyle.Foreground(lipgloss.Color("39"))
	helpStyle := c.helpStyle

	title := "Available Commands"
	monorepoInfo := cmdWindowState.GetMonorepoInfo()
	if monorepoInfo != nil && monorepoInfo.Root != "" {
		title += " (Monorepo detected)"
	}

	help := "↑↓ Navigate • Enter Run • Esc Cancel"

	// Update commands list with current state
	c.updateCommandsListFromState(cmdWindowState)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		c.commandsList.View(),
		helpStyle.Render(help),
	)

	return windowStyle.Render(content)
}

// renderCustomCommandDialog renders the custom command input dialog
func (c *CommandWindowViewComponentNew) renderCustomCommandDialog(cmdWindowState *state.CommandWindowState) string {
	// Get terminal size directly
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		termWidth, termHeight = cmdWindowState.GetTerminalSize()
	}

	windowWidth := cmdWindowState.GetCommandWindowWidth()
	if termWidth-CommandWindowPadding < windowWidth {
		windowWidth = termWidth - CommandWindowPadding
	}

	windowStyle := c.customDialogStyle.Width(windowWidth)
	titleStyle := c.titleStyle.Foreground(lipgloss.Color("39"))
	helpStyle := c.helpStyle

	title := titleStyle.Render("Custom Command")
	helpText := helpStyle.Render("Enter Run • Esc Cancel")

	// Update custom command input with current state
	c.customCommandInput.SetValue(cmdWindowState.GetCustomCommandText())

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		c.customCommandInput.View(),
		helpText,
	)

	window := windowStyle.Render(content)

	// Use Lipgloss to center the window in the terminal
	overlayStyle := c.overlayStyle.Width(termWidth).Height(termHeight).Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(window)
}

// updateCommandsListFromState updates the commands list using current state
func (c *CommandWindowViewComponentNew) updateCommandsListFromState(cmdWindowState *state.CommandWindowState) {
	detectedCommands := cmdWindowState.GetDetectedCommands()

	// Convert commands to list items
	var items []list.Item
	for _, cmd := range detectedCommands {
		items = append(items, CmdDialogItemWrapper{Command: cmd})
	}

	c.commandsList.SetItems(items)
	if len(items) > 0 {
		c.commandsList.Select(0) // Select first item by default
	}

	// Update list size
	termWidth, termHeight := cmdWindowState.GetTerminalSize()
	c.commandsList.SetSize(termWidth-4, termHeight-8)
}

// renderErrorState renders an error state when command window data is not available
func (c *CommandWindowViewComponentNew) renderErrorState(message string) string {
	var content strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Padding(0, 1).
		MarginBottom(1)

	content.WriteString(headerStyle.Render("⌨️  Command Window"))
	content.WriteString("\n")

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Margin(1)

	content.WriteString(errorStyle.Render("⚠️  "+message))
	content.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Margin(1)

	content.WriteString(helpStyle.Render("Command window functionality will be available when needed"))

	return content.String()
}

// getTerminalSize returns the terminal width and height
func getTerminalSize() (width, height int, err error) {
	fd := int(os.Stdout.Fd())
	width, height, err = term.GetSize(fd)
	if err == nil {
		return width, height, nil
	}
	// Try stderr as fallback
	fd = int(os.Stderr.Fd())
	return term.GetSize(fd)
}

// SetSize updates component dimensions
func (c *CommandWindowViewComponentNew) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// UpdateSize updates dimensions with pre-calculated content height
func (c *CommandWindowViewComponentNew) UpdateSize(width, height, headerHeight, footerHeight, contentHeight int) {
	c.width = width
	c.height = height
}

// View renders the component (legacy method for compatibility)
func (c *CommandWindowViewComponentNew) View() string {
	if !c.IsVisible() {
		return ""
	}
	return "" // Rendering is done via ViewWithState
}

// Update handles messages (minimal - stateless component)
func (c *CommandWindowViewComponentNew) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetSize(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Update custom command input if needed
		newInput, inputCmd := c.customCommandInput.Update(msg)
		c.customCommandInput = newInput
		if inputCmd != nil {
			cmd = inputCmd
		}

		// Update commands list if needed
		newList, listCmd := c.commandsList.Update(msg)
		c.commandsList = newList
		if listCmd != nil {
			cmd = tea.Batch(cmd, listCmd)
		}

		return c, cmd
	}

	return c, nil
}

// Init initializes the component
func (c *CommandWindowViewComponentNew) Init() tea.Cmd {
	return nil
}

// Wrapper types for list.Item interface compatibility

// CmdDialogItemWrapper implements list.Item for executable commands
type CmdDialogItemWrapper struct {
	Command parser.ExecutableCommand
}

func (i CmdDialogItemWrapper) FilterValue() string {
	return i.Command.Name
}

func (i CmdDialogItemWrapper) Title() string {
	return i.Command.Name
}

func (i CmdDialogItemWrapper) Description() string {
	return i.Command.Command
}