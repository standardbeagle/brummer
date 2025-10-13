package harness

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// KeyMappingManager manages key mappings for input simulation
type KeyMappingManager struct {
	mu          sync.RWMutex
	mappings    map[string]KeyMapping
	defaultMappings []KeyMapping
}

// NewKeyMappingManager creates a new key mapping manager
func NewKeyMappingManager() *KeyMappingManager {
	return &KeyMappingManager{
		mappings: make(map[string]KeyMapping),
		defaultMappings: getDefaultKeyMappings(),
	}
}

// AddMapping adds a key mapping
func (kmm *KeyMappingManager) AddMapping(mapping KeyMapping) error {
	kmm.mu.Lock()
	defer kmm.mu.Unlock()

	if mapping.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}

	kmm.mappings[mapping.Action] = mapping
	return nil
}

// GetMapping returns a key mapping by action
func (kmm *KeyMappingManager) GetMapping(action string) (KeyMapping, error) {
	kmm.mu.RLock()
	defer kmm.mu.RUnlock()

	if mapping, exists := kmm.mappings[action]; exists {
		return mapping, nil
	}

	// Check default mappings
	for _, mapping := range kmm.defaultMappings {
		if mapping.Action == action {
			return mapping, nil
		}
	}

	return KeyMapping{}, fmt.Errorf("key mapping not found: %s", action)
}

// SetKeyMappings sets multiple key mappings
func (kmm *KeyMappingManager) SetKeyMappings(mappings []KeyMapping) error {
	kmm.mu.Lock()
	defer kmm.mu.Unlock()

	// Validate mappings
	for _, mapping := range mappings {
		if mapping.Action == "" {
			return fmt.Errorf("action cannot be empty")
		}
		if len(mapping.Keys) == 0 {
			return fmt.Errorf("keys cannot be empty for action: %s", mapping.Action)
		}
	}

	// Clear existing mappings and set new ones
	kmm.mappings = make(map[string]KeyMapping)
	for _, mapping := range mappings {
		kmm.mappings[mapping.Action] = mapping
	}

	return nil
}

// GetAllMappings returns all key mappings
func (kmm *KeyMappingManager) GetAllMappings() map[string]KeyMapping {
	kmm.mu.RLock()
	defer kmm.mu.RUnlock()

	allMappings := make(map[string]KeyMapping)

	// Add default mappings
	for _, mapping := range kmm.defaultMappings {
		allMappings[mapping.Action] = mapping
	}

	// Add custom mappings (may override defaults)
	for action, mapping := range kmm.mappings {
		allMappings[action] = mapping
	}

	return allMappings
}

// ClearMappings clears all custom key mappings
func (kmm *KeyMappingManager) ClearMappings() {
	kmm.mu.Lock()
	defer kmm.mu.Unlock()

	kmm.mappings = make(map[string]KeyMapping)
}

// RemoveMapping removes a key mapping
func (kmm *KeyMappingManager) RemoveMapping(action string) {
	kmm.mu.Lock()
	defer kmm.mu.Unlock()

	delete(kmm.mappings, action)
}

// GetActions returns all available actions
func (kmm *KeyMappingManager) GetActions() []string {
	kmm.mu.RLock()
	defer kmm.mu.RUnlock()

	actions := make([]string, 0)
	seen := make(map[string]bool)

	// Add default actions
	for _, mapping := range kmm.defaultMappings {
		if !seen[mapping.Action] {
			actions = append(actions, mapping.Action)
			seen[mapping.Action] = true
		}
	}

	// Add custom actions
	for action := range kmm.mappings {
		if !seen[action] {
			actions = append(actions, action)
			seen[action] = true
		}
	}

	return actions
}

// ValidateMapping validates a key mapping
func (kmm *KeyMappingManager) ValidateMapping(mapping KeyMapping) error {
	if mapping.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}

	if len(mapping.Keys) == 0 {
		return fmt.Errorf("keys cannot be empty")
	}

	// Validate each key
	for i, key := range mapping.Keys {
		// Bubble Tea doesn't have KeyUnknown, so we check for valid key types
		if key.Type == tea.KeyRunes && len(key.Runes) == 0 {
			return fmt.Errorf("invalid key at position %d: empty runes", i)
		}
	}

	return nil
}

