package components

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/parser"
	"github.com/standardbeagle/brummer/internal/process"
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

// CommandWindowView manages command dialogs and their state
// Migrated from command_window_controller.go (T033-T037)
// Architecture: Component with direct process.Manager service access
type CommandWindowView struct {
	// Command window state
	showingCommandWindow bool
	commandAutocomplete  CommandAutocompleteInterface

	// Run dialog state
	showingRunDialog bool
	commandsList     list.Model
	detectedCommands []parser.ExecutableCommand
	monorepoInfo     *parser.MonorepoInfo

	// Custom command dialog state
	showingCustomCommand bool
	customCommandInput   textinput.Model

	// Script selector state (for initial view)
	scriptSelector CommandAutocompleteInterface

	// Direct service access
	processMgr *process.Manager
	width      int
	height     int

	// Factory function for creating autocomplete instances
	// Injected from parent to avoid import cycle
	autocompleteFactory func(scripts map[string]string, processMgr *process.Manager) CommandAutocompleteInterface
}

// NewCommandWindowView creates a new command window component
// The autocompleteFactory is injected to avoid import cycle with internal/tui
func NewCommandWindowView(processMgr *process.Manager, autocompleteFactory func(scripts map[string]string, processMgr *process.Manager) CommandAutocompleteInterface) *CommandWindowView {
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

	return &CommandWindowView{
		processMgr:          processMgr,
		customCommandInput:  customCommandInput,
		commandsList:        commandsList,
		autocompleteFactory: autocompleteFactory,
	}
}

