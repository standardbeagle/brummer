package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
)

// ScriptSelectorController manages the script selection UI
type ScriptSelectorController struct {
	// UI components
	scriptSelector CommandAutocomplete
	visible        bool
	quickMode      bool

	// Dependencies
	processMgr    *process.Manager
	logStore      *logs.Store
	updateChan    chan tea.Msg
	navController NavigationControllerInterface
}

// NewScriptSelectorController creates a new script selector controller
func NewScriptSelectorController(scripts map[string]string, processMgr *process.Manager, logStore *logs.Store, updateChan chan tea.Msg, navController NavigationControllerInterface) *ScriptSelectorController {
	scriptMap := scripts

	scriptSelector := NewScriptSelectorAutocompleteWithProcessManager(scriptMap, processMgr)
	// Width will be set dynamically in View() based on terminal size

	return &ScriptSelectorController{
		scriptSelector: scriptSelector,
		processMgr:     processMgr,
		logStore:       logStore,
		updateChan:     updateChan,
		navController:  navController,
	}
}

// IsVisible returns whether the script selector is visible
func (c *ScriptSelectorController) IsVisible() bool {
	return c.visible
}

// Show displays the script selector
func (c *ScriptSelectorController) Show(quickMode bool) {
	c.visible = true
	c.quickMode = quickMode
	c.scriptSelector.Focus()

	if quickMode {
		// Quick mode - clear for typing
		c.scriptSelector.input.SetValue("")
		c.scriptSelector.selected = 0
		c.scriptSelector.showDropdown = true
	} else {
		// Normal mode - ensure dropdown shows all scripts
		c.scriptSelector.input.SetValue("") // Start with empty to show all
		c.scriptSelector.updateScriptSelectorSuggestions()
		c.scriptSelector.showDropdown = true
	}
}

// Hide hides the script selector
func (c *ScriptSelectorController) Hide() {
	c.visible = false
	c.quickMode = false
	c.scriptSelector.input.SetValue("")
	c.scriptSelector.errorMessage = ""
}

// EnterArbitraryMode switches to arbitrary command mode
func (c *ScriptSelectorController) EnterArbitraryMode() {
	c.scriptSelector.input.SetValue("")
	c.scriptSelector.input.Placeholder = "Type any command (e.g., 'ls', 'node server.js')..."
	c.scriptSelector.suggestions = []string{}
	c.scriptSelector.showDropdown = false
	c.scriptSelector.errorMessage = ""
	c.scriptSelector.arbitraryMode = true
}

// HandleKeyMsg processes keyboard input for the script selector
func (c *ScriptSelectorController) HandleKeyMsg(msg tea.KeyMsg) (bool, tea.Cmd) {
	if c == nil || !c.visible {
		return false, nil
	}

	switch msg.String() {
	case "esc":
		c.Hide()
		return true, nil

	case "ctrl+n":
		c.EnterArbitraryMode()
		return true, nil

	case "enter":
		return c.handleEnter()

	case "up":
		c.navigateUp()
		return true, nil

	case "down":
		c.navigateDown()
		return true, nil

	case "tab":
		c.autocomplete()
		return true, nil
	}

	// Update input
	var cmd tea.Cmd
	c.scriptSelector.input, cmd = c.scriptSelector.input.Update(msg)

	// Update suggestions if not in arbitrary mode
	if !c.scriptSelector.arbitraryMode {
		c.updateSuggestions()
	}

	return true, cmd
}

