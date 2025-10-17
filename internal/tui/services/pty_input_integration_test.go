package services

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPTYInputIntegration tests the complete PTY input integration with InputService
func TestPTYInputIntegration(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.EnableDebug = true

	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Verify PTY is enabled
	assert.True(t, inputService.IsPTYEnabled(), "PTY should be enabled")
	assert.Equal(t, "test", inputService.GetPTYMode(), "PTY mode should be 'test'")

	// Verify PTY layer is accessible
	retrievedPTYLayer := inputService.GetPTYLayer()
	assert.Equal(t, ptyLayer, retrievedPTYLayer, "Retrieved PTY layer should match original")

	// Test PTY statistics
	stats := inputService.GetPTYStatistics()
	assert.True(t, stats.PTYEnabled, "PTY should be enabled in statistics")
	assert.Equal(t, "test", stats.PTYMode, "PTY mode should be 'test' in statistics")
	assert.True(t, stats.PTYLayerActive, "PTY layer should be active in statistics")
	assert.Equal(t, uint64(0), stats.PTYKeyCount, "PTY key count should start at 0")
	assert.Equal(t, uint64(0), stats.TotalKeyCount, "Total key count should start at 0")
}

// TestPTYInputAdapterIntegration tests the PTYInputAdapter with InputService
func TestPTYInputAdapterIntegration(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyConfig.EnableDebug = true

	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create PTY input adapter
	adapterConfig := DefaultPTYInputAdapterConfig()
	adapterConfig.EnableDebug = true

	adapter, err := NewPTYInputAdapter(ptyLayer, inputService, adapterConfig)
	require.NoError(t, err, "Failed to create PTY input adapter")
	require.NotNil(t, adapter, "Adapter should not be nil")

	// Start adapter
	err = adapter.Start()
	require.NoError(t, err, "Failed to start PTY input adapter")
	defer adapter.Stop()

	// Verify adapter is active
	assert.True(t, adapter.IsActive(), "Adapter should be active")

	// Test key simulation through adapter
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	err = adapter.SimulateKeyPress(testKey)
	require.NoError(t, err, "Failed to simulate key press through adapter")

	// Allow some time for processing
	time.Sleep(50 * time.Millisecond)

	// Verify statistics updated
	adapterStats := adapter.GetStatistics()
	assert.Greater(t, adapterStats.KeyPressCount, uint64(0), "Key press count should be greater than 0")

	serviceStats := inputService.GetPTYStatistics()
	assert.Greater(t, serviceStats.PTYKeyCount, uint64(0), "PTY key count should be greater than 0")
	assert.Greater(t, serviceStats.TotalKeyCount, uint64(0), "Total key count should be greater than 0")
}

// TestPTYInputKeySequence tests simulating key sequences through PTY
func TestPTYInputKeySequence(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Test key sequence
	testKeys := CreateTestKeySequence()
	require.NotEmpty(t, testKeys, "Test key sequence should not be empty")

	// Simulate each key through PTY
	for i, key := range testKeys {
		err := inputService.SimulatePTYKeyPress(key)
		require.NoError(t, err, "Failed to simulate key %d (%v) through PTY", i, key)
	}

	// Verify statistics
	stats := inputService.GetPTYStatistics()
	assert.Equal(t, uint64(len(testKeys)), stats.PTYKeyCount, "PTY key count should match test keys count")
	assert.Equal(t, uint64(len(testKeys)), stats.TotalKeyCount, "Total key count should match test keys count")
}

// TestPTYInputValidation tests PTY input validation
func TestPTYInputValidation(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService
	inputService := NewInputService()
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Test validation without PTY enabled
	testKeys := CreateTestKeySequence()
	err = inputService.ValidatePTYInput(testKeys)
	assert.Error(t, err, "Should fail validation when PTY is not enabled")
	assert.Contains(t, err.Error(), "PTY mode is not enabled", "Error should mention PTY mode")

	// Enable PTY
	err = inputService.EnablePTY(ptyLayer)
	require.NoError(t, err, "Failed to enable PTY")

	// Test validation with PTY enabled
	err = inputService.ValidatePTYInput(testKeys)
	assert.NoError(t, err, "Should pass validation when PTY is enabled")

	// Verify statistics updated
	stats := inputService.GetPTYStatistics()
	assert.Equal(t, uint64(len(testKeys)), stats.PTYKeyCount, "PTY key count should match test keys count")
}

// TestPTYInputModes tests different PTY modes
func TestPTYInputModes(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService
	inputService := NewInputService()
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Test initial mode
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled initially")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "Initial mode should be 'disabled'")

	// Test enabling test mode
	err = inputService.EnablePTY(ptyLayer)
	require.NoError(t, err, "Failed to enable PTY")
	assert.True(t, inputService.IsPTYEnabled(), "PTY should be enabled")
	assert.Equal(t, "test", inputService.GetPTYMode(), "Mode should be 'test'")

	// Test setting production mode
	err = inputService.SetPTYMode("production")
	require.NoError(t, err, "Failed to set production mode")
	assert.True(t, inputService.IsPTYEnabled(), "PTY should still be enabled")
	assert.Equal(t, "production", inputService.GetPTYMode(), "Mode should be 'production'")

	// Test setting invalid mode
	err = inputService.SetPTYMode("invalid")
	assert.Error(t, err, "Should fail to set invalid mode")
	assert.Equal(t, "production", inputService.GetPTYMode(), "Mode should remain unchanged")

	// Test disabling PTY
	err = inputService.DisablePTY()
	require.NoError(t, err, "Failed to disable PTY")
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "Mode should be 'disabled'")
}

