package services

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// PTYNavigationStateTracker tracks navigation state for PTY testing scenarios
// It provides comprehensive state validation and monitoring capabilities
type PTYNavigationStateTracker struct {
	// State tracking
	mu                sync.RWMutex
	currentView       atomic.Value // stores messages.ViewType
	previousView      atomic.Value // stores messages.ViewType
	viewHistory       []ViewTransition
	switchCount       atomic.Uint64
	switchErrors      atomic.Uint64
	lastSwitchTime    atomic.Value // stores time.Time
	initialView       messages.ViewType

	// Validation state
	expectedViews     map[messages.ViewType]bool
	viewTransitions   map[messages.ViewType][]messages.ViewType
	numberKeyMappings map[int]messages.ViewType
	viewOrder         []messages.ViewType

	// Performance tracking
	switchDurations   []time.Duration
	lastActivity      atomic.Value // stores time.Time

	// Configuration
	config PTYNavigationStateTrackerConfig

	// Statistics
	stats NavigationStateStats
}

// ViewTransition represents a single view transition
type ViewTransition struct {
	FromView    messages.ViewType
	ToView      messages.ViewType
	Trigger     string
	Timestamp   time.Time
	Duration    time.Duration
	Success     bool
	Error       string
	KeySequence string
}

// PTYNavigationStateTrackerConfig contains configuration for the state tracker
type PTYNavigationStateTrackerConfig struct {
	// History management
	MaxHistorySize      int
	TrackDurations      bool
	TrackErrors         bool
	TrackKeySequences   bool

	// Validation settings
	ValidateTransitions bool
	ValidateViews       bool
	ValidateNumberKeys  bool
	ValidateTabCycling  bool

	// Performance settings
	MaxSwitchDuration   time.Duration
	WarningThreshold    time.Duration

	// Debug settings
	EnableDebug         bool
	LogTransitions      bool
	LogValidationErrors bool
}

// NavigationStateStats contains statistics about navigation state
type NavigationStateStats struct {
	TotalTransitions    uint64
	SuccessfulTransitions uint64
	FailedTransitions   uint64
	AverageSwitchTime   time.Duration
	MaxSwitchTime       time.Duration
	MinSwitchTime       time.Duration
	ViewVisitCounts     map[messages.ViewType]uint64
	TriggerCounts       map[string]uint64
	ErrorCounts         map[string]uint64
	LastActivity        time.Time
	InitialState        messages.ViewType
	CurrentState        messages.ViewType
}

// DefaultPTYNavigationStateTrackerConfig returns a default configuration
func DefaultPTYNavigationStateTrackerConfig() PTYNavigationStateTrackerConfig {
	return PTYNavigationStateTrackerConfig{
		MaxHistorySize:     100,
		TrackDurations:     true,
		TrackErrors:        true,
		TrackKeySequences:  true,
		ValidateTransitions: true,
		ValidateViews:      true,
		ValidateNumberKeys: true,
		ValidateTabCycling: true,
		MaxSwitchDuration:  5 * time.Second,
		WarningThreshold:   1 * time.Second,
		EnableDebug:        false,
		LogTransitions:     false,
		LogValidationErrors: false,
	}
}

// NewPTYNavigationStateTracker creates a new navigation state tracker
func NewPTYNavigationStateTracker(initialView messages.ViewType, config PTYNavigationStateTrackerConfig) *PTYNavigationStateTracker {
	tracker := &PTYNavigationStateTracker{
		config:        config,
		viewHistory:   make([]ViewTransition, 0, config.MaxHistorySize),
		expectedViews: make(map[messages.ViewType]bool),
		viewTransitions: make(map[messages.ViewType][]messages.ViewType),
		numberKeyMappings: make(map[int]messages.ViewType),
		switchDurations: make([]time.Duration, 0, config.MaxHistorySize),
		stats: NavigationStateStats{
			ViewVisitCounts: make(map[messages.ViewType]uint64),
			TriggerCounts:   make(map[string]uint64),
			ErrorCounts:     make(map[string]uint64),
			MinSwitchTime:   time.Hour, // Initialize to large value
		},
	}

	// Initialize expected views
	expectedViews := []messages.ViewType{
		messages.ProcessesView,
		messages.LogsView,
		messages.ErrorsView,
		messages.AICodersView,
		messages.SettingsView,
	}

	for _, view := range expectedViews {
		tracker.expectedViews[view] = true
		tracker.viewTransitions[view] = expectedViews // All views can transition to all others
		tracker.stats.ViewVisitCounts[view] = 0
	}

	// Set up number key mappings (1-5)
	tracker.numberKeyMappings[1] = messages.ProcessesView
	tracker.numberKeyMappings[2] = messages.LogsView
	tracker.numberKeyMappings[3] = messages.ErrorsView
	tracker.numberKeyMappings[4] = messages.AICodersView
	tracker.numberKeyMappings[5] = messages.SettingsView

	// Set up view order
	tracker.viewOrder = expectedViews

	// Initialize state
	tracker.initialView = initialView
	tracker.currentView.Store(initialView)
	tracker.previousView.Store(initialView)
	tracker.lastSwitchTime.Store(time.Now())
	tracker.lastActivity.Store(time.Now())

	// Initialize stats
	tracker.stats.InitialState = initialView
	tracker.stats.CurrentState = initialView
	tracker.stats.ViewVisitCounts[initialView] = 1

	return tracker
}

