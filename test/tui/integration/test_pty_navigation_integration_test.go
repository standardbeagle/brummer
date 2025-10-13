//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// TestPTYNavigationIntegration tests the complete integration between PTY layer and NavigationService
// This test validates that view switching works correctly through the PTY layer connected to NavigationService
func TestPTYNavigationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY navigation integration test in short mode")
	}

	t.Log("🔗 TESTING: PTY-NavigationService Integration")
	t.Log("=========================================")

	// Create test components
	ctx := context.Background()
	eventBus := events.NewEventBus()
	navigationService := services.NewNavigationService(eventBus)

	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.EnableDebug = true
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create PTY navigation adapter
	adapterConfig := services.DefaultPTYNavigationAdapterConfig()
	adapterConfig.EnableDebug = true
	adapterConfig.EnableStatistics = true

	adapter, err := services.NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
	require.NoError(t, err, "Failed to create PTY navigation adapter")

	// Start adapter
	err = adapter.Start()
	require.NoError(t, err, "Failed to start PTY navigation adapter")
	defer adapter.Stop()

	// Create validator
	validatorConfig := services.DefaultPTYViewSwitchingValidatorConfig()
	validatorConfig.EnableDebug = true
	validatorConfig.ValidateStateChanges = true
	validatorConfig.ValidatePerformance = true

	validator, err := services.NewPTYViewSwitchingValidator(ptyLayer, adapter, validatorConfig)
	require.NoError(t, err, "Failed to create PTY view switching validator")

	t.Log("✅ PTY-NavigationService integration setup completed")

	// Test 1: Number Key Navigation
	t.Log("Test 1: Number Key Navigation")
	numberKeyTests := []struct {
		number       int
		expectedView messages.ViewType
	}{
		{1, messages.ProcessesView},
		{2, messages.LogsView},
		{3, messages.ErrorsView},
		{4, messages.AICodersView},
		{5, messages.SettingsView},
	}

	for _, test := range numberKeyTests {
		t.Logf("   Testing number key %d -> %s", test.number, test.expectedView)

		result := validator.ValidateNumberKey(test.number, test.expectedView)
		assert.True(t, result.Success, "Number key %d validation should succeed", test.number)
		assert.Equal(t, test.expectedView, result.ActualView, "Should switch to correct view")
		assert.Equal(t, "number_key", result.ActualTrigger, "Trigger should be number_key")

		if !result.Success {
			t.Logf("   ❌ Number key %d failed: %s", test.number, result.Error)
		} else {
			t.Logf("   ✅ Number key %d -> %s (duration: %v)", test.number, result.ActualView, result.Duration)
		}
	}

	// Test 2: Tab Cycling
	t.Log("Test 2: Tab Cycling")
	cyclingResult := validator.ValidateTabCycling(6) // Test 6 cycles to cover all views
	assert.True(t, cyclingResult.Success, "Tab cycling validation should succeed")
	assert.Greater(t, cyclingResult.Details["unique_views_visited"], 1, "Should visit multiple views during cycling")

	if cyclingResult.Success {
		t.Logf("   ✅ Tab cycling completed: visited %d unique views (duration: %v)",
			cyclingResult.Details["unique_views_visited"], cyclingResult.Duration)
	} else {
		t.Logf("   ❌ Tab cycling failed: %s", cyclingResult.Error)
	}

	// Test 3: Key Sequence Testing
	t.Log("Test 3: Key Sequence Testing")
	keySequence := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'2'}}, // Switch to Logs
		{Type: tea.KeyTab},                        // Cycle to next
		{Type: tea.KeyRunes, Runes: []rune{'4'}}, // Switch to AICoders
	}

	sequenceResult := validator.ValidateKeySequence(keySequence, messages.AICodersView, "number_key")
	assert.True(t, sequenceResult.Success, "Key sequence validation should succeed")
	assert.Equal(t, messages.AICodersView, sequenceResult.ActualView, "Should end on AICoders view")

	if sequenceResult.Success {
		t.Logf("   ✅ Key sequence completed: final view %s (duration: %v)",
			sequenceResult.ActualView, sequenceResult.Duration)
	} else {
		t.Logf("   ❌ Key sequence failed: %s", sequenceResult.Error)
	}

	// Test 4: State Consistency Validation
	t.Log("Test 4: State Consistency Validation")
	stateResult := validator.ValidateStateConsistency()
	assert.True(t, stateResult.Success, "State consistency validation should succeed")

	if stateResult.Success {
		t.Logf("   ✅ State consistency validated")
	} else {
		t.Logf("   ❌ State consistency failed: %s", stateResult.Error)
	}

	// Test 5: Performance Statistics
	t.Log("Test 5: Performance Statistics")
	adapterStats := adapter.GetStatistics()
	validatorStats := validator.GetStatistics()

	t.Logf("   Adapter Statistics:")
	t.Logf("     Total switches: %d", adapterStats.SwitchCount)
	t.Logf("     Key presses: %d", adapterStats.KeyPressCount)
	t.Logf("     Tab presses: %d", adapterStats.TabPresses)
	t.Logf("     View history length: %d", adapterStats.ViewHistoryLength)

	t.Logf("   Validator Statistics:")
	t.Logf("     Total validations: %d", validatorStats.TotalValidations)
	t.Logf("     Successful validations: %d", validatorStats.SuccessfulValidations)
	t.Logf("     Failed validations: %d", validatorStats.FailedValidations)
	t.Logf("     Average validation time: %v", validatorStats.AverageValidationTime)

	// Validate performance expectations
	assert.Greater(t, adapterStats.SwitchCount, uint64(0), "Should have recorded view switches")
	assert.Greater(t, adapterStats.KeyPressCount, uint64(0), "Should have recorded key presses")
	assert.Greater(t, validatorStats.SuccessfulValidations, 0, "Should have successful validations")
	assert.Less(t, validatorStats.AverageValidationTime, 2*time.Second, "Average validation time should be reasonable")

	t.Log("✅ PTY-NavigationService integration tests completed successfully")
}

