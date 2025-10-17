package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// ScriptSelectorAutocompleteInterface defines interface for script selector autocomplete
// This allows the component to work with autocomplete without tight coupling
type ScriptSelectorAutocompleteInterface interface {
	// Basic operations
	Focus()
	SetWidth(width int)
	View() string
	RenderDropdown(maxSuggestions int) string
	Update(msg tea.Msg) (ScriptSelectorAutocompleteInterface, tea.Cmd)

	// State accessors
	Value() string
	SetValue(value string)
	SetPlaceholder(placeholder string)
	GetSelected() int
	SetSelected(index int)
	GetSuggestions() []string
	SetSuggestions(suggestions []string)
	IsShowingDropdown() bool
	SetShowDropdown(show bool)
	GetErrorMessage() string
	SetErrorMessage(msg string)
	IsArbitraryMode() bool
	SetArbitraryMode(mode bool)
	UpdateSuggestions()
	GetAvailableScripts() map[string]string
}

// ScriptSelector manages the script selection UI
// Migrated from script_selector_controller.go (T068-T073)
// Architecture: Component with direct service access
type ScriptSelector struct {
	// UI components
	scriptSelector ScriptSelectorAutocompleteInterface
	visible        bool
	quickMode      bool

	// Direct service access
	processMgr    *process.Manager
	logStore      *logs.Store
	updateChan    chan tea.Msg
	navController NavigationControllerInterface
}

// NavigationControllerInterface defines minimal navigation interface
type NavigationControllerInterface interface {
	SwitchTo(view messages.ViewType)
}

// View constants for local use
const (
	ViewProcesses = messages.ViewType("processes")
	ViewLogs      = messages.ViewType("logs")
	ViewAICoders  = messages.ViewType("aicoders")
)

// Constants for terminal sizing
const (
	ScriptSelectorMargin         = 4
	ScriptSelectorMinWidth       = 40
	ScriptSelectorMaxWidth       = 100
	SmallTerminalHeightThreshold = 20
	// MaxDropdownSuggestions and MaxDropdownSuggestionsSmall defined in command_window_view.go
)

// Error constants
const (
	ErrProcessManagerNotInitialized = "process manager not initialized"
	ErrFailedToStartCommand         = "Failed to start command '%s' with args %v: %v"
	ErrFailedToStartScript          = "Failed to start script '%s': %v"
)

// NewScriptSelector creates a new script selector component
func NewScriptSelector(
	scriptSelector ScriptSelectorAutocompleteInterface,
	processMgr *process.Manager,
	logStore *logs.Store,
	updateChan chan tea.Msg,
	navController NavigationControllerInterface,
) *ScriptSelector {
	return &ScriptSelector{
		scriptSelector: scriptSelector,
		processMgr:     processMgr,
		logStore:       logStore,
		updateChan:     updateChan,
		navController:  navController,
	}
}

// Init initializes the component (BubbleTea interface)
func (s *ScriptSelector) Init() tea.Cmd {
	return nil
}

// Update handles messages (BubbleTea interface)
func (s *ScriptSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !s.visible {
		return s, nil
	}

	// Delegate to HandleKeyMsg for keyboard input
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		handled, cmd := s.HandleKeyMsg(keyMsg)
		if handled {
			return s, cmd
		}
	}

	return s, nil
}

// View renders the component (BubbleTea interface)
func (s *ScriptSelector) View() string {
	if !s.visible {
		return ""
	}
	return s.Render()
}

// SetSize updates dimensions (Component interface)
func (s *ScriptSelector) SetSize(width, height int) {
	// Script selector calculates size dynamically from terminal
}

// Focus gives keyboard focus (Component interface)
func (s *ScriptSelector) Focus() tea.Cmd {
	if s.scriptSelector != nil {
		s.scriptSelector.Focus()
	}
	return nil
}

// Blur removes keyboard focus (Component interface)
func (s *ScriptSelector) Blur() tea.Cmd {
	return nil
}

// GetSize returns current dimensions (Component interface)
func (s *ScriptSelector) GetSize() (width, height int) {
	w, h, err := getTerminalSize()
	if err != nil {
		return 80, 24
	}
	return w, h
}

// IsVisible returns whether the script selector is visible
func (s *ScriptSelector) IsVisible() bool {
	return s.visible
}