// handleEnter processes the enter key
func (c *ScriptSelectorController) handleEnter() (bool, tea.Cmd) {
	if c.scriptSelector.arbitraryMode {
		// Execute arbitrary command
		command := strings.TrimSpace(c.scriptSelector.input.Value())
		if command != "" {
			parts := strings.Fields(command)
			if len(parts) > 0 {
				cmdName := parts[0]
				args := parts[1:]

				// Start the command
				SafeGoroutine(
					fmt.Sprintf("start script selector command '%s'", cmdName),
					func() error {
						if c.processMgr == nil {
							return fmt.Errorf(ErrProcessManagerNotInitialized)
						}
						_, err := c.processMgr.StartCommand(command, cmdName, args)
						if err == nil && c.navController != nil {
							// Success - switch to logs view
							c.navController.SwitchTo(ViewLogs)
							if c.updateChan != nil {
								c.updateChan <- tea.Msg(nil)
							}
						}
						return err
					},
					func(err error) {
						errorMsg := fmt.Sprintf(ErrFailedToStartCommand, cmdName, args, err)
						c.scriptSelector.errorMessage = errorMsg
						if c.logStore != nil {
							c.logStore.Add("system", "System", errorMsg, true)
						}
						if c.updateChan != nil {
							c.updateChan <- tea.Msg(nil)
						}
					},
				)

				c.Hide()
			}
		}
		return true, nil
	}

	// Execute selected script
	if c.scriptSelector.selected >= 0 && c.scriptSelector.selected < len(c.scriptSelector.suggestions) {
		scriptName := c.scriptSelector.suggestions[c.scriptSelector.selected]

		SafeGoroutine(
			fmt.Sprintf("start selected script '%s'", scriptName),
			func() error {
				if c.processMgr == nil {
					return fmt.Errorf(ErrProcessManagerNotInitialized)
				}
				_, err := c.processMgr.StartScript(scriptName)
				if err == nil && c.navController != nil {
					// Success - switch to logs view
					c.navController.SwitchTo(ViewLogs)
					if c.updateChan != nil {
						c.updateChan <- tea.Msg(nil)
					}
				}
				return err
			},
			func(err error) {
				errorMsg := fmt.Sprintf(ErrFailedToStartScript, scriptName, err)
				c.scriptSelector.errorMessage = errorMsg
				if c.logStore != nil {
					c.logStore.Add("system", "System", errorMsg, true)
				}
				if c.updateChan != nil {
					c.updateChan <- tea.Msg(nil)
				}
			},
		)

		c.Hide()
	}

	return true, nil
}

// navigateUp moves selection up
func (c *ScriptSelectorController) navigateUp() {
	if len(c.scriptSelector.suggestions) > 0 && c.scriptSelector.showDropdown {
		if c.scriptSelector.selected > 0 {
			c.scriptSelector.selected--
		}
	}
}

// navigateDown moves selection down
func (c *ScriptSelectorController) navigateDown() {
	if len(c.scriptSelector.suggestions) > 0 && c.scriptSelector.showDropdown {
		if c.scriptSelector.selected < len(c.scriptSelector.suggestions)-1 {
			c.scriptSelector.selected++
		}
	}
}

// autocomplete fills in the selected suggestion
func (c *ScriptSelectorController) autocomplete() {
	if len(c.scriptSelector.suggestions) > 0 && c.scriptSelector.selected >= 0 {
		selected := c.scriptSelector.suggestions[c.scriptSelector.selected]
		c.scriptSelector.input.SetValue(selected)
		c.scriptSelector.showDropdown = false
	}
}

// updateSuggestions updates the suggestion list based on input
func (c *ScriptSelectorController) updateSuggestions() {
	inputValue := c.scriptSelector.input.Value()

	if inputValue == "" {
		// Show all scripts
		scripts := make([]string, 0, len(c.scriptSelector.availableScripts))
		for name := range c.scriptSelector.availableScripts {
			scripts = append(scripts, name)
		}
		c.scriptSelector.suggestions = scripts
		c.scriptSelector.showDropdown = true
	} else {
		// Filter scripts
		var filtered []string
		for script := range c.scriptSelector.availableScripts {
			if strings.HasPrefix(strings.ToLower(script), strings.ToLower(inputValue)) {
				filtered = append(filtered, script)
			}
		}
		c.scriptSelector.suggestions = filtered
		c.scriptSelector.showDropdown = len(filtered) > 0
	}

	// Reset selection if out of bounds
	if c.scriptSelector.selected >= len(c.scriptSelector.suggestions) {
		c.scriptSelector.selected = 0
	}
}

