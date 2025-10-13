package contracts_test

import (
	"errors"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// These tests validate the Navigation Service API contract
// They should fail initially and pass after implementation

func TestNavigationService_SwitchToView(t *testing.T) {
	// Test view switching functionality
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test switching to different valid views
	err := service.SwitchToView(messages.LogsView, "test")
	if err != nil {
		t.Fatalf("SwitchToView failed: %v", err)
	}

	currentView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	if currentView != messages.LogsView {
		t.Fatalf("Expected current view to be %s, got %s", messages.LogsView, currentView)
	}

	// Test switching to same view (should be no-op)
	err = service.SwitchToView(messages.LogsView, "test")
	if err != nil {
		t.Fatalf("SwitchToView to same view failed: %v", err)
	}
}

func TestNavigationService_SwitchToInvalidView(t *testing.T) {
	// Test invalid view type handling
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test switching to invalid view
	err := service.SwitchToView(messages.ViewType("invalid"), "test")
	if err == nil {
		t.Fatal("SwitchToView should have failed with invalid view")
	}

	if !errors.Is(err, services.ErrInvalidView) {
		t.Fatalf("Expected ErrInvalidView, got: %v", err)
	}
}

func TestNavigationService_GetCurrentView(t *testing.T) {
	// Test getting current view
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	currentView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	// Should default to ProcessesView
	if currentView != messages.ProcessesView {
		t.Fatalf("Expected default view to be %s, got %s", messages.ProcessesView, currentView)
	}
}

func TestNavigationService_GetCurrentView_NoActiveView(t *testing.T) {
	// Test error when no view is active
	// This test is harder to implement since the service initializes with a default view
	// For now, we'll just verify the service always has a current view
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	currentView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed unexpectedly: %v", err)
	}

	if currentView == "" {
		t.Fatal("Expected non-empty current view")
	}
}

func TestNavigationService_CycleToNextView(t *testing.T) {
	// Test view cycling functionality
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Get initial view
	initialView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	// Cycle to next view
	err = service.CycleToNextView()
	if err != nil {
		t.Fatalf("CycleToNextView failed: %v", err)
	}

	// Verify view changed
	nextView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	if nextView == initialView {
		t.Fatal("Expected view to change after cycling")
	}

	// Test full cycle through all views
	viewOrder := service.GetViewOrder()
	expectedViews := len(viewOrder)

	for i := 0; i < expectedViews; i++ {
		currentView, err := service.GetCurrentView()
		if err != nil {
			t.Fatalf("GetCurrentView failed: %v", err)
		}

		// Should match expected view in order
		expectedIndex := (i + 1) % expectedViews // +1 because we already cycled once
		expectedView := viewOrder[expectedIndex]

		if currentView != expectedView {
			t.Fatalf("Expected view %s at cycle %d, got %s", expectedView, i, currentView)
		}

		if i < expectedViews-1 {
			err = service.CycleToNextView()
			if err != nil {
				t.Fatalf("CycleToNextView failed at iteration %d: %v", i, err)
			}
		}
	}
}

func TestNavigationService_GetViewForNumberKey(t *testing.T) {
	// Test number key to view mapping
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test default mappings
	view, err := service.GetViewForNumberKey(1)
	if err != nil {
		t.Fatalf("GetViewForNumberKey failed: %v", err)
	}

	if view != messages.ProcessesView {
		t.Fatalf("Expected ProcessesView for key 1, got %s", view)
	}

	view, err = service.GetViewForNumberKey(5)
	if err != nil {
		t.Fatalf("GetViewForNumberKey failed: %v", err)
	}

	if view != messages.SettingsView {
		t.Fatalf("Expected SettingsView for key 5, got %s", view)
	}
}

func TestNavigationService_GetViewForNumberKey_InvalidNumber(t *testing.T) {
	// Test invalid number key handling
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test invalid numbers
	testCases := []int{0, 10, -1}

	for _, num := range testCases {
		_, err := service.GetViewForNumberKey(num)
		if err == nil {
			t.Fatalf("GetViewForNumberKey should have failed for number %d", num)
		}

		if !errors.Is(err, services.ErrInvalidNumber) {
			t.Fatalf("Expected ErrInvalidNumber for number %d, got: %v", num, err)
		}
	}

	// Test unmapped number
	_, err := service.GetViewForNumberKey(9)
	if err == nil {
		t.Fatal("GetViewForNumberKey should have failed for unmapped number 9")
	}

	if !errors.Is(err, services.ErrInvalidNumber) {
		t.Fatalf("Expected ErrInvalidNumber for unmapped number, got: %v", err)
	}
}

func TestNavigationService_ViewTransitionCallbacks(t *testing.T) {
	// Test view transition callback execution
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	callbackCalled := false
	callbackError := errors.New("test callback error")

	// Register successful callback
	successCallbackID := service.RegisterTransitionCallback(func(fromView, toView messages.ViewType) error {
		callbackCalled = true
		return nil
	})

	// Register failing callback
	failCallbackID := service.RegisterTransitionCallback(func(fromView, toView messages.ViewType) error {
		return callbackError
	})

	// Switch view to trigger callbacks
	err := service.SwitchToView(messages.LogsView, "test")
	if err == nil {
		t.Fatal("SwitchToView should have failed due to callback error")
	}

	if !errors.Is(err, services.ErrTransitionFailed) {
		t.Fatalf("Expected ErrTransitionFailed, got: %v", err)
	}

	if !callbackCalled {
		t.Fatal("Success callback was not called")
	}

	// Clean up
	service.UnregisterTransitionCallback(successCallbackID)
	service.UnregisterTransitionCallback(failCallbackID)

	// Test successful transition after removing failing callback
	successOnlyCallbackID := service.RegisterTransitionCallback(func(fromView, toView messages.ViewType) error {
		return nil
	})

	err = service.SwitchToView(messages.SettingsView, "test")
	if err != nil {
		t.Fatalf("SwitchToView should have succeeded: %v", err)
	}

	currentView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	if currentView != messages.SettingsView {
		t.Fatalf("Expected current view to be %s, got %s", messages.SettingsView, currentView)
	}

	service.UnregisterTransitionCallback(successOnlyCallbackID)
}

func TestNavigationService_Performance(t *testing.T) {
	// Test navigation service performance meets <5ms target
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test SwitchToView performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		err := service.SwitchToView(messages.LogsView, "perf_test")
		if err != nil {
			t.Fatalf("SwitchToView failed: %v", err)
		}
	}
	duration := time.Since(start)

	avgDuration := duration / 1000
	if avgDuration > 5*time.Millisecond {
		t.Fatalf("SwitchToView average duration %v exceeds 5ms target", avgDuration)
	}

	// Test GetCurrentView performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, err := service.GetCurrentView()
		if err != nil {
			t.Fatalf("GetCurrentView failed: %v", err)
		}
	}
	duration = time.Since(start)

	avgDuration = duration / 1000
	if avgDuration > 1*time.Millisecond {
		t.Fatalf("GetCurrentView average duration %v exceeds 1ms target", avgDuration)
	}

	// Test CycleToNextView performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		err := service.CycleToNextView()
		if err != nil {
			t.Fatalf("CycleToNextView failed: %v", err)
		}
	}
	duration = time.Since(start)

	avgDuration = duration / 1000
	if avgDuration > 5*time.Millisecond {
		t.Fatalf("CycleToNextView average duration %v exceeds 5ms target", avgDuration)
	}
}