// Init initializes the component (BubbleTea interface)
func (v *CommandWindowView) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (v *CommandWindowView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// If command window is showing, update the command autocomplete component
	if v.showingCommandWindow && v.commandAutocomplete != nil {
		switch keyMsg := msg.(type) {
		case tea.KeyMsg:
			// Forward keyboard events to the command autocomplete component
			newAutocomplete, cmd := v.commandAutocomplete.Update(keyMsg)
			v.commandAutocomplete = newAutocomplete
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update custom command input if showing
	if v.showingCustomCommand {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			newInput, cmd := v.customCommandInput.Update(msg)
			v.customCommandInput = newInput
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update commands list if run dialog is showing
	if v.showingRunDialog {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			newList, cmd := v.commandsList.Update(msg)
			v.commandsList = newList
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return v, tea.Batch(cmds...)
}

// View renders the component (BubbleTea interface)
func (v *CommandWindowView) View() string {
	// Rendering is done via Render* methods called by parent model
	return ""
}

// SetSize updates the component's dimensions (Component interface)
func (v *CommandWindowView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Focus gives keyboard focus to the component (Component interface)
func (v *CommandWindowView) Focus() tea.Cmd {
	return nil
}

// Blur removes keyboard focus from the component (Component interface)
func (v *CommandWindowView) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (v *CommandWindowView) GetSize() (width, height int) {
	return v.width, v.height
}

// UpdateSize updates the controller dimensions
func (v *CommandWindowView) UpdateSize(width, height int) {
	v.width = width
	v.height = height

	// Update command autocomplete width
	if v.showingCommandWindow && v.commandAutocomplete != nil {
		windowWidth := DefaultCommandWindowWidth
		if width-CommandWindowPadding < windowWidth {
			windowWidth = width - CommandWindowPadding
		}
		v.commandAutocomplete.SetWidth(windowWidth)
	}

	// Update commands list size for run dialog
	if v.showingRunDialog {
		v.commandsList.SetSize(width-4, height-8)
	}
}

// ShowCommandWindow shows the command palette window
func (v *CommandWindowView) ShowCommandWindow(scripts map[string]string) {
	v.showingCommandWindow = true
	v.commandAutocomplete = v.autocompleteFactory(scripts, v.processMgr)

	windowWidth := DefaultCommandWindowWidth
	if v.width-CommandWindowPadding < windowWidth {
		windowWidth = v.width - CommandWindowPadding
	}
	v.commandAutocomplete.SetWidth(windowWidth)

	v.commandAutocomplete.Focus()
}

// HideCommandWindow hides the command palette window
func (v *CommandWindowView) HideCommandWindow() {
	v.showingCommandWindow = false
}

// IsShowingCommandWindow returns whether the command window is visible
func (v *CommandWindowView) IsShowingCommandWindow() bool {
	return v.showingCommandWindow
}

// ShowRunDialog shows the run dialog with detected commands
func (v *CommandWindowView) ShowRunDialog(commands []parser.ExecutableCommand, monorepoInfo *parser.MonorepoInfo) {
	v.showingRunDialog = true
	v.detectedCommands = commands
	v.monorepoInfo = monorepoInfo

	// Convert commands to list items
	var items []list.Item
	for _, cmd := range commands {
		items = append(items, CmdDialogItem{Command: cmd})
	}

	v.commandsList.SetItems(items)
	if len(items) > 0 {
		v.commandsList.Select(0)
	}
}

// HideRunDialog hides the run dialog
func (v *CommandWindowView) HideRunDialog() {
	v.showingRunDialog = false
}

// IsShowingRunDialog returns whether the run dialog is visible
func (v *CommandWindowView) IsShowingRunDialog() bool {
	return v.showingRunDialog
}

// ShowCustomCommandDialog shows the custom command input dialog
func (v *CommandWindowView) ShowCustomCommandDialog() {
	v.showingCustomCommand = true
	v.customCommandInput.Focus()
	v.customCommandInput.SetValue("")
}

// HideCustomCommandDialog hides the custom command dialog
func (v *CommandWindowView) HideCustomCommandDialog() {
	v.showingCustomCommand = false
}

// IsShowingCustomCommand returns whether the custom command dialog is visible
func (v *CommandWindowView) IsShowingCustomCommand() bool {
	return v.showingCustomCommand
}

// GetCommandAutocomplete returns the command autocomplete for input handling
func (v *CommandWindowView) GetCommandAutocomplete() CommandAutocompleteInterface {
	return v.commandAutocomplete
}

// SetCommandAutocomplete sets the command autocomplete (used by DialogMessageHandler)
func (v *CommandWindowView) SetCommandAutocomplete(autocomplete CommandAutocompleteInterface) {
	v.commandAutocomplete = autocomplete
}

// GetCustomCommandInput returns the custom command input for input handling
func (v *CommandWindowView) GetCustomCommandInput() *textinput.Model {
	return &v.customCommandInput
}

// GetCommandsList returns the commands list for input handling
func (v *CommandWindowView) GetCommandsList() *list.Model {
	return &v.commandsList
}

// GetSelectedCommand returns the currently selected command from the run dialog
func (v *CommandWindowView) GetSelectedCommand() *parser.ExecutableCommand {
	if !v.showingRunDialog || len(v.detectedCommands) == 0 {
		return nil
	}

	selectedItem := v.commandsList.SelectedItem()
	if selectedItem == nil {
		return nil
	}

	if cmdItem, ok := selectedItem.(CmdDialogItem); ok {
		return &cmdItem.Command
	}

	return nil
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

// RenderCommandWindow renders the command palette window
func (v *CommandWindowView) RenderCommandWindow() string {
	// Get terminal size directly to avoid dependency on model updates
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		// Fallback to stored values if terminal size can't be determined
		termWidth = v.width
		termHeight = v.height
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

	inputView := inputStyle.Render(v.commandAutocomplete.View())

	// Get the dropdown suggestions
	dropdownView := v.commandAutocomplete.RenderDropdown(maxSuggestions)

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	helpText := helpStyle.Render("↑↓ Navigate • Tab/Enter Select • Esc Cancel")

	// Error message if any
	errorMsg := v.commandAutocomplete.GetErrorMessage()
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
func (v *CommandWindowView) RenderRunDialog() string {
	windowStyle := lipgloss.NewStyle().
		Width(v.width - 4).
		Height(v.height - 4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	title := "Available Commands"
	if v.monorepoInfo != nil && v.monorepoInfo.Root != "" {
		title += " (Monorepo detected)"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(1)

	help := "↑↓ Navigate • Enter Run • Esc Cancel"

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		v.commandsList.View(),
		helpStyle.Render(help),
	)

	return windowStyle.Render(content)
}

// RenderCustomCommandDialog renders the custom command input dialog
func (v *CommandWindowView) RenderCustomCommandDialog() string {
	// Get terminal size directly
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth < MinTerminalWidth || termHeight < MinTerminalHeight {
		termWidth = v.width
		termHeight = v.height
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
		v.customCommandInput.View(),
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

// CmdDialogItem implements list.Item for executable commands
type CmdDialogItem struct {
	Command parser.ExecutableCommand
}

func (i CmdDialogItem) FilterValue() string {
	return i.Command.Name
}

func (i CmdDialogItem) Title() string {
	return i.Command.Name
}

func (i CmdDialogItem) Description() string {
	return i.Command.Command
}
