package harness

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
	"github.com/stretchr/testify/require"
)

// TestInputSimulator_BasicFunctionality tests basic InputSimulator functionality
func TestInputSimulator_BasicFunctionality(t *testing.T) {
	t.Run("NewInputSimulator", func(t *testing.T) {
		// Create a simple PTY layer for testing
		ptyConfig := pty.DefaultPTYConfig()
		ptyConfig.Timeout = 5 * time.Second
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err, "Failed to create PTY layer")

		// Allocate PTY
		err = ptyLayer.Allocate()
		require.NoError(t, err, "Failed to allocate PTY")
		defer ptyLayer.Cleanup()

		// Create InputSimulator
		config := InputSimulatorConfig{
			KeyInputDelay: 10 * time.Millisecond,
			PTYLayer:      ptyLayer,
		}

		simulator, err := NewInputSimulator(config)
		require.NoError(t, err, "Failed to create InputSimulator")
		require.NotNil(t, simulator, "InputSimulator should not be nil")
	})

	t.Run("GenerateKeySequence", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0, // No delay for testing
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Test basic action generation
		sequence, err := simulator.GenerateKeySequence("quit")
		require.NoError(t, err, "GenerateKeySequence should not fail for 'quit'")
		require.NotEmpty(t, sequence, "Generated sequence should not be empty")
		require.Len(t, sequence, 1, "Quit should generate 1 key")

		// Test navigation action
		sequence, err = simulator.GenerateKeySequence("up")
		require.NoError(t, err, "GenerateKeySequence should not fail for 'up'")
		require.Len(t, sequence, 1, "Navigation should generate 1 key")
		require.Equal(t, tea.KeyUp, sequence[0].Type, "Should generate up key")

		// Test complex action
		sequence, err = simulator.GenerateKeySequence("type:hello")
		require.NoError(t, err, "GenerateKeySequence should handle type actions")
		require.Len(t, sequence, 5, "Should generate 5 characters for 'hello'")

		// Test error case
		_, err = simulator.GenerateKeySequence("")
		require.Error(t, err, "GenerateKeySequence should fail for empty action")
	})

	t.Run("SimulateKeyPress", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0,
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Test simulating a key press
		testKey := tea.KeyMsg{
			Type: tea.KeyRunes,
			Runes: []rune{'a'},
		}

		err = simulator.SimulateKeyPress(testKey)
		require.NoError(t, err, "SimulateKeyPress should not fail")

		// Verify key was added to sequence
		currentSequence := simulator.GetCurrentSequence()
		require.Len(t, currentSequence, 1, "Current sequence should contain one key")
		require.Equal(t, testKey.Type, currentSequence[0].Type, "Key type should match")
		require.Equal(t, testKey.Runes, currentSequence[0].Runes, "Key runes should match")
	})

	t.Run("SimulateKeySequence", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0,
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Clear sequence first
		err = simulator.ClearSequence()
		require.NoError(t, err, "ClearSequence should not fail")

		// Test simulating a sequence
		testSequence := []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune{'h'}},
			{Type: tea.KeyRunes, Runes: []rune{'e'}},
			{Type: tea.KeyRunes, Runes: []rune{'l'}},
			{Type: tea.KeyRunes, Runes: []rune{'l'}},
			{Type: tea.KeyRunes, Runes: []rune{'o'}},
			{Type: tea.KeyEnter},
		}

		err = simulator.SimulateKeySequence(testSequence)
		require.NoError(t, err, "SimulateKeySequence should not fail")

		// Verify keys were added
		currentSequence := simulator.GetCurrentSequence()
		require.Len(t, currentSequence, 6, "Sequence should contain 6 keys")
	})

	t.Run("SetKeyMappings", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0,
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Test setting key mappings
		mappings := []KeyMapping{
			{
				Action: "custom_action",
				Keys: []tea.KeyMsg{
					{Type: tea.KeyRunes, Runes: []rune{'x'}},
				},
			},
		}

		err = simulator.SetKeyMappings(mappings)
		require.NoError(t, err, "SetKeyMappings should not fail")

		// Test that mappings affect key sequence generation
		sequence, err := simulator.GenerateKeySequence("custom_action")
		require.NoError(t, err, "GenerateKeySequence should work with custom mapping")
		require.NotEmpty(t, sequence, "Should generate sequence for custom action")
	})

	t.Run("SetTimingDelay", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0,
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Test setting timing delay
		testDelay := 50 * time.Millisecond
		err = simulator.SetTimingDelay(testDelay)
		require.NoError(t, err, "SetTimingDelay should not fail")

		// Test negative delay (should be rejected)
		err = simulator.SetTimingDelay(-time.Millisecond)
		require.Error(t, err, "SetTimingDelay should reject negative delays")
	})

	t.Run("GetCurrentSequence", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0,
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Test initial state
		sequence := simulator.GetCurrentSequence()
		require.NotNil(t, sequence, "GetCurrentSequence should never return nil")
		require.Empty(t, sequence, "Initial sequence should be empty")

		// Add some keys and verify sequence
		testKeys := []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune{'a'}},
			{Type: tea.KeyRunes, Runes: []rune{'b'}},
			{Type: tea.KeyEnter},
		}

		for _, key := range testKeys {
			err := simulator.SimulateKeyPress(key)
			require.NoError(t, err, "SimulateKeyPress should not fail")
		}

		sequence = simulator.GetCurrentSequence()
		require.Len(t, sequence, len(testKeys), "Sequence should contain all added keys")
	})

	t.Run("ClearSequence", func(t *testing.T) {
		// Setup PTY and simulator
		ptyConfig := pty.DefaultPTYConfig()
		ptyLayer, err := pty.NewPTYLayer(ptyConfig)
		require.NoError(t, err)
		err = ptyLayer.Allocate()
		require.NoError(t, err)
		defer ptyLayer.Cleanup()

		simulator, err := NewInputSimulator(InputSimulatorConfig{
			KeyInputDelay: 0,
			PTYLayer:      ptyLayer,
		})
		require.NoError(t, err)

		// Add a key
		err = simulator.SimulateKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		require.NoError(t, err, "SimulateKeyPress should not fail")

		// Verify sequence has content
		sequence := simulator.GetCurrentSequence()
		require.NotEmpty(t, sequence, "Sequence should have content before clearing")

		// Clear sequence
		err = simulator.ClearSequence()
		require.NoError(t, err, "ClearSequence should not fail")

		// Verify sequence is empty
		sequence = simulator.GetCurrentSequence()
		require.Empty(t, sequence, "Sequence should be empty after clearing")
		require.NotNil(t, sequence, "Sequence should still be non-nil after clearing")
	})
}