// getDefaultKeyMappings returns default key mappings
func getDefaultKeyMappings() []KeyMapping {
	return []KeyMapping{
		// Basic navigation
		{
			Action:      "up",
			Keys:        []tea.KeyMsg{{Type: tea.KeyUp}},
			Description: "Move cursor up",
		},
		{
			Action:      "down",
			Keys:        []tea.KeyMsg{{Type: tea.KeyDown}},
			Description: "Move cursor down",
		},
		{
			Action:      "left",
			Keys:        []tea.KeyMsg{{Type: tea.KeyLeft}},
			Description: "Move cursor left",
		},
		{
			Action:      "right",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRight}},
			Description: "Move cursor right",
		},

		// Basic keys
		{
			Action:      "enter",
			Keys:        []tea.KeyMsg{{Type: tea.KeyEnter}},
			Description: "Press Enter key",
		},
		{
			Action:      "escape",
			Keys:        []tea.KeyMsg{{Type: tea.KeyEscape}},
			Description: "Press Escape key",
		},
		{
			Action:      "tab",
			Keys:        []tea.KeyMsg{{Type: tea.KeyTab}},
			Description: "Press Tab key",
		},
		{
			Action:      "backspace",
			Keys:        []tea.KeyMsg{{Type: tea.KeyBackspace}},
			Description: "Press Backspace key",
		},
		{
			Action:      "space",
			Keys:        []tea.KeyMsg{{Type: tea.KeySpace}},
			Description: "Press Space key",
		},

		// Control keys
		{
			Action:      "ctrl+c",
			Keys:        []tea.KeyMsg{{Type: tea.KeyCtrlC}},
			Description: "Press Ctrl+C",
		},
		{
			Action:      "ctrl+d",
			Keys:        []tea.KeyMsg{{Type: tea.KeyCtrlD}},
			Description: "Press Ctrl+D",
		},
		{
			Action:      "ctrl+z",
			Keys:        []tea.KeyMsg{{Type: tea.KeyCtrlZ}},
			Description: "Press Ctrl+Z",
		},
		{
			Action:      "ctrl+s",
			Keys:        []tea.KeyMsg{{Type: tea.KeyCtrlS}},
			Description: "Press Ctrl+S",
		},
		{
			Action:      "ctrl+o",
			Keys:        []tea.KeyMsg{{Type: tea.KeyCtrlO}},
			Description: "Press Ctrl+O",
		},

		// View switching (Brummer specific)
		{
			Action:      "view:processes",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'1'}}},
			Description: "Switch to Processes view",
		},
		{
			Action:      "view:logs",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'2'}}},
			Description: "Switch to Logs view",
		},
		{
			Action:      "view:errors",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'3'}}},
			Description: "Switch to Errors view",
		},
		{
			Action:      "view:ai-coders",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'4'}}},
			Description: "Switch to AI Coders view",
		},
		{
			Action:      "view:settings",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'5'}}},
			Description: "Switch to Settings view",
		},

		// Command palette
		{
			Action:      "command:palette",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'/'}}},
			Description: "Open command palette",
		},
		{
			Action:      "command:quit",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'q', 'u', 'i', 't'}}},
			Description: "Quit command",
		},
		{
			Action:      "command:help",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'p'}}},
			Description: "Show help",
		},

		// Brummer specific commands
		{
			Action:      "brummer:dev",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'d', 'e', 'v'}}},
			Description: "Run dev command",
		},
		{
			Action:      "brummer:test",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'t', 'e', 's', 't'}}},
			Description: "Run test command",
		},
		{
			Action:      "brummer:build",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'b', 'u', 'i', 'l', 'd'}}},
			Description: "Run build command",
		},

		// Complex sequences
		{
			Action:      "navigate:home",
			Keys:        []tea.KeyMsg{{Type: tea.KeyHome}},
			Description: "Go to beginning of line",
		},
		{
			Action:      "navigate:end",
			Keys:        []tea.KeyMsg{{Type: tea.KeyEnd}},
			Description: "Go to end of line",
		},
		{
			Action:      "navigate:page-up",
			Keys:        []tea.KeyMsg{{Type: tea.KeyPgUp}},
			Description: "Page up",
		},
		{
			Action:      "navigate:page-down",
			Keys:        []tea.KeyMsg{{Type: tea.KeyPgDown}},
			Description: "Page down",
		},
		{
			Action:      "navigate:delete",
			Keys:        []tea.KeyMsg{{Type: tea.KeyDelete}},
			Description: "Delete character",
		},

		// Number keys
		{
			Action:      "key:0",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'0'}}},
			Description: "Press 0 key",
		},
		{
			Action:      "key:1",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'1'}}},
			Description: "Press 1 key",
		},
		{
			Action:      "key:2",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'2'}}},
			Description: "Press 2 key",
		},
		{
			Action:      "key:3",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'3'}}},
			Description: "Press 3 key",
		},
		{
			Action:      "key:4",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'4'}}},
			Description: "Press 4 key",
		},
		{
			Action:      "key:5",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'5'}}},
			Description: "Press 5 key",
		},
		{
			Action:      "key:6",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'6'}}},
			Description: "Press 6 key",
		},
		{
			Action:      "key:7",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'7'}}},
			Description: "Press 7 key",
		},
		{
			Action:      "key:8",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'8'}}},
			Description: "Press 8 key",
		},
		{
			Action:      "key:9",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'9'}}},
			Description: "Press 9 key",
		},

		// Common text
		{
			Action:      "text:yes",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'y', 'e', 's'}}},
			Description: "Type 'yes'",
		},
		{
			Action:      "text:no",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'n', 'o'}}},
			Description: "Type 'no'",
		},
		{
			Action:      "text:exit",
			Keys:        []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'e', 'x', 'i', 't'}}},
			Description: "Type 'exit'",
		},
	}
}

