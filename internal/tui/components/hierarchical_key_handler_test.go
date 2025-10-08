package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// MockKeyComponent is a simple mock component for key handler testing
type MockKeyComponent struct {
	name    string
	model   tea.Model
	handled map[string]bool
}

func (m *MockKeyComponent) Init() tea.Cmd {
	return nil
}

func (m *MockKeyComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *MockKeyComponent) View() string {
	return ""
}

func (m *MockKeyComponent) SetModel(model tea.Model) {
	m.model = model
}

func (m *MockKeyComponent) GetModel() tea.Model {
	return m.model
}

// TestHierarchicalKeyHandler_PriorityOrdering tests that handlers are called in priority order
func TestHierarchicalKeyHandler_PriorityOrdering(t *testing.T) {
	handler := NewHierarchicalKeyHandler()

	// Create handlers with different priorities
	catchAll := NewComponentKeyHandler("catch-all", PriorityLowest, &MockKeyComponent{})
	catchAll.AddKey("test", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return catchAll.model, nil
	})

	global := NewComponentKeyHandler("global", PriorityGlobal, &MockKeyComponent{})
	global.AddKey("test", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return global.model, nil
	})

	focused := NewComponentKeyHandler("focused", PriorityFocused, &MockKeyComponent{})
	focused.AddKey("test", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return focused.model, nil
	})

	overlay := NewComponentKeyHandler("overlay", PriorityOverlay, &MockKeyComponent{})
	overlay.AddKey("test", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return overlay.model, nil
	})

	// Set up context
	ctx := &KeyHandlerContext{
		CatchAllHandler:  catchAll,
		GlobalHandlers:   []KeyHandler{global},
		FocusedComponent: focused,
		ActiveOverlays:   []KeyHandler{overlay},
	}

	handler.SetContext(ctx)

	// Verify the order is correct (highest priority first)
	activeHandlers := handler.GetActiveHandlers()
	if len(activeHandlers) != 4 {
		t.Fatalf("Expected 4 handlers, got %d", len(activeHandlers))
	}

	// Should be: overlay (1), focused (2), global (4), catch-all (5)
	expectedOrder := []string{"overlay", "focused", "global", "catch-all"}
	for i, expected := range expectedOrder {
		if activeHandlers[i].Name() != expected {
			t.Errorf("Expected handler %d to be %s, got %s", i, expected, activeHandlers[i].Name())
		}
	}
}

// TestHierarchicalKeyHandler_OverlayPriority tests that overlays take priority over other handlers
func TestHierarchicalKeyHandler_OverlayPriority(t *testing.T) {
	handler := NewHierarchicalKeyHandler()

	// Create handlers
	overlay := NewComponentKeyHandler("overlay", PriorityOverlay, &MockKeyComponent{})
	var overlayHandled bool
	overlay.AddKey("enter", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		overlayHandled = true
		return overlay.model, nil
	})

	focused := NewComponentKeyHandler("focused", PriorityFocused, &MockKeyComponent{})
	var focusedHandled bool
	focused.AddKey("enter", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		focusedHandled = true
		return focused.model, nil
	})

	// Set up context with overlay and focused component
	ctx := &KeyHandlerContext{
		ActiveOverlays:   []KeyHandler{overlay},
		FocusedComponent: focused,
	}

	handler.SetContext(ctx)

	// Test that overlay handles the key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = handler.HandleKey(msg, nil)

	if !overlayHandled {
		t.Error("Overlay should have handled the key")
	}
	if focusedHandled {
		t.Error("Focused component should not have handled the key when overlay is active")
	}
}

// TestHierarchicalKeyHandler_FocusedComponentPriority tests that focused components take priority
func TestHierarchicalKeyHandler_FocusedComponentPriority(t *testing.T) {
	handler := NewHierarchicalKeyHandler()

	// Create handlers
	focused := NewComponentKeyHandler("focused", PriorityFocused, &MockKeyComponent{})
	var focusedHandled bool
	focused.AddKey("enter", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		focusedHandled = true
		return focused.model, nil
	})

	view := NewComponentKeyHandler("view", PriorityView, &MockKeyComponent{})
	var viewHandled bool
	view.AddKey("enter", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		viewHandled = true
		return view.model, nil
	})

	// Set up context with focused component and view handler
	ctx := &KeyHandlerContext{
		FocusedComponent: focused,
		ViewHandler:      view,
	}

	handler.SetContext(ctx)

	// Test that focused component handles the key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = handler.HandleKey(msg, nil)

	if !focusedHandled {
		t.Error("Focused component should have handled the key")
	}
	if viewHandled {
		t.Error("View handler should not have handled the key when component is focused")
	}
}

// TextInputWrapper wraps textinput.Model to implement tea.Model properly
type TextInputWrapper struct {
	ti textinput.Model
}

func (w *TextInputWrapper) Init() tea.Cmd {
	return nil
}

func (w *TextInputWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newTI, cmd := w.ti.Update(msg)
	w.ti = newTI
	return w, cmd
}

func (w *TextInputWrapper) View() string {
	return w.ti.View()
}

