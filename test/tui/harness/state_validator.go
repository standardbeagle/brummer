package harness

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StateValidator validates TUI state changes during testing
type StateValidator struct {
	mu             sync.RWMutex
	config         StateValidatorConfig
	expectedStates []ExpectedState
	actualStates   []StateSnapshot
	validationRules []ValidationRule
}

// NewStateValidator creates a new state validator instance
func NewStateValidator(config StateValidatorConfig) *StateValidator {
	return &StateValidator{
		config:           config,
		expectedStates:   make([]ExpectedState, 0),
		actualStates:     make([]StateSnapshot, 0),
		validationRules: config.ValidationRules,
	}
}

// CaptureState captures the current state of the TUI model
func (sv *StateValidator) CaptureState(model interface{}) (StateSnapshot, error) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	snapshot := StateSnapshot{
		Timestamp: time.Now(),
		Model:     model,
	}

	// Extract content from PTY if available
	if sv.config.PTYLayer != nil && sv.config.PTYLayer.IsAllocated() {
		snapshot.Content = sv.config.PTYLayer.GetContent()
		row, col := sv.config.PTYLayer.GetCursorPosition()
		snapshot.Cursor = CursorPosition{Row: row, Col: col}

		size := sv.config.PTYLayer.GetSize()
		snapshot.Size = TerminalSize{
			Rows: int(size.Rows),
			Cols: int(size.Cols),
		}
	}

	// Extract model state
	modelState, err := sv.extractModelState(model)
	if err != nil {
		return StateSnapshot{}, fmt.Errorf("failed to extract model state: %w", err)
	}
	snapshot.Metadata = modelState

	// Store actual state
	sv.actualStates = append(sv.actualStates, snapshot)

	// Keep only last 100 states to prevent memory issues
	if len(sv.actualStates) > 100 {
		sv.actualStates = sv.actualStates[len(sv.actualStates)-100:]
	}

	return snapshot, nil
}

// extractModelState extracts relevant state information from the model
func (sv *StateValidator) extractModelState(model interface{}) (map[string]interface{}, error) {
	if model == nil {
		return map[string]interface{}{}, nil
	}

	state := make(map[string]interface{})

	// Use reflection to extract public fields from the model
	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}

	// Extract common fields that TUI models typically have
	modelType := modelValue.Type()
	for i := 0; i < modelValue.NumField(); i++ {
		field := modelValue.Field(i)
		fieldType := modelType.Field(i)

		// Skip unexported fields - use CanInterface() instead of IsExported()
		if !field.CanInterface() {
			continue
		}

		fieldName := fieldType.Name
		fieldValue := field.Interface()

		// Convert complex types to simpler representations
		switch v := fieldValue.(type) {
		case tea.Model:
			// For nested models, try to extract their state recursively
			nestedState, err := sv.extractModelState(v)
			if err == nil {
				state[fieldName] = nestedState
			}
		default:
			// For basic types, store as-is
			if sv.isSimpleType(fieldValue) {
				state[fieldName] = fieldValue
			} else {
				// For complex types, try to convert to JSON
				if jsonBytes, err := json.Marshal(fieldValue); err == nil {
					var jsonValue interface{}
					if err := json.Unmarshal(jsonBytes, &jsonValue); err == nil {
						state[fieldName] = jsonValue
					}
				}
			}
		}
	}

	return state, nil
}

// isSimpleType checks if a value is a simple type that can be directly stored
func (sv *StateValidator) isSimpleType(value interface{}) bool {
	switch value.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
			float32, float64, bool:
		return true
	default:
		return false
	}
}

