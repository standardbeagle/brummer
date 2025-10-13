package harness

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ExpectedStateManager manages expected states for testing
type ExpectedStateManager struct {
	mu           sync.RWMutex
	states       []ExpectedState
	defaultState ExpectedState
}

// NewExpectedStateManager creates a new expected state manager
func NewExpectedStateManager() *ExpectedStateManager {
	return &ExpectedStateManager{
		states: make([]ExpectedState, 0),
		defaultState: ExpectedState{
			TriggerAction: "",
			Timeout:       1 * time.Second,
			ModelState:    make(map[string]interface{}),
		},
	}
}

// AddExpectedState adds an expected state
func (esm *ExpectedStateManager) AddExpectedState(state ExpectedState) error {
	esm.mu.Lock()
	defer esm.mu.Unlock()

	// Validate expected state
	if err := esm.validateExpectedState(state); err != nil {
		return err
	}

	esm.states = append(esm.states, state)
	return nil
}

// GetExpectedState returns an expected state by index
func (esm *ExpectedStateManager) GetExpectedState(index int) (ExpectedState, error) {
	esm.mu.RLock()
	defer esm.mu.RUnlock()

	if index < 0 || index >= len(esm.states) {
		return ExpectedState{}, fmt.Errorf("index out of range: %d", index)
	}

	return esm.states[index], nil
}

// GetExpectedStates returns all expected states
func (esm *ExpectedStateManager) GetExpectedStates() []ExpectedState {
	esm.mu.RLock()
	defer esm.mu.RUnlock()

	states := make([]ExpectedState, len(esm.states))
	copy(states, esm.states)
	return states
}

// ClearExpectedStates clears all expected states
func (esm *ExpectedStateManager) ClearExpectedStates() {
	esm.mu.Lock()
	defer esm.mu.Unlock()

	esm.states = esm.states[:0]
}

// validateExpectedState validates an expected state
func (esm *ExpectedStateManager) validateExpectedState(state ExpectedState) error {
	if state.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	// Validate view type if specified
	if state.ViewType != nil {
		// Check if it's a valid view type
		switch state.ViewType.(type) {
		case string, int, tea.Model:
			// Valid types
		default:
			return fmt.Errorf("invalid view type: %T", state.ViewType)
		}
	}

	return nil
}

// ExpectedStateBuilder helps build expected states
type ExpectedStateBuilder struct {
	state ExpectedState
}

// NewExpectedStateBuilder creates a new expected state builder
func NewExpectedStateBuilder() *ExpectedStateBuilder {
	return &ExpectedStateBuilder{
		state: ExpectedState{
			Timeout:    1 * time.Second,
			ModelState: make(map[string]interface{}),
		},
	}
}

// WithTriggerAction sets the trigger action
func (esb *ExpectedStateBuilder) WithTriggerAction(action string) *ExpectedStateBuilder {
	esb.state.TriggerAction = action
	return esb
}

// WithViewType sets the expected view type
func (esb *ExpectedStateBuilder) WithViewType(viewType interface{}) *ExpectedStateBuilder {
	esb.state.ViewType = viewType
	return esb
}

// WithSelectedProcess sets the expected selected process
func (esb *ExpectedStateBuilder) WithSelectedProcess(process string) *ExpectedStateBuilder {
	esb.state.SelectedProcess = process
	return esb
}

// WithModelState adds model state
func (esb *ExpectedStateBuilder) WithModelState(key string, value interface{}) *ExpectedStateBuilder {
	if esb.state.ModelState == nil {
		esb.state.ModelState = make(map[string]interface{})
	}
	esb.state.ModelState[key] = value
	return esb
}

// WithModelStateMap sets the entire model state
func (esb *ExpectedStateBuilder) WithModelStateMap(state map[string]interface{}) *ExpectedStateBuilder {
	esb.state.ModelState = state
	return esb
}

// WithTimeout sets the timeout
func (esb *ExpectedStateBuilder) WithTimeout(timeout time.Duration) *ExpectedStateBuilder {
	esb.state.Timeout = timeout
	return esb
}

// Build builds the expected state
func (esb *ExpectedStateBuilder) Build() ExpectedState {
	return esb.state
}

// ExpectedStateValidator validates expected states
type ExpectedStateValidator struct {
	mu sync.RWMutex
}

// NewExpectedStateValidator creates a new expected state validator
func NewExpectedStateValidator() *ExpectedStateValidator {
	return &ExpectedStateValidator{}
}

// ValidateExpectedState validates an expected state
func (esv *ExpectedStateValidator) ValidateExpectedState(state ExpectedState) ValidationResult {
	esv.mu.RLock()
	defer esv.mu.RUnlock()

	// Check required fields
	if state.TriggerAction == "" {
		return ValidationResult{
			Valid:   false,
			Message: "trigger action cannot be empty",
			Details: map[string]interface{}{
				"field": "trigger_action",
				"reason": "required field missing",
			},
		}
	}

	// Check timeout
	if state.Timeout <= 0 {
		return ValidationResult{
			Valid:   false,
			Message: "timeout must be positive",
			Details: map[string]interface{}{
				"field":    "timeout",
				"value":    state.Timeout,
				"required": "> 0",
			},
		}
	}

	// Validate view type if specified
	if state.ViewType != nil {
		if err := esv.validateViewType(state.ViewType); err != nil {
			return ValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("invalid view type: %v", err),
				Details: map[string]interface{}{
					"field": "view_type",
					"value": state.ViewType,
					"error": err.Error(),
				},
			}
		}
	}

	// Validate model state
	if state.ModelState != nil {
		if err := esv.validateModelState(state.ModelState); err != nil {
			return ValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("invalid model state: %v", err),
				Details: map[string]interface{}{
					"field": "model_state",
					"error": err.Error(),
				},
			}
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "expected state is valid",
		Details: map[string]interface{}{
			"trigger_action": state.TriggerAction,
			"timeout":        state.Timeout.String(),
		},
	}
}