// TestPTYNavigationAdapterIsolated tests the PTY navigation adapter in isolation
func TestPTYNavigationAdapterIsolated(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY navigation adapter isolated test in short mode")
	}

	t.Log("🧪 TESTING: PTY Navigation Adapter Isolated")
	t.Log("==========================================")

	// Create isolated test environment
	eventBus := events.NewEventBus()
	navigationService := services.NewNavigationService(eventBus)

	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err)

	err = ptyLayer.Allocate()
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	// Create adapter with testing configuration
	adapterConfig := services.DefaultPTYNavigationAdapterConfig()
	adapterConfig.EnableDebug = true
	adapterConfig.EnableStatistics = true
	adapterConfig.ValidateNumberKeys = true
	adapterConfig.ValidateTabCycling = true

	adapter, err := services.NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
	require.NoError(t, err)

	err = adapter.Start()
	require.NoError(t, err)
	defer adapter.Stop()

	// Test 1: Individual Key Processing
	t.Log("Test 1: Individual Key Processing")
	testKeys := map[tea.KeyMsg]messages.ViewType{
		{Type: tea.KeyRunes, Runes: []rune{'1'}}: messages.ProcessesView,
		{Type: tea.KeyRunes, Runes: []rune{'3'}}: messages.ErrorsView,
		{Type: tea.KeyRunes, Runes: []rune{'5'}}: messages.SettingsView,
	}

	for key, expectedView := range testKeys {
		t.Logf("   Testing key %v -> %s", key, expectedView)

		initialView := adapter.GetCurrentView()
		initialCount := adapter.GetSwitchCount()

		err := adapter.SimulateKeyPress(key)
		assert.NoError(t, err, "Should successfully simulate key press")

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		finalView := adapter.GetCurrentView()
		finalCount := adapter.GetSwitchCount()

		// Validate view change
		if initialView != expectedView {
			assert.Equal(t, expectedView, finalView, "Should switch to expected view")
			assert.Greater(t, finalCount, initialCount, "Switch count should increase")
		}

		t.Logf("   ✅ Key processed: %s -> %s", initialView, finalView)
	}

	// Test 2: Tab Cycling Test
	t.Log("Test 2: Tab Cycling Test")
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	cycles := 3

	initialView := adapter.GetCurrentView()
	t.Logf("   Starting tab cycling from view: %s", initialView)

	for i := 0; i < cycles; i++ {
		err := adapter.SimulateKeyPress(tabKey)
		assert.NoError(t, err, "Should successfully simulate tab press")
		time.Sleep(100 * time.Millisecond)
	}

	finalView := adapter.GetCurrentView()
	t.Logf("   After %d tab cycles, current view: %s", cycles, finalView)

	// Should have changed from initial view if we have multiple views
	viewOrder := navigationService.GetViewOrder()
	if len(viewOrder) > 1 {
		assert.NotEqual(t, initialView, finalView, "View should have changed after tab cycling")
	}

	// Test 3: Statistics Validation
	t.Log("Test 3: Statistics Validation")
	stats := adapter.GetStatistics()

	assert.True(t, stats.Active, "Adapter should be active")
	assert.Greater(t, stats.KeyPressCount, uint64(0), "Should have recorded key presses")
	assert.Greater(t, stats.SwitchCount, uint64(0), "Should have recorded view switches")
	assert.Greater(t, stats.TabPresses, uint64(0), "Should have recorded tab presses")

	t.Logf("   Statistics:")
	t.Logf("     Key presses: %d", stats.KeyPressCount)
	t.Logf("     View switches: %d", stats.SwitchCount)
	t.Logf("     Tab presses: %d", stats.TabPresses)
	t.Logf("     View history length: %d", stats.ViewHistoryLength)

	// Test 4: View History Validation
	t.Log("Test 4: View History Validation")
	history := adapter.GetViewHistory()
	assert.Greater(t, len(history), 0, "Should have view history")

	for i, transition := range history {
		t.Logf("   History[%d]: %s -> %s (trigger: %s)",
			i, transition.FromView, transition.ToView, transition.Trigger)
	}

	// Test 5: Wait for View Switch
	t.Log("Test 5: Wait for View Switch")
	targetCount := adapter.GetSwitchCount() + 1

	err = adapter.SimulateKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	require.NoError(t, err)

	err = adapter.WaitForViewSwitch(targetCount, 1*time.Second)
	assert.NoError(t, err, "Should wait for view switch successfully")

	currentView := adapter.GetCurrentView()
	expectedView, _ := navigationService.GetViewForNumberKey(2)
	assert.Equal(t, expectedView, currentView, "Should be on Logs view after pressing 2")

	t.Logf("   ✅ Wait for view switch successful: current view = %s", currentView)

	t.Log("✅ PTY Navigation Adapter isolated tests completed successfully")
}

