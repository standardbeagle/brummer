package state

import (
	"time"

	"github.com/standardbeagle/brummer/internal/parser"
)

// CommandWindowState manages all command window UI state
// This is a sub-model that encapsulates command window display according to Elm/Redux patterns
type CommandWindowState struct {
	// Command window state
	ShowingCommandWindow bool
	CommandWindowWidth   int
	CommandInputText     string   // Text being typed in command palette
	CommandSuggestions   []string // Autocomplete suggestions

	// Run dialog state
	ShowingRunDialog     bool
	DetectedCommands     []parser.ExecutableCommand
	MonorepoInfo         *parser.MonorepoInfo
	SelectedCommandIndex int

	// Custom command dialog state
	ShowingCustomCommand bool
	CustomCommandText    string

	// Terminal size state
	TerminalWidth  int
	TerminalHeight int

	// UI state
	Loading     bool
	LastError   string
	LastUpdated time.Time

	// Version for atomic updates and state tracking
	version uint64
}

// NewCommandWindowState creates a new command window sub-model with initial state
func NewCommandWindowState() *CommandWindowState {
	return &CommandWindowState{
		ShowingCommandWindow: false,
		CommandWindowWidth:   60, // DefaultCommandWindowWidth
		CommandInputText:     "",
		ShowingRunDialog:     false,
		DetectedCommands:     []parser.ExecutableCommand{},
		MonorepoInfo:         nil,
		SelectedCommandIndex: 0,
		ShowingCustomCommand: false,
		CustomCommandText:    "",
		TerminalWidth:        80,
		TerminalHeight:       24,
		Loading:              false,
		LastError:            "",
		LastUpdated:          time.Time{},
		version:              0,
	}
}

// ShowCommandWindow shows the command palette window (immutable)
func (s *CommandWindowState) ShowCommandWindow() *CommandWindowState {

	newState := s.copy()
	newState.ShowingCommandWindow = true
	newState.incrementVersion()
	return newState
}

// HideCommandWindow hides the command palette window (immutable)
func (s *CommandWindowState) HideCommandWindow() *CommandWindowState {

	newState := s.copy()
	newState.ShowingCommandWindow = false
	newState.incrementVersion()
	return newState
}

// SetCommandWindowWidth updates the command window width (immutable)
func (s *CommandWindowState) SetCommandWindowWidth(width int) *CommandWindowState {

	newState := s.copy()
	newState.CommandWindowWidth = width
	newState.incrementVersion()
	return newState
}

// ShowRunDialog shows the run dialog with detected commands (immutable)
func (s *CommandWindowState) ShowRunDialog(commands []parser.ExecutableCommand, monorepoInfo *parser.MonorepoInfo) *CommandWindowState {

	newState := s.copy()
	newState.ShowingRunDialog = true
	newState.DetectedCommands = make([]parser.ExecutableCommand, len(commands))
	copy(newState.DetectedCommands, commands)

	if monorepoInfo != nil {
		newState.MonorepoInfo = monorepoInfo
	}

	newState.SelectedCommandIndex = 0
	newState.incrementVersion()
	return newState
}

// HideRunDialog hides the run dialog (immutable)
func (s *CommandWindowState) HideRunDialog() *CommandWindowState {

	newState := s.copy()
	newState.ShowingRunDialog = false
	newState.incrementVersion()
	return newState
}

// SetSelectedCommand updates the selected command index (immutable)
func (s *CommandWindowState) SetSelectedCommand(index int) *CommandWindowState {

	newState := s.copy()
	if index >= 0 && index < len(newState.DetectedCommands) {
		newState.SelectedCommandIndex = index
	}
	newState.incrementVersion()
	return newState
}

// GetSelectedCommand returns the currently selected command
func (s *CommandWindowState) GetSelectedCommand() *parser.ExecutableCommand {

	if !s.ShowingRunDialog || len(s.DetectedCommands) == 0 {
		return nil
	}

	if s.SelectedCommandIndex >= 0 && s.SelectedCommandIndex < len(s.DetectedCommands) {
		return &s.DetectedCommands[s.SelectedCommandIndex]
	}

	return nil
}

// ShowCustomCommandDialog shows the custom command input dialog (immutable)
func (s *CommandWindowState) ShowCustomCommandDialog() *CommandWindowState {

	newState := s.copy()
	newState.ShowingCustomCommand = true
	newState.CustomCommandText = ""
	newState.incrementVersion()
	return newState
}

// HideCustomCommandDialog hides the custom command dialog (immutable)
func (s *CommandWindowState) HideCustomCommandDialog() *CommandWindowState {

	newState := s.copy()
	newState.ShowingCustomCommand = false
	newState.incrementVersion()
	return newState
}

// SetCustomCommandText updates the custom command text (immutable)
func (s *CommandWindowState) SetCustomCommandText(text string) *CommandWindowState {

	newState := s.copy()
	newState.CustomCommandText = text
	newState.incrementVersion()
	return newState
}

// SetTerminalSize updates the terminal size (immutable)
func (s *CommandWindowState) SetTerminalSize(width, height int) *CommandWindowState {

	newState := s.copy()
	newState.TerminalWidth = width
	newState.TerminalHeight = height
	newState.incrementVersion()
	return newState
}

// SetLoading updates the loading state (immutable)
func (s *CommandWindowState) SetLoading(loading bool) *CommandWindowState {

	newState := s.copy()
	newState.Loading = loading
	newState.incrementVersion()
	return newState
}