// View returns the rendered view
func (c *ScriptSelectorController) View() string {
	if !c.visible {
		return ""
	}
	
	// Update autocomplete width based on terminal size
	c.updateAutocompleteWidth()
	
	// Get terminal dimensions and max suggestions
	termWidth, termHeight := c.getTerminalDimensions()
	maxSuggestions := c.getMaxSuggestions(termHeight)
	
	// Render input and dropdown components
	input := c.scriptSelector.View()
	dropdown := c.scriptSelector.RenderDropdown(maxSuggestions)
	
	// Calculate layout dimensions
	contentHeight := c.calculateContentHeight(dropdown)
	topPadding := c.calculateTopPadding(termHeight, contentHeight)
	
	// Build the complete view
	return c.buildView(termWidth, termHeight, topPadding, input, dropdown)
}

// updateAutocompleteWidth sets the width of the autocomplete based on terminal size
func (c *ScriptSelectorController) updateAutocompleteWidth() {
	if termWidth, _, err := getTerminalSize(); err == nil && termWidth > 0 {
		autocompleteWidth := termWidth - ScriptSelectorMargin
		if autocompleteWidth < ScriptSelectorMinWidth {
			autocompleteWidth = ScriptSelectorMinWidth
		}
		if autocompleteWidth > ScriptSelectorMaxWidth {
			autocompleteWidth = ScriptSelectorMaxWidth
		}
		c.scriptSelector.SetWidth(autocompleteWidth)
	}
}

// getTerminalDimensions returns the terminal width and height with fallback values
func (c *ScriptSelectorController) getTerminalDimensions() (int, int) {
	termWidth, termHeight, _ := getTerminalSize()
	if termWidth == 0 || termHeight == 0 {
		// Fallback dimensions
		termWidth = 80
		termHeight = 24
	}
	return termWidth, termHeight
}

// getMaxSuggestions returns the maximum number of suggestions based on terminal height
func (c *ScriptSelectorController) getMaxSuggestions(termHeight int) int {
	if termHeight > 0 && termHeight < SmallTerminalHeightThreshold {
		return MaxDropdownSuggestionsSmall
	}
	return MaxDropdownSuggestions
}

// calculateContentHeight calculates the total height of the content
func (c *ScriptSelectorController) calculateContentHeight(dropdown string) int {
	contentHeight := 2 // input height
	if dropdown != "" {
		contentHeight += strings.Count(dropdown, "\n") + 1
	}
	return contentHeight
}

// calculateTopPadding calculates the top padding for vertical centering
func (c *ScriptSelectorController) calculateTopPadding(termHeight, contentHeight int) int {
	topPadding := (termHeight - contentHeight - TitleAndHelpTextHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	return topPadding
}

// buildView constructs the complete view with all components using Lipgloss layouts
func (c *ScriptSelectorController) buildView(termWidth, termHeight, topPadding int, input, dropdown string) string {
	// Create base container style
	containerStyle := lipgloss.NewStyle().
		Width(termWidth).
		Height(termHeight).
		Align(lipgloss.Center, lipgloss.Center)
	
	// Create content sections
	title := c.renderTitle(termWidth)
	content := c.renderContent(termWidth, input, dropdown)
	helpText := c.renderHelpText(termWidth)
	
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
func (c *ScriptSelectorController) renderTitle(termWidth int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("226")).
		Width(termWidth).
		Align(lipgloss.Center).
		MarginBottom(1)
	
	return titleStyle.Render("🐝 Select a Script to Run")
}

// renderContent creates the input and dropdown section using Lipgloss
func (c *ScriptSelectorController) renderContent(termWidth int, input, dropdown string) string {
	// Create content container with proper width
	contentWidth := c.scriptSelector.width
	if contentWidth > termWidth-10 {
		contentWidth = termWidth - 10
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
func (c *ScriptSelectorController) renderHelpText(termWidth int) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Width(termWidth).
		Align(lipgloss.Center).
		MarginTop(1)
	
	return helpStyle.Render("↑/↓ Navigate • Enter Run • Tab Complete • Esc Cancel")
}

// GetScriptSelector returns the underlying CommandAutocomplete for backward compatibility
func (c *ScriptSelectorController) GetScriptSelector() *CommandAutocomplete {
	return &c.scriptSelector
}