// TestPTYInputWithKeyBindings tests PTY input with key bindings
func TestPTYInputWithKeyBindings(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Register a test key binding
	testBinding := KeyBinding{
		Key:      tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}},
		Action:   "test_action",
		Context:  "test_context",
		Priority: 1,
	}

	err = inputService.RegisterKeyBinding(ctx, testBinding)
	require.NoError(t, err, "Failed to register test key binding")

	// Test PTY key press that matches binding
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	handled, _ := inputService.RouteKeyMessagePTY(testKey, "test_context")
	assert.True(t, handled, "Key should be handled by binding")

	// Test PTY key press that doesn't match binding
	testKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	handled, _ = inputService.RouteKeyMessagePTY(testKey, "test_context")
	assert.False(t, handled, "Key should not be handled (no matching binding)")

	// Verify statistics
	stats := inputService.GetPTYStatistics()
	assert.Equal(t, uint64(2), stats.PTYKeyCount, "Should have processed 2 PTY keys")
	assert.Equal(t, uint64(2), stats.TotalKeyCount, "Should have processed 2 total keys")
}

// TestPTYInputSequential tests sequential PTY input processing
// TODO: Replace with proper command pattern for concurrent testing
// For now, disable concurrent test to avoid goroutines
func TestPTYInputSequential(t *testing.T) {
	// Create PTY layer
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err, "Failed to create PTY layer")

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err, "Failed to allocate PTY")
	defer ptyLayer.Cleanup()

	// Create InputService with PTY support
	inputService := NewInputServiceWithPTY(ptyLayer)
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Create PTY input adapter
	adapterConfig := DefaultPTYInputAdapterConfig()
	adapterConfig.EnableDebug = false

	adapter, err := NewPTYInputAdapter(ptyLayer, inputService, adapterConfig)
	require.NoError(t, err, "Failed to create PTY input adapter")

	// Start adapter
	err = adapter.Start()
	require.NoError(t, err, "Failed to start PTY input adapter")
	defer adapter.Stop()

	// Test sequential key simulation
	numKeys := 100

	for i := 0; i < numKeys; i++ {
		testKey := tea.KeyMsg{
			Type: tea.KeyRunes,
			Runes: []rune{rune('a' + (i % 26))},
		}

		err := adapter.SimulateKeyPress(testKey)
		require.NoError(t, err, "Failed to simulate key %d: %v", i, testKey)
	}

	// Allow some time for processing
	time.Sleep(50 * time.Millisecond)

	// Verify statistics
	adapterStats := adapter.GetStatistics()
	assert.Equal(t, uint64(numKeys), adapterStats.KeyPressCount,
		"Should have processed all keys")

	serviceStats := inputService.GetPTYStatistics()
	assert.Equal(t, uint64(numKeys), serviceStats.PTYKeyCount,
		"Should have processed all PTY keys")
}

// TestPTYInputBackwardCompatibility tests backward compatibility
func TestPTYInputBackwardCompatibility(t *testing.T) {
	// Create InputService without PTY (traditional way)
	inputService := NewInputService()
	require.NotNil(t, inputService, "InputService should not be nil")

	// Start InputService
	ctx := context.Background()
	err := inputService.Start(ctx)
	require.NoError(t, err, "Failed to start InputService")
	defer inputService.Shutdown(ctx)

	// Test that traditional methods still work
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	handled, _ := inputService.RouteKeyMessage(testKey, "test_context")
	assert.False(t, handled, "Key should not be handled (no handlers registered)")

	// Test that PTY methods return appropriate errors when PTY is disabled
	assert.False(t, inputService.IsPTYEnabled(), "PTY should be disabled")
	assert.Equal(t, "disabled", inputService.GetPTYMode(), "Mode should be 'disabled'")
	assert.Nil(t, inputService.GetPTYLayer(), "PTY layer should be nil")

	// Test PTY key simulation when PTY is disabled
	err = inputService.SimulatePTYKeyPress(testKey)
	assert.Error(t, err, "Should fail to simulate PTY key when PTY is disabled")
	assert.Contains(t, err.Error(), "PTY mode is not enabled", "Error should mention PTY mode")

	// Verify statistics
	stats := inputService.GetPTYStatistics()
	assert.False(t, stats.PTYEnabled, "PTY should be disabled in statistics")
	assert.Equal(t, "disabled", stats.PTYMode, "Mode should be 'disabled' in statistics")
	assert.Equal(t, uint64(0), stats.PTYKeyCount, "PTY key count should be 0")
	assert.Equal(t, uint64(0), stats.TotalKeyCount, "Total key count should be 0") // Shouldn't count PTY keys when disabled
}