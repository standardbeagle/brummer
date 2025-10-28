package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/internal/tui/state"
	"github.com/stretchr/testify/assert"
)

// createTestModel creates a properly initialized Model for testing
func createTestModel(view View, withAICoderSession bool) *Model {
	model := &Model{
		masterData: state.NewMasterData(),
	}
	model.viewManager = NewSimpleViewManager(view)

	if withAICoderSession {
		model.masterData.AICoderState = state.NewAICoderState()
		model.masterData.AICoderState = model.masterData.AICoderState.
			AddSession("test-session", "Test Session").
			SetFocusedSession("test-session")
	} else {
		model.masterData.AICoderState = state.NewAICoderState()
	}

	return model
}

// TestAICoderKeyHandling tests all key handling scenarios for AI Coder
func TestAICoderKeyHandling(t *testing.T) {
	t.Run("No AI Coder Session", func(t *testing.T) {
		// Setup
		model := createTestModel(ViewAICoders, false)

		// Create key registry and handlers
		registry := NewKeyHandlerRegistry()
		registry.Register(NewAICoderKeyHandler(nil))
		registry.Register(NewGlobalNavigationHandler(model.viewManager, false))

		// Test: Navigation keys should work when no session is active
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "normal",
		}

		// Number keys should switch views
		key1 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		cmd := registry.HandleKey(key1, ctx, model)
		assert.NotNil(t, cmd, "Number key '1' should switch to Processes view")

		// Arrow keys should navigate
		leftKey := tea.KeyMsg{Type: tea.KeyLeft}
		cmd = registry.HandleKey(leftKey, ctx, model)
		assert.NotNil(t, cmd, "Left arrow should navigate to previous view")
	})

	t.Run("AI Coder Session Focused but Terminal Not Focused", func(t *testing.T) {
		// Setup
		model := createTestModel(ViewAICoders, true)
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false) // Terminal NOT focused

		// Create key registry and handlers
		registry := NewKeyHandlerRegistry()
		aiHandler := NewAICoderKeyHandler(nil)
		navHandler := NewGlobalNavigationHandler(model.viewManager, false)
		registry.Register(aiHandler)
		registry.Register(navHandler)

		// Context should be "normal" when terminal is not focused
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "normal",
		}

		// Test: Navigation keys should work
		key6 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}}  // AI Coders is view 6
		cmd := registry.HandleKey(key6, ctx, model)
		assert.NotNil(t, cmd, "Number key '6' should work (stays on AI Coders view)")

		// Verify we're still in AI Coders view
		assert.Equal(t, ViewAICoders, model.viewManager.GetCurrentView(), "Should still be in AI Coders view")

		leftKey := tea.KeyMsg{Type: tea.KeyLeft}
		cmd = registry.HandleKey(leftKey, ctx, model)
		assert.NotNil(t, cmd, "Left arrow should navigate when terminal not focused")

		// Switch back to AI Coders view after left arrow
		model.viewManager.SwitchToView(ViewAICoders)

		// Test: Enter key should focus terminal
		enterKey := tea.KeyMsg{Type: tea.KeyEnter}
		cmd = registry.HandleKey(enterKey, ctx, model)
		assert.NotNil(t, cmd, "Enter key should refocus terminal")
		assert.True(t, model.masterData.AICoderState.TerminalFocused, "Terminal should be focused after Enter")

		// Reset terminal focus
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false)

		// Test: 'q' key should exit session
		qKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		cmd = registry.HandleKey(qKey, ctx, model)
		assert.NotNil(t, cmd, "'q' key should exit session when terminal not focused")
		assert.Equal(t, "", model.masterData.AICoderState.FocusedSession, "Session should be unfocused after 'q'")

		// Reset session
		model.masterData.AICoderState = model.masterData.AICoderState.SetFocusedSession("test-session")

		// Test: Ctrl+T should toggle focus
		ctrlTKey := tea.KeyMsg{Type: tea.KeyCtrlT}
		cmd = registry.HandleKey(ctrlTKey, ctx, model)
		assert.NotNil(t, cmd, "Ctrl+T should toggle terminal focus")
		assert.True(t, model.masterData.AICoderState.TerminalFocused, "Terminal should be focused after Ctrl+T")
	})

	t.Run("AI Coder Terminal Focused", func(t *testing.T) {
		// Setup
		model := createTestModel(ViewAICoders, true)
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(true) // Terminal IS focused
		model.aiCoderService = &services.AICoderService{} // Mock service

		// Create key registry and handlers
		registry := NewKeyHandlerRegistry()
		aiHandler := NewAICoderKeyHandler(nil)
		navHandler := NewGlobalNavigationHandler(model.viewManager, false)
		registry.Register(aiHandler)
		registry.Register(navHandler)

		// Context should be "terminal_focused" when terminal is focused
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "terminal_focused",
		}

		// Test: Navigation keys should be captured by AI Coder handler
		key1 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
		handled := aiHandler.CanHandle(key1, ctx)
		assert.True(t, handled, "AI Coder should handle '1' key when terminal focused")

		key2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
		handled = aiHandler.CanHandle(key2, ctx)
		assert.True(t, handled, "AI Coder should handle '2' key when terminal focused")

		leftKey := tea.KeyMsg{Type: tea.KeyLeft}
		handled = aiHandler.CanHandle(leftKey, ctx)
		assert.True(t, handled, "AI Coder should handle left arrow when terminal focused")

		// Test: Regular characters should be handled
		aKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		handled = aiHandler.CanHandle(aKey, ctx)
		assert.True(t, handled, "AI Coder should handle 'a' key when terminal focused")

		// Test: Ctrl+T should release focus
		ctrlTKey := tea.KeyMsg{Type: tea.KeyCtrlT}
		cmd := registry.HandleKey(ctrlTKey, ctx, model)
		assert.NotNil(t, cmd, "Ctrl+T should release terminal focus")
		assert.False(t, model.masterData.AICoderState.TerminalFocused, "Terminal should not be focused after Ctrl+T")

		// Reset and test ESC key
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(true)
		escKey := tea.KeyMsg{Type: tea.KeyEsc}
		cmd = registry.HandleKey(escKey, ctx, model)
		assert.NotNil(t, cmd, "ESC should release terminal focus")
		assert.False(t, model.masterData.AICoderState.TerminalFocused, "Terminal should not be focused after ESC")
	})

	t.Run("View Switching", func(t *testing.T) {
		// Setup
		model := createTestModel(ViewProcesses, false)

		// Create key registry and handlers
		registry := NewKeyHandlerRegistry()
		registry.Register(NewAICoderKeyHandler(nil))
		registry.Register(NewGlobalNavigationHandler(model.viewManager, false))

		// Test: Keys should work in other views even with AI Coder session
		// Add a session to test that it doesn't interfere
		model.masterData.AICoderState = model.masterData.AICoderState.
			AddSession("test-session", "Test Session").
			SetFocusedSession("test-session").
			SetTerminalFocus(true) // Terminal focused

		// Switch to Processes view
		model.viewManager.SwitchToView(ViewProcesses)
		ctx := KeyContext{
			View:    ViewProcesses,
			Context: "normal",
		}

		// Navigation should work in other views
		key2 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
		cmd := registry.HandleKey(key2, ctx, model)
		assert.NotNil(t, cmd, "Number key '2' should work in Processes view")

		// AI Coder handler should not interfere
		aiHandler := NewAICoderKeyHandler(nil)
		handled := aiHandler.CanHandle(key2, ctx)
		assert.False(t, handled, "AI Coder should not handle keys in other views")
	})
}

