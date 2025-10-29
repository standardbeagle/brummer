package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestKeyHandlerRegistry tests the basic functionality of the key handler registry
func TestKeyHandlerRegistry(t *testing.T) {
	// Create registry
	registry := NewKeyHandlerRegistry()
	assert.NotNil(t, registry)

	// Create test handlers with different priorities
	var handler1Called, handler2Called, handler3Called bool

	handler1 := &testKeyHandler{
		name:     "Handler1",
		priority: 100,
		canHandle: func(key tea.KeyMsg, ctx KeyContext) bool {
			return key.Type == tea.KeyTab
		},
		handle: func(key tea.KeyMsg, model *Model) tea.Cmd {
			handler1Called = true
			return func() tea.Msg { return nil } // Return non-nil command
		},
	}

	handler2 := &testKeyHandler{
		name:     "Handler2",
		priority: 200,
		canHandle: func(key tea.KeyMsg, ctx KeyContext) bool {
			return key.Type == tea.KeyTab
		},
		handle: func(key tea.KeyMsg, model *Model) tea.Cmd {
			handler2Called = true
			return func() tea.Msg { return nil } // Return non-nil command
		},
	}

	handler3 := &testKeyHandler{
		name:     "Handler3",
		priority: 50, // Highest priority
		canHandle: func(key tea.KeyMsg, ctx KeyContext) bool {
			return key.Type == tea.KeyEnter
		},
		handle: func(key tea.KeyMsg, model *Model) tea.Cmd {
			handler3Called = true
			return func() tea.Msg { return nil } // Return non-nil command
		},
	}

	// Register handlers (out of order to test sorting)
	registry.Register(handler2)
	registry.Register(handler1)
	registry.Register(handler3)

	// Verify sorting - handlers should be sorted by priority
	assert.Equal(t, 3, len(registry.handlers))
	assert.Equal(t, "Handler3", registry.handlers[0].Name())
	assert.Equal(t, "Handler1", registry.handlers[1].Name())
	assert.Equal(t, "Handler2", registry.handlers[2].Name())

	// Test Tab key - should be handled by Handler1 (priority 100)
	ctx := KeyContext{View: ViewProcesses}
	model := &Model{}
	tabKey := tea.KeyMsg{Type: tea.KeyTab}

	registry.HandleKey(tabKey, ctx, model)
	assert.True(t, handler1Called, "Handler1 should have been called for Tab key")
	assert.False(t, handler2Called, "Handler2 should not have been called (lower priority)")

	// Test Enter key - should be handled by Handler3
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	registry.HandleKey(enterKey, ctx, model)
	assert.True(t, handler3Called, "Handler3 should have been called for Enter key")
}

// TestContextAwareSlashCommand tests the context-aware slash command handler
func TestContextAwareSlashCommand(t *testing.T) {
	var handlerCalled bool
	handler := NewSlashCommandHandler(func() tea.Cmd {
		handlerCalled = true
		return func() tea.Msg {
			return OpenCommandPaletteMsg{}
		}
	})

	slashKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}

	// Test 1: In normal context, slash should be handled
	ctx := KeyContext{View: ViewProcesses, Context: "normal"}
	assert.True(t, handler.CanHandle(slashKey, ctx))

	// Test 2: In AI Coder PTY mode at start of line, should NOT be handled (AI Coder has absolute priority)
	ctx = KeyContext{View: ViewAICoders, Context: "pty", CursorPosition: 0}
	assert.False(t, handler.CanHandle(slashKey, ctx))

	// Test 3: In AI Coder PTY mode mid-line, should NOT be handled (AI Coder has absolute priority)
	ctx = KeyContext{View: ViewAICoders, Context: "pty", CursorPosition: 5}
	assert.False(t, handler.CanHandle(slashKey, ctx))

	// Test 4: In AI Coder list mode, should NOT be handled (AI Coder has absolute priority)
	ctx = KeyContext{View: ViewAICoders, Context: "normal"}
	assert.False(t, handler.CanHandle(slashKey, ctx))

	// Test handling with callback
	model := &Model{}
	ctx = KeyContext{View: ViewProcesses, Context: "normal"}
	cmd := handler.Handle(slashKey, model)
	assert.NotNil(t, cmd, "Command should be returned")
	assert.True(t, handlerCalled, "Handler callback should have been called")

	// Execute the command
	msg := cmd()
	assert.IsType(t, OpenCommandPaletteMsg{}, msg)

	// Test handling without callback (fallback)
	handlerWithoutCallback := NewSlashCommandHandler(nil)
	cmd = handlerWithoutCallback.Handle(slashKey, model)
	assert.NotNil(t, cmd, "Command should still be returned even without callback")
	msg = cmd()
	assert.IsType(t, OpenCommandPaletteMsg{}, msg, "Should return OpenCommandPaletteMsg as fallback")
}