// Show displays the script selector
func (s *ScriptSelector) Show(quickMode bool) {
	s.visible = true
	s.quickMode = quickMode

	if s.scriptSelector == nil {
		return
	}

	s.scriptSelector.Focus()

	if quickMode {
		// Quick mode - clear for typing
		s.scriptSelector.SetValue("")
		s.scriptSelector.SetSelected(0)
		s.scriptSelector.SetShowDropdown(true)
	} else {
		// Normal mode - ensure dropdown shows all scripts
		s.scriptSelector.SetValue("") // Start with empty to show all
		s.scriptSelector.UpdateSuggestions()
		s.scriptSelector.SetShowDropdown(true)
	}
}

// Hide hides the script selector
func (s *ScriptSelector) Hide() {
	s.visible = false
	s.quickMode = false
	if s.scriptSelector != nil {
		s.scriptSelector.SetValue("")
		s.scriptSelector.SetErrorMessage("")
	}
}

// EnterArbitraryMode switches to arbitrary command mode
func (s *ScriptSelector) EnterArbitraryMode() {
	if s.scriptSelector == nil {
		return
	}

	s.scriptSelector.SetValue("")
	s.scriptSelector.SetPlaceholder("Type any command (e.g., 'ls', 'node server.js')...")
	s.scriptSelector.SetSuggestions([]string{})
	s.scriptSelector.SetShowDropdown(false)
	s.scriptSelector.SetErrorMessage("")
	s.scriptSelector.SetArbitraryMode(true)
}

// EnterAICoderMode starts an AI coder session
func (s *ScriptSelector) EnterAICoderMode() {
	s.Hide()

	// Start AI coder session
	safeGoroutine(
		"start AI coder session",
		func() error {
			if s.processMgr == nil {
				return fmt.Errorf(ErrProcessManagerNotInitialized)
			}

			// Start AI coder with default provider (can be configured)
			_, err := s.processMgr.StartCommand("ai-coder", "ai-coder", []string{"start"})
			if err == nil && s.navController != nil {
				// Success - switch to logs view to see AI coder output
				s.navController.SwitchTo(ViewLogs)
				if s.updateChan != nil {
					s.updateChan <- tea.Msg(nil)
				}
			}
			return err
		},
		func(err error) {
			errorMsg := fmt.Sprintf("Failed to start AI coder: %v", err)
			if s.scriptSelector != nil {
				s.scriptSelector.SetErrorMessage(errorMsg)
			}
			if s.logStore != nil {
				s.logStore.Add("system", "System", errorMsg, true)
			}
			if s.updateChan != nil {
				s.updateChan <- tea.Msg(nil)
			}
		},
	)
}

// HandleKeyMsg processes keyboard input for the script selector
func (s *ScriptSelector) HandleKeyMsg(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !s.visible || s.scriptSelector == nil {
		return false, nil
	}

	switch msg.String() {
	case "esc":
		s.Hide()
		return true, nil

	case "ctrl+s":
		// Skip startup screen - go directly to AI coder screen
		s.Hide()
		if s.navController != nil {
			s.navController.SwitchTo(ViewAICoders)
			if s.updateChan != nil {
				s.updateChan <- tea.Msg(nil)
			}
		}
		return true, nil

	case "ctrl+n":
		s.EnterArbitraryMode()
		return true, nil

	case "ctrl+a":
		s.EnterAICoderMode()
		return true, nil

	case "enter":
		return s.handleEnter()

	case "up":
		s.navigateUp()
		return true, nil

	case "down":
		s.navigateDown()
		return true, nil

	case "tab":
		s.autocomplete()
		return true, nil
	}

	// Update input
	newAutocomplete, cmd := s.scriptSelector.Update(msg)
	s.scriptSelector = newAutocomplete

	// Update suggestions if not in arbitrary mode
	if !s.scriptSelector.IsArbitraryMode() {
		s.updateSuggestions()
	}

	return true, cmd
}

