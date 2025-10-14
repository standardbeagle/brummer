package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestViewSpecificKeyHandling tests that the same key can be handled differently in different views
func TestViewSpecificKeyHandling(t *testing.T) {
	registry := NewKeyHandlerRegistry()

	var logsHandlerCalled, errorsHandlerCalled bool

	// Register handler for 'c' key in Logs view (clear logs)
	logsHandler := NewViewSpecificKeyHandler(ViewLogs, func(key tea.KeyMsg, model *Model) tea.Cmd {
		if key.Type == tea.KeyRunes && len(key.Runes) == 1 && key.Runes[0] == 'c' {
			logsHandlerCalled = true
			return func() tea.Msg {
				return StatusMsg{Message: "Cleared logs"}
			}
		}
		return nil
	})

	// Register handler for 'c' key in Errors view (copy error)
	errorsHandler := NewViewSpecificKeyHandler(ViewErrors, func(key tea.KeyMsg, model *Model) tea.Cmd {
		if key.Type == tea.KeyRunes && len(key.Runes) == 1 && key.Runes[0] == 'c' {
			errorsHandlerCalled = true
			return func() tea.Msg {
				return StatusMsg{Message: "Copied error"}
			}
		}
		return nil
	})

	registry.Register(logsHandler)
	registry.Register(errorsHandler)

	// Test 'c' key in Logs view
	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	model := &Model{}

	// In Logs view, only logs handler should be called
	ctxLogs := KeyContext{View: ViewLogs, Context: "normal"}
	logsHandlerCalled = false
	errorsHandlerCalled = false

	cmd := registry.HandleKey(cKey, ctxLogs, model)
	assert.NotNil(t, cmd, "Command should be returned for 'c' in logs view")

	msg := cmd()
	statusMsg, ok := msg.(StatusMsg)
	assert.True(t, ok, "Should return StatusMsg")
	assert.Equal(t, "Cleared logs", statusMsg.Message)
	assert.True(t, logsHandlerCalled, "Logs handler should be called")
	assert.False(t, errorsHandlerCalled, "Errors handler should NOT be called")

	// In Errors view, only errors handler should be called
	ctxErrors := KeyContext{View: ViewErrors, Context: "normal"}
	logsHandlerCalled = false
	errorsHandlerCalled = false

	cmd = registry.HandleKey(cKey, ctxErrors, model)
	assert.NotNil(t, cmd, "Command should be returned for 'c' in errors view")

	msg = cmd()
	statusMsg, ok = msg.(StatusMsg)
	assert.True(t, ok, "Should return StatusMsg")
	assert.Equal(t, "Copied error", statusMsg.Message)
	assert.False(t, logsHandlerCalled, "Logs handler should NOT be called")
	assert.True(t, errorsHandlerCalled, "Errors handler should be called")

	// In a different view (e.g., Processes), neither handler should be called
	ctxProcesses := KeyContext{View: ViewProcesses, Context: "normal"}
	logsHandlerCalled = false
	errorsHandlerCalled = false

	cmd = registry.HandleKey(cKey, ctxProcesses, model)
	assert.Nil(t, cmd, "No command should be returned for 'c' in processes view")
	assert.False(t, logsHandlerCalled, "Logs handler should NOT be called")
	assert.False(t, errorsHandlerCalled, "Errors handler should NOT be called")
}