// KeyMappingValidator validates key mappings
type KeyMappingValidator struct {
	mu    sync.RWMutex
	rules []ValidationRule
}

// NewKeyMappingValidator creates a new key mapping validator
func NewKeyMappingValidator() *KeyMappingValidator {
	return &KeyMappingValidator{
		rules: make([]ValidationRule, 0),
	}
}

// ValidateMappings validates a set of key mappings
func (kmv *KeyMappingValidator) ValidateMappings(mappings map[string]KeyMapping) []ValidationResult {
	kmv.mu.RLock()
	defer kmv.mu.RUnlock()

	results := make([]ValidationResult, 0)

	// Validate each mapping
	for action, mapping := range mappings {
		result := kmv.validateMapping(action, mapping)
		results = append(results, result)
	}

	// Run custom validation rules
	for _, rule := range kmv.rules {
		if rule.Enabled {
			var result ValidationResult
			if rule.IsKeyMappingRule && rule.KeyMappingValidator != nil {
				result = rule.KeyMappingValidator(KeyMappingValidationContext{
					Mappings: mappings,
				})
			} else if !rule.IsKeyMappingRule && rule.Validator != nil {
				// For state validation rules, we'd need a StateSnapshot
				// For now, skip these in key mapping validation
				continue
			}
			results = append(results, result)
		}
	}

	return results
}

// validateMapping validates a single key mapping
func (kmv *KeyMappingValidator) validateMapping(action string, mapping KeyMapping) ValidationResult {
	if mapping.Action == "" {
		return ValidationResult{
			Valid:   false,
			Message: "action cannot be empty",
		}
	}

	if len(mapping.Keys) == 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("keys cannot be empty for action: %s", action),
		}
	}

	// Validate each key
	for i, key := range mapping.Keys {
		// Bubble Tea doesn't have KeyUnknown, so we check for valid key types
		if key.Type == tea.KeyRunes && len(key.Runes) == 0 {
			return ValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("invalid key at position %d for action %s: empty runes", i, action),
			}
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: fmt.Sprintf("key mapping '%s' is valid", action),
	}
}

// AddValidationRule adds a validation rule
func (kmv *KeyMappingValidator) AddValidationRule(rule ValidationRule) {
	kmv.mu.Lock()
	defer kmv.mu.Unlock()

	kmv.rules = append(kmv.rules, rule)
}

// KeyMappingValidationContext provides context for key mapping validation
type KeyMappingValidationContext struct {
	Mappings map[string]KeyMapping
	Timestamp time.Time
}