package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/internal/process"
)

// CommandWindowController manages command dialogs and their state
type CommandWindowController struct {
	// Command window state
	showingCommandWindow bool
	commandAutocomplete  CommandAutocomplete

	// Run dialog state
	showingRunDialog bool
	commandsList     list.Model
	detectedCommands []parser.ExecutableCommand
	monorepoInfo     *parser.MonorepoInfo

	// Custom command dialog state
	showingCustomCommand bool
	customCommandInput   textinput.Model

	// Script selector state (for initial view)
	scriptSelector CommandAutocomplete

	// Dependencies injected from parent Model
	processMgr *process.Manager
	width      int
	height     int
}

// NewCommandWindowController creates a new command window controller
func NewCommandWindowController(processMgr *process.Manager) *CommandWindowController {
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

	return &CommandWindowController{
		processMgr:         processMgr,
		customCommandInput: customCommandInput,
		commandsList:       commandsList,
	}
}

// UpdateSize updates the controller dimensions
func (c *CommandWindowController) UpdateSize(width, height int) {
	c.width = width
	c.height = height

	// Update command autocomplete width
	if c.showingCommandWindow {
		windowWidth := DefaultCommandWindowWidth
		if width-CommandWindowPadding < windowWidth {
			windowWidth = width - CommandWindowPadding
		}
		c.commandAutocomplete.SetWidth(windowWidth)
	}

	// Update commands list size for run dialog
	if c.showingRunDialog {
		c.commandsList.SetSize(width-4, height-8)
	}
}

// ShowCommandWindow shows the command palette window
func (c *CommandWindowController) ShowCommandWindow(scripts map[string]string, aiProviders []string) {
	c.showingCommandWindow = true
	c.commandAutocomplete = NewCommandAutocompleteWithProcessManager(scripts, c.processMgr)

	windowWidth := DefaultCommandWindowWidth
	if c.width-CommandWindowPadding < windowWidth {
		windowWidth = c.width - CommandWindowPadding
	}
	c.commandAutocomplete.SetWidth(windowWidth)

	// Set AI providers if available
	if len(aiProviders) > 0 {
		c.commandAutocomplete.SetAIProviders(aiProviders)
	}

	c.commandAutocomplete.Focus()
}

// HideCommandWindow hides the command palette window
func (c *CommandWindowController) HideCommandWindow() {
	c.showingCommandWindow = false
}

// IsShowingCommandWindow returns whether the command window is visible
func (c *CommandWindowController) IsShowingCommandWindow() bool {
	return c.showingCommandWindow
}

// ShowRunDialog shows the run dialog with detected commands
func (c *CommandWindowController) ShowRunDialog(commands []parser.ExecutableCommand, monorepoInfo *parser.MonorepoInfo) {
	c.showingRunDialog = true
	c.detectedCommands = commands
	c.monorepoInfo = monorepoInfo

	// Convert commands to list items
	var items []list.Item
	for _, cmd := range commands {
		items = append(items, cmdDialogItem{command: cmd})
	}

	c.commandsList.SetItems(items)
	if len(items) > 0 {
		c.commandsList.Select(0)
	}
}

// HideRunDialog hides the run dialog
func (c *CommandWindowController) HideRunDialog() {
	c.showingRunDialog = false
}

// IsShowingRunDialog returns whether the run dialog is visible
func (c *CommandWindowController) IsShowingRunDialog() bool {
	return c.showingRunDialog
}

// ShowCustomCommandDialog shows the custom command input dialog
func (c *CommandWindowController) ShowCustomCommandDialog() {
	c.showingCustomCommand = true
	c.customCommandInput.Focus()
	c.customCommandInput.SetValue("")
}

// HideCustomCommandDialog hides the custom command dialog
func (c *CommandWindowController) HideCustomCommandDialog() {
	c.showingCustomCommand = false
}

// IsShowingCustomCommand returns whether the custom command dialog is visible
func (c *CommandWindowController) IsShowingCustomCommand() bool {
	return c.showingCustomCommand
}

// GetCommandAutocomplete returns the command autocomplete for input handling
func (c *CommandWindowController) GetCommandAutocomplete() *CommandAutocomplete {
	return &c.commandAutocomplete
}

// GetCustomCommandInput returns the custom command input for input handling
func (c *CommandWindowController) GetCustomCommandInput() *textinput.Model {
	return &c.customCommandInput
}