// TestConflictingGlobalHandlers tests that registering multiple global handlers for the same key
// results in priority-based resolution (no error, but deterministic behavior)
func TestConflictingGlobalHandlers(t *testing.T) {
	registry := NewKeyHandlerRegistry()

	var handler1Called, handler2Called bool

	// Create two global handlers that both want to handle 'g' key
	// Handler 1 with priority 150
	handler1 := &testKeyHandler{
		name:     "GlobalHandler1",
		priority: 150,
		canHandle: func(key tea.KeyMsg, ctx KeyContext) bool {
			return key.Type == tea.KeyRunes && len(key.Runes) == 1 && key.Runes[0] == 'g'
		},
		handle: func(key tea.KeyMsg, model *Model) tea.Cmd {
			handler1Called = true
			return func() tea.Msg {
				return StatusMsg{Message: "Handler1 processed 'g'"}
			}
		},
	}

	// Handler 2 with priority 250 (lower priority)
	handler2 := &testKeyHandler{
		name:     "GlobalHandler2",
		priority: 250,
		canHandle: func(key tea.KeyMsg, ctx KeyContext) bool {
			return key.Type == tea.KeyRunes && len(key.Runes) == 1 && key.Runes[0] == 'g'
		},
		handle: func(key tea.KeyMsg, model *Model) tea.Cmd {
			handler2Called = true
			return func() tea.Msg {
				return StatusMsg{Message: "Handler2 processed 'g'"}
			}
		},
	}

	// Register both handlers
	registry.Register(handler1)
	registry.Register(handler2)

	// Press 'g' key
	gKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	ctx := KeyContext{View: ViewProcesses, Context: "normal"}
	model := &Model{}

	cmd := registry.HandleKey(gKey, ctx, model)
	assert.NotNil(t, cmd, "Command should be returned")

	// Only handler1 (higher priority) should be called
	assert.True(t, handler1Called, "Handler1 (priority 150) should be called")
	assert.False(t, handler2Called, "Handler2 (priority 250) should NOT be called due to lower priority")

	msg := cmd()
	statusMsg, ok := msg.(StatusMsg)
	assert.True(t, ok, "Should return StatusMsg")
	assert.Equal(t, "Handler1 processed 'g'", statusMsg.Message)
}

// TestModalContextPriority tests that modal/overlay contexts take precedence
func TestModalContextPriority(t *testing.T) {
	registry := NewKeyHandlerRegistry()

	var normalHandlerCalled, modalHandlerCalled bool

	// Normal view handler for Enter key (priority 200)
	normalHandler := NewViewSpecificKeyHandler(ViewProcesses, func(key tea.KeyMsg, model *Model) tea.Cmd {
		if key.Type == tea.KeyEnter {
			normalHandlerCalled = true
			return func() tea.Msg {
				return StatusMsg{Message: "Normal Enter handled"}
			}
		}
		return nil
	})

	// Modal handler with higher priority (priority 50)
	modalHandler := &testKeyHandler{
		name:     "ModalHandler",
		priority: 50, // Higher priority than view-specific (200)
		canHandle: func(key tea.KeyMsg, ctx KeyContext) bool {
			// Only handle if in modal context
			return ctx.Context == "modal" && key.Type == tea.KeyEnter
		},
		handle: func(key tea.KeyMsg, model *Model) tea.Cmd {
			modalHandlerCalled = true
			return func() tea.Msg {
				return StatusMsg{Message: "Modal Enter handled"}
			}
		},
	}

	registry.Register(normalHandler)
	registry.Register(modalHandler)

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	model := &Model{}

	// Test 1: Normal context - view handler should handle it
	ctxNormal := KeyContext{View: ViewProcesses, Context: "normal"}
	normalHandlerCalled = false
	modalHandlerCalled = false

	cmd := registry.HandleKey(enterKey, ctxNormal, model)
	assert.NotNil(t, cmd)
	assert.True(t, normalHandlerCalled, "Normal handler should be called in normal context")
	assert.False(t, modalHandlerCalled, "Modal handler should NOT be called in normal context")

	// Test 2: Modal context - modal handler should handle it (higher priority)
	ctxModal := KeyContext{View: ViewProcesses, Context: "modal"}
	normalHandlerCalled = false
	modalHandlerCalled = false

	cmd = registry.HandleKey(enterKey, ctxModal, model)
	assert.NotNil(t, cmd)
	assert.False(t, normalHandlerCalled, "Normal handler should NOT be called in modal context")
	assert.True(t, modalHandlerCalled, "Modal handler should be called in modal context")

	msg := cmd()
	statusMsg, ok := msg.(StatusMsg)
	assert.True(t, ok)
	assert.Equal(t, "Modal Enter handled", statusMsg.Message)
}

