package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

func TestNewInputComponent(t *testing.T) {
	// Test creating component with provided service
	inputService := services.NewInputService()
	component := NewInputComponent(inputService)

	if component == nil {
		t.Fatal("Expected component to be created")
	}

	if component.inputService != inputService {
		t.Error("Expected component to use provided input service")
	}

	if component.context != "global" {
		t.Errorf("Expected initial context to be 'global', got %s", component.context)
	}
}

func TestNewInputComponentWithNilService(t *testing.T) {
	// Test creating component with nil service (should create one)
	component := NewInputComponent(nil)

	if component == nil {
		t.Fatal("Expected component to be created")
	}

	if component.inputService == nil {
		t.Error("Expected component to create input service when nil provided")
	}
}

func TestInputComponentInit(t *testing.T) {
	component := NewInputComponent(nil)
	cmd := component.Init()

	if cmd == nil {
		t.Fatal("Expected Init to return a command")
	}

	// Execute the command to start the service
	msg := cmd()
	if msg == nil {
		t.Fatal("Expected command to return a message")
	}

	// Should receive ComponentReadyMsg
	readyMsg, ok := msg.(messages.ComponentReadyMsg)
	if !ok {
		t.Fatalf("Expected ComponentReadyMsg, got %T", msg)
	}

	if readyMsg.ComponentID != "input" {
		t.Errorf("Expected component ID 'input', got %s", readyMsg.ComponentID)
	}
}

func TestInputComponentKeyHandling(t *testing.T) {
	component := NewInputComponent(nil)

	// Test key message handling
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	model, cmd := component.Update(keyMsg)

	if model != component {
		t.Error("Expected model to remain the same after key handling")
	}

	if cmd == nil {
		t.Error("Expected command to be returned after key handling")
	}

	// Execute command to get the key pressed message
	keyPressedMsg := cmd()
	if keyPressedMsg == nil {
		t.Fatal("Expected command to return a message")
	}

	// Should receive InputKeyPressedMsg
	inputMsg, ok := keyPressedMsg.(messages.InputKeyPressedMsg)
	if !ok {
		t.Fatalf("Expected InputKeyPressedMsg, got %T", keyPressedMsg)
	}

	if inputMsg.Context != "global" {
		t.Errorf("Expected context 'global', got %s", inputMsg.Context)
	}

	if !inputMsg.Time.Equal(inputMsg.Time) {
		t.Error("Expected time to be set")
	}
}

func TestInputComponentContextChange(t *testing.T) {
	component := NewInputComponent(nil)

	// Test context change message
	contextMsg := messages.InputContextChangedMsg{
		OldContext: "global",
		NewContext: "aicoder",
		Time:       time.Now(),
	}

	model, _ := component.Update(contextMsg)

	if model != component {
		t.Error("Expected model to remain the same after context change")
	}

	if component.GetContext() != "aicoder" {
		t.Errorf("Expected context to be 'aicoder', got %s", component.GetContext())
	}
}

func TestInputComponentQuitHandling(t *testing.T) {
	component := NewInputComponent(nil)

	// Test first quit press (should show notification)
	quitMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	model, cmd := component.Update(quitMsg)

	if model != component {
		t.Error("Expected model to remain the same after first quit press")
	}

	if cmd == nil {
		t.Error("Expected command to be returned after quit press")
	}

	// Execute command to get notification message
	notificationMsg := cmd()
	if notificationMsg == nil {
		t.Fatal("Expected command to return a message")
	}

	// Should receive SystemNotificationMsg
	sysNotifMsg, ok := notificationMsg.(messages.SystemNotificationMsg)
	if !ok {
		t.Fatalf("Expected SystemNotificationMsg, got %T", notificationMsg)
	}

	if sysNotifMsg.Title != "Quit" {
		t.Errorf("Expected title 'Quit', got %s", sysNotifMsg.Title)
	}

	if sysNotifMsg.Level != messages.InfoNotification {
		t.Errorf("Expected info notification level, got %s", sysNotifMsg.Level)
	}
}

func TestInputComponentFocusBlur(t *testing.T) {
	component := NewInputComponent(nil)

	// Test focus
	cmd := component.Focus()
	if cmd != nil {
		t.Error("Expected Focus to return nil command")
	}

	if !component.IsFocused() {
		t.Error("Expected component to be focused after Focus()")
	}

	// Test blur
	cmd = component.Blur()
	if cmd != nil {
		t.Error("Expected Blur to return nil command")
	}

	if component.IsFocused() {
		t.Error("Expected component to not be focused after Blur()")
	}
}

func TestInputComponentKeyBinding(t *testing.T) {
	component := NewInputComponent(nil)

	// Test key binding registration
	bindingMsg := messages.KeyBindingRegisterMsg{
		Key:      tea.Key{Type: tea.KeyEnter},
		Action:   "test_action",
		Context:  "test",
		Priority: 100,
	}

	model, _ := component.Update(bindingMsg)

	if model != component {
		t.Error("Expected model to remain the same after key binding registration")
	}

	// Check if binding was added (simplified test)
	bindings := component.GetKeyBindings()
	if len(bindings) == 0 {
		t.Error("Expected key binding to be registered")
	}
}

func TestInputComponentCleanup(t *testing.T) {
	component := NewInputComponent(nil)

	// Test cleanup
	err := component.Cleanup()
	if err != nil {
		t.Errorf("Expected cleanup to succeed, got error: %v", err)
	}
}

func TestInputComponentSetContext(t *testing.T) {
	component := NewInputComponent(nil)

	// Test setting context
	component.SetContext("test_context")
	if component.GetContext() != "test_context" {
		t.Errorf("Expected context to be 'test_context', got %s", component.GetContext())
	}
}

func TestInputComponentView(t *testing.T) {
	component := NewInputComponent(nil)

	// Test view (should return empty string for input component)
	view := component.View()
	if view != "" {
		t.Errorf("Expected empty view for input component, got '%s'", view)
	}
}

func TestInputComponentEventHandling(t *testing.T) {
	component := NewInputComponent(nil)

	// Test service event handling
	event := events.Event{
		Type: "input.key.pressed",
		Data: map[string]interface{}{
			"key":     "test_key",
			"context": "test",
			"handled": true,
		},
		Timestamp: time.Now(),
	}

	model, cmd := component.Update(event)

	if model != component {
		t.Error("Expected model to remain the same after event handling")
	}

	if cmd != nil {
		t.Error("Expected no command to be returned for handled key event")
	}
}