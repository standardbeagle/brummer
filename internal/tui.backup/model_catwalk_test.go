package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/catwalk"
	"github.com/stretchr/testify/require"
)

// TestModelWithCatwalk uses the catwalk library to test TUI model interactions
func TestModelWithCatwalk(t *testing.T) {
	// Use existing test helper that should create a properly initialized model
	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("Initial Model State", func(t *testing.T) {
		// Test that the model initializes correctly
		require.NotNil(t, model)
		require.NotNil(t, model.keys)
		require.NotNil(t, model.processMgr)
		require.NotNil(t, model.logStore)
	})
}

// TestModelBasicInteractions tests basic model interactions using catwalk
func TestModelBasicInteractions(t *testing.T) {
	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("Tab Navigation", func(t *testing.T) {
		// Test Tab navigation using catwalk
		catwalk.RunModelFromString(t, "\t", model,
			catwalk.WithWindowSize(80, 24),
		)
		// If we get here without panic, the test passed
	})

	t.Run("Quit Key", func(t *testing.T) {
		// Create a fresh model for quit test
		model2 := createTestModelWithDefaults()
		require.NotNil(t, model2)

		// Test quit key using catwalk
		catwalk.RunModelFromString(t, "q", model2,
			catwalk.WithWindowSize(80, 24),
		)
		// If we get here without panic, the test passed
	})

	t.Run("Help Key", func(t *testing.T) {
		// Create a fresh model for help test
		model3 := createTestModelWithDefaults()
		require.NotNil(t, model3)

		// Test help key using catwalk
		catwalk.RunModelFromString(t, "?", model3,
			catwalk.WithWindowSize(80, 24),
		)
		// If we get here without panic, the test passed
	})
}

// TestModelMultipleKeys tests sequences of key presses
func TestModelMultipleKeys(t *testing.T) {
	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("Multiple Tab Presses", func(t *testing.T) {
		// Test multiple Tab presses to cycle through views
		catwalk.RunModelFromString(t, "\t\t\t", model,
			catwalk.WithWindowSize(80, 24),
		)
		// If we get here without panic, the test passed
	})

	t.Run("Command Keys", func(t *testing.T) {
		// Create fresh model for command test
		model2 := createTestModelWithDefaults()
		require.NotNil(t, model2)

		// Test command mode with various keys
		catwalk.RunModelFromString(t, "/fsr", model2,
			catwalk.WithWindowSize(80, 24),
		)
		// If we get here without panic, the test passed
	})
}

// TestModelRendering tests that model rendering works without panics
func TestModelRendering(t *testing.T) {
	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("View Method", func(t *testing.T) {
		// Test that View() method doesn't panic
		viewOutput := model.View()
		require.NotNil(t, viewOutput)
		require.NotEmpty(t, viewOutput)

		// Basic sanity check - should contain some TUI elements
		// but not panic messages
		require.NotContains(t, viewOutput, "panic")
		require.NotContains(t, viewOutput, "runtime error")
	})

	t.Run("Basic Updates", func(t *testing.T) {
		// Test that Update() method doesn't panic for basic keys
		// cmd can be nil, but should not panic
		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyTab})
		})

		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		})

		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		})

		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyUp})
		})

		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyDown})
		})

		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		})

		require.NotPanics(t, func() {
			model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		})
	})
}

// TestModelInvalidInput tests that model handles invalid input gracefully
func TestModelInvalidInput(t *testing.T) {
	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("Various Key Inputs", func(t *testing.T) {
		// Test various key inputs that should not panic
		testKeys := []tea.Msg{
			tea.KeyMsg{Type: tea.KeyCtrlA},
			tea.KeyMsg{Type: tea.KeyF1},
			tea.KeyMsg{Type: tea.KeyHome},
			tea.KeyMsg{Type: tea.KeyEnd},
			tea.KeyMsg{Type: tea.KeyPgUp},
			tea.KeyMsg{Type: tea.KeyPgDown},
			tea.KeyMsg{Type: tea.KeyInsert},
			tea.KeyMsg{Type: tea.KeyDelete},
			tea.KeyMsg{Type: tea.KeyBackspace},
			tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, // Random character
			tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, // Random character
			tea.MouseMsg{Type: tea.MouseLeft},                  // Mouse event
			tea.WindowSizeMsg{Width: 80, Height: 24},           // Window resize
		}

		for _, key := range testKeys {
			require.NotPanics(t, func() {
				model.Update(key)
			}, "Model should handle key: %v", key)
		}
	})

	t.Run("Unknown Messages", func(t *testing.T) {
		// Test unknown message types - should be handled gracefully
		unknownMsg := struct{ tea.Msg }{}
		require.NotPanics(t, func() {
			model.Update(unknownMsg)
		})
	})
}

// TestModelWindowResize tests that the model handles window resizing
func TestModelWindowResize(t *testing.T) {
	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("Window Resize Events", func(t *testing.T) {
		// Test various window sizes
		sizes := []struct {
			width  int
			height int
		}{
			{80, 24},  // Standard terminal
			{120, 30}, // Large terminal
			{40, 10},  // Small terminal
			{200, 50}, // Very large terminal
		}

		for _, size := range sizes {
			require.NotPanics(t, func() {
				model.Update(tea.WindowSizeMsg{
					Width:  size.width,
					Height: size.height,
				})
			}, "Model should handle resize to %dx%d", size.width, size.height)

			// Test that View() still works after resize
			viewOutput := model.View()
			require.NotNil(t, viewOutput)
			require.NotEmpty(t, viewOutput)
		}
	})
}

// TestModelPerformance tests basic performance characteristics
func TestModelPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	model := createTestModelWithDefaults()
	require.NotNil(t, model)

	t.Run("Update Performance", func(t *testing.T) {
		// Measure update performance
		start := time.Now()
		updates := 1000

		for i := 0; i < updates; i++ {
			_, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
		}

		duration := time.Since(start)
		avgUpdateTime := duration / time.Duration(updates)

		t.Logf("Performance: %d updates in %v (avg: %v per update)",
			updates, duration, avgUpdateTime)

		// Should be reasonably fast (less than 1ms per update on average)
		require.Less(t, avgUpdateTime, 1*time.Millisecond,
			"Model updates should be fast")
	})

	t.Run("View Performance", func(t *testing.T) {
		// Measure view rendering performance
		start := time.Now()
		renders := 1000

		for i := 0; i < renders; i++ {
			_ = model.View()
		}

		duration := time.Since(start)
		avgRenderTime := duration / time.Duration(renders)

		t.Logf("Performance: %d renders in %v (avg: %v per render)",
			renders, duration, avgRenderTime)

		// Rendering should be reasonably fast (less than 10ms per render on average)
		require.Less(t, avgRenderTime, 10*time.Millisecond,
			"Model rendering should be fast")
	})
}
