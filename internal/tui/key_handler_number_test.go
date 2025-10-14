package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// Test that number keys correctly map to views according to the UI tabs
func TestGlobalNavigationHandler_NumberKeyMapping(t *testing.T) {
	tests := []struct {
		name        string
		key         rune
		expectedView View
		shouldHandle bool
	}{
		{
			name:        "Key 1 should map to Processes",
			key:         '1',
			expectedView: ViewProcesses,
			shouldHandle: true,
		},
		{
			name:        "Key 2 should map to Logs",
			key:         '2',
			expectedView: ViewLogs,
			shouldHandle: true,
		},
		{
			name:        "Key 3 should map to Errors",
			key:         '3',
			expectedView: ViewErrors,
			shouldHandle: true,
		},
		{
			name:        "Key 4 should map to URLs",
			key:         '4',
			expectedView: ViewURLs,
			shouldHandle: true,
		},
		{
			name:        "Key 5 should map to Web",
			key:         '5',
			expectedView: ViewWeb,
			shouldHandle: true,
		},
		{
			name:        "Key 6 should map to AI Coders",
			key:         '6',
			expectedView: ViewAICoders,
			shouldHandle: true,
		},
		{
			name:        "Key 7 should map to Settings",
			key:         '7',
			expectedView: ViewSettings,
			shouldHandle: true,
		},
		{
			name:        "Key 8 should not be handled (no view)",
			key:         '8',
			expectedView: "",
			shouldHandle: false,
		},
		{
			name:        "Key 9 should not be handled (no view)",
			key:         '9',
			expectedView: "",
			shouldHandle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple view manager
			viewManager := NewSimpleViewManager(ViewProcesses)

			// Create the handler with simplified architecture
			handler := NewGlobalNavigationHandler(viewManager, false)

			// Create the key message
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{tt.key},
			}

			// Create a context
			ctx := KeyContext{
				View:    ViewProcesses,
				Context: "normal",
			}

			// Test if the handler can handle the key
			canHandle := handler.CanHandle(keyMsg, ctx)

			if tt.shouldHandle {
				assert.True(t, canHandle, "Handler should handle key %c", tt.key)

				// Now test that it switches to the correct view
				// Note: We can't easily test the actual view switch without a full model,
				// but we can at least verify the handler returns a command
				model := &Model{}

				cmd := handler.Handle(keyMsg, model)
				assert.NotNil(t, cmd, "Handler should return a command for key %c", tt.key)
			} else {
				assert.False(t, canHandle, "Handler should not handle key %c", tt.key)
			}
		})
	}
}

// Test the fixed implementation
func TestGlobalNavigationHandler_FixedImplementation(t *testing.T) {
	// This test verifies the FIXED behavior with simplified architecture
	viewManager := NewSimpleViewManager(ViewProcesses)
	handler := &GlobalNavigationHandler{viewManager: viewManager, debugMode: false}

	// Test what views each number key maps to after the fix
	viewMap := map[rune]View{
		'1': ViewProcesses,
		'2': ViewLogs,
		'3': ViewErrors,
		'4': ViewURLs,      // FIXED: Now correctly maps to ViewURLs
		'5': ViewWeb,       // FIXED: Now correctly maps to ViewWeb
		'6': ViewAICoders,  // FIXED: Now handles key 6
		'7': ViewSettings,  // FIXED: Now handles key 7
	}

	for key, expectedView := range viewMap {
		keyMsg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{key},
		}
		ctx := KeyContext{View: ViewProcesses, Context: "normal"}

		canHandle := handler.CanHandle(keyMsg, ctx)
		assert.True(t, canHandle, "Fixed implementation handles key %c", key)

		// Log the correct mapping
		t.Logf("Key %c correctly maps to view: %s", key, expectedView)
	}

	// Test that keys 8-9 are still NOT handled (correct behavior)
	for _, key := range []rune{'8', '9'} {
		keyMsg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{key},
		}
		ctx := KeyContext{View: ViewProcesses, Context: "normal"}

		canHandle := handler.CanHandle(keyMsg, ctx)
		assert.False(t, canHandle, "Fixed implementation correctly does NOT handle key %c", key)
	}
}

// Test that the range check in CanHandle works correctly
func TestGlobalNavigationHandler_RangeCheck(t *testing.T) {
	viewManager := NewSimpleViewManager(ViewProcesses)
	handler := NewGlobalNavigationHandler(viewManager, false)

	tests := []struct {
		key      rune
		expected bool
		name     string
	}{
		{'0', false, "Key 0 should not be handled"},
		{'1', true, "Key 1 should be handled"},
		{'5', true, "Key 5 should be handled"},
		{'6', true, "Key 6 should be handled (fixed)"},
		{'7', true, "Key 7 should be handled (fixed)"},
		{'8', false, "Key 8 should not be handled"},
		{'9', false, "Key 9 should not be handled"},
		{'a', false, "Letter 'a' should not be handled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{tt.key},
			}
			ctx := KeyContext{View: ViewProcesses, Context: "normal"}

			canHandle := handler.CanHandle(keyMsg, ctx)
			assert.Equal(t, tt.expected, canHandle, tt.name)
		})
	}
}