// TestBubbleTeaComponentAdapter_WithTextInput tests integration with Bubble Tea text input
func TestBubbleTeaComponentAdapter_WithTextInput(t *testing.T) {
	// Create a text input component
	ti := textinput.New()
	ti.Placeholder = "Type here..."
	ti.Focus()

	wrapper := &TextInputWrapper{ti: ti}

	// Create adapter with wrapper (which properly implements tea.Model)
	adapter := NewBubbleTeaComponentAdapter("text-input", PriorityFocused, wrapper, wrapper)
	adapter.SetHandledKeys("enter", "backspace", "ctrl+c", "q")

	// Test that it can handle keys it's configured for
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	if !adapter.CanHandle(enterMsg) {
		t.Error("Adapter should handle enter key")
	}

	backspaceMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	if !adapter.CanHandle(backspaceMsg) {
		t.Error("Adapter should handle backspace key")
	}

	// Test that it doesn't handle keys it's not configured for
	escapeMsg := tea.KeyMsg{Type: tea.KeyEsc}
	if adapter.CanHandle(escapeMsg) {
		t.Error("Adapter should not handle escape key")
	}

	// Test handling a key
	originalValue := wrapper.ti.Value()
	adapter.SetHandledKeys("a", "b", "c")

	aMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	if !adapter.CanHandle(aMsg) {
		t.Error("Adapter should handle 'a' key")
	}

	// Handle the key through the adapter
	newModel, _ := adapter.Handle(aMsg)
	newWrapper := newModel.(*TextInputWrapper)

	if newWrapper.ti.Value() == originalValue {
		t.Error("Text input value should have changed after handling 'a' key")
	}
}

// TestItem implements list.Item for testing
type TestItem struct {
	title string
}

func (i TestItem) Title() string       { return i.title }
func (i TestItem) Description() string { return "" }
func (i TestItem) FilterValue() string { return i.title }

// ListWrapper wraps list.Model to implement tea.Model properly
type ListWrapper struct {
	l list.Model
}

func (w *ListWrapper) Init() tea.Cmd {
	return nil
}

func (w *ListWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newL, cmd := w.l.Update(msg)
	w.l = newL
	return w, cmd
}

func (w *ListWrapper) View() string {
	return w.l.View()
}

// TestBubbleTeaComponentAdapter_WithList tests integration with Bubble Tea list component
func TestBubbleTeaComponentAdapter_WithList(t *testing.T) {
	// Create a list component with proper items
	items := []list.Item{
		TestItem{title: "Item 1"},
		TestItem{title: "Item 2"},
		TestItem{title: "Item 3"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Test List"

	wrapper := &ListWrapper{l: l}

	// Create adapter with wrapper (which properly implements tea.Model)
	adapter := NewBubbleTeaComponentAdapter("list", PriorityFocused, wrapper, wrapper)

	// Test navigation keys
	adapter.SetHandledKeys("up", "down", "enter", "k", "j")

	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	if !adapter.CanHandle(upMsg) {
		t.Error("Adapter should handle up key")
	}

	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	if !adapter.CanHandle(downMsg) {
		t.Error("Adapter should handle down key")
	}

	// Test handling navigation
	originalIndex := wrapper.l.Index()
	adapter.Handle(downMsg)
	newModel := adapter.GetModel()
	newWrapper := newModel.(*ListWrapper)

	if newWrapper.l.Index() != originalIndex+1 {
		t.Errorf("List index should have changed from %d to %d", originalIndex, originalIndex+1)
	}
}

// TestHierarchicalKeyHandler_DynamicContext tests dynamic context changes
func TestHierarchicalKeyHandler_DynamicContext(t *testing.T) {
	handler := NewHierarchicalKeyHandler()

	// Create handlers
	global := NewComponentKeyHandler("global", PriorityGlobal, &MockKeyComponent{})
	global.AddKey("test", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		return global.model, nil
	})

	focused := NewComponentKeyHandler("focused", PriorityFocused, nil)
	var focusedHandled bool
	focused.AddKey("test", func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		focusedHandled = true
		return focused.model, nil
	})

	// Initially no focused component
	ctx := &KeyHandlerContext{
		GlobalHandlers: []KeyHandler{global},
	}
	handler.SetContext(ctx)

	// Test that global handles the key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'e', 's', 't'}}
	_, _ = handler.HandleKey(msg, nil)

	// Now add focused component
	ctx.FocusedComponent = focused
	handler.SetContext(ctx)

	// Reset flag and test again
	focusedHandled = false
	_, _ = handler.HandleKey(msg, nil)

	if !focusedHandled {
		t.Error("Focused component should have handled the key after being added")
	}
}

// TestComponentKeyHandler_KeyBinding tests key binding functionality
func TestComponentKeyHandler_KeyBinding(t *testing.T) {
	handler := NewComponentKeyHandler("test", PriorityGlobal, &MockKeyComponent{})

	var handledKeys []string

	// Add multiple key bindings
	handler.AddKeys(map[string]func(tea.KeyMsg) (tea.Model, tea.Cmd){
		"a": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			handledKeys = append(handledKeys, "a")
			return handler.model, nil
		},
		"b": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			handledKeys = append(handledKeys, "b")
			return handler.model, nil
		},
		"enter": func(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			handledKeys = append(handledKeys, "enter")
			return handler.model, nil
		},
	})

	// Test each key
	keys := []string{"a", "b", "enter"}
	for _, key := range keys {
		var msg tea.KeyMsg
		if key == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(key[0])}}
		}
		if !handler.CanHandle(msg) {
			t.Errorf("Handler should be able to handle key: %s", key)
		}
		handler.Handle(msg)
	}

	// Verify all keys were handled
	if len(handledKeys) != 3 {
		t.Errorf("Expected 3 handled keys, got %d", len(handledKeys))
	}

	// Test unknown key
	unknownMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	if handler.CanHandle(unknownMsg) {
		t.Error("Handler should not handle unknown key")
	}
}
