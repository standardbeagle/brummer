package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/standardbeagle/brummer/internal/config"
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// CommandAutocomplete handles multi-level command completion
type CommandAutocomplete struct {
	input         textinput.Model
	segments      []string // Parsed command segments
	currentIndex  int      // Which segment we're editing
	suggestions   []string // Current suggestions
	selected      int      // Selected suggestion index
	showDropdown  bool
	width         int
	errorMessage  string // Error message to display
	arbitraryMode bool   // True when in arbitrary command mode (Ctrl+N)

	// Command-specific data
	availableScripts map[string]string   // Script name -> script command
	processMgr       *processes.Manager  // Reference to process manager to check running scripts
	cfg              *config.Config      // Configuration for AI providers and other settings
}

func NewCommandAutocomplete(scripts map[string]string) CommandAutocomplete {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = "/"
	ti.Focus()
	ti.CharLimit = 100

	c := CommandAutocomplete{
		input:            ti,
		availableScripts: scripts,
		showDropdown:     true,
	}

	// Initialize suggestions
	c.updateSuggestions()

	return c
}

// NewCommandAutocompleteWithProcessManager creates a command autocomplete with process manager and config
func NewCommandAutocompleteWithProcessManager(scripts map[string]string, processMgr *processes.Manager, cfg *config.Config) CommandAutocomplete {
	c := NewCommandAutocomplete(scripts)
	c.processMgr = processMgr
	c.cfg = cfg
	return c
}

func (c CommandAutocomplete) Init() tea.Cmd {
	return textinput.Blink
}

func (c CommandAutocomplete) Update(msg tea.Msg) (CommandAutocomplete, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Validate and execute command
			if isValid, errorMsg := c.ValidateInput(); isValid {
				// Parse command and create execute message
				value := c.input.Value()
				parts := strings.Fields(value)
				if len(parts) == 0 {
					c.errorMessage = "Please enter a command"
					return c, nil
				}

				// Remove leading slash from command name
				commandName := strings.TrimPrefix(parts[0], "/")
				arguments := parts[1:]

				return c, func() tea.Msg {
					return messages.CommandExecuteMsg{
						CommandName: commandName,
						Arguments:   arguments,
					}
				}
			} else {
				// Show error message
				c.errorMessage = errorMsg
				return c, nil
			}

		case "tab":
			if c.showDropdown && len(c.suggestions) > 0 {
				c.applySelectedSuggestion()
				c.updateSuggestions()
			}
			return c, nil

		case "up":
			if c.showDropdown && c.selected > 0 {
				c.selected--
			}
			return c, nil

		case "down":
			if c.showDropdown && c.selected < len(c.suggestions)-1 {
				c.selected++
			}
			return c, nil

		default:
			// Update input
			prevValue := c.input.Value()
			c.input, cmd = c.input.Update(msg)

			// Update suggestions if value changed
			if c.input.Value() != prevValue {
				c.updateSuggestions()
				// Clear error when user is typing
				c.errorMessage = ""
			}
			return c, cmd
		}
	}

	return c, nil
}

func (c *CommandAutocomplete) SetWidth(width int) {
	c.width = width
	c.input.Width = width - 4 // Account for borders/padding
}


func (c *CommandAutocomplete) Value() string {
	return c.input.Value()
}

func (c *CommandAutocomplete) SetValue(value string) {
	c.input.SetValue(value)
	c.updateSuggestions()
}

func (c *CommandAutocomplete) Focus() {
	c.input.Focus()
}

func (c *CommandAutocomplete) Blur() {
	c.input.Blur()
}