// TestAICoderDirectHandler tests the AI Coder handler directly without registry
func TestAICoderDirectHandler(t *testing.T) {
	// Setup model with AI Coder session
	model := createTestModel(ViewAICoders, true)
	model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false)

	// Create handler
	handler := NewAICoderKeyHandler(nil).(*AICoderKeyHandler)

	// Test Enter key handling
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	ctx := KeyContext{View: ViewAICoders, Context: "normal"}

	// Check if handler says it can handle Enter key
	canHandle := handler.CanHandle(enterKey, ctx)
	assert.True(t, canHandle, "Handler should say it can handle Enter key")

	// Handle the Enter key
	cmd := handler.Handle(enterKey, model)
	assert.NotNil(t, cmd, "Handler should return command for Enter key")
	assert.True(t, model.masterData.AICoderState.TerminalFocused, "Terminal should be focused after Enter")

	// Reset and test 'q' key
	model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false)
	qKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	canHandle = handler.CanHandle(qKey, ctx)
	assert.True(t, canHandle, "Handler should say it can handle 'q' key")

	cmd = handler.Handle(qKey, model)
	assert.NotNil(t, cmd, "Handler should return command for 'q' key")
	assert.Equal(t, "", model.masterData.AICoderState.FocusedSession, "Session should be unfocused after 'q'")
}