// TestPTYNavigationStateTracker tests the navigation state tracker component
func TestPTYNavigationStateTracker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY navigation state tracker test in short mode")
	}

	t.Log("📊 TESTING: PTY Navigation State Tracker")
	t.Log("=======================================")

	// Create state tracker
	config := services.DefaultPTYNavigationStateTrackerConfig()
	config.EnableDebug = true
	config.ValidateTransitions = true
	config.TrackDurations = true

	tracker := services.NewPTYNavigationStateTracker(messages.ProcessesView, config)

	// Test 1: Initial State Validation
	t.Log("Test 1: Initial State Validation")
	currentView := tracker.GetCurrentView()
	assert.Equal(t, messages.ProcessesView, currentView, "Initial view should be ProcessesView")

	err := tracker.ValidateCurrentState()
	assert.NoError(t, err, "Initial state should be valid")

	statistics := tracker.GetStatistics()
	assert.Equal(t, uint64(0), statistics.TotalTransitions, "Should have no transitions initially")
	assert.Equal(t, messages.ProcessesView, statistics.CurrentState, "Current state should be ProcessesView")

	t.Log("   ✅ Initial state validated")

	// Test 2: Valid Transition Recording
	t.Log("Test 2: Valid Transition Recording")
	fromView := messages.ProcessesView
	toView := messages.LogsView
	trigger := "number_key"
	duration := 50 * time.Millisecond

	err = tracker.RecordTransition(fromView, toView, trigger, duration, "2")
	assert.NoError(t, err, "Should successfully record valid transition")

	currentView = tracker.GetCurrentView()
	assert.Equal(t, toView, currentView, "Current view should be updated")

	switchCount := tracker.GetSwitchCount()
	assert.Equal(t, uint64(1), switchCount, "Should have one transition recorded")

	t.Log("   ✅ Valid transition recorded")

	// Test 3: View History Tracking
	t.Log("Test 3: View History Tracking")
	history := tracker.GetViewHistory()
	assert.Len(t, history, 1, "Should have one transition in history")

	transition := history[0]
	assert.Equal(t, fromView, transition.FromView, "Should record correct from view")
	assert.Equal(t, toView, transition.ToView, "Should record correct to view")
	assert.Equal(t, trigger, transition.Trigger, "Should record correct trigger")
	assert.Equal(t, duration, transition.Duration, "Should record correct duration")
	assert.True(t, transition.Success, "Transition should be marked as successful")

	t.Log("   ✅ View history tracking validated")

	// Test 4: Multiple Transitions
	t.Log("Test 4: Multiple Transitions")
	transitions := []struct {
		from     messages.ViewType
		to       messages.ViewType
		trigger  string
		duration time.Duration
		key      string
	}{
		{messages.LogsView, messages.ErrorsView, "number_key", 45 * time.Millisecond, "3"},
		{messages.ErrorsView, messages.AICodersView, "number_key", 55 * time.Millisecond, "4"},
		{messages.AICodersView, messages.SettingsView, "cycle", 40 * time.Millisecond, "\t"},
	}

	for _, trans := range transitions {
		err = tracker.RecordTransition(trans.from, trans.to, trans.trigger, trans.duration, trans.key)
		assert.NoError(t, err, "Should record transition %s -> %s", trans.from, trans.to)
	}

	switchCount = tracker.GetSwitchCount()
	assert.Equal(t, uint64(4), switchCount, "Should have four transitions recorded")

	history = tracker.GetViewHistory()
	assert.Len(t, history, 4, "Should have four transitions in history")

	t.Log("   ✅ Multiple transitions recorded")

	// Test 5: Statistics Validation
	t.Log("Test 5: Statistics Validation")
	statistics = tracker.GetStatistics()
	assert.Equal(t, uint64(4), statistics.TotalTransitions, "Should have 4 total transitions")
	assert.Equal(t, uint64(4), statistics.SuccessfulTransitions, "Should have 4 successful transitions")
	assert.Equal(t, uint64(0), statistics.FailedTransitions, "Should have 0 failed transitions")
	assert.Greater(t, statistics.AverageSwitchTime, time.Duration(0), "Should have positive average switch time")

	// Check visit counts
	assert.Equal(t, uint64(1), statistics.ViewVisitCounts[messages.ProcessesView], "Should have visited ProcessesView once")
	assert.Equal(t, uint64(1), statistics.ViewVisitCounts[messages.LogsView], "Should have visited LogsView once")
	assert.Equal(t, uint64(1), statistics.ViewVisitCounts[messages.ErrorsView], "Should have visited ErrorsView once")
	assert.Equal(t, uint64(1), statistics.ViewVisitCounts[messages.AICodersView], "Should have visited AICodersView once")
	assert.Equal(t, uint64(1), statistics.ViewVisitCounts[messages.SettingsView], "Should have visited SettingsView once")

	// Check trigger counts
	assert.Equal(t, uint64(3), statistics.TriggerCounts["number_key"], "Should have 3 number_key triggers")
	assert.Equal(t, uint64(1), statistics.TriggerCounts["cycle"], "Should have 1 cycle trigger")

	t.Logf("   Statistics:")
	t.Logf("     Total transitions: %d", statistics.TotalTransitions)
	t.Logf("     Successful transitions: %d", statistics.SuccessfulTransitions)
	t.Logf("     Average switch time: %v", statistics.AverageSwitchTime)
	t.Logf("     Visit counts: %+v", statistics.ViewVisitCounts)
	t.Logf("     Trigger counts: %+v", statistics.TriggerCounts)

	t.Log("   ✅ Statistics validated")

	// Test 6: Failed Transition Recording
	t.Log("Test 6: Failed Transition Recording")
	initialCount := tracker.GetSwitchCount()
	errorCount := tracker.GetErrorCount()

	tracker.RecordFailedTransition(messages.SettingsView, messages.ProcessesView, "invalid", "1", "validation error")

	// Switch count should not increase for failed transitions
	assert.Equal(t, initialCount, tracker.GetSwitchCount(), "Switch count should not increase for failed transitions")
	assert.Equal(t, errorCount+1, tracker.GetErrorCount(), "Error count should increase for failed transitions")

	t.Log("   ✅ Failed transition recording validated")

	// Test 7: Number Key Mapping Validation
	t.Log("Test 7: Number Key Mapping Validation")
	err = tracker.ValidateNumberKeyMapping(1, messages.ProcessesView)
	assert.NoError(t, err, "Number key 1 should map to ProcessesView")

	err = tracker.ValidateNumberKeyMapping(9, messages.ProcessesView)
	assert.Error(t, err, "Number key 9 should not be mapped")

	t.Log("   ✅ Number key mapping validation completed")

	// Test 8: Reset Functionality
	t.Log("Test 8: Reset Functionality")
	tracker.Reset(messages.AICodersView)

	currentView = tracker.GetCurrentView()
	assert.Equal(t, messages.AICodersView, currentView, "Should reset to AICodersView")

	switchCount = tracker.GetSwitchCount()
	assert.Equal(t, uint64(0), switchCount, "Switch count should be reset to 0")

	history = tracker.GetViewHistory()
	assert.Len(t, history, 0, "History should be cleared after reset")

	statistics = tracker.GetStatistics()
	assert.Equal(t, uint64(0), statistics.TotalTransitions, "Statistics should be reset")
	assert.Equal(t, messages.AICodersView, statistics.CurrentState, "Current state should be reset")

	t.Log("   ✅ Reset functionality validated")

	t.Log("✅ PTY Navigation State Tracker tests completed successfully")
}

