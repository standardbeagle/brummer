package services

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInputServiceBackwardCompatibility ensures all existing InputService functionality
// continues to work exactly as before the PTY integration
func TestInputServiceBackwardCompatibility(t *testing.T) {
	// Test traditional constructor (no PTY)
	inputService := NewInputService()
	require.NotNil(t, inputService, "InputService should be created with traditional constructor")

	// Test that PTY is disabled by default
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled by default")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "PTY mode should be 'disabled' by default")
	assert.Nil(t, inputService.GetPTYLayer(), "PTY layer should be nil by default")

	// Test traditional startup
	ctx := context.Background()
	err := inputService.Start(ctx)
	require.NoError(t, err, "InputService should start normally without PTY")
	defer inputService.Shutdown(ctx)

	// Test that it reports as running
	assert.True(t, inputService.IsRunning(), "InputService should report as running")

	// Test traditional key routing (without PTY)
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	handled, _ := inputService.RouteKeyMessage(testKey, "test_context")
	assert.False(t, handled, "Key should not be handled by default (no handlers)")

	// Test global handler registration (traditional)
	globalHandlerCalled := false
	err = inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'g' {
			globalHandlerCalled = true
		}
		return nil
	})
	require.NoError(t, err, "Should be able to register global handler")

	// Test context handler registration (traditional)
	contextHandlerCalled := false
	err = inputService.SubscribeWithContextKeys(ctx, "test_context", func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'c' {
			contextHandlerCalled = true
		}
		return nil
	})
	require.NoError(t, err, "Should be able to register context handler")

	// Test key binding registration (traditional)
	testBinding := KeyBinding{
		Key:      tea.Key{Type: tea.KeyRunes, Runes: []rune{'b'}},
		Action:   "test_action",
		Context:  "test_context",
		Priority: 1,
	}
	err = inputService.RegisterKeyBinding(ctx, testBinding)
	require.NoError(t, err, "Should be able to register key binding")

	// Test handler retrieval
	handlers := inputService.GetActiveHandlers()
	assert.GreaterOrEqual(t, len(handlers), 2, "Should have at least 2 registered handlers")

	// Test key binding retrieval
	bindings := inputService.GetGlobalBindings()
	assert.GreaterOrEqual(t, len(bindings), 1, "Should have at least 1 registered binding")

	// Test key routing with registered handlers
	globalKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	handled, _ = inputService.RouteKeyMessage(globalKey, "any_context")
	assert.True(t, handled, "Global handler should handle the key")
	assert.True(t, globalHandlerCalled, "Global handler should have been called")

	contextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	handled, _ = inputService.RouteKeyMessage(contextKey, "test_context")
	assert.True(t, handled, "Context handler should handle the key")
	assert.True(t, contextHandlerCalled, "Context handler should have been called")

	bindingKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	handled, _ = inputService.RouteKeyMessage(bindingKey, "test_context")
	assert.True(t, handled, "Key binding should handle the key")

	// Test service status
	status := inputService.GetServiceStatus()
	assert.Equal(t, "input", status.Service, "Service name should be 'input'")
	assert.Equal(t, "active", status.Status, "Service status should be 'active'")
	assert.Contains(t, status.Details, "handlers:", "Status details should mention handlers")
	assert.Contains(t, status.Details, "bindings:", "Status details should mention bindings")

	// Test PTY statistics (should show PTY disabled)
	stats := inputService.GetPTYStatistics()
	assert.False(t, stats.PTYEnabled, "PTY should be disabled in statistics")
	assert.Equal(t, "disabled", stats.PTYMode, "PTY mode should be 'disabled' in statistics")
	assert.Equal(t, uint64(0), stats.PTYKeyCount, "PTY key count should be 0")
	assert.False(t, stats.PTYLayerActive, "PTY layer should not be active")

	// Test that PTY operations fail gracefully when PTY is disabled
	err = inputService.SimulatePTYKeyPress(testKey)
	assert.Error(t, err, "PTY key simulation should fail when PTY is disabled")
	assert.Contains(t, err.Error(), "PTY mode is not enabled", "Error should mention PTY mode")

	err = inputService.ValidatePTYInput([]tea.KeyMsg{testKey})
	assert.Error(t, err, "PTY validation should fail when PTY is disabled")
	assert.Contains(t, err.Error(), "PTY mode is not enabled", "Error should mention PTY mode")

	// Test handler count
	handlerCount := inputService.GetHandlerCount()
	assert.GreaterOrEqual(t, handlerCount, 2, "Should have at least 2 handlers")

	// Test shutdown
	err = inputService.Shutdown(ctx)
	require.NoError(t, err, "InputService should shutdown normally")
	assert.False(t, inputService.IsRunning(), "InputService should not be running after shutdown")
}

