package services

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// PTYViewSwitchingValidator provides comprehensive validation for view switching through PTY input
// It validates that PTY input simulation correctly triggers view switching through the NavigationService
type PTYViewSwitchingValidator struct {
	// Core components
	ptyLayer          *pty.PTYLayer
	navigationAdapter *PTYNavigationAdapter
	stateTracker      *PTYNavigationStateTracker

	// Validation state
	mu                sync.RWMutex
	validations       []ValidationResult
	currentValidation *ValidationSession
	validationCount   int

	// Configuration
	config PTYViewSwitchingValidatorConfig

	// Statistics
	stats ValidatorStatistics
}

// ValidationResult represents the result of a validation test
type ValidationResult struct {
	TestName        string
	TestType        ValidationTestType
	Success         bool
	ExpectedView    messages.ViewType
	ActualView      messages.ViewType
	ExpectedTrigger string
	ActualTrigger   string
	Duration        time.Duration
	Error           string
	Details         map[string]interface{}
	Timestamp       time.Time
}

// ValidationTestType represents the type of validation test
type ValidationTestType int

const (
	ValidationTestNumberKey ValidationTestType = iota
	ValidationTestTabCycling
	ValidationTestKeySequence
	ValidationTestRapidSwitching
	ValidationTestInvalidKeys
	ValidationTestStateConsistency
	ValidationTestPerformance
)

// ValidationSession represents an active validation session
type ValidationSession struct {
	ID          string
	StartTime   time.Time
	EndTime     time.Time
	TestType    ValidationTestType
	TargetView  messages.ViewType
	Trigger     string
	KeySequence []tea.KeyMsg
	Timeout     time.Duration
	Completed   bool
	Results     []ValidationResult
}

// PTYViewSwitchingValidatorConfig contains configuration for the validator
type PTYViewSwitchingValidatorConfig struct {
	// Validation settings
	EnableValidation      bool
	ValidateStateChanges  bool
	ValidateTriggers      bool
	ValidatePerformance   bool
	ValidateHistory       bool

	// Timing settings
	DefaultTimeout        time.Duration
	MaxValidationTime     time.Duration
	MinSwitchTime         time.Duration
	MaxSwitchTime         time.Duration
	PerformanceThreshold  time.Duration

	// Test settings
	TestInvalidKeys       bool
	TestRapidSwitching    bool
	TestKeySequences      bool
	TestStateConsistency  bool

	// History settings
	MaxValidationHistory  int
	TrackValidationStats  bool

	// Debug settings
	EnableDebug           bool
	LogValidations        bool
	LogValidationDetails  bool
}

// ValidatorStatistics contains statistics about validation operations
type ValidatorStatistics struct {
	TotalValidations    int
	SuccessfulValidations int
	FailedValidations   int
	AverageValidationTime time.Duration
	MaxValidationTime   time.Duration
	MinValidationTime   time.Duration
	TestTypeCounts      map[ValidationTestType]int
	ErrorCounts         map[string]int
	LastValidation      time.Time
}

// DefaultPTYViewSwitchingValidatorConfig returns a default configuration
func DefaultPTYViewSwitchingValidatorConfig() PTYViewSwitchingValidatorConfig {
	return PTYViewSwitchingValidatorConfig{
		EnableValidation:     true,
		ValidateStateChanges: true,
		ValidateTriggers:     true,
		ValidatePerformance:  true,
		ValidateHistory:      true,
		DefaultTimeout:       5 * time.Second,
		MaxValidationTime:    30 * time.Second,
		MinSwitchTime:        1 * time.Millisecond,
		MaxSwitchTime:        2 * time.Second,
		PerformanceThreshold: 1 * time.Second,
		TestInvalidKeys:      true,
		TestRapidSwitching:   true,
		TestKeySequences:     true,
		TestStateConsistency: true,
		MaxValidationHistory: 100,
		TrackValidationStats: true,
		EnableDebug:          false,
		LogValidations:       false,
		LogValidationDetails: false,
	}
}