// ValidateTransition validates the transition between two states
func (sv *StateValidator) ValidateTransition(from, to StateSnapshot) ValidationResult {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	// Basic validation checks
	if to.Timestamp.Before(from.Timestamp) {
		return ValidationResult{
			Valid:   false,
			Message: "timestamp went backwards",
			Details: map[string]interface{}{
				"from_timestamp": from.Timestamp,
				"to_timestamp":   to.Timestamp,
			},
		}
	}

	// Check if content changed (if available)
	if from.Content != "" && to.Content != "" && from.Content == to.Content {
		return ValidationResult{
			Valid:   true,
			Message: "no state change detected",
			Details: map[string]interface{}{
				"content_same": true,
			},
		}
	}

	// Validate against validation rules
	for _, rule := range sv.validationRules {
		if rule.Enabled {
			result := rule.Validator(to)
			if !result.Valid {
				return ValidationResult{
					Valid:   false,
					Message: fmt.Sprintf("validation rule '%s' failed: %s", rule.Name, result.Message),
					Details: map[string]interface{}{
						"rule_id":   rule.ID,
						"rule_name": rule.Name,
						"rule_details": result.Details,
					},
				}
			}
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "transition is valid",
		Details: map[string]interface{}{
			"duration_ms": to.Timestamp.Sub(from.Timestamp).Milliseconds(),
		},
	}
}

// CompareStates compares an expected state with an actual state
func (sv *StateValidator) CompareStates(expected interface{}, actual StateSnapshot) ComparisonResult {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	// Handle different types of expected states
	switch expected := expected.(type) {
	case ExpectedState:
		return sv.compareExpectedState(expected, actual)
	case map[string]interface{}:
		return sv.compareStateMaps(expected, actual.Metadata)
	default:
		// Try to convert to map and compare
		if expectedBytes, err := json.Marshal(expected); err == nil {
			var expectedMap map[string]interface{}
			if err := json.Unmarshal(expectedBytes, &expectedMap); err == nil {
				return sv.compareStateMaps(expectedMap, actual.Metadata)
			}
		}

		return ComparisonResult{
			Match:   false,
			Message: fmt.Sprintf("unsupported expected state type: %T", expected),
			Details: map[string]interface{}{
				"expected_type": fmt.Sprintf("%T", expected),
				"actual_type":   "StateSnapshot",
			},
		}
	}
}

// compareExpectedState compares an ExpectedState with an actual StateSnapshot
func (sv *StateValidator) compareExpectedState(expected ExpectedState, actual StateSnapshot) ComparisonResult {
	result := ComparisonResult{
		Match:   true,
		Message: "expected state matches",
		Details: make(map[string]interface{}),
		Diff:    make([]StateDiff, 0),
	}

	// Compare model state if specified
	if len(expected.ModelState) > 0 {
		stateResult := sv.compareStateMaps(expected.ModelState, actual.Metadata)
		if !stateResult.Match {
			result.Match = false
			result.Message = "model state mismatch"
			result.Diff = append(result.Diff, stateResult.Diff...)
		}
	}

	// Check timeout
	if expected.Timeout > 0 {
		// Assuming expected.Timeout is a duration in milliseconds, convert to time.Duration
		timeoutDuration := time.Duration(expected.Timeout) * time.Millisecond
		elapsed := time.Since(actual.Timestamp) // Time since the actual state was captured
		if elapsed > timeoutDuration {
			result.Match = false
			result.Message = "state validation timeout"
			result.Details["timeout_exceeded"] = true
			result.Details["elapsed_ms"] = elapsed.Milliseconds()
			result.Details["timeout_ms"] = expected.Timeout
		}
	}

	// Store comparison details
	result.Details["expected_state"] = expected
	result.Details["actual_state"] = actual

	return result
}

// compareStateMaps compares two state maps and generates differences
func (sv *StateValidator) compareStateMaps(expected, actual map[string]interface{}) ComparisonResult {
	result := ComparisonResult{
		Match:   true,
		Message: "state maps match",
		Details: make(map[string]interface{}),
		Diff:    make([]StateDiff, 0),
	}

	// Check for missing keys in actual
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			result.Match = false
			result.Message = "missing keys in actual state"
			result.Diff = append(result.Diff, StateDiff{
				Path:     key,
				Expected: expectedValue,
				Actual:   nil,
				Type:     DiffTypeRemoved,
			})
			continue
		}

		// Compare values
		if !sv.areEqual(expectedValue, actualValue) {
			result.Match = false
			result.Message = "value mismatch"
			result.Diff = append(result.Diff, StateDiff{
				Path:     key,
				Expected: expectedValue,
				Actual:   actualValue,
				Type:     DiffTypeModified,
			})
		}
	}

	// Check for extra keys in actual
	for key, actualValue := range actual {
		_, exists := expected[key]
		if !exists {
			result.Match = false
			result.Message = "extra keys in actual state"
			result.Diff = append(result.Diff, StateDiff{
				Path:     key,
				Expected: nil,
				Actual:   actualValue,
				Type:     DiffTypeAdded,
			})
		}
	}

	// Store comparison details
	result.Details["expected_keys"] = len(expected)
	result.Details["actual_keys"] = len(actual)
	result.Details["differences"] = len(result.Diff)

	return result
}