// TestGlobalNavigationHandler tests view switching with number keys
func TestGlobalNavigationHandler(t *testing.T) {
	viewManager := NewSimpleViewManager(ViewProcesses)
	handler := NewGlobalNavigationHandler(viewManager, false)

	// Test Tab key handling - GlobalNavigationHandler does NOT handle Tab (reserved for input navigation)
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	ctx := KeyContext{View: ViewProcesses}
	assert.False(t, handler.CanHandle(tabKey, ctx), "GlobalNavigationHandler should not handle Tab key")

	// Test number keys 1-7 (fixed to include all views)
	for i := '1'; i <= '7'; i++ {
		numberKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{i}}
		assert.True(t, handler.CanHandle(numberKey, ctx), "Key %c should be handled", i)
	}

	// Test invalid numbers (should not be handled)
	invalidKey8 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}}
	assert.False(t, handler.CanHandle(invalidKey8, ctx), "Key 8 should not be handled")

	invalidKey9 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}}
	assert.False(t, handler.CanHandle(invalidKey9, ctx), "Key 9 should not be handled")

	// Test other keys (should not be handled)
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	assert.False(t, handler.CanHandle(enterKey, ctx))

	// Test that global navigation CAN handle keys in AI Coder view as fallback
	// AI Coder handler has absolute priority (1) vs GlobalNavigation (900)
	// So AI Coder handler gets keys first, but GlobalNavigation is available as fallback
	aiCoderCtx := KeyContext{View: ViewAICoders}
	numberKey1 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	leftKey := tea.KeyMsg{Type: tea.KeyLeft}
	rightKey := tea.KeyMsg{Type: tea.KeyRight}

	assert.True(t, handler.CanHandle(numberKey1, aiCoderCtx), "Number keys should be handled in AI Coder view as fallback")
	assert.True(t, handler.CanHandle(leftKey, aiCoderCtx), "Left arrow should be handled in AI Coder view as fallback")
	assert.True(t, handler.CanHandle(rightKey, aiCoderCtx), "Right arrow should be handled in AI Coder view as fallback")
}

// TestQuitHandler tests the double-press quit protection
func TestQuitHandler(t *testing.T) {
	handler := NewQuitHandler()

	// Test Ctrl+C handling
	ctrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	ctx := KeyContext{}
	assert.True(t, handler.CanHandle(ctrlC, ctx))

	// Test Escape handling
	esc := tea.KeyMsg{Type: tea.KeyEsc}
	assert.True(t, handler.CanHandle(esc, ctx))

	// Test other keys (should not be handled)
	tab := tea.KeyMsg{Type: tea.KeyTab}
	assert.False(t, handler.CanHandle(tab, ctx))

	// Test that quit handler handles Ctrl+C in AI Coder view (for double-press quit)
	// but does NOT handle Esc (reserved for AI tools/PTY)
	aiCoderCtx := KeyContext{View: ViewAICoders}
	assert.True(t, handler.CanHandle(ctrlC, aiCoderCtx), "Ctrl+C should be handled in AI Coder view for double-press quit")
	assert.False(t, handler.CanHandle(esc, aiCoderCtx), "Esc should not be handled in AI Coder view (reserved for AI tools)")

	// Test double-press protection
	model := &Model{}

	// First press should return status message
	cmd := handler.Handle(ctrlC, model)
	assert.NotNil(t, cmd)
	msg := cmd()
	statusMsg, ok := msg.(StatusMsg)
	assert.True(t, ok)
	assert.Equal(t, "Press ESC or Ctrl+C again to quit", statusMsg.Message)

	// Second press within 500ms should quit
	cmd = handler.Handle(ctrlC, model)
	assert.NotNil(t, cmd)
	// Note: We can't directly test tea.Quit as it's a function,
	// but the command should be the quit command
}

