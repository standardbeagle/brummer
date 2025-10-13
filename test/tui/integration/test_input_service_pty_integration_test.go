package integration

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// TestInputServicePTY_Integration tests the integration between InputService and PTY layer
func TestInputServicePTY_Integration(t *testing.T) {
	// Create PTY layer for testing
	config := pty.DefaultPTYConfig()
	config.Timeout = 10 * time.Second
	ptyLayer, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	// Allocate PTY
	err = ptyLayer.Allocate()
	require.NoError(t, err)

	// Create InputService with PTY support
	inputService := services.NewInputServiceWithPTY(ptyLayer)
	defer inputService.Stop()

	// Start the service
	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err)

	// Verify PTY is enabled
	assert.True(t, inputService.IsPTYEnabled())
	assert.Equal(t, "test", inputService.GetPTYMode())

	// Test PTY key simulation
	testKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'h'}},
		{Type: tea.KeyRunes, Runes: []rune{'e'}},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyRunes, Runes: []rune{'o'}},
		{Type: tea.KeyEnter},
	}

	// Simulate key presses through InputService
	for _, key := range testKeys {
		err := inputService.SimulatePTYKeyPress(key)
		assert.NoError(t, err)
	}

	// Validate PTY input
	err = inputService.ValidatePTYInput(testKeys)
	assert.NoError(t, err)

	// Check PTY statistics
	stats := inputService.GetPTYStatistics()
	assert.True(t, stats.PTYEnabled)
	assert.Equal(t, "test", stats.PTYMode)
	assert.True(t, stats.PTYLayerActive)
	assert.Greater(t, stats.PTYKeyCount, uint64(0))
	assert.Greater(t, stats.TotalKeyCount, uint64(0))

	// Test PTY layer access
	ptyLayerFromService := inputService.GetPTYLayer()
	assert.NotNil(t, ptyLayerFromService)
	assert.Equal(t, ptyLayer, ptyLayerFromService)

	// Test PTY mode changes
	err = inputService.SetPTYMode("production")
	assert.NoError(t, err)
	assert.Equal(t, "production", inputService.GetPTYMode())

	err = inputService.SetPTYMode("disabled")
	assert.NoError(t, err)
	assert.Equal(t, "disabled", inputService.GetPTYMode())
	assert.False(t, inputService.IsPTYEnabled())

	// Re-enable for cleanup
	err = inputService.SetPTYMode("test")
	assert.NoError(t, err)
	assert.True(t, inputService.IsPTYEnabled())

	// Shutdown service
	err = inputService.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestInputServicePTY_KeyBindingIntegration tests PTY integration with key bindings
func TestInputServicePTY_KeyBindingIntegration(t *testing.T) {
	// Create PTY layer
	config := pty.DefaultPTYConfig()
	config.Timeout = 10 * time.Second
	ptyLayer, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	err = ptyLayer.Allocate()
	require.NoError(t, err)

	// Create InputService
	inputService := services.NewInputServiceWithPTY(ptyLayer)
	defer inputService.Stop()

	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err)

	// Register key bindings
	bindings := []services.KeyBinding{
		{
			Key:      tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}},
			Action:   "quit",
			Context:  "pty_test",
			Priority: 1,
		},
		{
			Key:      tea.Key{Type: tea.KeyCtrlC},
			Action:   "quit",
			Context:  "pty_test",
			Priority: 1,
		},
		{
			Key:      tea.Key{Type: tea.KeyRunes, Runes: []rune{'h'}},
			Action:   "help",
			Context:  "pty_test",
			Priority: 2,
		},
	}

	for _, binding := range bindings {
		err := inputService.RegisterKeyBinding(ctx, binding)
		assert.NoError(t, err)
	}

	// Test PTY key routing with bindings
	testKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'h'}}, // Should trigger help
		{Type: tea.KeyRunes, Runes: []rune{'q'}}, // Should trigger quit
		{Type: tea.KeyCtrlC},                    // Should trigger quit
	}

	for _, key := range testKeys {
		handled := inputService.RouteKeyMessagePTY(key, "pty_test")
		assert.True(t, handled, "Key should be handled: %v", key)
	}

	// Verify key binding statistics
	globalBindings := inputService.GetGlobalBindings()
	assert.GreaterOrEqual(t, len(globalBindings), len(bindings))
}