// RecordTransition records a view transition with validation
func (t *PTYNavigationStateTracker) RecordTransition(fromView, toView, trigger string, duration time.Duration, keySequence string) error {
	// Record start time for duration tracking
	startTime := time.Now()

	// Validate transition if enabled
	if t.config.ValidateTransitions {
		if err := t.validateTransition(fromView, toView, trigger); err != nil {
			t.recordFailedTransition(fromView, toView, trigger, duration, keySequence, err.Error())
			return fmt.Errorf("transition validation failed: %w", err)
		}
	}

	// Update atomic state
	t.previousView.Store(fromView)
	t.currentView.Store(toView)
	t.lastSwitchTime.Store(startTime)
	t.lastActivity.Store(startTime)

	// Create transition record
	transition := ViewTransition{
		FromView:    messages.ViewType(fromView),
		ToView:      messages.ViewType(toView),
		Trigger:     trigger,
		Timestamp:   startTime,
		Duration:    duration,
		Success:     true,
		KeySequence: keySequence,
	}

	// Update history
	t.mu.Lock()
	t.viewHistory = append(t.viewHistory, transition)
	if len(t.viewHistory) > t.config.MaxHistorySize {
		t.viewHistory = t.viewHistory[1:]
	}
	t.mu.Unlock()

	// Update statistics
	t.updateStatistics(transition)

	if t.config.LogTransitions || t.config.EnableDebug {
		fmt.Printf("Navigation transition: %s -> %s (trigger: %s, duration: %v)\n",
			fromView, toView, trigger, duration)
	}

	return nil
}

// RecordFailedTransition records a failed transition attempt
func (t *PTYNavigationStateTracker) RecordFailedTransition(fromView, toView, trigger, keySequence, errorMsg string) {
	t.recordFailedTransition(fromView, toView, trigger, 0, keySequence, errorMsg)
}

// recordFailedTransition is the internal implementation for recording failed transitions
func (t *PTYNavigationStateTracker) recordFailedTransition(fromView, toView, trigger string, duration time.Duration, keySequence, errorMsg string) {
	startTime := time.Now()
	t.lastActivity.Store(startTime)

	transition := ViewTransition{
		FromView:    messages.ViewType(fromView),
		ToView:      messages.ViewType(toView),
		Trigger:     trigger,
		Timestamp:   startTime,
		Duration:    duration,
		Success:     false,
		Error:       errorMsg,
		KeySequence: keySequence,
	}

	t.mu.Lock()
	t.viewHistory = append(t.viewHistory, transition)
	if len(t.viewHistory) > t.config.MaxHistorySize {
		t.viewHistory = t.viewHistory[1:]
	}
	t.mu.Unlock()

	t.switchErrors.Add(1)
	t.stats.FailedTransitions++
	t.stats.ErrorCounts[errorMsg]++

	if t.config.LogValidationErrors || t.config.EnableDebug {
		fmt.Printf("Navigation transition failed: %s -> %s (trigger: %s, error: %s)\n",
			fromView, toView, trigger, errorMsg)
	}
}

// GetCurrentView returns the current view
func (t *PTYNavigationStateTracker) GetCurrentView() messages.ViewType {
	if view := t.currentView.Load(); view != nil {
		return view.(messages.ViewType)
	}
	return ""
}

// GetPreviousView returns the previous view
func (t *PTYNavigationStateTracker) GetPreviousView() messages.ViewType {
	if view := t.previousView.Load(); view != nil {
		return view.(messages.ViewType)
	}
	return ""
}