// handleEnter processes the enter key
func (s *ScriptSelector) handleEnter() (bool, tea.Cmd) {
	if s.scriptSelector == nil {
		return true, nil
	}

	if s.scriptSelector.IsArbitraryMode() {
		// Execute arbitrary command
		command := strings.TrimSpace(s.scriptSelector.Value())
		if command != "" {
			parts := strings.Fields(command)
			if len(parts) > 0 {
				cmdName := parts[0]
				args := parts[1:]

				// Start the command
				safeGoroutine(
					fmt.Sprintf("start script selector command '%s'", cmdName),
					func() error {
						if s.processMgr == nil {
							return fmt.Errorf(ErrProcessManagerNotInitialized)
						}
						_, err := s.processMgr.StartCommand(command, cmdName, args)
						if err == nil && s.navController != nil {
							// Success - switch to logs view
							s.navController.SwitchTo(ViewLogs)
							if s.updateChan != nil {
								s.updateChan <- tea.Msg(nil)
							}
						}
						return err
					},
					func(err error) {
						errorMsg := fmt.Sprintf(ErrFailedToStartCommand, cmdName, args, err)
						if s.scriptSelector != nil {
							s.scriptSelector.SetErrorMessage(errorMsg)
						}
						if s.logStore != nil {
							s.logStore.Add("system", "System", errorMsg, true)
						}
						if s.updateChan != nil {
							s.updateChan <- tea.Msg(nil)
						}
					},
				)

				s.Hide()
			}
		}
		return true, nil
	}

	// Execute selected script
	suggestions := s.scriptSelector.GetSuggestions()
	selected := s.scriptSelector.GetSelected()

	if selected >= 0 && selected < len(suggestions) {
		scriptName := suggestions[selected]

		safeGoroutine(
			fmt.Sprintf("start selected script '%s'", scriptName),
			func() error {
				if s.processMgr == nil {
					return fmt.Errorf(ErrProcessManagerNotInitialized)
				}
				_, err := s.processMgr.StartScript(scriptName)
				if err == nil && s.navController != nil {
					// Success - switch to logs view
					s.navController.SwitchTo(ViewLogs)
					if s.updateChan != nil {
						s.updateChan <- tea.Msg(nil)
					}
				}
				return err
			},
			func(err error) {
				errorMsg := fmt.Sprintf(ErrFailedToStartScript, scriptName, err)
				if s.scriptSelector != nil {
					s.scriptSelector.SetErrorMessage(errorMsg)
				}
				if s.logStore != nil {
					s.logStore.Add("system", "System", errorMsg, true)
				}
				if s.updateChan != nil {
					s.updateChan <- tea.Msg(nil)
				}
			},
		)

		s.Hide()
	}

	return true, nil
}

// navigateUp moves selection up
func (s *ScriptSelector) navigateUp() {
	if s.scriptSelector == nil {
		return
	}

	suggestions := s.scriptSelector.GetSuggestions()
	if len(suggestions) > 0 && s.scriptSelector.IsShowingDropdown() {
		selected := s.scriptSelector.GetSelected()
		if selected > 0 {
			s.scriptSelector.SetSelected(selected - 1)
		}
	}
}

// navigateDown moves selection down
func (s *ScriptSelector) navigateDown() {
	if s.scriptSelector == nil {
		return
	}

	suggestions := s.scriptSelector.GetSuggestions()
	if len(suggestions) > 0 && s.scriptSelector.IsShowingDropdown() {
		selected := s.scriptSelector.GetSelected()
		if selected < len(suggestions)-1 {
			s.scriptSelector.SetSelected(selected + 1)
		}
	}
}

// autocomplete fills in the selected suggestion
func (s *ScriptSelector) autocomplete() {
	if s.scriptSelector == nil {
		return
	}

	suggestions := s.scriptSelector.GetSuggestions()
	selected := s.scriptSelector.GetSelected()

	if len(suggestions) > 0 && selected >= 0 && selected < len(suggestions) {
		s.scriptSelector.SetValue(suggestions[selected])
		s.scriptSelector.SetShowDropdown(false)
	}
}

// updateSuggestions updates the suggestion list based on input
func (s *ScriptSelector) updateSuggestions() {
	if s.scriptSelector == nil {
		return
	}

	inputValue := s.scriptSelector.Value()
	availableScripts := s.scriptSelector.GetAvailableScripts()

	if inputValue == "" {
		// Show all scripts
		scripts := make([]string, 0, len(availableScripts))
		for name := range availableScripts {
			scripts = append(scripts, name)
		}
		s.scriptSelector.SetSuggestions(scripts)
		s.scriptSelector.SetShowDropdown(true)
	} else {
		// Filter scripts
		var filtered []string
		for script := range availableScripts {
			if strings.HasPrefix(strings.ToLower(script), strings.ToLower(inputValue)) {
				filtered = append(filtered, script)
			}
		}
		s.scriptSelector.SetSuggestions(filtered)
		s.scriptSelector.SetShowDropdown(len(filtered) > 0)
	}

	// Reset selection if out of bounds
	suggestions := s.scriptSelector.GetSuggestions()
	if s.scriptSelector.GetSelected() >= len(suggestions) {
		s.scriptSelector.SetSelected(0)
	}
}