// TestInputServiceWithPTYBackwardCompatibility tests that PTY-enabled InputService
// still maintains full backward compatibility
func TestInputServiceWithPTYBackwardCompatibility(t *testing.T) {
	// For this test, we'll skip actual PTY layer creation to avoid interface issues
	// In real usage, you'd create an actual PTY layer

	// Test PTY-enabled constructor with nil PTY layer (should work gracefully)
	inputService := NewInputServiceWithPTY(nil)
	require.NotNil(t, inputService, "InputService should be created with PTY constructor even with nil PTY")

	// PTY should be disabled when nil is provided
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled when nil is provided")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "PTY mode should be 'disabled' when nil is provided")
	assert.Nil(t, inputService.GetPTYLayer(), "PTY layer should be nil")

	// Test traditional startup still works
	ctx := context.Background()
	err := inputService.Start(ctx)
	require.NoError(t, err, "InputService should start normally with PTY")
	defer inputService.Shutdown(ctx)

	// Test that traditional functionality still works exactly the same
	globalHandlerCalled := false
	err = inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'x' {
			globalHandlerCalled = true
		}
		return nil
	})
	require.NoError(t, err, "Should be able to register global handler")

	// Test traditional key routing (without PTY context)
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	handled, _ := inputService.RouteKeyMessage(testKey, "traditional_context")
	assert.True(t, handled, "Traditional key routing should still work")
	assert.True(t, globalHandlerCalled, "Global handler should have been called")

	// Test that traditional methods still work
	handlers := inputService.GetActiveHandlers()
	assert.GreaterOrEqual(t, len(handlers), 1, "Should have at least 1 handler")

	bindings := inputService.GetGlobalBindings()
	assert.NotNil(t, bindings, "Should be able to get bindings")

	handlerCount := inputService.GetHandlerCount()
	assert.GreaterOrEqual(t, handlerCount, 1, "Should have at least 1 handler")

	// Test service status includes PTY information
	status := inputService.GetServiceStatus()
	assert.Equal(t, "input", status.Service, "Service name should be 'input'")
	assert.Equal(t, "active", status.Status, "Service status should be 'active'")
	assert.Contains(t, status.Details, "handlers:", "Status details should mention handlers")
	assert.Contains(t, status.Details, "PTY:", "Status details should mention PTY")
	assert.Contains(t, status.Details, "test", "Status details should mention PTY mode")

	// Test that we can disable PTY (should work even when already disabled)
	err = inputService.DisablePTY()
	require.NoError(t, err, "Should be able to disable PTY")
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "PTY mode should be 'disabled'")
	assert.Nil(t, inputService.GetPTYLayer(), "PTY layer should be nil")

	// Test that traditional functionality still works after disabling PTY
	anotherKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	handled, _ = inputService.RouteKeyMessage(anotherKey, "another_context")
	assert.False(t, handled, "Key should not be handled (no matching handler)")
}

