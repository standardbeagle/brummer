package harness

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// InputSimulatorImpl provides realistic keyboard input simulation for TUI testing
// with PTY layer integration and configurable timing delays.
// This type implements the InputSimulator interface defined in test contracts.
type InputSimulatorImpl struct {
	mu            sync.RWMutex
	ptyLayer      *pty.PTYLayer
	sequence      []tea.KeyMsg
	keyMappings   map[string][]tea.KeyMsg
	timingDelay   time.Duration
	defaultDelay  time.Duration
	actionMappings map[string][]string // Action to key string mappings
}

// NewInputSimulator creates a new InputSimulator instance with the given configuration
func NewInputSimulator(config InputSimulatorConfig) (*InputSimulatorImpl, error) {
	if config.PTYLayer == nil {
		return nil, fmt.Errorf("PTYLayer is required for InputSimulator")
	}

	if !config.PTYLayer.IsAllocated() {
		return nil, fmt.Errorf("PTYLayer must be allocated before creating InputSimulator")
	}

	simulator := &InputSimulatorImpl{
		ptyLayer:     config.PTYLayer,
		sequence:     make([]tea.KeyMsg, 0),
		keyMappings:  make(map[string][]tea.KeyMsg),
		timingDelay:  config.KeyInputDelay,
		defaultDelay: 100 * time.Millisecond,
		actionMappings: map[string][]string{
			// Navigation actions
			"up":        {"up"},
			"down":      {"down"},
			"left":      {"left"},
			"right":     {"right"},
			"home":      {"home"},
			"end":       {"end"},
			"pgup":      {"pgup"},
			"pgdown":    {"pgdown"},
			"tab":       {"tab"},
			"shift_tab": {"shift+tab"},

			// Control actions
			"enter":     {"enter"},
			"space":     {"space"},
			"backspace": {"backspace"},
			"delete":    {"delete"},
			"escape":    {"escape"},
			"ctrl_c":    {"ctrl+c"},
			"ctrl_d":    {"ctrl+d"},
			"ctrl_z":    {"ctrl+z"},

			// Common TUI actions
			"quit":      {"q"},
			"exit":      {"ctrl+c"},
			"help":      {"h"},
			"refresh":   {"r"},
			"save":      {"ctrl+s"},
			"load":      {"ctrl+o"},
			"search":    {"/"},
			"command":   {":"},

			// Brummer-specific actions
			"select":    {"enter"},
			"back":      {"escape"},
			"confirm":   {"enter"},
			"cancel":    {"escape"},
			"next_view": {"tab"},
			"prev_view": {"shift+tab"},
			"command_palette": {"/"},
			"dev":       {"d", "e", "v"},
			"test":      {"t", "e", "s", "t"},
			"build":     {"b", "u", "i", "l", "d"},
		},
	}

	// Initialize default key mappings
	simulator.initializeDefaultKeyMappings()

	return simulator, nil
}