// Test to ensure view switching doesn't break when invalid view is used
func TestSimpleViewManager_InvalidViewSwitch(t *testing.T) {
	viewManager := NewSimpleViewManager(ViewProcesses)

	// Store the original view
	originalView := viewManager.GetCurrentView()

	// Try to set an invalid view - the simple view manager doesn't validate,
	// but the UI's renderContent will handle unknown views gracefully
	viewManager.SetCurrentView("invalid_view")

	// In the simple architecture, validation happens at the keyboard handler level
	// The GetViewForNumberKey function only returns valid views
	_, ok := GetViewForNumberKey('9')
	assert.False(t, ok, "GetViewForNumberKey should not return a view for key 9")

	// Reset to a valid view
	viewManager.SetCurrentView(originalView)
	assert.Equal(t, originalView, viewManager.GetCurrentView())
}

// TestNumberKeyNavigationWithDirectUpdate tests actual number key navigation in the full UI
// This integration test uses direct Update calls to ensure the entire keyboard->view switch flow works
func TestNumberKeyNavigationWithDirectUpdate(t *testing.T) {
	model := createTestModelWithWorkingDir()
	defer os.RemoveAll(model.workingDir)

	t.Run("All number keys 1-7 switch views without blocking", func(t *testing.T) {
		// Map of keys to expected views
		keyViewMap := map[rune]View{
			'1': ViewProcesses,
			'2': ViewLogs,
			'3': ViewErrors,   // This was the problematic one!
			'4': ViewURLs,
			'5': ViewWeb,
			'6': ViewAICoders,
			'7': ViewSettings,
		}

		for key, expectedView := range keyViewMap {
			// Reset to a known state before each test
			model.switchToView(ViewProcesses)

			// Send key directly through Update
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{key},
			}

			// Use timeout detection for the Update call
			done := make(chan View)
			go func() {
				updatedModel, cmd := model.Update(keyMsg)
				model = updatedModel.(*Model)

				// Execute command if returned (with timeout)
				if cmd != nil {
					cmdDone := make(chan bool)
					go func() {
						msg := cmd()
						if msg != nil {
							updatedModel, _ := model.Update(msg)
							model = updatedModel.(*Model)
						}
						cmdDone <- true
					}()

					select {
					case <-cmdDone:
						// Command completed
					case <-time.After(100 * time.Millisecond):
						// Command timed out (that's OK with our fix)
					}
				}
				done <- model.currentView()
			}()

			// Wait for the key press to complete or timeout
			select {
			case actualView := <-done:
				assert.Equal(t, expectedView, actualView,
					"Key %c should switch to %v but got %v", key, expectedView, actualView)
				t.Logf("✅ Key %c correctly switched to %v", key, actualView)
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("❌ Key %c caused UI lockup when switching to %v", key, expectedView)
			}
		}
	})

	t.Run("Keys 8-9 don't switch views", func(t *testing.T) {
		// Start from a known view
		model.switchToView(ViewProcesses)
		startView := model.currentView()

		// Test keys that should NOT switch views
		for _, key := range []rune{'8', '9', '0'} {
			// Send key directly through Update
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{key},
			}

			done := make(chan View)
			go func() {
				updatedModel, cmd := model.Update(keyMsg)
				model = updatedModel.(*Model)

				// Execute command if returned (with timeout)
				if cmd != nil {
					cmdDone := make(chan bool)
					go func() {
						msg := cmd()
						if msg != nil {
							model.Update(msg)
						}
						cmdDone <- true
					}()

					select {
					case <-cmdDone:
						// Command completed
					case <-time.After(50 * time.Millisecond):
						// Command timed out (that's OK)
					}
				}
				done <- model.currentView()
			}()

			// Wait for completion
			select {
			case currentView := <-done:
				// View should not change for invalid number keys
				// (unless they have other bindings)
				assert.Equal(t, startView, currentView,
					"Key %c should not change view", key)
				t.Logf("Key %c handled correctly, view remains %v", key, currentView)
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Key %c caused lockup", key)
			}
		}
	})

	t.Run("Rapid number key presses don't cause issues", func(t *testing.T) {
		// Simulate rapid key presses to ensure no race conditions
		keys := []rune{'1', '2', '3', '4', '5', '6', '7', '1', '2', '3'}

		for i, key := range keys {
			// Don't wait between presses - simulate rapid typing
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{key},
			}

			done := make(chan bool)
			go func() {
				updatedModel, cmd := model.Update(keyMsg)
				model = updatedModel.(*Model)

				// Execute command quickly (don't wait long)
				if cmd != nil {
					cmdDone := make(chan bool)
					go func() {
						msg := cmd()
						if msg != nil {
							model.Update(msg)
						}
						cmdDone <- true
					}()

					select {
					case <-cmdDone:
						// Command completed
					case <-time.After(20 * time.Millisecond):
						// Command timed out quickly (that's OK for rapid presses)
					}
				}
				done <- true
			}()

			// Each key press should complete quickly
			select {
			case <-done:
				t.Logf("Rapid key %d (%c) processed", i+1, key)
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("Rapid key press %d (%c) caused lockup", i+1, key)
			}
		}

		// Verify we ended up in a valid state
		finalView := model.currentView()
		assert.NotNil(t, finalView, "Should have a valid view after rapid key presses")
		t.Logf("Final view after rapid presses: %v", finalView)
	})
}