// TestPTYNavigationIntegrationPerformance tests performance characteristics of the integration
func TestPTYNavigationIntegrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PTY navigation integration performance test in short mode")
	}

	t.Log("⚡ TESTING: PTY Navigation Integration Performance")
	t.Log("==============================================")

	// Setup components
	eventBus := events.NewEventBus()
	navigationService := services.NewNavigationService(eventBus)

	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err)

	err = ptyLayer.Allocate()
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	adapterConfig := services.DefaultPTYNavigationAdapterConfig()
	adapterConfig.EnableStatistics = true
	adapterConfig.EnableDebug = false // Disable debug for performance testing

	adapter, err := services.NewPTYNavigationAdapter(ptyLayer, navigationService, eventBus, adapterConfig)
	require.NoError(t, err)

	err = adapter.Start()
	require.NoError(t, err)
	defer adapter.Stop()

	// Performance Test 1: Rapid Key Presses
	t.Log("Performance Test 1: Rapid Key Presses")
	rapidKeyCount := 100
	startTime := time.Now()

	for i := 0; i < rapidKeyCount; i++ {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('1' + (i%5))}} // Cycle through 1-5
		err := adapter.SimulateKeyPress(key)
		assert.NoError(t, err, "Should handle rapid key presses")
	}

	rapidKeyDuration := time.Since(startTime)
	avgKeyTime := rapidKeyDuration / time.Duration(rapidKeyCount)

	t.Logf("   Rapid key presses: %d keys in %v (avg: %v per key)",
		rapidKeyCount, rapidKeyDuration, avgKeyTime)

	// Performance expectations
	assert.Less(t, avgKeyTime, 10*time.Millisecond, "Average key press time should be under 10ms")
	assert.Less(t, rapidKeyDuration, 2*time.Second, "Total rapid key test should complete quickly")

	// Performance Test 2: Rapid Tab Cycling
	t.Log("Performance Test 2: Rapid Tab Cycling")
	tabCount := 50
	startTime = time.Now()

	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	for i := 0; i < tabCount; i++ {
		err := adapter.SimulateKeyPress(tabKey)
		assert.NoError(t, err, "Should handle rapid tab presses")
	}

	tabDuration := time.Since(startTime)
	avgTabTime := tabDuration / time.Duration(tabCount)

	t.Logf("   Rapid tab cycling: %d tabs in %v (avg: %v per tab)",
		tabCount, tabDuration, avgTabTime)

	assert.Less(t, avgTabTime, 10*time.Millisecond, "Average tab press time should be under 10ms")
	assert.Less(t, tabDuration, 1*time.Second, "Total tab test should complete quickly")

	// Performance Test 3: Statistics Collection Overhead
	t.Log("Performance Test 3: Statistics Collection Overhead")
	statsIterations := 1000
	startTime = time.Now()

	for i := 0; i < statsIterations; i++ {
		_ = adapter.GetStatistics()
	}

	statsDuration := time.Since(startTime)
	avgStatsTime := statsDuration / time.Duration(statsIterations)

	t.Logf("   Statistics collection: %d calls in %v (avg: %v per call)",
		statsIterations, statsDuration, avgStatsTime)

	assert.Less(t, avgStatsTime, 1*time.Millisecond, "Statistics collection should be very fast")

	// Performance Test 4: View History Access
	t.Log("Performance Test 4: View History Access")
	historyIterations := 500
	startTime = time.Now()

	for i := 0; i < historyIterations; i++ {
		_ = adapter.GetViewHistory()
	}

	historyDuration := time.Since(startTime)
	avgHistoryTime := historyDuration / time.Duration(historyIterations)

	t.Logf("   View history access: %d calls in %v (avg: %v per call)",
		historyIterations, historyDuration, avgHistoryTime)

	assert.Less(t, avgHistoryTime, 1*time.Millisecond, "History access should be very fast")

	// Final Statistics Check
	t.Log("Final Statistics Check")
	finalStats := adapter.GetStatistics()

	t.Logf("   Final performance statistics:")
	t.Logf("     Total key presses: %d", finalStats.KeyPressCount)
	t.Logf("     Total view switches: %d", finalStats.SwitchCount)
	t.Logf("     Total tab presses: %d", finalStats.TabPresses)
	t.Logf("     View history length: %d", finalStats.ViewHistoryLength)

	// Validate that we processed all the keys
	assert.Equal(t, uint64(rapidKeyCount+tabCount), finalStats.KeyPressCount, "Should have recorded all key presses")
	assert.Greater(t, finalStats.SwitchCount, uint64(0), "Should have recorded view switches")
	assert.Equal(t, uint64(tabCount), finalStats.TabPresses, "Should have recorded all tab presses")

	t.Log("✅ PTY Navigation Integration Performance tests completed successfully")
}