// initializeDefaultKeyMappings sets up default key mappings for common actions
func (is *InputSimulatorImpl) initializeDefaultKeyMappings() {
	defaultMappings := map[string][]tea.KeyMsg{
		"up":        {{Type: tea.KeyUp}},
		"down":      {{Type: tea.KeyDown}},
		"left":      {{Type: tea.KeyLeft}},
		"right":     {{Type: tea.KeyRight}},
		"home":      {{Type: tea.KeyHome}},
		"end":       {{Type: tea.KeyEnd}},
		"pgup":      {{Type: tea.KeyPgUp}},
		"pgdown":    {{Type: tea.KeyPgDown}},
		"tab":       {{Type: tea.KeyTab}},
		"enter":     {{Type: tea.KeyEnter}},
		"space":     {{Type: tea.KeySpace}},
		"backspace": {{Type: tea.KeyBackspace}},
		"delete":    {{Type: tea.KeyDelete}},
		"escape":    {{Type: tea.KeyEsc}},
		"ctrl_c":    {{Type: tea.KeyCtrlC}},
		"ctrl_d":    {{Type: tea.KeyCtrlD}},
		"ctrl_z":    {{Type: tea.KeyCtrlZ}},
		"quit":      {{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		"help":      {{Type: tea.KeyRunes, Runes: []rune{'h'}}},
		"refresh":   {{Type: tea.KeyRunes, Runes: []rune{'r'}}},
		"save":      {{Type: tea.KeyCtrlS}},
		"load":      {{Type: tea.KeyCtrlO}},
		"search":    {{Type: tea.KeyRunes, Runes: []rune{'/'}}},
		"command":   {{Type: tea.KeyRunes, Runes: []rune{':'}}},
		"select":    {{Type: tea.KeyEnter}},
		"back":      {{Type: tea.KeyEsc}},
		"confirm":   {{Type: tea.KeyEnter}},
		"cancel":    {{Type: tea.KeyEsc}},
		"next_view": {{Type: tea.KeyTab}},
		"dev":       {
			{Type: tea.KeyRunes, Runes: []rune{'d'}},
			{Type: tea.KeyRunes, Runes: []rune{'e'}},
			{Type: tea.KeyRunes, Runes: []rune{'v'}},
		},
		"test":      {
			{Type: tea.KeyRunes, Runes: []rune{'t'}},
			{Type: tea.KeyRunes, Runes: []rune{'e'}},
			{Type: tea.KeyRunes, Runes: []rune{'s'}},
			{Type: tea.KeyRunes, Runes: []rune{'t'}},
		},
	}

	for action, keys := range defaultMappings {
		is.keyMappings[action] = keys
	}
}

// GenerateKeySequence converts an action string into a sequence of keyboard events
func (is *InputSimulatorImpl) GenerateKeySequence(action string) ([]tea.KeyMsg, error) {
	if strings.TrimSpace(action) == "" {
		return nil, fmt.Errorf("action cannot be empty")
	}

	is.mu.RLock()
	defer is.mu.RUnlock()

	// First check if we have a direct mapping for the action
	if keys, exists := is.keyMappings[action]; exists {
		// Return a copy to avoid modification
		result := make([]tea.KeyMsg, len(keys))
		copy(result, keys)
		return result, nil
	}

	// Check if it's a complex action (e.g., "navigate:up", "type:hello")
	if strings.Contains(action, ":") {
		return is.generateComplexActionSequence(action)
	}

	// If no mapping found, try to interpret as a type action
	if len(action) > 1 && !strings.HasPrefix(action, "ctrl+") &&
	   !strings.HasPrefix(action, "shift+") && !strings.HasPrefix(action, "alt+") {
		return is.generateTypeSequence(action)
	}

	return nil, fmt.Errorf("unknown action: %s", action)
}

// generateComplexActionSequence handles complex actions with parameters
func (is *InputSimulatorImpl) generateComplexActionSequence(action string) ([]tea.KeyMsg, error) {
	parts := strings.SplitN(action, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid complex action format: %s", action)
	}

	prefix := parts[0]
	param := parts[1]

	switch prefix {
	case "type", "input", "text":
		return is.generateTypeSequence(param)
	case "navigate", "move", "cursor":
		return is.generateNavigationSequence(param)
	case "command", "cmd":
		return is.generateCommandSequence(param)
	case "shortcut", "hotkey":
		return is.generateShortcutSequence(param)
	default:
		return nil, fmt.Errorf("unknown complex action prefix: %s", prefix)
	}
}

// generateTypeSequence creates a key sequence for typing text
func (is *InputSimulatorImpl) generateTypeSequence(text string) ([]tea.KeyMsg, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty for type action")
	}

	keys := make([]tea.KeyMsg, len(text))
	for i, char := range text {
		keys[i] = tea.KeyMsg{
			Type: tea.KeyRunes,
			Runes: []rune{char},
		}
	}

	return keys, nil
}

// generateNavigationSequence creates keys for navigation commands
func (is *InputSimulatorImpl) generateNavigationSequence(direction string) ([]tea.KeyMsg, error) {
	navigationMap := map[string]string{
		"up":       "up",
		"down":     "down",
		"left":     "left",
		"right":    "right",
		"home":     "home",
		"end":      "end",
		"pageup":   "pgup",
		"pagedown": "pgdown",
		"pgup":     "pgup",
		"pgdown":   "pgdown",
		"begin":    "home",
		"start":    "home",
	}

	// Handle compound directions like "up+enter" or "down+down+enter"
	if strings.Contains(direction, "+") {
		parts := strings.Split(direction, "+")
		var keys []tea.KeyMsg
		for _, part := range parts {
			partKeys, err := is.generateNavigationSequence(part)
			if err != nil {
				return nil, err
			}
			keys = append(keys, partKeys...)
		}
		return keys, nil
	}

	if mapped, exists := navigationMap[strings.ToLower(direction)]; exists {
		return is.GenerateKeySequence(mapped)
	}

	return nil, fmt.Errorf("unknown navigation direction: %s", direction)
}

// generateCommandSequence creates keys for command execution
func (is *InputSimulatorImpl) generateCommandSequence(command string) ([]tea.KeyMsg, error) {
	commandMap := map[string]string{
		"quit":       "quit",
		"exit":       "exit",
		"help":       "help",
		"refresh":    "refresh",
		"search":     "search",
		"command":    "command",
		"palette":    "command_palette",
		"dev":        "dev",
		"test":       "test",
		"build":      "build",
		"save":       "save",
		"load":       "load",
		"confirm":    "confirm",
		"cancel":     "cancel",
		"back":       "back",
		"next":       "next_view",
		"previous":   "prev_view",
	}

	if mapped, exists := commandMap[strings.ToLower(command)]; exists {
		return is.GenerateKeySequence(mapped)
	}

	// If it's not a known command, treat it as text to type followed by enter
	textKeys, err := is.generateTypeSequence(command)
	if err != nil {
		return nil, err
	}

	// Add enter at the end
	enterKey, err := is.GenerateKeySequence("enter")
	if err != nil {
		return nil, err
	}

	return append(textKeys, enterKey...), nil
}

// generateShortcutSequence creates keys for keyboard shortcuts
func (is *InputSimulatorImpl) generateShortcutSequence(shortcut string) ([]tea.KeyMsg, error) {
	// Handle shortcuts like "ctrl+s", "shift+tab", "alt+f"
	shortcut = strings.ToLower(shortcut)

	parts := strings.Split(shortcut, "+")
	if len(parts) == 1 {
		// No modifier, treat as regular key
		return is.GenerateKeySequence(parts[0])
	}

	modifiers := parts[:len(parts)-1]
	key := parts[len(parts)-1]

	// Convert modifiers and key to tea.KeyMsg
	keyMsg := tea.KeyMsg{}

	// Apply modifiers
	for _, modifier := range modifiers {
		switch modifier {
		case "ctrl", "control":
			switch key {
			case "c":
				keyMsg.Type = tea.KeyCtrlC
			case "d":
				keyMsg.Type = tea.KeyCtrlD
			case "z":
				keyMsg.Type = tea.KeyCtrlZ
			case "s":
				keyMsg.Type = tea.KeyCtrlS
			case "o":
				keyMsg.Type = tea.KeyCtrlO
			default:
				return nil, fmt.Errorf("unsupported ctrl+key combination: ctrl+%s", key)
			}
		case "shift":
			// Shift is handled by the key itself for most keys
			if key == "tab" {
				keyMsg.Type = tea.KeyShiftTab
			} else {
				// For other keys, uppercase the rune
				if len(key) == 1 {
					keyMsg.Type = tea.KeyRunes
					keyMsg.Runes = []rune{rune(strings.ToUpper(key)[0])}
				}
			}
		default:
			return nil, fmt.Errorf("unsupported modifier: %s", modifier)
		}
	}

	return []tea.KeyMsg{keyMsg}, nil
}

// SimulateKeyPress simulates a single key press by writing it to the PTY
func (is *InputSimulatorImpl) SimulateKeyPress(key tea.KeyMsg) error {

	is.mu.Lock()
	defer is.mu.Unlock()

	// Add to sequence
	is.sequence = append(is.sequence, key)

	// Write to PTY if available
	if is.ptyLayer != nil && is.ptyLayer.IsAllocated() {
		if err := is.ptyLayer.SimulateKeyPress(key); err != nil {
			return fmt.Errorf("failed to write key to PTY: %w", err)
		}

		// Apply timing delay
		if is.timingDelay > 0 {
			time.Sleep(is.timingDelay)
		}
	}

	return nil
}

// SimulateKeySequence simulates a sequence of key presses
func (is *InputSimulatorImpl) SimulateKeySequence(sequence []tea.KeyMsg) error {
	if sequence == nil {
		return fmt.Errorf("key sequence cannot be nil")
	}

	for i, key := range sequence {
		if err := is.SimulateKeyPress(key); err != nil {
			return fmt.Errorf("failed to simulate key at index %d: %w", i, err)
		}
	}

	return nil
}

// SetKeyMappings updates the key mappings for action-to-key conversion
func (is *InputSimulatorImpl) SetKeyMappings(mappings []KeyMapping) error {
	if mappings == nil {
		return fmt.Errorf("mappings cannot be nil")
	}

	is.mu.Lock()
	defer is.mu.Unlock()

	// Validate mappings before applying
	for _, mapping := range mappings {
		if strings.TrimSpace(mapping.Action) == "" {
			return fmt.Errorf("mapping action cannot be empty")
		}
		if len(mapping.Keys) == 0 {
			return fmt.Errorf("mapping keys cannot be empty for action: %s", mapping.Action)
		}
		// Validate each key
		for range mapping.Keys {
			// Key validation - skip unknown types for now
		}
	}

	// Apply new mappings
	is.keyMappings = make(map[string][]tea.KeyMsg)
	for _, mapping := range mappings {
		// Store a copy of the keys
		keys := make([]tea.KeyMsg, len(mapping.Keys))
		copy(keys, mapping.Keys)
		is.keyMappings[mapping.Action] = keys
	}

	// Restore default mappings for any actions not provided
	is.initializeDefaultKeyMappings()

	return nil
}

// SetTimingDelay updates the delay between key presses
func (is *InputSimulatorImpl) SetTimingDelay(delay time.Duration) error {
	if delay < 0 {
		return fmt.Errorf("timing delay cannot be negative")
	}

	is.mu.Lock()
	defer is.mu.Unlock()

	is.timingDelay = delay
	return nil
}

// GetCurrentSequence returns the current sequence of simulated keys
func (is *InputSimulatorImpl) GetCurrentSequence() []tea.KeyMsg {
	is.mu.RLock()
	defer is.mu.RUnlock()

	// Return a copy to avoid modification
	result := make([]tea.KeyMsg, len(is.sequence))
	copy(result, is.sequence)
	return result
}

// ClearSequence clears the current key sequence
func (is *InputSimulatorImpl) ClearSequence() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.sequence = make([]tea.KeyMsg, 0)
	return nil
}

