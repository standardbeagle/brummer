package components

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// OptimizedCommandInputComponent provides high-performance command input with debounced suggestions
// Uses caching and debouncing to eliminate input stalls during typing
type OptimizedCommandInputComponent struct {
	BaseComponent

	input              textinput.Model
	registry           *CommandRegistry
	active             bool
	suggestions        []string
	selectedSuggestion int
	showSuggestions    bool
	currentCommand     *messages.Command
	argumentIndex      int

	// Performance optimizations
	lastInputValue     string        // Cache to avoid unnecessary updates
	cachedSuggestions  map[string][]string // Cache for filtered suggestions
	suggestionMutex    sync.RWMutex  // Thread-safe cache access
	debounceTimer      *time.Timer   // Debounce timer for suggestion updates
	debounceDelay      time.Duration // Delay for debouncing (100ms for responsive feel)
	updatePending      bool          // Flag to prevent multiple concurrent updates
}

// NewOptimizedCommandInputComponent creates a new optimized command input
func NewOptimizedCommandInputComponent(registry *CommandRegistry) *OptimizedCommandInputComponent {
	input := textinput.New()
	input.Placeholder = "Enter command..."
	input.Prompt = "/"
	input.CharLimit = 200

	component := &OptimizedCommandInputComponent{
		registry:          registry,
		input:             input,
		selectedSuggestion: 0,
		cachedSuggestions: make(map[string][]string),
		debounceDelay:     100 * time.Millisecond, // Responsive but not too aggressive
	}

	// Pre-populate cache with all commands
	component.prepopulateCache()

	return component
}

// prepopulateCache fills the cache with common command lookups
func (c *OptimizedCommandInputComponent) prepopulateCache() {
	c.suggestionMutex.Lock()
	defer c.suggestionMutex.Unlock()

	// Cache all commands for empty prefix
	allCommands := c.registry.GetNames()
	c.cachedSuggestions[""] = allCommands

	// Cache common single-letter prefixes
	commonPrefixes := []string{"a", "c", "h", "l", "p", "r", "s", "q", "f"}
	for _, prefix := range commonPrefixes {
		c.cachedSuggestions[prefix] = c.filterCommandsUncached(prefix)
	}
}

// IsActive returns whether the command input is currently active
func (c *OptimizedCommandInputComponent) IsActive() bool {
	return c.active
}

// Open activates the command input
func (c *OptimizedCommandInputComponent) Open() {
	c.active = true
	c.input.Reset()
	c.input.Focus()
	c.lastInputValue = ""
	c.updateSuggestionsImmediate() // Immediate update on open
}

// Close deactivates the command input
func (c *OptimizedCommandInputComponent) Close() {
	c.active = false
	c.input.Blur()
	c.input.Reset()
	c.suggestions = nil
	c.selectedSuggestion = 0
	c.showSuggestions = false
	c.currentCommand = nil
	c.argumentIndex = 0
	c.lastInputValue = ""

	// Cancel any pending debounce timer
	if c.debounceTimer != nil {
		c.debounceTimer.Stop()
		c.debounceTimer = nil
	}
}

// Init initializes the component
func (c *OptimizedCommandInputComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages with optimized suggestion updates
func (c *OptimizedCommandInputComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				c.updateSuggestionsImmediate() // Immediate update for tab completion
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
			// Update input and schedule debounced suggestions
			var cmd tea.Cmd
			c.input, cmd = c.input.Update(msg)
			c.scheduleSuggestionsUpdate()
			return c, cmd
		}
	}

	return c, nil
}

// scheduleSuggestionsUpdate schedules a debounced suggestion update
func (c *OptimizedCommandInputComponent) scheduleSuggestionsUpdate() {
	currentValue := c.input.Value()

	// Skip if input hasn't actually changed
	if currentValue == c.lastInputValue {
		return
	}

	c.lastInputValue = currentValue

	// Cancel existing timer
	if c.debounceTimer != nil {
		c.debounceTimer.Stop()
	}

	// Start new timer
	c.debounceTimer = time.AfterFunc(c.debounceDelay, func() {
		c.updateSuggestionsImmediate()
	})
}