// GetViewHistory returns the view transition history
func (t *PTYNavigationStateTracker) GetViewHistory() []ViewTransition {
	t.mu.RLock()
	defer t.mu.RUnlock()

	history := make([]ViewTransition, len(t.viewHistory))
	copy(history, t.viewHistory)
	return history
}

// GetSwitchCount returns the total number of switches
func (t *PTYNavigationStateTracker) GetSwitchCount() uint64 {
	return t.switchCount.Load()
}

// GetErrorCount returns the total number of errors
func (t *PTYNavigationStateTracker) GetErrorCount() uint64 {
	return t.switchErrors.Load()
}

// GetLastSwitchTime returns the timestamp of the last switch
func (t *PTYNavigationStateTracker) GetLastSwitchTime() time.Time {
	if t := t.lastSwitchTime.Load(); t != nil {
		return t.(time.Time)
	}
	return time.Time{}
}

// ValidateCurrentState validates the current navigation state
func (t *PTYNavigationStateTracker) ValidateCurrentState() error {
	currentView := t.GetCurrentView()

	if t.config.ValidateViews {
		if !t.expectedViews[currentView] {
			return fmt.Errorf("current view %s is not a valid view", currentView)
		}
	}

	return nil
}

// ValidateNumberKeyMapping validates a number key to view mapping
func (t *PTYNavigationStateTracker) ValidateNumberKeyMapping(number int, expectedView messages.ViewType) error {
	if t.config.ValidateNumberKeys {
		mappedView, exists := t.numberKeyMappings[number]
		if !exists {
			return fmt.Errorf("number key %d is not mapped", number)
		}
		if mappedView != expectedView {
			return fmt.Errorf("number key %d maps to %s, expected %s", number, mappedView, expectedView)
		}
	}
	return nil
}

// ValidateViewOrder validates that the current view follows the expected order
func (t *PTYNavigationStateTracker) ValidateViewOrder() error {
	if len(t.viewOrder) < 2 {
		return nil // No order to validate
	}

	currentView := t.GetCurrentView()
	previousView := t.GetPreviousView()

	if previousView == "" || currentView == "" {
		return nil // Not enough data to validate
	}

	// Find positions in order
	prevIndex := -1
	currIndex := -1
	for i, view := range t.viewOrder {
		if view == previousView {
			prevIndex = i
		}
		if view == currentView {
			currIndex = i
		}
	}

	// If both views are in the order, validate cycling
	if prevIndex != -1 && currIndex != -1 {
		// For tab cycling, we expect to move to the next view (with wrap-around)
		expectedNextIndex := (prevIndex + 1) % len(t.viewOrder)
		if currIndex != expectedNextIndex {
			return fmt.Errorf("view order validation failed: expected %s after %s, got %s",
				t.viewOrder[expectedNextIndex], previousView, currentView)
		}
	}

	return nil
}

// GetStatistics returns current navigation statistics
func (t *PTYNavigationStateTracker) GetStatistics() NavigationStateStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Update current state
	stats := t.stats
	stats.CurrentState = t.GetCurrentView()
	stats.TotalTransitions = t.switchCount.Load()
	stats.FailedTransitions = t.switchErrors.Load()
	stats.SuccessfulTransitions = stats.TotalTransitions - stats.FailedTransitions

	if lastActivity, ok := t.lastActivity.Load().(time.Time); ok {
		stats.LastActivity = lastActivity
	}

	// Calculate average switch time
	if len(t.switchDurations) > 0 {
		var total time.Duration
		max := time.Duration(0)
		min := time.Hour

		for _, duration := range t.switchDurations {
			total += duration
			if duration > max {
				max = duration
			}
			if duration < min {
				min = duration
			}
		}

		stats.AverageSwitchTime = total / time.Duration(len(t.switchDurations))
		stats.MaxSwitchTime = max
		stats.MinSwitchTime = min
	}

	return stats
}

// Reset resets the tracker state
func (t *PTYNavigationStateTracker) Reset(newInitialView messages.ViewType) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Reset atomic values
	t.currentView.Store(newInitialView)
	t.previousView.Store(newInitialView)
	t.lastSwitchTime.Store(time.Now())
	t.lastActivity.Store(time.Now())

	// Reset counters
	t.switchCount.Store(0)
	t.switchErrors.Store(0)

	// Clear history
	t.viewHistory = t.viewHistory[:0]
	t.switchDurations = t.switchDurations[:0]

	// Reset statistics
	t.stats = NavigationStateStats{
		ViewVisitCounts: make(map[messages.ViewType]uint64),
		TriggerCounts:   make(map[string]uint64),
		ErrorCounts:     make(map[string]uint64),
		MinSwitchTime:   time.Hour,
		InitialState:    newInitialView,
		CurrentState:    newInitialView,
	}

	// Reinitialize visit counts
	for view := range t.expectedViews {
		t.stats.ViewVisitCounts[view] = 0
	}
	t.stats.ViewVisitCounts[newInitialView] = 1
}