// Render returns the rendered view
func (s *ScriptSelector) Render() string {
	if !s.visible || s.scriptSelector == nil {
		return ""
	}

	// Update autocomplete width based on terminal size
	s.updateAutocompleteWidth()

	// Get terminal dimensions and max suggestions
	termWidth, termHeight := s.getTerminalDimensions()
	maxSuggestions := s.getMaxSuggestions(termHeight)

	// Render input and dropdown components
	input := s.scriptSelector.View()
	dropdown := s.scriptSelector.RenderDropdown(maxSuggestions)

	// Build the complete view using Lipgloss centering
	return s.buildView(termWidth, termHeight, input, dropdown)
}

// updateAutocompleteWidth sets the width of the autocomplete based on terminal size
func (s *ScriptSelector) updateAutocompleteWidth() {
	if s.scriptSelector == nil {
		return
	}

	termWidth, _, err := getTerminalSize()
	if err == nil && termWidth > 0 {
		autocompleteWidth := termWidth - ScriptSelectorMargin
		if autocompleteWidth < ScriptSelectorMinWidth {
			autocompleteWidth = ScriptSelectorMinWidth
		}
		if autocompleteWidth > ScriptSelectorMaxWidth {
			autocompleteWidth = ScriptSelectorMaxWidth
		}
		s.scriptSelector.SetWidth(autocompleteWidth)
	}
}

// getTerminalDimensions returns the terminal width and height with fallback values
func (s *ScriptSelector) getTerminalDimensions() (int, int) {
	termWidth, termHeight, err := getTerminalSize()
	if err != nil || termWidth == 0 || termHeight == 0 {
		// Fallback dimensions
		termWidth = 80
		termHeight = 24
	}
	return termWidth, termHeight
}

// getMaxSuggestions returns the maximum number of suggestions based on terminal height
func (s *ScriptSelector) getMaxSuggestions(termHeight int) int {
	if termHeight > 0 && termHeight < SmallTerminalHeightThreshold {
		return MaxDropdownSuggestionsSmall
	}
	return MaxDropdownSuggestions
}

// buildView constructs the complete view with all components using Lipgloss layouts
func (s *ScriptSelector) buildView(termWidth, termHeight int, input, dropdown string) string {
	// Create base container style
	containerStyle := lipgloss.NewStyle().
		Width(termWidth).
		Height(termHeight).
		Align(lipgloss.Center, lipgloss.Center)

	// Create content sections
	title := s.renderTitle(termWidth)
	content := s.renderContent(termWidth, input, dropdown)
	helpText := s.renderHelpText(termWidth)

	// Join sections vertically with proper spacing
	fullContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		content,
		helpText,
	)

	// Apply container style for centering
	return containerStyle.Render(fullContent)
}

// renderTitle creates the centered title using Lipgloss
func (s *ScriptSelector) renderTitle(termWidth int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Width(termWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	return titleStyle.Render("🐝 Select a Script to Run")
}

// renderContent creates the input and dropdown section using Lipgloss
func (s *ScriptSelector) renderContent(termWidth int, input, dropdown string) string {
	// Create content container with proper width
	contentWidth := termWidth - 10
	if contentWidth < ScriptSelectorMinWidth {
		contentWidth = ScriptSelectorMinWidth
	}
	if contentWidth > ScriptSelectorMaxWidth {
		contentWidth = ScriptSelectorMaxWidth
	}

	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Left)

	// Container for centering content horizontally
	containerStyle := lipgloss.NewStyle().
		Width(termWidth).
		Align(lipgloss.Center)

	// Build content sections
	var sections []string

	// Add input
	sections = append(sections, contentStyle.Render(input))

	// Add dropdown if present
	if dropdown != "" {
		sections = append(sections, contentStyle.Render(dropdown))
	}

	// Join sections and center
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return containerStyle.Render(content)
}

// renderHelpText creates the centered help text using Lipgloss
func (s *ScriptSelector) renderHelpText(termWidth int) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Width(termWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	return helpStyle.Render("↑/↓ Navigate • Enter Run • Tab Complete • Esc Cancel")
}

// GetScriptSelector returns the underlying autocomplete for backward compatibility
func (s *ScriptSelector) GetScriptSelector() ScriptSelectorAutocompleteInterface {
	return s.scriptSelector
}

// Helper functions
// Note: getTerminalSize() is defined in command_window_view.go and reused here

// safeGoroutine wraps operation execution with panic recovery
// TODO: Replace with proper command pattern for async operations
// For now, execute synchronously to avoid goroutines
func safeGoroutine(operationName string, fn func() error, onError func(error)) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in %s: %v", operationName, r)
				if onError != nil {
					onError(err)
				}
			}
		}()

		if err := fn(); err != nil && onError != nil {
			onError(err)
		}
	}()
}