// Helper test handler for testing
type testKeyHandler struct {
	name      string
	priority  int
	canHandle func(tea.KeyMsg, KeyContext) bool
	handle    func(tea.KeyMsg, *Model) tea.Cmd
}

func (h *testKeyHandler) Name() string     { return h.name }
func (h *testKeyHandler) Priority() int    { return h.priority }
func (h *testKeyHandler) CanHandle(key tea.KeyMsg, ctx KeyContext) bool {
	return h.canHandle(key, ctx)
}
func (h *testKeyHandler) Handle(key tea.KeyMsg, model *Model) tea.Cmd {
	return h.handle(key, model)
}

// TestGlobalNavigationWithDirectUpdate tests keyboard navigation using direct Update calls
// This ensures the full UI interaction flow works correctly
func TestGlobalNavigationWithDirectUpdate(t *testing.T) {
	model := createTestModelWithWorkingDir()
	defer os.RemoveAll(model.workingDir)

	t.Run("Number keys switch views correctly", func(t *testing.T) {
		// Test switching to each view with number keys
		testCases := []struct {
			key          rune
			expectedView View
		}{
			{'1', ViewProcesses},
			{'2', ViewLogs},
			{'3', ViewErrors}, // This was the one that was locking up!
			{'4', ViewURLs},
			{'5', ViewWeb},
			{'6', ViewAICoders},
			{'7', ViewSettings},
		}

		for _, tc := range testCases {
			// Reset to a known state
			model.switchToView(ViewProcesses)

			// Send the key directly through Update
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{tc.key},
			}

			// Run the update with timeout detection
			done := make(chan bool)
			go func() {
				updatedModel, cmd := model.Update(keyMsg)
				model = updatedModel.(*Model)

				// Execute any returned command
				if cmd != nil {
					// Try to execute with timeout
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
					case <-time.After(100 * time.Millisecond):
						// Command timed out (but that's OK with our fix)
					}
				}
				done <- true
			}()

			// Wait with timeout to detect lockups
			select {
			case <-done:
				assert.Equal(t, tc.expectedView, model.currentView(),
					"Key %c should switch to %v", tc.key, tc.expectedView)
			case <-time.After(500 * time.Millisecond):
				t.Fatalf("Key %c caused lockup when switching to %v", tc.key, tc.expectedView)
			}
		}
	})

	t.Run("Tab key cycles through views", func(t *testing.T) {
		// Start from a known state
		model.switchToView(ViewProcesses)
		startView := model.currentView()

		// Press Tab multiple times and verify cycling
		// There are 7 views total: Processes, Logs, Errors, URLs, Web, AICoders, Settings
		for i := 0; i < 7; i++ { // Go through all views and wrap around
			previousView := model.currentView()

			// Send Tab key directly
			tabMsg := tea.KeyMsg{Type: tea.KeyTab}
			updatedModel, cmd := model.Update(tabMsg)
			model = updatedModel.(*Model)

			// Execute command if returned
			if cmd != nil {
				msg := cmd()
				if msg != nil {
					updatedModel, _ := model.Update(msg)
					model = updatedModel.(*Model)
				}
			}

			currentView := model.currentView()

			// Verify we moved to a different view
			if i < 6 { // Before wrapping (0-5 = 6 transitions)
				assert.NotEqual(t, previousView, currentView,
					"Tab should move to next view")
			} else { // Should wrap back to start (i=6, 7th tab press)
				assert.Equal(t, startView, currentView,
					"Tab should wrap back to start view")
			}
		}
	})

	t.Run("Invalid keys don't crash", func(t *testing.T) {
		// Test that invalid number keys don't cause issues
		invalidKeys := []rune{'8', '9', '0', 'a', 'b', 'z'}

		for _, key := range invalidKeys {
			previousView := model.currentView()

			// Send key directly through Update
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{key},
			}

			// This should not panic or lock up
			done := make(chan bool)
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
				done <- true
			}()

			select {
			case <-done:
				// Good, it completed without locking up
				// View might change if the key has another binding
				t.Logf("Key %c handled without lockup (view: %v -> %v)", key, previousView, model.currentView())
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Key %c caused lockup", key)
			}
		}
	})
}