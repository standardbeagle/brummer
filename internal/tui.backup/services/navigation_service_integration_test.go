package services_test

import (
	"testing"

	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestNavigationService_Integration tests the navigation service with event integration
func TestNavigationService_Integration(t *testing.T) {
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test basic functionality
	currentView, err := service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	if currentView != messages.ProcessesView {
		t.Fatalf("Expected default view to be ProcessesView, got %s", currentView)
	}

	// Test view switching
	err = service.SwitchToView(messages.LogsView, "integration_test")
	if err != nil {
		t.Fatalf("SwitchToView failed: %v", err)
	}

	currentView, err = service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	if currentView != messages.LogsView {
		t.Fatalf("Expected current view to be LogsView, got %s", currentView)
	}

	// Test view cycling
	err = service.CycleToNextView()
	if err != nil {
		t.Fatalf("CycleToNextView failed: %v", err)
	}

	currentView, err = service.GetCurrentView()
	if err != nil {
		t.Fatalf("GetCurrentView failed: %v", err)
	}

	if currentView == messages.LogsView {
		t.Fatal("Expected view to change after cycling")
	}

	// Test number key mapping
	view, err := service.GetViewForNumberKey(1)
	if err != nil {
		t.Fatalf("GetViewForNumberKey failed: %v", err)
	}

	if view != messages.ProcessesView {
		t.Fatalf("Expected ProcessesView for key 1, got %s", view)
	}

	// Test view order
	order := service.GetViewOrder()
	if len(order) != 5 {
		t.Fatalf("Expected 5 views in order, got %d", len(order))
	}

	if order[0] != messages.ProcessesView {
		t.Fatalf("Expected first view in order to be ProcessesView, got %s", order[0])
	}

	// Test number key mappings
	mappings := service.GetNumberKeyMappings()
	if len(mappings) < 5 {
		t.Fatalf("Expected at least 5 number key mappings, got %d", len(mappings))
	}

	if mappings[1] != messages.ProcessesView {
		t.Fatalf("Expected mapping 1 to be ProcessesView, got %s", mappings[1])
	}
}

// TestNavigationService_CallbackIntegration tests callback functionality
func TestNavigationService_CallbackIntegration(t *testing.T) {
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	callbackCalled := false
	callbackFrom := messages.ViewType("")
	callbackTo := messages.ViewType("")

	// Register callback
	callbackID := service.RegisterTransitionCallback(func(fromView, toView messages.ViewType) error {
		callbackCalled = true
		callbackFrom = fromView
		callbackTo = toView
		return nil
	})

	// Switch view to trigger callback
	err := service.SwitchToView(messages.SettingsView, "callback_test")
	if err != nil {
		t.Fatalf("SwitchToView failed: %v", err)
	}

	if !callbackCalled {
		t.Fatal("Callback was not called")
	}

	if callbackFrom != messages.ProcessesView {
		t.Fatalf("Expected callback from view to be ProcessesView, got %s", callbackFrom)
	}

	if callbackTo != messages.SettingsView {
		t.Fatalf("Expected callback to view to be SettingsView, got %s", callbackTo)
	}

	// Clean up
	service.UnregisterTransitionCallback(callbackID)
}

// TestNavigationService_ErrorHandling tests error handling scenarios
func TestNavigationService_ErrorHandling(t *testing.T) {
	eventBus := events.NewEventBus()
	service := services.NewNavigationService(eventBus)

	// Test invalid view
	err := service.SwitchToView(messages.ViewType("invalid"), "test")
	if err == nil {
		t.Fatal("SwitchToView should have failed with invalid view")
	}

	// Test invalid number key
	_, err = service.GetViewForNumberKey(0)
	if err == nil {
		t.Fatal("GetViewForNumberKey should have failed with invalid number")
	}

	_, err = service.GetViewForNumberKey(10)
	if err == nil {
		t.Fatal("GetViewForNumberKey should have failed with invalid number")
	}

	// Test unmapped number key
	_, err = service.GetViewForNumberKey(9)
	if err == nil {
		t.Fatal("GetViewForNumberKey should have failed with unmapped number")
	}
}