// SetError updates the error state (immutable)
func (s *CommandWindowState) SetError(err error) *CommandWindowState {

	newState := s.copy()
	if err != nil {
		newState.LastError = err.Error()
	} else {
		newState.LastError = ""
	}
	newState.Loading = false
	newState.incrementVersion()
	return newState
}

// ClearError clears the error state (immutable)
func (s *CommandWindowState) ClearError() *CommandWindowState {

	newState := s.copy()
	newState.LastError = ""
	newState.incrementVersion()
	return newState
}

// UpdateLastUpdated updates the last updated timestamp (immutable)
func (s *CommandWindowState) UpdateLastUpdated() *CommandWindowState {

	newState := s.copy()
	newState.LastUpdated = time.Now()
	newState.incrementVersion()
	return newState
}

// GetStateSummary returns a summary of current state
func (s *CommandWindowState) GetStateSummary() map[string]interface{} {

	return map[string]interface{}{
		"showing_command_window": s.ShowingCommandWindow,
		"command_window_width":   s.CommandWindowWidth,
		"showing_run_dialog":     s.ShowingRunDialog,
		"detected_commands":      len(s.DetectedCommands),
		"selected_command_index": s.SelectedCommandIndex,
		"showing_custom_command": s.ShowingCustomCommand,
		"custom_command_text":    s.CustomCommandText,
		"terminal_width":         s.TerminalWidth,
		"terminal_height":        s.TerminalHeight,
		"loading":                s.Loading,
		"last_error":             s.LastError,
		"last_updated":           s.LastUpdated,
		"version":                s.version,
	}
}

// Validate checks for state consistency
func (s *CommandWindowState) Validate() []string {

	var issues []string

	// Basic validation
	if s.CommandWindowWidth <= 0 {
		issues = append(issues, "invalid command window width")
	}

	if s.TerminalWidth <= 0 || s.TerminalHeight <= 0 {
		issues = append(issues, "invalid terminal size")
	}

	// Validate selected command index
	if s.SelectedCommandIndex < 0 || s.SelectedCommandIndex >= len(s.DetectedCommands) {
		issues = append(issues, "selected command index out of bounds")
	}

	// Validate dialog states - only one should be visible at a time
	visibleDialogs := 0
	if s.ShowingCommandWindow {
		visibleDialogs++
	}
	if s.ShowingRunDialog {
		visibleDialogs++
	}
	if s.ShowingCustomCommand {
		visibleDialogs++
	}

	if visibleDialogs > 1 {
		issues = append(issues, "multiple dialogs visible simultaneously")
	}

	return issues
}

// Getters for current state values
func (s *CommandWindowState) IsShowingCommandWindow() bool {
	return s.ShowingCommandWindow
}

func (s *CommandWindowState) IsShowingRunDialog() bool {
	return s.ShowingRunDialog
}

func (s *CommandWindowState) IsShowingCustomCommand() bool {
	return s.ShowingCustomCommand
}

func (s *CommandWindowState) GetCommandWindowWidth() int {
	return s.CommandWindowWidth
}

func (s *CommandWindowState) GetDetectedCommands() []parser.ExecutableCommand {

	commands := make([]parser.ExecutableCommand, len(s.DetectedCommands))
	copy(commands, s.DetectedCommands)
	return commands
}

func (s *CommandWindowState) GetMonorepoInfo() *parser.MonorepoInfo {
	return s.MonorepoInfo
}

func (s *CommandWindowState) GetCustomCommandText() string {
	return s.CustomCommandText
}

func (s *CommandWindowState) GetCommandInputText() string {
	return s.CommandInputText
}

// SetCommandInputText updates the command input text (immutable)
func (s *CommandWindowState) SetCommandInputText(text string) *CommandWindowState {
	newState := s.copy()
	newState.CommandInputText = text
	newState.incrementVersion()
	return newState
}

func (s *CommandWindowState) GetCommandSuggestions() []string {
	suggestions := make([]string, len(s.CommandSuggestions))
	copy(suggestions, s.CommandSuggestions)
	return suggestions
}

// SetCommandSuggestions updates the command suggestions (immutable)
func (s *CommandWindowState) SetCommandSuggestions(suggestions []string) *CommandWindowState {
	newState := s.copy()
	newState.CommandSuggestions = make([]string, len(suggestions))
	copy(newState.CommandSuggestions, suggestions)
	newState.incrementVersion()
	return newState
}

func (s *CommandWindowState) GetTerminalSize() (int, int) {
	return s.TerminalWidth, s.TerminalHeight
}

func (s *CommandWindowState) GetLoading() bool {
	return s.Loading
}

func (s *CommandWindowState) GetError() string {
	return s.LastError
}

func (s *CommandWindowState) GetLastUpdated() time.Time {
	return s.LastUpdated
}

func (s *CommandWindowState) GetVersion() uint64 {
	return s.version
}

// Helper methods
func (s *CommandWindowState) copy() *CommandWindowState {
	newState := *s

	// Deep copy slices
	newState.DetectedCommands = make([]parser.ExecutableCommand, len(s.DetectedCommands))
	copy(newState.DetectedCommands, s.DetectedCommands)

	return &newState
}

func (s *CommandWindowState) incrementVersion() {
	s.version++
}