// GetCommandsList returns the commands list for input handling
func (c *CommandWindowController) GetCommandsList() *list.Model {
	return &c.commandsList
}

// GetSelectedCommand returns the currently selected command from the run dialog
func (c *CommandWindowController) GetSelectedCommand() *parser.ExecutableCommand {
	if !c.showingRunDialog || len(c.detectedCommands) == 0 {
		return nil
	}

	selectedItem := c.commandsList.SelectedItem()
	if selectedItem == nil {
		return nil
	}

	if cmdItem, ok := selectedItem.(cmdDialogItem); ok {
		return &cmdItem.command
	}

	return nil
}

// RenderCommandWindow renders the command palette window
func (c *CommandWindowController) RenderCommandWindow() string {
	// Get terminal size directly to avoid dependency on model updates
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		// Fallback to stored values if terminal size can't be determined
		termWidth = c.width
		termHeight = c.height
	}
	
	// Safety check for minimum dimensions - use fallback for small terminals
	if termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		// Render a minimal command window for small terminals
		return fmt.Sprintf("Command Window (%s)", ErrTerminalTooSmall)
	}

	// Create the command window
	windowWidth := DefaultCommandWindowWidth
	if termWidth-CommandWindowPadding < windowWidth {
		windowWidth = termWidth - CommandWindowPadding
	}
	maxSuggestions := MaxDropdownSuggestions

	windowStyle := lipgloss.NewStyle().
		Width(windowWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226")).
		Background(lipgloss.Color("235")).
		Padding(1, 2)

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		MarginBottom(1)

	title := titleStyle.Render("Command Palette")

	// Input
	inputStyle := lipgloss.NewStyle().
		Width(windowWidth - 6).
		MarginBottom(1)

	inputView := inputStyle.Render(c.commandAutocomplete.View())

	// Get the dropdown suggestions
	dropdownView := c.commandAutocomplete.RenderDropdown(maxSuggestions)

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	helpText := helpStyle.Render("↑↓ Navigate • Tab/Enter Select • Esc Cancel")

	// Error message if any
	errorMsg := c.commandAutocomplete.GetErrorMessage()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			MarginTop(1)
		helpText = errorStyle.Render(errorMsg)
	}

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Left, title, inputView, dropdownView, helpText)
	window := windowStyle.Render(content)

	// Use Lipgloss to center the window in the terminal
	overlayStyle := lipgloss.NewStyle().
		Width(termWidth).
		Height(termHeight).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(window)
}

// RenderRunDialog renders the run dialog with available commands
func (c *CommandWindowController) RenderRunDialog() string {
	windowStyle := lipgloss.NewStyle().
		Width(c.width - 4).
		Height(c.height - 4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	title := "Available Commands"
	if c.monorepoInfo != nil && c.monorepoInfo.Root != "" {
		title += " (Monorepo detected)"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	help := "↑↓ Navigate • Enter Run • Esc Cancel"

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		c.commandsList.View(),
		helpStyle.Render(help),
	)

	return windowStyle.Render(content)
}

// RenderCustomCommandDialog renders the custom command input dialog
func (c *CommandWindowController) RenderCustomCommandDialog() string {
	// Get terminal size directly
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		termWidth = c.width
		termHeight = c.height
	}
	
	windowWidth := DefaultCommandWindowWidth
	if termWidth-CommandWindowPadding < windowWidth {
		windowWidth = termWidth - CommandWindowPadding
	}

	windowStyle := lipgloss.NewStyle().
		Width(windowWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	title := titleStyle.Render("Custom Command")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	helpText := helpStyle.Render("Enter Run • Esc Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		c.customCommandInput.View(),
		helpText,
	)

	window := windowStyle.Render(content)

	// Use Lipgloss to center the window in the terminal
	overlayStyle := lipgloss.NewStyle().
		Width(termWidth).
		Height(termHeight).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(window)
}

// cmdDialogItem implements list.Item for executable commands (to avoid conflict with existing commandItem)
type cmdDialogItem struct {
	command parser.ExecutableCommand
}

func (i cmdDialogItem) FilterValue() string {
	return i.command.Name
}

func (i cmdDialogItem) Title() string {
	return i.command.Name
}

func (i cmdDialogItem) Description() string {
	return i.command.Command
}