// updateSuggestionsImmediate performs an immediate suggestion update
func (c *OptimizedCommandInputComponent) updateSuggestionsImmediate() {
	if !c.active {
		return
	}

	value := strings.TrimSpace(c.input.Value())

	// Remove leading slash for parsing
	value = strings.TrimPrefix(value, "/")

	parts := strings.Fields(value)

	if len(parts) == 0 {
		// Show all commands (cached)
		c.suggestions = c.getCachedSuggestions("")
		c.showSuggestions = true
		c.selectedSuggestion = 0
		c.currentCommand = nil
		c.argumentIndex = 0
		return
	}

	commandName := parts[0]

	// If we're still typing the command name (no space after)
	if !strings.HasSuffix(c.input.Value(), " ") && len(parts) == 1 {
		// Use cached filtered commands
		c.suggestions = c.getCachedSuggestions(commandName)
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

// getCachedSuggestions retrieves suggestions from cache, computing if necessary
func (c *OptimizedCommandInputComponent) getCachedSuggestions(prefix string) []string {
	c.suggestionMutex.RLock()
	if suggestions, exists := c.cachedSuggestions[prefix]; exists {
		c.suggestionMutex.RUnlock()
		return suggestions
	}
	c.suggestionMutex.RUnlock()

	// Cache miss - compute and cache
	c.suggestionMutex.Lock()
	defer c.suggestionMutex.Unlock()

	// Double-check after acquiring write lock
	if suggestions, exists := c.cachedSuggestions[prefix]; exists {
		return suggestions
	}

	suggestions := c.filterCommandsUncached(prefix)
	c.cachedSuggestions[prefix] = suggestions

	// Limit cache size to prevent memory bloat
	if len(c.cachedSuggestions) > 100 {
		c.evictOldestCacheEntries()
	}

	return suggestions
}

// filterCommandsUncached performs the actual command filtering (without cache)
func (c *OptimizedCommandInputComponent) filterCommandsUncached(prefix string) []string {
	allCommands := c.registry.GetNames()
	if prefix == "" {
		return allCommands
	}

	// Optimize for common cases
	if len(prefix) == 1 {
		return c.filterSingleLetterPrefix(prefix, allCommands)
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

// filterSingleLetterPrefix optimized filtering for single-letter prefixes
func (c *OptimizedCommandInputComponent) filterSingleLetterPrefix(prefix string, allCommands []string) []string {
	if len(prefix) != 1 {
		return allCommands
	}

	lowerPrefix := strings.ToLower(prefix)
	var filtered []string
	for _, cmd := range allCommands {
		if len(cmd) > 0 && strings.ToLower(string(cmd[0])) == lowerPrefix {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// evictOldestCacheEntries removes oldest cache entries to prevent memory bloat
func (c *OptimizedCommandInputComponent) evictOldestCacheEntries() {
	// Keep only the most recent 50 entries
	count := 0
	for key := range c.cachedSuggestions {
		if count > 50 {
			delete(c.cachedSuggestions, key)
		}
		count++
	}
}

// View renders the command input
func (c *OptimizedCommandInputComponent) View() string {
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

	// Suggestions dropdown (if any) - optimized rendering
	var suggestionsBox string
	if c.showSuggestions && len(c.suggestions) > 0 {
		var lines []string
		maxSuggestions := 8
		suggestionCount := len(c.suggestions)
		if suggestionCount < maxSuggestions {
			maxSuggestions = suggestionCount
		}

		// Pre-allocate slice for better performance
		lines = make([]string, 0, maxSuggestions)
		for i := 0; i < maxSuggestions; i++ {
			suggestion := c.suggestions[i]

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
func (c *OptimizedCommandInputComponent) parseCommand(input string) messages.CommandExecuteMsg {
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

// filterSuggestions returns suggestions matching the prefix (optimized)
func (c *OptimizedCommandInputComponent) filterSuggestions(suggestions []string, prefix string) []string {
	if prefix == "" {
		return suggestions
	}

	if len(prefix) == 1 {
		// Optimized single-letter prefix filtering
		var filtered []string
		lowerPrefix := strings.ToLower(prefix)
		for _, suggestion := range suggestions {
			if len(suggestion) > 0 && strings.ToLower(string(suggestion[0])) == lowerPrefix {
				filtered = append(filtered, suggestion)
			}
		}
		return filtered
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
func (c *OptimizedCommandInputComponent) applySuggestion() {
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

// ClearCache clears the suggestion cache (useful for testing or when commands change)
func (c *OptimizedCommandInputComponent) ClearCache() {
	c.suggestionMutex.Lock()
	defer c.suggestionMutex.Unlock()

	c.cachedSuggestions = make(map[string][]string)
	c.prepopulateCache()
}