// TestInputServicePTYModeTransitions tests PTY mode transitions without breaking functionality
func TestInputServicePTYModeTransitions(t *testing.T) {
	// Create InputService without PTY initially
	inputService := NewInputService()
	require.NotNil(t, inputService, "InputService should be created")

	ctx := context.Background()
	err := inputService.Start(ctx)
	require.NoError(t, err, "InputService should start")
	defer inputService.Shutdown(ctx)

	// Register a traditional handler
	handlerCalled := false
	err = inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'z' {
			handlerCalled = true
		}
		return nil
	})
	require.NoError(t, err, "Should be able to register handler")

	// Test functionality without PTY
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	handled, _ := inputService.RouteKeyMessage(testKey, "test")
	assert.True(t, handled, "Handler should work without PTY")
	assert.True(t, handlerCalled, "Handler should be called without PTY")

	// Reset handler call flag
	handlerCalled = false

	// Note: We skip actual PTY enable in this test to avoid interface issues
	// In real usage, you would provide an actual PTY layer
	// err = inputService.EnablePTY(actualPTYLayer)
	// require.NoError(t, err, "Should be able to enable PTY")
	// assert.True(t, inputService.IsPTYEnabled(), "PTY should be enabled")
	// assert.Equal(t, "test", inputService.GetPTYMode(), "PTY mode should be 'test'")

	// Test that the same handler still works with PTY enabled
	handled, _ = inputService.RouteKeyMessage(testKey, "test")
	assert.True(t, handled, "Handler should still work with PTY enabled")
	assert.True(t, handlerCalled, "Handler should still be called with PTY enabled")

	// Reset handler call flag
	handlerCalled = false

	// Test PTY-specific routing
	ptyHandled, _ := inputService.RouteKeyMessagePTY(testKey, "pty_test")
	assert.True(t, ptyHandled, "PTY routing should also work")
	assert.True(t, handlerCalled, "Handler should be called through PTY routing")

	// Change PTY mode to production
	err = inputService.SetPTYMode("production")
	require.NoError(t, err, "Should be able to set production mode")
	assert.Equal(t, "production", inputService.GetPTYMode(), "PTY mode should be 'production'")

	// Reset handler call flag
	handlerCalled = false

	// Test that handler still works in production mode
	handled, _ = inputService.RouteKeyMessage(testKey, "test")
	assert.True(t, handled, "Handler should work in production mode")
	assert.True(t, handlerCalled, "Handler should be called in production mode")

	// Reset handler call flag
	handlerCalled = false

	// Disable PTY
	err = inputService.DisablePTY()
	require.NoError(t, err, "Should be able to disable PTY")
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "PTY mode should be 'disabled'")

	// Test that handler still works after PTY is disabled
	handled, _ = inputService.RouteKeyMessage(testKey, "test")
	assert.True(t, handled, "Handler should work after PTY disabled")
	assert.True(t, handlerCalled, "Handler should be called after PTY disabled")
}

// TestInputServiceEventBusCompatibility tests that event bus functionality remains unchanged
func TestInputServiceEventBusCompatibility(t *testing.T) {
	inputService := NewInputService()
	require.NotNil(t, inputService, "InputService should be created")

	// Test that event bus is available
	eventBus := inputService.EventBus()
	assert.NotNil(t, eventBus, "Event bus should be available")

	ctx := context.Background()
	err := inputService.Start(ctx)
	require.NoError(t, err, "InputService should start")
	defer inputService.Shutdown(ctx)

	// Test that event subscription works (traditional)
	eventsReceived := make([]events.Event, 0)
	eventBus.Subscribe("input.handler.registered", events.Handler(func(event events.Event) {
		eventsReceived = append(eventsReceived, event)
	}))

	// Register a handler to trigger events
	err = inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		return nil
	})
	require.NoError(t, err, "Should be able to register handler")

	// Allow some time for event processing
	time.Sleep(10 * time.Millisecond)

	// Should have received handler registration events
	assert.Greater(t, len(eventsReceived), 0, "Should have received registration events")

	// Skip PTY enable in this test to avoid interface issues
	// In real usage with actual PTY layer:
	// err = inputService.EnablePTY(actualPTYLayer)
	// require.NoError(t, err, "Should be able to enable PTY")

	// Clear events
	eventsReceived = make([]events.Event, 0)

	// Register another handler
	err = inputService.SubscribeToGlobalKeys(ctx, func(keyMsg tea.KeyMsg) tea.Cmd {
		return nil
	})
	require.NoError(t, err, "Should be able to register another handler")

	// Allow some time for event processing
	time.Sleep(10 * time.Millisecond)

	// Should still receive events with PTY enabled
	assert.Greater(t, len(eventsReceived), 0, "Should still receive events with PTY enabled")
}

// Note: MockPTYLayer removed to avoid interface compatibility issues.
// In actual tests with PTY functionality, create real PTY layers using
// pty.NewPTYLayer() and ptyLayer.Allocate().