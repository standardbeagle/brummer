package tui

import (
	"testing"

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
			return nil
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
			return nil
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
			return nil
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

	// Test 2: In AI Coder PTY mode at start of line, should be handled
	ctx = KeyContext{View: ViewAICoders, Context: "pty", CursorPosition: 0}
	assert.True(t, handler.CanHandle(slashKey, ctx))

	// Test 3: In AI Coder PTY mode mid-line, should NOT be handled
	ctx = KeyContext{View: ViewAICoders, Context: "pty", CursorPosition: 5}
	assert.False(t, handler.CanHandle(slashKey, ctx))

	// Test 4: In AI Coder list mode, should be handled
	ctx = KeyContext{View: ViewAICoders, Context: "normal"}
	assert.True(t, handler.CanHandle(slashKey, ctx))

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

	// Test Tab key handling
	tabKey := tea.KeyMsg{Type: tea.KeyTab}
	ctx := KeyContext{View: ViewProcesses}
	assert.True(t, handler.CanHandle(tabKey, ctx))

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

	// Test double-press protection
	model := &Model{}

	// First press should return status message
	cmd := handler.Handle(ctrlC, model)
	assert.NotNil(t, cmd)
	msg := cmd()
	statusMsg, ok := msg.(StatusMsg)
	assert.True(t, ok)
	assert.Equal(t, "Press Ctrl+C again to quit", statusMsg.Message)

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