// TestInputServicePTY_HandlerIntegration tests PTY integration with input handlers
func TestInputServicePTY_HandlerIntegration(t *testing.T) {
	// Create PTY layer
	config := pty.DefaultPTYConfig()
	config.Timeout = 10 * time.Second
	ptyLayer, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	err = ptyLayer.Allocate()
	require.NoError(t, err)

	// Create InputService
	inputService := services.NewInputServiceWithPTY(ptyLayer)
	defer inputService.Stop()

	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err)

	// Track received key messages
	receivedKeys := make([]tea.KeyMsg, 0)
	keyHandler := func(keyMsg tea.KeyMsg) tea.Cmd {
		receivedKeys = append(receivedKeys, keyMsg)
		return nil
	}

	// Register handlers
	err = inputService.SubscribeWithContextKeys(ctx, "pty_test", keyHandler)
	assert.NoError(t, err)

	err = inputService.SubscribeToGlobalKeys(ctx, keyHandler)
	assert.NoError(t, err)

	// Test PTY key routing through handlers
	testKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
		{Type: tea.KeyRunes, Runes: []rune{'b'}},
		{Type: tea.KeyRunes, Runes: []rune{'c'}},
	}

	for _, key := range testKeys {
		handled := inputService.RouteKeyMessagePTY(key, "pty_test")
		assert.True(t, handled, "Key should be handled: %v", key)
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify keys were received by handlers
	assert.GreaterOrEqual(t, len(receivedKeys), len(testKeys))

	// Verify PTY statistics updated
	stats := inputService.GetPTYStatistics()
	assert.Greater(t, stats.PTYKeyCount, uint64(0))
	assert.Greater(t, stats.TotalKeyCount, uint64(0))
}

// TestInputServicePTY_ErrorHandling tests PTY error handling scenarios
func TestInputServicePTY_ErrorHandling(t *testing.T) {
	inputService := services.NewInputService()
	defer inputService.Stop()

	ctx := context.Background()
	err := inputService.Start(ctx)
	require.NoError(t, err)

	// Test PTY operations without PTY layer
	assert.False(t, inputService.IsPTYEnabled())
	assert.Equal(t, "disabled", inputService.GetPTYMode())

	// Should fail when PTY is not enabled
	testKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	err = inputService.SimulatePTYKeyPress(testKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PTY mode is not enabled")

	// Test enabling invalid PTY layer
	err = inputService.EnablePTY(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PTY layer cannot be nil")

	// Test invalid PTY mode
	err = inputService.SetPTYMode("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PTY mode")

	// Create unallocated PTY layer
	config := pty.DefaultPTYConfig()
	unallocatedPTY, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer unallocatedPTY.Cleanup()

	err = inputService.EnablePTY(unallocatedPTY)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PTY layer must be allocated")
}

// TestInputServicePTY_ConcurrentAccess tests concurrent PTY access
func TestInputServicePTY_ConcurrentAccess(t *testing.T) {
	// Create PTY layer
	config := pty.DefaultPTYConfig()
	config.Timeout = 15 * time.Second
	ptyLayer, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	err = ptyLayer.Allocate()
	require.NoError(t, err)

	// Create InputService
	inputService := services.NewInputServiceWithPTY(ptyLayer)
	defer inputService.Stop()

	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err)

	// Test concurrent key simulation
	concurrency := 10
	keysPerGoroutine := 5

	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < keysPerGoroutine; j++ {
				key := tea.KeyMsg{
					Type: tea.KeyRunes,
					Runes: []rune{rune('a' + id%26)},
				}

				err := inputService.SimulatePTYKeyPress(key)
				if err != nil {
					errors <- err
					return
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
		t.Errorf("Concurrent PTY access failed: %v", err)
	default:
		// No errors
	}

	// Verify statistics
	stats := inputService.GetPTYStatistics()
	expectedKeyCount := uint64(concurrency * keysPerGoroutine)
	assert.GreaterOrEqual(t, stats.PTYKeyCount, expectedKeyCount)
	assert.GreaterOrEqual(t, stats.TotalKeyCount, expectedKeyCount)
}

// TestInputServicePTY_Performance tests PTY performance characteristics
func TestInputServicePTY_Performance(t *testing.T) {
	// Create PTY layer
	config := pty.DefaultPTYConfig()
	config.Timeout = 10 * time.Second
	ptyLayer, err := pty.NewPTYLayer(config)
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	err = ptyLayer.Allocate()
	require.NoError(t, err)

	// Create InputService
	inputService := services.NewInputServiceWithPTY(ptyLayer)
	defer inputService.Stop()

	ctx := context.Background()
	err = inputService.Start(ctx)
	require.NoError(t, err)

	// Performance test: simulate rapid key presses
	keyCount := 1000
	start := time.Now()

	for i := 0; i < keyCount; i++ {
		key := tea.KeyMsg{
			Type: tea.KeyRunes,
			Runes: []rune{rune('a' + i%26)},
		}

		err := inputService.SimulatePTYKeyPress(key)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	keysPerSecond := float64(keyCount) / duration.Seconds()

	// Performance should be reasonable (at least 100 keys/second)
	assert.Greater(t, keysPerSecond, 100.0, "PTY key simulation performance too slow: %.2f keys/sec", keysPerSecond)

	t.Logf("PTY Performance: %d keys in %v (%.2f keys/sec)", keyCount, duration, keysPerSecond)

	// Verify statistics
	stats := inputService.GetPTYStatistics()
	assert.Equal(t, uint64(keyCount), stats.PTYKeyCount)
	assert.Equal(t, uint64(keyCount), stats.TotalKeyCount)
}