// TestAICoderRegistryDebug tests the registry routing with AI Coder handler
func TestAICoderRegistryDebug(t *testing.T) {
	// Setup model with AI Coder session
	model := createTestModel(ViewAICoders, true)
	model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false)

	// Create registry
	registry := NewKeyHandlerRegistry()

	// Register both handlers like in the failing test
	aiHandler := NewAICoderKeyHandler(nil)
	navHandler := NewGlobalNavigationHandler(model.viewManager, false)
	registry.Register(aiHandler)
	registry.Register(navHandler)

	// Test Enter key
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	ctx := KeyContext{View: ViewAICoders, Context: "normal"}

	// Check registry has handlers in correct order
	assert.Equal(t, 2, len(registry.handlers), "Registry should have 2 handlers")
	assert.Equal(t, "AICoderKeys", registry.handlers[0].Name(), "First handler should be AICoderKeys")
	assert.Equal(t, "GlobalNavigation", registry.handlers[1].Name(), "Second handler should be GlobalNavigation")

	// Try to handle key
	cmd := registry.HandleKey(enterKey, ctx, model)
	assert.NotNil(t, cmd, "Registry should return command for Enter key")
	assert.True(t, model.masterData.AICoderState.TerminalFocused, "Terminal should be focused after Enter")
}

// TestAICoderPriorityHandling tests that AI Coder handler has correct priority
func TestAICoderPriorityHandling(t *testing.T) {
	registry := NewKeyHandlerRegistry()

	// Register handlers
	aiHandler := NewAICoderKeyHandler(nil)
	navHandler := NewGlobalNavigationHandler(NewSimpleViewManager(ViewProcesses), false)
	quitHandler := NewQuitHandler()

	registry.Register(aiHandler)
	registry.Register(navHandler)
	registry.Register(quitHandler)

	// Verify priority order
	assert.Equal(t, 1, aiHandler.Priority(), "AI Coder should have highest priority")
	assert.Equal(t, 50, quitHandler.Priority(), "Quit handler should have medium priority")
	assert.Equal(t, 900, navHandler.Priority(), "Navigation should have lowest priority")

	// Verify handlers are sorted correctly
	assert.Equal(t, "AICoderKeys", registry.handlers[0].Name(), "AI Coder should be first")
	assert.Equal(t, "Quit", registry.handlers[1].Name(), "Quit should be second")
	assert.Equal(t, "GlobalNavigation", registry.handlers[2].Name(), "Navigation should be last")
}