// validateTransition validates a view transition
func (t *PTYNavigationStateTracker) validateTransition(fromView, toView, trigger string) error {
	// Validate from and to views
	if t.config.ValidateViews {
		if !t.expectedViews[messages.ViewType(fromView)] {
			return fmt.Errorf("from view %s is not valid", fromView)
		}
		if !t.expectedViews[messages.ViewType(toView)] {
			return fmt.Errorf("to view %s is not valid", toView)
		}
	}

	// Validate trigger
	if trigger == "" {
		return fmt.Errorf("trigger cannot be empty")
	}

	// For cycle trigger, validate view order
	if trigger == "cycle" && t.config.ValidateTabCycling {
		return t.validateCyclingTransition(messages.ViewType(fromView), messages.ViewType(toView))
	}

	// For number_key trigger, validate mapping
	if trigger == "number_key" {
		// This would need the number key to validate properly
		// For now, just ensure it's a valid transition
	}

	return nil
}

// validateCyclingTransition validates a tab cycling transition
func (t *PTYNavigationStateTracker) validateCyclingTransition(fromView, toView messages.ViewType) error {
	if len(t.viewOrder) < 2 {
		return nil // No cycling order defined
	}

	// Find current view in order
	fromIndex := -1
	for i, view := range t.viewOrder {
		if string(view) == string(fromView) {
			fromIndex = i
			break
		}
	}

	if fromIndex == -1 {
		return fmt.Errorf("from view %s not found in view order", fromView)
	}

	// Calculate expected next view
	expectedNextIndex := (fromIndex + 1) % len(t.viewOrder)
	expectedView := t.viewOrder[expectedNextIndex]

	if string(toView) != string(expectedView) {
		return fmt.Errorf("cycling transition invalid: expected %s after %s, got %s",
			expectedView, fromView, toView)
	}

	return nil
}

// updateStatistics updates the statistics with a new transition
func (t *PTYNavigationStateTracker) updateStatistics(transition ViewTransition) {
	t.switchCount.Add(1)

	// Update visit counts
	t.stats.ViewVisitCounts[transition.ToView]++

	// Update trigger counts
	t.stats.TriggerCounts[transition.Trigger]++

	// Update duration statistics
	if t.config.TrackDurations && transition.Duration > 0 {
		t.mu.Lock()
		t.switchDurations = append(t.switchDurations, transition.Duration)
		if len(t.switchDurations) > t.config.MaxHistorySize {
			t.switchDurations = t.switchDurations[1:]
		}
		t.mu.Unlock()

		// Check for performance warnings
		if transition.Duration > t.config.WarningThreshold {
			if t.config.EnableDebug {
				fmt.Printf("Warning: Slow navigation transition detected: %v (threshold: %v)\n",
					transition.Duration, t.config.WarningThreshold)
			}
		}
	}

	// Update successful transition count
	t.stats.SuccessfulTransitions++
}

// GetExpectedViews returns the set of expected views
func (t *PTYNavigationStateTracker) GetExpectedViews() map[messages.ViewType]bool {
	result := make(map[messages.ViewType]bool)
	for view, valid := range t.expectedViews {
		result[view] = valid
	}
	return result
}

// GetNumberKeyMappings returns the number key to view mappings
func (t *PTYNavigationStateTracker) GetNumberKeyMappings() map[int]messages.ViewType {
	t.mu.RLock()
	defer t.mu.RUnlock()

	mappings := make(map[int]messages.ViewType)
	for k, v := range t.numberKeyMappings {
		mappings[k] = v
	}
	return mappings
}

// GetViewOrder returns the expected view order
func (t *PTYNavigationStateTracker) GetViewOrder() []messages.ViewType {
	t.mu.RLock()
	defer t.mu.RUnlock()

	order := make([]messages.ViewType, len(t.viewOrder))
	copy(order, t.viewOrder)
	return order
}

// HasVisited checks if a view has been visited
func (t *PTYNavigationStateTracker) HasVisited(view messages.ViewType) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.stats.ViewVisitCounts[view] > 0
}

// GetVisitCount returns the number of times a view has been visited
func (t *PTYNavigationStateTracker) GetVisitCount(view messages.ViewType) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.stats.ViewVisitCounts[view]
}