func (c *CommandAutocomplete) updateSuggestions() {
	value := c.input.Value()
	// Add the slash prefix since it's in the prompt
	if !strings.HasPrefix(value, "/") && value != "" {
		value = "/" + value
	}
	c.segments = strings.Fields(value)

	// Determine which segment we're currently editing
	if strings.HasSuffix(value, " ") {
		// We're starting a new segment
		c.currentIndex = len(c.segments)
		c.segments = append(c.segments, "")
	} else if len(c.segments) > 0 {
		c.currentIndex = len(c.segments) - 1
	} else {
		c.currentIndex = 0
		c.segments = []string{""}
	}

	// Get suggestions based on the command path
	c.suggestions = c.getSuggestionsForCurrentPosition()
	c.selected = 0
	c.showDropdown = len(c.suggestions) > 0

	// Always show dropdown if we have suggestions or if we're at the beginning
	if len(c.suggestions) == 0 && c.currentIndex == 0 && (value == "" || value == "/") {
		// Show initial commands when empty (with / prefix for clarity)
		c.suggestions = []string{"/ai", "/run", "/restart", "/stop", "/clear", "/show", "/hide", "/proxy", "/toggle-proxy", "/term", "/docker-up", "/docker-down", "/docker-build", "/compose-up", "/compose-down", "/help", "/exit", "/quit"}
		c.showDropdown = true
	}

	// Ensure selected index is valid
	if c.selected >= len(c.suggestions) {
		c.selected = 0
	}
}

func (c *CommandAutocomplete) getSuggestionsForCurrentPosition() []string {
	if c.currentIndex == 0 {
		// First segment - show root commands (with / prefix)
		rootCommands := []string{"/ai", "/run", "/restart", "/stop", "/clear", "/show", "/hide", "/proxy", "/toggle-proxy", "/term", "/docker-up", "/docker-down", "/docker-build", "/compose-up", "/compose-down", "/help", "/exit", "/quit"}
		currentText := ""
		if len(c.segments) > 0 {
			currentText = c.segments[0]
			// Commands are already prefixed with /, so match directly
		}
		return c.filterSuggestions(rootCommands, currentText)
	}

	// For subsequent segments, look up based on the first command
	if c.currentIndex == 1 && len(c.segments) > 0 {
		switch c.segments[0] {
		case "/run":
			// Get script names, excluding already running ones
			scripts := make([]string, 0, len(c.availableScripts))

			// Get running scripts if process manager is available
			runningScripts := make(map[string]bool)
			if c.processMgr != nil {
				for _, proc := range c.processMgr.GetAllProcesses() {
					if proc.GetStatus() == processes.StatusRunning {
						runningScripts[proc.Name] = true
					}
				}
			}

			// Only add scripts that aren't already running
			for name := range c.availableScripts {
				if !runningScripts[name] {
					scripts = append(scripts, name)
				}
			}
			sort.Strings(scripts)

			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(scripts, currentText)

		case "/restart", "/stop":
			// Get running processes with "all" as default option
			processNames := []string{"all"}

			if c.processMgr != nil {
				for _, proc := range c.processMgr.GetAllProcesses() {
					if proc.GetStatus() == processes.StatusRunning {
						processNames = append(processNames, proc.Name)
					}
				}
			}

			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(processNames, currentText)

		case "/clear":
			// Options: all, logs, errors, web, or script names
			options := []string{"all", "logs", "errors", "web"}

			// Add all script names (both running and not running)
			if c.availableScripts != nil {
				for scriptName := range c.availableScripts {
					options = append(options, scriptName)
				}
			}

			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(options, currentText)

		case "/show", "/hide":
			// Common patterns for log filtering
			patterns := []string{"error", "warn", "info", "debug", "^\\[", "\\]$", "|"}
			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(patterns, currentText)

		case "/proxy":
			// First argument: subcommands or URL suggestions
			if c.currentIndex == 1 {
				subcommands := []string{"list", "http://localhost:3000", "http://localhost:8080", "https://example.com"}
				currentText := ""
				if c.currentIndex < len(c.segments) {
					currentText = c.segments[c.currentIndex]
				}
				return c.filterSuggestions(subcommands, currentText)
			}
			// For subsequent arguments, suggest URL formats if not "list"
			if len(c.segments) > 1 && c.segments[1] != "list" {
				urlExamples := []string{"http://localhost:3000", "http://localhost:8080", "https://example.com"}
				currentText := ""
				if c.currentIndex < len(c.segments) {
					currentText = c.segments[c.currentIndex]
				}
				return c.filterSuggestions(urlExamples, currentText)
			}
			return []string{}

		case "/ai":
			// Suggest AI providers from configuration
			if c.currentIndex == 1 {
				// Get available AI providers from config
				var providers []string
				if c.cfg != nil && c.cfg.AICoders != nil && c.cfg.AICoders.Providers != nil && len(c.cfg.AICoders.Providers) > 0 {
					// Extract provider names from config
					for providerName := range c.cfg.AICoders.Providers {
						providers = append(providers, providerName)
					}
					// Sort for consistent ordering
					sort.Strings(providers)
				} else {
					// Fallback if config not available - use comprehensive list from .brum.example.toml
					providers = []string{
						"aider", "bash", "claude", "claude-secure", "copilot", "copilot-secure",
						"fish", "gemini", "gemini-secure", "local", "opencode", "opus", "sonnet", "zsh",
					}
				}

				currentText := ""
				if c.currentIndex < len(c.segments) {
					currentText = c.segments[c.currentIndex]
				}
				return c.filterSuggestions(providers, currentText)
			}
			return []string{}

		case "/docker-up", "/docker-down":
			// Suggest common container names or "all"
			options := []string{"all"}
			// TODO: Could be enhanced to list actual running containers from docker ps
			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(options, currentText)

		case "/docker-build":
			// Suggest common tag patterns
			tags := []string{"latest", "dev", "prod", "v1.0.0"}
			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(tags, currentText)

		case "/compose-up", "/compose-down":
			// Suggest service names or empty for all services
			// TODO: Could be enhanced to parse docker-compose.yml for actual service names
			options := []string{"all", "web", "api", "db", "redis", "worker"}
			currentText := ""
			if c.currentIndex < len(c.segments) {
				currentText = c.segments[c.currentIndex]
			}
			return c.filterSuggestions(options, currentText)
		}
	}

	return []string{}
}