// NewPTYViewSwitchingValidator creates a new view switching validator
func NewPTYViewSwitchingValidator(
	ptyLayer *pty.PTYLayer,
	navigationAdapter *PTYNavigationAdapter,
	config PTYViewSwitchingValidatorConfig,
) (*PTYViewSwitchingValidator, error) {
	if ptyLayer == nil {
		return nil, fmt.Errorf("PTY layer is required")
	}
	if navigationAdapter == nil {
		return nil, fmt.Errorf("navigation adapter is required")
	}

	// Create state tracker
	stateTrackerConfig := DefaultPTYNavigationStateTrackerConfig()
	stateTrackerConfig.EnableDebug = config.EnableDebug
	stateTracker := NewPTYNavigationStateTracker(
		navigationAdapter.GetCurrentView(),
		stateTrackerConfig,
	)

	validator := &PTYViewSwitchingValidator{
		ptyLayer:          ptyLayer,
		navigationAdapter: navigationAdapter,
		stateTracker:      stateTracker,
		config:            config,
		validations:       make([]ValidationResult, 0, config.MaxValidationHistory),
		stats: ValidatorStatistics{
			TestTypeCounts: make(map[ValidationTestType]int),
			ErrorCounts:    make(map[string]int),
			MinValidationTime: time.Hour,
		},
	}

	return validator, nil
}