// validateViewType validates a view type
func (esv *ExpectedStateValidator) validateViewType(viewType interface{}) error {
	switch viewType.(type) {
	case string:
		// Check for common view names
		if str, ok := viewType.(string); ok {
			validViews := []string{
				"processes", "logs", "errors", "ai-coders", "settings",
				" Processes", " Logs", " Errors", " AI Coders", " Settings",
				"1", "2", "3", "4", "5",
			}
			for _, valid := range validViews {
				if str == valid {
					return nil
				}
			}
			return fmt.Errorf("unknown view type: %s", str)
		}
	case int:
		// Check for numeric view indices
		if num, ok := viewType.(int); ok {
			if num >= 1 && num <= 5 {
				return nil
			}
			return fmt.Errorf("invalid view index: %d (must be 1-5)", num)
		}
	case tea.Model:
		// Tea models are always valid
		return nil
	default:
		return fmt.Errorf("unsupported view type: %T", viewType)
	}

	return nil
}

// validateModelState validates model state
func (esv *ExpectedStateValidator) validateModelState(state map[string]interface{}) error {
	if len(state) == 0 {
		return fmt.Errorf("model state cannot be empty")
	}

	// Check for common required fields
	requiredFields := []string{"view_type"}
	for _, field := range requiredFields {
		if _, exists := state[field]; !exists {
			return fmt.Errorf("missing required model state field: %s", field)
		}
	}

	return nil
}

// CommonExpectedStates provides common expected state configurations
var CommonExpectedStates = map[string]ExpectedState{
	"processes-view": {
		TriggerAction: "view:processes",
		ViewType:      "processes",
		Timeout:       1 * time.Second,
		ModelState: map[string]interface{}{
			"view_type":  "processes",
			"active":    true,
		},
	},
	"logs-view": {
		TriggerAction: "view:logs",
		ViewType:      "logs",
		Timeout:       1 * time.Second,
		ModelState: map[string]interface{}{
			"view_type":  "logs",
			"active":    true,
		},
	},
	"errors-view": {
		TriggerAction: "view:errors",
		ViewType:      "errors",
		Timeout:       1 * time.Second,
		ModelState: map[string]interface{}{
			"view_type":  "errors",
			"active":    true,
		},
	},
	"command-palette-open": {
		TriggerAction: "command:palette",
		ModelState: map[string]interface{}{
			"command_window_visible": true,
			"active":                true,
		},
		Timeout: 1 * time.Second,
	},
	"command-palette-close": {
		TriggerAction: "escape",
		ModelState: map[string]interface{}{
			"command_window_visible": false,
			"active":                true,
		},
		Timeout: 1 * time.Second,
	},
}

// GetCommonExpectedState returns a common expected state by name
func GetCommonExpectedState(name string) (ExpectedState, error) {
	state, exists := CommonExpectedStates[name]
	if !exists {
		return ExpectedState{}, fmt.Errorf("common expected state not found: %s", name)
	}
	return state, nil
}

// ExpectedStateMatcher helps match actual states with expected states
type ExpectedStateMatcher struct {
	mu sync.RWMutex
}

// NewExpectedStateMatcher creates a new expected state matcher
func NewExpectedStateMatcher() *ExpectedStateMatcher {
	return &ExpectedStateMatcher{}
}

// MatchState matches an actual state against expected states
func (esm *ExpectedStateMatcher) MatchState(actual ActualState, expectedStates []ExpectedState) (ExpectedState, error) {
	esm.mu.RLock()
	defer esm.mu.RUnlock()

	for _, expected := range expectedStates {
		if esm.statesMatch(expected, actual) {
			return expected, nil
		}
	}

	return ExpectedState{}, fmt.Errorf("no matching expected state found")
}

// statesMatch checks if an expected state matches an actual state
func (esm *ExpectedStateMatcher) statesMatch(expected ExpectedState, actual ActualState) bool {
	// Check view type
	if expected.ViewType != nil && actual.ViewType != nil {
		if !esm.compareValues(expected.ViewType, actual.ViewType) {
			return false
		}
	}

	// Check selected process
	if expected.SelectedProcess != "" && actual.SelectedProcess != "" {
		if expected.SelectedProcess != actual.SelectedProcess {
			return false
		}
	}

	// Check model state
	if len(expected.ModelState) > 0 && len(actual.ModelState) > 0 {
		if !esm.compareStateMaps(expected.ModelState, actual.ModelState) {
			return false
		}
	}

	return true
}

// compareValues compares two values
func (esm *ExpectedStateMatcher) compareValues(expected, actual interface{}) bool {
	switch expected := expected.(type) {
	case string:
		actualStr, ok := actual.(string)
		if !ok {
			return false
		}
		return expected == actualStr
	case int:
		actualInt, ok := actual.(int)
		if !ok {
			return false
		}
		return expected == actualInt
	default:
		// Try to convert to string and compare
		return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", actual)
	}
}

// compareStateMaps compares two state maps
func (esm *ExpectedStateMatcher) compareStateMaps(expected, actual map[string]interface{}) bool {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			return false
		}

		if !esm.compareValues(expectedValue, actualValue) {
			return false
		}
	}
	return true
}