// TestInputSimulator_BrummerSpecific tests Brummer-specific functionality
func TestInputSimulator_BrummerSpecific(t *testing.T) {
	// Setup PTY and simulator
	ptyConfig := pty.DefaultPTYConfig()
	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	require.NoError(t, err)
	err = ptyLayer.Allocate()
	require.NoError(t, err)
	defer ptyLayer.Cleanup()

	simulator, err := NewInputSimulator(InputSimulatorConfig{
		KeyInputDelay: 0,
		PTYLayer:      ptyLayer,
	})
	require.NoError(t, err)

	t.Run("BrummerCommands", func(t *testing.T) {
		// Test Brummer-specific commands
		commands := []string{"dev", "test", "build", "quit", "help"}

		for _, cmd := range commands {
			sequence, err := simulator.GenerateKeySequence(cmd)
			require.NoError(t, err, "GenerateKeySequence should work for %s", cmd)
			require.NotEmpty(t, sequence, "Should generate sequence for %s", cmd)
		}
	})

	t.Run("CommandPalette", func(t *testing.T) {
		// Test command palette action
		sequence, err := simulator.GenerateKeySequence("command_palette")
		require.NoError(t, err, "GenerateKeySequence should work for command_palette")
		require.NotEmpty(t, sequence, "Should generate sequence for command_palette")
	})
}