// ValidateNumberKey validates that a number key press switches to the correct view
func (v *PTYViewSwitchingValidator) ValidateNumberKey(number int, expectedView messages.ViewType) ValidationResult {
	testName := fmt.Sprintf("Number Key %d -> %s", number, expectedView)

	if v.config.LogValidations || v.config.EnableDebug {
		fmt.Printf("Starting validation: %s\n", testName)
	}

	session := &ValidationSession{
		ID:          fmt.Sprintf("numkey_%d_%d", number, time.Now().UnixNano()),
		StartTime:   time.Now(),
		TestType:    ValidationTestNumberKey,
		TargetView:  expectedView,
		Trigger:     "number_key",
		Timeout:     v.config.DefaultTimeout,
		Completed:   false,
		Results:     make([]ValidationResult, 0),
	}

	result := ValidationResult{
		TestName:        testName,
		TestType:        ValidationTestNumberKey,
		ExpectedView:    expectedView,
		ExpectedTrigger: "number_key",
		Timestamp:       time.Now(),
		Details:         make(map[string]interface{}),
	}

	// Record initial state
	initialView := v.navigationAdapter.GetCurrentView()
	initialSwitchCount := v.navigationAdapter.GetSwitchCount()

	result.Details["initial_view"] = initialView
	result.Details["initial_switch_count"] = initialSwitchCount

	// Validate number key mapping
	if err := v.stateTracker.ValidateNumberKeyMapping(number, expectedView); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Number key mapping validation failed: %v", err)
		result.Duration = time.Since(session.StartTime)
		return v.completeValidation(session, result)
	}

	// Simulate number key press
	key := tea.KeyMsg{
		Type: tea.KeyRunes,
		Runes: []rune{rune('0' + number)},
	}

	keyPressStart := time.Now()
	if err := v.navigationAdapter.SimulateKeyPress(key); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to simulate key press: %v", err)
		result.Duration = time.Since(session.StartTime)
		return v.completeValidation(session, result)
	}

	// Wait for view switch
	switchDeadline := time.Now().Add(v.config.DefaultTimeout)
	success := false
	var finalView messages.ViewType

	for time.Now().Before(switchDeadline) {
		currentView := v.navigationAdapter.GetCurrentView()
		if currentView != initialView {
			finalView = currentView
			success = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !success {
		result.Success = false
		result.Error = "View switch did not occur within timeout"
		result.Duration = time.Since(session.StartTime)
		return v.completeValidation(session, result)
	}

	// Validate final state
	result.ActualView = finalView
	result.Duration = time.Since(session.StartTime)

	if finalView != expectedView {
		result.Success = false
		result.Error = fmt.Sprintf("View mismatch: expected %s, got %s", expectedView, finalView)
		return v.completeValidation(session, result)
	}

	// Validate performance
	if v.config.ValidatePerformance && result.Duration > v.config.MaxSwitchTime {
		result.Success = false
		result.Error = fmt.Sprintf("Performance validation failed: switch took %v (max: %v)",
			result.Duration, v.config.MaxSwitchTime)
		return v.completeValidation(session, result)
	}

	// Record successful transition
	if err := v.stateTracker.RecordTransition(string(initialView), string(finalView), "number_key", result.Duration, string(rune('0'+number))); err != nil {
		if v.config.LogValidationDetails {
			fmt.Printf("Warning: Failed to record transition: %v\n", err)
		}
	}

	result.Success = true
	result.Details["key_press_duration"] = time.Since(keyPressStart)
	result.Details["final_switch_count"] = v.navigationAdapter.GetSwitchCount()

	return v.completeValidation(session, result)
}

// ValidateTabCycling validates that tab key cycling works correctly
func (v *PTYViewSwitchingValidator) ValidateTabCycling(cycles int) ValidationResult {
	testName := fmt.Sprintf("Tab Cycling (%d cycles)", cycles)

	if v.config.LogValidations || v.config.EnableDebug {
		fmt.Printf("Starting validation: %s\n", testName)
	}

	session := &ValidationSession{
		ID:          fmt.Sprintf("tabcycle_%d_%d", cycles, time.Now().UnixNano()),
		StartTime:   time.Now(),
		TestType:    ValidationTestTabCycling,
		Timeout:     time.Duration(cycles) * v.config.DefaultTimeout,
		Completed:   false,
		Results:     make([]ValidationResult, 0),
	}

	result := ValidationResult{
		TestName:     testName,
		TestType:     ValidationTestTabCycling,
		ExpectedTrigger: "cycle",
		Timestamp:    time.Now(),
		Details:      make(map[string]interface{}),
	}

	// Record initial state
	initialView := v.navigationAdapter.GetCurrentView()
	initialSwitchCount := v.navigationAdapter.GetSwitchCount()

	result.Details["initial_view"] = initialView
	result.Details["initial_switch_count"] = initialSwitchCount
	result.Details["requested_cycles"] = cycles

	// Get expected view order
	expectedOrder := v.stateTracker.GetViewOrder()
	if len(expectedOrder) == 0 {
		result.Success = false
		result.Error = "No view order configured for validation"
		result.Duration = time.Since(session.StartTime)
		return v.completeValidation(session, result)
	}

	// Track visited views
	visitedViews := make(map[messages.ViewType]bool)
	var viewSequence []messages.ViewType
	var cycleErrors []string

	// Perform tab cycling
	for i := 0; i < cycles; i++ {
		// Simulate tab key press
		key := tea.KeyMsg{Type: tea.KeyTab}
		keyPressStart := time.Now()

		if err := v.navigationAdapter.SimulateKeyPress(key); err != nil {
			cycleErrors = append(cycleErrors, fmt.Sprintf("Cycle %d: Failed to press tab: %v", i+1, err))
			continue
		}

		// Wait for view switch
		switchDeadline := time.Now().Add(v.config.DefaultTimeout)
		var currentView messages.ViewType
		switchOccurred := false

		for time.Now().Before(switchDeadline) {
			newView := v.navigationAdapter.GetCurrentView()
			if newView != currentView {
				currentView = newView
				switchOccurred = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if !switchOccurred {
			cycleErrors = append(cycleErrors, fmt.Sprintf("Cycle %d: No view switch occurred", i+1))
			continue
		}

		// Record view
		viewSequence = append(viewSequence, currentView)
		visitedViews[currentView] = true

		// Validate view order if enabled
		if v.config.ValidateStateChanges && i > 0 {
			if err := v.stateTracker.ValidateViewOrder(); err != nil {
				cycleErrors = append(cycleErrors, fmt.Sprintf("Cycle %d: View order validation failed: %v", i+1, err))
			}
		}

		result.Details[fmt.Sprintf("cycle_%d_view", i+1)] = currentView
		result.Details[fmt.Sprintf("cycle_%d_duration", i+1)] = time.Since(keyPressStart)
	}

	result.Duration = time.Since(session.StartTime)
	result.ActualView = v.navigationAdapter.GetCurrentView()

	// Validate results
	if len(cycleErrors) > 0 {
		result.Success = false
		result.Error = fmt.Sprintf("Tab cycling failed with %d errors: %s", len(cycleErrors), cycleErrors[0])
		result.Details["cycle_errors"] = cycleErrors
		return v.completeValidation(session, result)
	}

	// Validate that we visited multiple views
	if len(visitedViews) < 2 && cycles > 1 {
		result.Success = false
		result.Error = fmt.Sprintf("Tab cycling visited only %d unique views (expected at least 2)", len(visitedViews))
		return v.completeValidation(session, result)
	}

	// Validate performance
	if v.config.ValidatePerformance && result.Duration > time.Duration(cycles)*v.config.MaxSwitchTime {
		result.Success = false
		result.Error = fmt.Sprintf("Performance validation failed: %d cycles took %v (avg: %v, max avg: %v)",
			cycles, result.Duration, result.Duration/time.Duration(cycles), v.config.MaxSwitchTime)
		return v.completeValidation(session, result)
	}

	result.Success = true
	result.Details["unique_views_visited"] = len(visitedViews)
	result.Details["view_sequence"] = viewSequence
	result.Details["final_switch_count"] = v.navigationAdapter.GetSwitchCount()

	return v.completeValidation(session, result)
}

// ValidateKeySequence validates a sequence of key presses
func (v *PTYViewSwitchingValidator) ValidateKeySequence(keys []tea.KeyMsg, expectedFinalView messages.ViewType, expectedTrigger string) ValidationResult {
	testName := fmt.Sprintf("Key Sequence (%d keys)", len(keys))

	if v.config.LogValidations || v.config.EnableDebug {
		fmt.Printf("Starting validation: %s\n", testName)
	}

	session := &ValidationSession{
		ID:          fmt.Sprintf("keyseq_%d_%d", len(keys), time.Now().UnixNano()),
		StartTime:   time.Now(),
		TestType:    ValidationTestKeySequence,
		TargetView:  expectedFinalView,
		Trigger:     expectedTrigger,
		KeySequence: keys,
		Timeout:     time.Duration(len(keys)) * v.config.DefaultTimeout,
		Completed:   false,
		Results:     make([]ValidationResult, 0),
	}

	result := ValidationResult{
		TestName:        testName,
		TestType:        ValidationTestKeySequence,
		ExpectedView:    expectedFinalView,
		ExpectedTrigger: expectedTrigger,
		Timestamp:       time.Now(),
		Details:         make(map[string]interface{}),
	}

	// Record initial state
	initialView := v.navigationAdapter.GetCurrentView()
	initialSwitchCount := v.navigationAdapter.GetSwitchCount()

	result.Details["initial_view"] = initialView
	result.Details["initial_switch_count"] = initialSwitchCount
	result.Details["key_sequence_length"] = len(keys)

	// Execute key sequence
	var sequenceErrors []string
	for i, key := range keys {
		keyPressStart := time.Now()

		if err := v.navigationAdapter.SimulateKeyPress(key); err != nil {
			sequenceErrors = append(sequenceErrors, fmt.Sprintf("Key %d: Failed to simulate: %v", i, err))
			continue
		}

		// Small delay between keys
		time.Sleep(10 * time.Millisecond)
		result.Details[fmt.Sprintf("key_%d_duration", i)] = time.Since(keyPressStart)
	}

	result.Duration = time.Since(session.StartTime)
	result.ActualView = v.navigationAdapter.GetCurrentView()

	// Validate sequence execution
	if len(sequenceErrors) > 0 {
		result.Success = false
		result.Error = fmt.Sprintf("Key sequence failed with %d errors: %s", len(sequenceErrors), sequenceErrors[0])
		result.Details["sequence_errors"] = sequenceErrors
		return v.completeValidation(session, result)
	}

	// Validate final view if specified
	if expectedFinalView != "" && result.ActualView != expectedFinalView {
		result.Success = false
		result.Error = fmt.Sprintf("Final view mismatch: expected %s, got %s", expectedFinalView, result.ActualView)
		return v.completeValidation(session, result)
	}

	// Validate performance
	if v.config.ValidatePerformance && result.Duration > v.config.MaxValidationTime {
		result.Success = false
		result.Error = fmt.Sprintf("Performance validation failed: sequence took %v (max: %v)",
			result.Duration, v.config.MaxValidationTime)
		return v.completeValidation(session, result)
	}

	result.Success = true
	result.Details["final_switch_count"] = v.navigationAdapter.GetSwitchCount()

	return v.completeValidation(session, result)
}

// ValidateStateConsistency validates the overall navigation state consistency
func (v *PTYViewSwitchingValidator) ValidateStateConsistency() ValidationResult {
	testName := "State Consistency Validation"

	if v.config.LogValidations || v.config.EnableDebug {
		fmt.Printf("Starting validation: %s\n", testName)
	}

	session := &ValidationSession{
		ID:          fmt.Sprintf("statecons_%d", time.Now().UnixNano()),
		StartTime:   time.Now(),
		TestType:    ValidationTestStateConsistency,
		Timeout:     v.config.DefaultTimeout,
		Completed:   false,
		Results:     make([]ValidationResult, 0),
	}

	result := ValidationResult{
		TestName: testName,
		TestType: ValidationTestStateConsistency,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Get current states from different components
	adapterView := v.navigationAdapter.GetCurrentView()
	trackerView := v.stateTracker.GetCurrentView()
	adapterSwitchCount := v.navigationAdapter.GetSwitchCount()
	trackerSwitchCount := v.stateTracker.GetSwitchCount()

	result.Details["adapter_view"] = adapterView
	result.Details["tracker_view"] = trackerView
	result.Details["adapter_switch_count"] = adapterSwitchCount
	result.Details["tracker_switch_count"] = trackerSwitchCount

	var consistencyErrors []string

	// Validate view consistency
	if adapterView != trackerView {
		consistencyErrors = append(consistencyErrors,
			fmt.Sprintf("View mismatch: adapter=%s, tracker=%s", adapterView, trackerView))
	}

	// Validate switch count consistency (allow small differences due to timing)
	if adapterSwitchCount != trackerSwitchCount {
		diff := int(adapterSwitchCount) - int(trackerSwitchCount)
		if diff < -1 || diff > 1 {
			consistencyErrors = append(consistencyErrors,
				fmt.Sprintf("Switch count mismatch: adapter=%d, tracker=%d", adapterSwitchCount, trackerSwitchCount))
		}
	}

	// Validate tracker state
	if err := v.stateTracker.ValidateCurrentState(); err != nil {
		consistencyErrors = append(consistencyErrors,
			fmt.Sprintf("Tracker state validation failed: %v", err))
	}

	// Get statistics for validation
	adapterStats := v.navigationAdapter.GetStatistics()
	trackerStats := v.stateTracker.GetStatistics()

	result.Details["adapter_stats"] = adapterStats
	result.Details["tracker_stats"] = trackerStats

	result.Duration = time.Since(session.StartTime)

	if len(consistencyErrors) > 0 {
		result.Success = false
		result.Error = fmt.Sprintf("State consistency validation failed: %s", consistencyErrors[0])
		result.Details["consistency_errors"] = consistencyErrors
		return v.completeValidation(session, result)
	}

	result.Success = true
	result.Details["validation_time"] = result.Duration

	return v.completeValidation(session, result)
}

// GetValidationHistory returns the history of validation results
func (v *PTYViewSwitchingValidator) GetValidationHistory() []ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	history := make([]ValidationResult, len(v.validations))
	copy(history, v.validations)
	return history
}

// GetStatistics returns validation statistics
func (v *PTYViewSwitchingValidator) GetStatistics() ValidatorStatistics {
	v.mu.RLock()
	defer v.mu.RUnlock()

	stats := v.stats
	stats.TotalValidations = len(v.validations)

	successCount := 0
	var totalTime time.Duration
	maxTime := time.Duration(0)
	minTime := time.Hour

	for _, validation := range v.validations {
		if validation.Success {
			successCount++
		}
		totalTime += validation.Duration
		if validation.Duration > maxTime {
			maxTime = validation.Duration
		}
		if validation.Duration < minTime {
			minTime = validation.Duration
		}
		stats.TestTypeCounts[validation.TestType]++
	}

	stats.SuccessfulValidations = successCount
	stats.FailedValidations = stats.TotalValidations - successCount

	if stats.TotalValidations > 0 {
		stats.AverageValidationTime = totalTime / time.Duration(stats.TotalValidations)
		stats.MaxValidationTime = maxTime
		if successCount > 0 {
			stats.MinValidationTime = minTime
		}
	}

	return stats
}

// Reset clears validation history and statistics
func (v *PTYViewSwitchingValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.validations = v.validations[:0]
	v.validationCount = 0
	v.currentValidation = nil

	// Reset statistics
	v.stats = ValidatorStatistics{
		TestTypeCounts: make(map[ValidationTestType]int),
		ErrorCounts:    make(map[string]int),
		MinValidationTime: time.Hour,
	}

	// Reset state tracker
	v.stateTracker.Reset(v.navigationAdapter.GetCurrentView())
}

// completeValidation completes a validation session and records the result
func (v *PTYViewSwitchingValidator) completeValidation(session *ValidationSession, result ValidationResult) ValidationResult {
	session.EndTime = time.Now()
	session.Completed = true
	session.Results = append(session.Results, result)

	v.mu.Lock()
	defer v.mu.Unlock()

	// Add to validation history
	v.validations = append(v.validations, result)
	if len(v.validations) > v.config.MaxValidationHistory {
		v.validations = v.validations[1:]
	}

	v.validationCount++
	v.currentValidation = session

	// Update statistics
	if result.Success {
		v.stats.SuccessfulValidations++
	} else {
		v.stats.FailedValidations++
		if result.Error != "" {
			v.stats.ErrorCounts[result.Error]++
		}
	}
	v.stats.LastValidation = session.EndTime

	if v.config.LogValidations || v.config.EnableDebug {
		status := "PASS"
		if !result.Success {
			status = "FAIL"
		}
		fmt.Printf("Validation completed: %s [%s] (duration: %v)\n", result.TestName, status, result.Duration)
		if !result.Success && result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
	}

	return result
}

// GetStateTracker returns the underlying state tracker
func (v *PTYViewSwitchingValidator) GetStateTracker() *PTYNavigationStateTracker {
	return v.stateTracker
}

// GetNavigationAdapter returns the underlying navigation adapter
func (v *PTYViewSwitchingValidator) GetNavigationAdapter() *PTYNavigationAdapter {
	return v.navigationAdapter
}