func TestCtrlTUniversalExit(t *testing.T) {
	// Create AI Coder key handler
	handler := NewAICoderKeyHandler(nil)

	// Test case 1: Ctrl+T with no focused session should exit AI Coder view
	t.Run("Ctrl+T with no focused session exits AI Coder view", func(t *testing.T) {
		// Setup model with AI Coder view but no focused session
		model := createTestModel(ViewAICoders, false)
		model.masterData.AICoderState = model.masterData.AICoderState.SetFocusedSession("")
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false)

		// Create Ctrl+T key message
		ctrlTKey := tea.KeyMsg{Type: tea.KeyCtrlT}

		// Create key context
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "",
		}

		// Test CanHandle
		if !handler.CanHandle(ctrlTKey, ctx) {
			t.Error("AICoderKeyHandler should be able to handle Ctrl+T")
		}

		// Test Handle
		cmd := handler.Handle(ctrlTKey, model)
		if cmd == nil {
			t.Error("AICoderKeyHandler should return a command for Ctrl+T")
		}

		// Execute the command to switch view
		msg := cmd()
		if msg == nil {
			t.Error("Command should return a message")
		}

		// Apply the message to the model (this should switch the view)
		newModel, _ := model.Update(msg)
		model = newModel.(*Model)

		// Verify we switched to Processes view
		if model.currentView() != ViewProcesses {
			t.Errorf("Expected to switch to Processes view, but got %v", model.currentView())
		}
	})

	// Test case 2: Ctrl+T with focused session and terminal focus should release focus
	t.Run("Ctrl+T with terminal focus releases focus", func(t *testing.T) {
		// Setup model with focused session and terminal focus
		model := createTestModel(ViewAICoders, true)
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(true)

		// Create Ctrl+T key message
		ctrlTKey := tea.KeyMsg{Type: tea.KeyCtrlT}

		// Create key context
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "",
		}

		// Test CanHandle
		if !handler.CanHandle(ctrlTKey, ctx) {
			t.Error("AICoderKeyHandler should be able to handle Ctrl+T")
		}

		// Test Handle
		cmd := handler.Handle(ctrlTKey, model)
		if cmd == nil {
			t.Error("AICoderKeyHandler should return a command for Ctrl+T")
		}

		// Execute the command
		msg := cmd()
		if msg == nil {
			t.Error("Command should return a message")
		}

		// Apply the message to the model
		newModel, _ := model.Update(msg)
		model = newModel.(*Model)

		// Verify terminal focus was released but session is still focused
		if model.masterData.AICoderState.TerminalFocused {
			t.Error("Terminal focus should be released")
		}
		if model.masterData.AICoderState.FocusedSession != "test-session" {
			t.Error("Focused session should remain the same")
		}
		if model.currentView() != ViewAICoders {
			t.Error("Should remain in AI Coder view")
		}
	})

	// Test case 3: Ctrl+T with focused session but no terminal focus should enable focus
	t.Run("Ctrl+T without terminal focus enables focus", func(t *testing.T) {
		// Setup model with focused session but no terminal focus
		model := createTestModel(ViewAICoders, true)
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(false)

		// Create Ctrl+T key message
		ctrlTKey := tea.KeyMsg{Type: tea.KeyCtrlT}

		// Create key context
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "",
		}

		// Test CanHandle
		if !handler.CanHandle(ctrlTKey, ctx) {
			t.Error("AICoderKeyHandler should be able to handle Ctrl+T")
		}

		// Test Handle
		cmd := handler.Handle(ctrlTKey, model)
		if cmd == nil {
			t.Error("AICoderKeyHandler should return a command for Ctrl+T")
		}

		// Execute the command
		msg := cmd()
		if msg == nil {
			t.Error("Command should return a message")
		}

		// Apply the message to the model
		newModel, _ := model.Update(msg)
		model = newModel.(*Model)

		// Verify terminal focus was enabled
		if !model.masterData.AICoderState.TerminalFocused {
			t.Error("Terminal focus should be enabled")
		}
		if model.masterData.AICoderState.FocusedSession != "test-session" {
			t.Error("Focused session should remain the same")
		}
		if model.currentView() != ViewAICoders {
			t.Error("Should remain in AI Coder view")
		}
	})

	// Test case 4: ESC key should NOT release terminal focus - should go to PTY
	t.Run("ESC key goes to PTY when terminal focused", func(t *testing.T) {
		// Setup model with focused session and terminal focus
		model := createTestModel(ViewAICoders, true)
		model.masterData.AICoderState = model.masterData.AICoderState.SetTerminalFocus(true)

		// Create ESC key message
		escKey := tea.KeyMsg{Type: tea.KeyEscape}

		// Create key context
		ctx := KeyContext{
			View:    ViewAICoders,
			Context: "terminal_focused",
		}

		// Test CanHandle - should handle all keys when terminal focused
		if !handler.CanHandle(escKey, ctx) {
			t.Error("AICoderKeyHandler should be able to handle ESC when terminal focused")
		}

		// Test Handle - should return nil when sending to PTY (input goes directly to service)
		cmd := handler.Handle(escKey, model)
		if cmd != nil {
			t.Error("AICoderKeyHandler should return nil for ESC when terminal focused (input goes to PTY)")
		}

		// Verify terminal focus is STILL enabled (ESC should not release it)
		if !model.masterData.AICoderState.TerminalFocused {
			t.Error("Terminal focus should remain enabled after ESC (ESC should go to PTY)")
		}
		if model.masterData.AICoderState.FocusedSession != "test-session" {
			t.Error("Focused session should remain the same")
		}
		if model.currentView() != ViewAICoders {
			t.Error("Should remain in AI Coder view")
		}
	})
}