// GetSequenceCount returns the number of keys in the current sequence
func (is *InputSimulatorImpl) GetSequenceCount() int {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return len(is.sequence)
}

// GetLastKey returns the last key in the sequence, or nil if empty
func (is *InputSimulatorImpl) GetLastKey() *tea.KeyMsg {
	is.mu.RLock()
	defer is.mu.RUnlock()

	if len(is.sequence) == 0 {
		return nil
	}

	// Return a copy
	lastKey := is.sequence[len(is.sequence)-1]
	return &lastKey
}

// ResetToDefaults resets the simulator to default state and mappings
func (is *InputSimulatorImpl) ResetToDefaults() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	is.sequence = make([]tea.KeyMsg, 0)
	is.timingDelay = is.defaultDelay
	is.initializeDefaultKeyMappings()

	return nil
}

// GetPTYLayer returns the associated PTY layer (for debugging)
func (is *InputSimulatorImpl) GetPTYLayer() *pty.PTYLayer {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return is.ptyLayer
}

// IsPTYReady returns true if the PTY layer is ready for input
func (is *InputSimulatorImpl) IsPTYReady() bool {
	is.mu.RLock()
	defer is.mu.RUnlock()

	return is.ptyLayer != nil && is.ptyLayer.IsAllocated() && is.ptyLayer.Err() == nil
}

// ValidateKeyMapping validates a single key mapping
func (is *InputSimulatorImpl) ValidateKeyMapping(mapping KeyMapping) error {
	if strings.TrimSpace(mapping.Action) == "" {
		return fmt.Errorf("action cannot be empty")
	}
	if len(mapping.Keys) == 0 {
		return fmt.Errorf("keys cannot be empty")
	}
	for range mapping.Keys {
		// Key validation - skip unknown types for now
	}
	return nil
}