// areEqual checks if two values are equal, with special handling for common types
func (sv *StateValidator) areEqual(expected, actual interface{}) bool {
	// Handle nil values
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}

	// Use reflect for deep comparison
	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)

	// Check if types are different
	if expectedValue.Type() != actualValue.Type() {
		// Try to convert to common type
		if expectedStr, ok := expected.(string); ok {
			if actualStr, ok := actual.(string); ok {
				return expectedStr == actualStr
			}
		}
		return false
	}

	// Compare based on type
	switch expected.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
			float32, float64, bool:
		return expected == actual
	default:
		// For complex types, try JSON comparison
		expectedBytes, err1 := json.Marshal(expected)
		actualBytes, err2 := json.Marshal(actual)
		if err1 == nil && err2 == nil {
			return string(expectedBytes) == string(actualBytes)
		}
		return false
	}
}

// AddExpectedState adds an expected state to the validator
func (sv *StateValidator) AddExpectedState(state ExpectedState) error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.expectedStates = append(sv.expectedStates, state)
	return nil
}

// ClearExpectedStates clears all expected states
func (sv *StateValidator) ClearExpectedStates() error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.expectedStates = sv.expectedStates[:0]
	return nil
}

// AddValidationRule adds a validation rule to the validator
func (sv *StateValidator) AddValidationRule(rule ValidationRule) error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.validationRules = append(sv.validationRules, rule)
	return nil
}

// ValidateState validates a state against all registered validation rules
func (sv *StateValidator) ValidateState(state StateSnapshot) []ValidationResult {
	sv.mu.RLock()
	rules := make([]ValidationRule, len(sv.validationRules))
	copy(rules, sv.validationRules)
	sv.mu.RUnlock()

	results := make([]ValidationResult, 0, len(rules))

	for _, rule := range rules {
		if rule.Enabled {
			result := rule.Validator(state)
			results = append(results, result)
		}
	}

	return results
}

// GetExpectedStates returns all expected states
func (sv *StateValidator) GetExpectedStates() []ExpectedState {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	states := make([]ExpectedState, len(sv.expectedStates))
	copy(states, sv.expectedStates)
	return states
}

// GetActualStates returns all actual states
func (sv *StateValidator) GetActualStates() []StateSnapshot {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	states := make([]StateSnapshot, len(sv.actualStates))
	copy(states, sv.actualStates)
	return states
}

// GetValidationRules returns all validation rules
func (sv *StateValidator) GetValidationRules() []ValidationRule {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	rules := make([]ValidationRule, len(sv.validationRules))
	copy(rules, sv.validationRules)
	return rules
}

// ClearActualStates clears all actual states
func (sv *StateValidator) ClearActualStates() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.actualStates = sv.actualStates[:0]
}

// ClearValidationRules clears all validation rules
func (sv *StateValidator) ClearValidationRules() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.validationRules = sv.validationRules[:0]
}

// EnableValidationRule enables or disables a validation rule by ID
func (sv *StateValidator) EnableValidationRule(ruleID string, enabled bool) error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	for i, rule := range sv.validationRules {
		if rule.ID == ruleID {
			sv.validationRules[i].Enabled = enabled
			return nil
		}
	}

	return fmt.Errorf("validation rule not found: %s", ruleID)
}