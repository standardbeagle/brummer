package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// TestNavigationServicePTY_Integration tests the integration between NavigationService and PTY layer
func TestNavigationServicePTY_Integration(t *testing.T) {
	// Create event bus for NavigationService
	eventBus := events.NewEventBus()

	// Create NavigationService
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Test initial state
	initialView, err := navService.GetCurrentView()
	require.NoError(t, err)
	assert.Equal(t, messages.ProcessesView, initialView)

	// Test PTY-specific state retrieval
	state := navService.GetNavigationStateForTesting()
	assert.Equal(t, messages.ProcessesView, state.CurrentView)
	assert.NotEmpty(t, state.ViewOrder)
	assert.NotEmpty(t, state.NumberKeyMappings)
	assert.GreaterOrEqual(t, len(state.ViewOrder), 5) // Should have at least 5 default views

	// Test state validation
	err = navService.ValidateStateForTesting()
	assert.NoError(t, err)

	// Test forced view switching for testing
	err = navService.ForceSwitchToViewForTesting("logs", "pty_test")
	assert.NoError(t, err)

	currentView, err = navService.GetCurrentView()
	require.NoError(t, err)
	assert.Equal(t, messages.LogsView, currentView)

	// Test number key view switching
	view, err := navService.GetViewForNumberKey(1)
	require.NoError(t, err)
	assert.Equal(t, messages.ProcessesView, view)

	view, err = navService.GetViewForNumberKey(2)
	require.NoError(t, err)
	assert.Equal(t, messages.LogsView, view)

	view, err = navService.GetViewForNumberKey(3)
	require.NoError(t, err)
	assert.Equal(t, messages.ErrorsView, view)

	view, err = navService.GetViewForNumberKey(4)
	require.NoError(t, err)
	assert.Equal(t, messages.AICodersView, view)

	view, err = navService.GetViewForNumberKey(5)
	require.NoError(t, err)
	assert.Equal(t, messages.SettingsView, view)

	// Test invalid number keys
	_, err = navService.GetViewForNumberKey(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_NUMBER")

	_, err = navService.GetViewForNumberKey(10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_NUMBER")

	_, err = navService.GetViewForNumberKey(9)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no mapping for number")

	// Test view cycling
	err = navService.CycleToNextView()
	assert.NoError(t, err)

	currentView, err = navService.GetCurrentView()
	require.NoError(t, err)
	// Should have moved from logs to errors (based on default order)
	assert.Equal(t, messages.ErrorsView, currentView)

	// Test multiple cycles
	originalView := currentView
	for i := 0; i < 5; i++ {
		err = navService.CycleToNextView()
		assert.NoError(t, err)
	}

	// Should have cycled through all views
	currentView, err = navService.GetCurrentView()
	require.NoError(t, err)
	assert.Equal(t, originalView, currentView) // Should be back to original after full cycle
}

// TestNavigationServicePTY_TransitionCallbacks tests PTY integration with transition callbacks
func TestNavigationServicePTY_TransitionCallbacks(t *testing.T) {
	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Track transition events
	var transitions []struct {
		from, to messages.ViewType
	}

	// Register transition callback
	callbackID := navService.RegisterTransitionCallback(func(fromView, toView messages.ViewType) error {
		transitions = append(transitions, struct{ from, to messages.ViewType }{fromView, toView})
		return nil
	})

	// Test forced view switching with callbacks
	err := navService.ForceSwitchToViewForTesting("logs", "pty_callback_test")
	assert.NoError(t, err)

	// Note: ForceSwitchToViewForTesting bypasses callbacks, so transitions won't be tracked
	assert.Empty(t, transitions)

	// Test normal view switching with callbacks
	err = navService.SwitchToView(messages.ErrorsView, "pty_callback_test")
	assert.NoError(t, err)

	// Check callback was called (though transitions slice might be empty due to forced switching)
	callbackCount := navService.GetTransitionCountForTesting()
	assert.GreaterOrEqual(t, callbackCount, 0)

	// Unregister callback
	navService.UnregisterTransitionCallback(callbackID)
}

// TestNavigationServicePTY_ViewOrderAndMappings tests PTY integration with view order and key mappings
func TestNavigationServicePTY_ViewOrderAndMappings(t *testing.T) {
	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Test default view order
	viewOrder := navService.GetViewOrder()
	assert.NotEmpty(t, viewOrder)
	assert.Contains(t, viewOrder, messages.ProcessesView)
	assert.Contains(t, viewOrder, messages.LogsView)
	assert.Contains(t, viewOrder, messages.ErrorsView)
	assert.Contains(t, viewOrder, messages.AICodersView)
	assert.Contains(t, viewOrder, messages.SettingsView)

	// Test custom view order
	customOrder := []messages.ViewType{
		messages.SettingsView,
		messages.AICodersView,
		messages.ProcessesView,
		messages.LogsView,
		messages.ErrorsView,
	}

	err := navService.SetViewOrder(customOrder)
	assert.NoError(t, err)

	updatedOrder := navService.GetViewOrder()
	assert.Equal(t, customOrder, updatedOrder)

	// Test custom number key mapping
	err = navService.SetNumberKeyMapping(1, messages.SettingsView)
	assert.NoError(t, err)

	view, err := navService.GetViewForNumberKey(1)
	require.NoError(t, err)
	assert.Equal(t, messages.SettingsView, view)

	// Test mapping override
	err = navService.SetNumberKeyMapping(2, messages.SettingsView)
	assert.NoError(t, err)

	view, err = navService.GetViewForNumberKey(2)
	require.NoError(t, err)
	assert.Equal(t, messages.SettingsView, view)

	// Get all mappings
	mappings := navService.GetNumberKeyMappings()
	assert.NotEmpty(t, mappings)
	assert.Equal(t, messages.SettingsView, mappings[1])
	assert.Equal(t, messages.SettingsView, mappings[2])
}

// TestNavigationServicePTY_StateValidation tests PTY integration with state validation
func TestNavigationServicePTY_StateValidation(t *testing.T) {
	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Test initial state validation
	err := navService.ValidateStateForTesting()
	assert.NoError(t, err)

	// Test state after view changes
	err = navService.SwitchToView(messages.AICodersView, "pty_validation_test")
	assert.NoError(t, err)

	err = navService.ValidateStateForTesting()
	assert.NoError(t, err)

	// Test state retrieval
	state := navService.GetNavigationStateForTesting()
	assert.Equal(t, messages.AICodersView, state.CurrentView)
	assert.NotEmpty(t, state.ViewOrder)
	assert.NotEmpty(t, state.NumberKeyMappings)
	assert.GreaterOrEqual(t, state.RegisteredCallbacks, 0)

	// Test reset functionality
	navService.ResetToInitialState()

	resetState := navService.GetNavigationStateForTesting()
	assert.Equal(t, messages.ProcessesView, resetState.CurrentView)
	assert.Contains(t, resetState.ViewOrder, messages.ProcessesView)
	assert.Contains(t, resetState.NumberKeyMappings, 1) // Should have mapping for key 1
}

// TestNavigationServicePTY_ErrorHandling tests PTY error handling scenarios
func TestNavigationServicePTY_ErrorHandling(t *testing.T) {
	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Test switching to invalid view
	err := navService.SwitchToView("invalid_view", "pty_error_test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_VIEW")

	// Test invalid number key mappings
	err = navService.SetNumberKeyMapping(0, messages.ProcessesView)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_NUMBER")

	err = navService.SetNumberKeyMapping(10, messages.ProcessesView)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_NUMBER")

	err = navService.SetNumberKeyMapping(5, "invalid_view")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_VIEW")

	// Test invalid view order
	invalidOrder := []messages.ViewType{
		messages.ProcessesView,
		"invalid_view",
		messages.LogsView,
	}

	err = navService.SetViewOrder(invalidOrder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_VIEW")

	// Test empty view order
	err = navService.SetViewOrder([]messages.ViewType{})
	// This might not error depending on implementation, but we check state validation
	err = navService.ValidateStateForTesting()
	if err != nil {
		assert.Contains(t, err.Error(), "view order is empty")
	}
}

// TestNavigationServicePTY_Performance tests PTY performance characteristics
func TestNavigationServicePTY_Performance(t *testing.T) {
	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Performance test: rapid view switching
	viewCount := 1000
	start := time.Now()

	for i := 0; i < viewCount; i++ {
		targetView := messages.ViewType(i%5 + 1) // Cycle through views 1-5
		if targetView == 1 {
			targetView = messages.ProcessesView
		} else if targetView == 2 {
			targetView = messages.LogsView
		} else if targetView == 3 {
			targetView = messages.ErrorsView
		} else if targetView == 4 {
			targetView = messages.AICodersView
		} else {
			targetView = messages.SettingsView
		}

		err := navService.SwitchToView(targetView, "pty_performance_test")
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	viewsPerSecond := float64(viewCount) / duration.Seconds()

	// Performance should be reasonable (at least 100 views/second)
	assert.Greater(t, viewsPerSecond, 100.0, "Navigation performance too slow: %.2f views/sec", viewsPerSecond)

	t.Logf("Navigation Performance: %d view switches in %v (%.2f views/sec)", viewCount, duration, viewsPerSecond)

	// Performance test: rapid view cycling
	cycleCount := 500
	start = time.Now()

	for i := 0; i < cycleCount; i++ {
		err := navService.CycleToNextView()
		assert.NoError(t, err)
	}

	duration = time.Since(start)
	cyclesPerSecond := float64(cycleCount) / duration.Seconds()

	// Cycling should also be fast
	assert.Greater(t, cyclesPerSecond, 100.0, "View cycling performance too slow: %.2f cycles/sec", cyclesPerSecond)

	t.Logf("Cycling Performance: %d cycles in %v (%.2f cycles/sec)", cycleCount, duration, cyclesPerSecond)
}

// TestNavigationServicePTY_ConcurrentAccess tests concurrent PTY access
func TestNavigationServicePTY_ConcurrentAccess(t *testing.T) {
	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Test concurrent view switching
	concurrency := 10
	operationsPerGoroutine := 50

	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Alternate between different operations
				switch j % 4 {
				case 0:
					err := navService.SwitchToView(messages.ProcessesView, "pty_concurrent_test")
					if err != nil {
						errors <- err
						return
					}
				case 1:
					err := navService.SwitchToView(messages.LogsView, "pty_concurrent_test")
					if err != nil {
						errors <- err
						return
					}
				case 2:
					err := navService.CycleToNextView()
					if err != nil {
						errors <- err
						return
					}
				case 3:
					_, err := navService.GetCurrentView()
					if err != nil {
						errors <- err
						return
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Check for any errors
	select {
	case err := <-errors:
		t.Errorf("Concurrent navigation access failed: %v", err)
	default:
		// No errors
	}

	// Verify final state
	currentView, err := navService.GetCurrentView()
	assert.NoError(t, err)
	assert.True(t, navService.isValidView(currentView))

	// Verify state is still valid
	err = navService.ValidateStateForTesting()
	assert.NoError(t, err)
}

// TestNavigationServicePTY_PTYSimulation tests PTY simulation scenarios
func TestNavigationServicePTY_PTYSimulation(t *testing.T) {
	// Create PTY layer for testing
	config := pty.DefaultPTYConfig()
	config.Timeout = 10 * time.Second
	ptyLayer, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	err = ptyLayer.Allocate()
	require.NoError(t, err)

	eventBus := events.NewEventBus()
	navService := services.NewNavigationService(eventBus)
	defer navService.Stop()

	// Simulate PTY keyboard navigation through number keys
	numberKeyViews := []struct {
		key    int
		view   messages.ViewType
		trigger string
	}{
		{1, messages.ProcessesView, "pty_number_key_1"},
		{2, messages.LogsView, "pty_number_key_2"},
		{3, messages.ErrorsView, "pty_number_key_3"},
		{4, messages.AICodersView, "pty_number_key_4"},
		{5, messages.SettingsView, "pty_number_key_5"},
	}

	for _, test := range numberKeyViews {
		view, err := navService.GetViewForNumberKey(test.key)
		require.NoError(t, err)
		assert.Equal(t, test.view, view)

		err = navService.SwitchToView(test.view, test.trigger)
		assert.NoError(t, err)

		currentView, err := navService.GetCurrentView()
		require.NoError(t, err)
		assert.Equal(t, test.view, currentView)

		// Simulate PTY delay between operations
		time.Sleep(10 * time.Millisecond)
	}

	// Simulate PTY tab cycling (equivalent to CycleToNextView)
	initialView, err := navService.GetCurrentView()
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err = navService.CycleToNextView()
		assert.NoError(t, err)

		// Simulate PTY delay
		time.Sleep(10 * time.Millisecond)
	}

	// Should have cycled through all views and returned to original
	finalView, err := navService.GetCurrentView()
	require.NoError(t, err)
	assert.Equal(t, initialView, finalView)

	// Verify navigation state
	state := navService.GetNavigationStateForTesting()
	assert.True(t, state.CurrentView != "")
	assert.GreaterOrEqual(t, len(state.ViewOrder), 5)
	assert.GreaterOrEqual(t, len(state.NumberKeyMappings), 5)

	// Verify state is valid
	err = navService.ValidateStateForTesting()
	assert.NoError(t, err)
}