func (c *CommandAutocomplete) filterSuggestions(options []string, prefix string) []string {
	if prefix == "" {
		return options
	}

	var filtered []string
	prefixLower := strings.ToLower(prefix)
	for _, opt := range options {
		if strings.HasPrefix(strings.ToLower(opt), prefixLower) {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

func (c *CommandAutocomplete) applySelectedSuggestion() {
	if len(c.suggestions) == 0 || c.selected >= len(c.suggestions) {
		return
	}

	selected := c.suggestions[c.selected]

	// Build the new command string
	parts := strings.Fields(c.input.Value())
	if strings.HasSuffix(c.input.Value(), " ") {
		// We're completing a new segment
		parts = append(parts, selected)
	} else if len(parts) > 0 {
		// Replace the last segment
		parts[len(parts)-1] = selected
	} else {
		parts = []string{selected}
	}

	// Set the new value with a trailing space (except for commands)
	newValue := strings.Join(parts, " ")
	if c.currentIndex == 0 {
		newValue += " "
	}
	c.input.SetValue(newValue)
	c.input.CursorEnd()
}

func (c CommandAutocomplete) View() string {
	// Just return the input view - the dropdown will be rendered separately
	return c.input.View()
}

func (c CommandAutocomplete) RenderDropdown(maxSuggestions int) string {
	if !c.showDropdown || len(c.suggestions) == 0 {
		return ""
	}

	var s strings.Builder

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Background(lipgloss.Color("237")).
		Width(c.width - 4)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(c.width - 4)

	count := len(c.suggestions)
	if count > maxSuggestions {
		count = maxSuggestions
	}

	for i := 0; i < count; i++ {
		suggestion := c.suggestions[i]

		// Add visual indicator for selection
		display := "  " + suggestion
		if i == c.selected {
			display = "▶ " + suggestion
			s.WriteString(selectedStyle.Render(display))
		} else {
			s.WriteString(normalStyle.Render(display))
		}

		if i < count-1 {
			s.WriteString("\n")
		}
	}

	// Show more indicator if there are more suggestions
	if len(c.suggestions) > maxSuggestions {
		moreStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		s.WriteString("\n")
		moreCount := len(c.suggestions) - maxSuggestions
		s.WriteString(moreStyle.Render(fmt.Sprintf("  ... and %d more", moreCount)))
	}

	return s.String()
}

func (c CommandAutocomplete) GetSuggestions() []string {
	return c.suggestions
}

func (c CommandAutocomplete) GetSelected() int {
	return c.selected
}

// ValidateInput checks if the current input is valid and can be executed
func (c *CommandAutocomplete) ValidateInput() (bool, string) {
	value := c.input.Value()
	if value == "" {
		return false, "Please enter a command"
	}

	// If it doesn't start with /, treat as arbitrary command (always valid)
	if !strings.HasPrefix(value, "/") {
		return true, ""
	}

	parts := strings.Fields(value)
	if len(parts) == 0 {
		return false, "Please enter a command"
	}

	command := parts[0]

	switch command {
	case "/run":
		if len(parts) < 2 {
			return false, "Please specify a script name (e.g., /run dev)"
		}
		scriptName := parts[1]
		// Check if script exists
		if _, exists := c.availableScripts[scriptName]; !exists {
			return false, fmt.Sprintf("Script '%s' not found. Available: %s", scriptName, c.getAvailableScriptsString())
		}
		// Check if script is already running
		if c.processMgr != nil {
			for _, proc := range c.processMgr.GetAllProcesses() {
				if proc.Name == scriptName && proc.GetStatus() == processes.StatusRunning {
					return false, fmt.Sprintf("Script '%s' is already running", scriptName)
				}
			}
		}
		return true, ""

	case "/restart", "/stop":
		if len(parts) < 2 {
			// Default to "all" if no process specified
			return true, ""
		}
		processName := parts[1]

		// Check if it's "all" or a valid running process
		if processName == "all" {
			return true, ""
		}

		// Check if process exists and is running
		if c.processMgr != nil {
			for _, proc := range c.processMgr.GetAllProcesses() {
				if proc.Name == processName && proc.GetStatus() == processes.StatusRunning {
					return true, ""
				}
			}
			return false, fmt.Sprintf("Process '%s' is not running", processName)
		}
		return true, ""

	case "/clear":
		if len(parts) < 2 {
			// Default to "all" if no target specified
			return true, ""
		}
		target := parts[1]

		// Check if it's a valid clear target
		validTargets := map[string]bool{
			"all":    true,
			"logs":   true,
			"errors": true,
			"web":    true,
		}

		if validTargets[target] {
			return true, ""
		}

		// Check if it's a valid script name
		if _, exists := c.availableScripts[target]; exists {
			return true, ""
		}

		return false, fmt.Sprintf("Invalid clear target '%s'. Use: all, logs, errors, or a script name", target)

	case "/show", "/hide":
		if len(parts) < 2 {
			return false, fmt.Sprintf("Please specify a pattern for %s", command)
		}
		return true, ""

	case "/proxy":
		if len(parts) < 2 {
			return false, "Please specify a subcommand or URL (e.g. /proxy list or /proxy http://localhost:3000)"
		}

		subcommand := parts[1]
		if subcommand == "list" {
			return true, ""
		}

		// Validate as URL
		urlStr := strings.Join(parts[1:], " ")
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			// Allow URLs without scheme - we'll add http:// automatically
			if strings.Contains(urlStr, "://") {
				return false, "URL must start with http:// or https://"
			}
		}
		return true, ""

	case "/toggle-proxy":
		// No additional parameters needed
		return true, ""

	case "/ai":
		if len(parts) < 2 {
			return false, "Please specify an AI provider (e.g., /ai claude)"
		}
		providerName := parts[1]
		// Validate against available providers from config
		validProviders := make(map[string]bool)

		// Try to get providers from config
		if c.cfg != nil && c.cfg.AICoders != nil && c.cfg.AICoders.Providers != nil && len(c.cfg.AICoders.Providers) > 0 {
			for name := range c.cfg.AICoders.Providers {
				validProviders[name] = true
			}
		} else {
			// Comprehensive fallback list matching .brum.example.toml
			fallbackProviders := []string{
				"aider", "bash", "claude", "claude-secure", "copilot", "copilot-secure",
				"fish", "gemini", "gemini-secure", "local", "opencode", "opus", "sonnet", "zsh",
			}
			for _, name := range fallbackProviders {
				validProviders[name] = true
			}
		}

		if !validProviders[providerName] {
			// Build available providers list for error message
			availableList := make([]string, 0, len(validProviders))
			for name := range validProviders {
				availableList = append(availableList, name)
			}
			return false, fmt.Sprintf("Invalid AI provider '%s'. Available: %s", providerName, strings.Join(availableList, ", "))
		}
		return true, ""

	case "/term":
		// No additional parameters needed for terminal
		return true, ""

	case "/help":
		// No additional parameters needed
		return true, ""

	case "/docker-up":
		// Optional container name parameter
		return true, ""

	case "/docker-down":
		// Optional container name parameter
		return true, ""

	case "/docker-build":
		// Optional tag parameter (defaults to "latest")
		return true, ""

	case "/compose-up":
		// Optional service name parameter
		return true, ""

	case "/compose-down":
		// Optional service name parameter
		return true, ""

	case "/exit", "/quit":
		// No additional parameters needed
		return true, ""

	default:
		// Check if it's a partial command
		for _, cmd := range []string{"run", "restart", "stop", "clear", "show", "hide", "proxy", "toggle-proxy", "ai", "term", "docker-up", "docker-down", "docker-build", "compose-up", "compose-down", "help", "exit", "quit"} {
			if strings.HasPrefix(cmd, strings.TrimPrefix(command, "/")) {
				return false, fmt.Sprintf("Incomplete command. Did you mean /%s?", cmd)
			}
		}
		return false, fmt.Sprintf("Unknown command: %s. Available commands: /run, /restart, /stop, /clear, /show, /hide, /proxy, /toggle-proxy, /ai, /term, /docker-up, /docker-down, /docker-build, /compose-up, /compose-down, /help, /exit, /quit", command)
	}
}

func (c *CommandAutocomplete) getAvailableScriptsString() string {
	// Get running scripts if process manager is available
	runningScripts := make(map[string]bool)
	if c.processMgr != nil {
		for _, proc := range c.processMgr.GetAllProcesses() {
			if proc.GetStatus() == processes.StatusRunning {
				runningScripts[proc.Name] = true
			}
		}
	}

	// Only show scripts that aren't already running
	scripts := make([]string, 0, len(c.availableScripts))
	for name := range c.availableScripts {
		if !runningScripts[name] {
			scripts = append(scripts, name)
		}
	}
	sort.Strings(scripts)
	if len(scripts) > 5 {
		return strings.Join(scripts[:5], ", ") + "..."
	}
	return strings.Join(scripts, ", ")
}

// GetErrorMessage returns the current error message
func (c CommandAutocomplete) GetErrorMessage() string {
	return c.errorMessage
}

// ClearError clears the error message
func (c *CommandAutocomplete) ClearError() {
	c.errorMessage = ""
}

// SetError sets an error message
func (c *CommandAutocomplete) SetError(message string) {
	c.errorMessage = message
}

// Accessor methods for ScriptSelectorAutocompleteInterface

// SetPlaceholder sets the input placeholder text
func (c *CommandAutocomplete) SetPlaceholder(placeholder string) {
	c.input.Placeholder = placeholder
}

// SetSelected sets the selected suggestion index
func (c *CommandAutocomplete) SetSelected(index int) {
	c.selected = index
}

// SetSuggestions sets the suggestion list
func (c *CommandAutocomplete) SetSuggestions(suggestions []string) {
	c.suggestions = suggestions
}

// IsShowingDropdown returns whether the dropdown is visible
func (c CommandAutocomplete) IsShowingDropdown() bool {
	return c.showDropdown
}

// SetShowDropdown sets dropdown visibility
func (c *CommandAutocomplete) SetShowDropdown(show bool) {
	c.showDropdown = show
}

// SetErrorMessage sets an error message (same as SetError but matches interface)
func (c *CommandAutocomplete) SetErrorMessage(msg string) {
	c.errorMessage = msg
}

// IsArbitraryMode returns whether in arbitrary command mode
func (c CommandAutocomplete) IsArbitraryMode() bool {
	return c.arbitraryMode
}

// SetArbitraryMode sets arbitrary command mode
func (c *CommandAutocomplete) SetArbitraryMode(mode bool) {
	c.arbitraryMode = mode
}

// UpdateSuggestions updates suggestions (delegates to updateSuggestions)
func (c *CommandAutocomplete) UpdateSuggestions() {
	c.updateSuggestions()
}

// GetAvailableScripts returns the map of available scripts
func (c CommandAutocomplete) GetAvailableScripts() map[string]string {
	return c.availableScripts
}
