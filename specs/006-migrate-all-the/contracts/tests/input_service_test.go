package contracts_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/services"
)

// These tests validate the Input Service API contract
// They should fail initially and pass after implementation

func TestInputService_SubscribeToGlobalKeys(t *testing.T) {
	// Test subscription to global keyboard events
	// This should fail before implementation

	ctx := context.Background()
	handler := func(key tea.KeyMsg) tea.Cmd {
		return nil
	}

	// Attempt to use input service - will fail due to missing implementation
	service := NewInputService()
	err := service.SubscribeToGlobalKeys(ctx, handler)

	if err != nil {
		t.Fatalf("Expected SubscribeToGlobalKeys to work, got error: %v", err)
	}

	// Verify handler is registered
	handlers := service.GetActiveHandlers()
	if len(handlers) == 0 {
		t.Fatalf("Expected handlers to be registered, but got empty list")
	}
	// Note: Can't compare functions directly, so just check that handlers exist
	// In actual implementation, we'd use a more sophisticated approach
}

func TestInputService_RegisterKeyBinding(t *testing.T) {
	// Test key binding registration
	// This should fail before implementation

	ctx := context.Background()
	binding := KeyBinding{
		Key:      tea.Key{Type: tea.KeyEnter},
		Action:   "confirm",
		Context:  "global",
		Priority: 1,
	}

	// Attempt to register key binding - will fail due to missing implementation
	service := NewInputService()
	err := service.RegisterKeyBinding(ctx, binding)

	if err != nil {
		t.Fatalf("Expected RegisterKeyBinding to work, got error: %v", err)
	}

	// Verify binding is registered
	bindings := service.GetGlobalBindings()
	found := false
	for _, b := range bindings {
		if b.Action == binding.Action && b.Context == binding.Context {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected key binding to be in global bindings list")
	}
}

func TestInputService_RegisterKeyBinding_Conflict(t *testing.T) {
	// Test key binding conflict detection
	// This should fail before implementation

	ctx := context.Background()
	conflictingBinding := KeyBinding{
		Key:      tea.Key{Type: tea.KeyEnter},
		Action:   "different_action",
		Context:  "global",
		Priority: 2,
	}

	// Attempt to test conflict detection - will fail due to missing implementation
	service := NewInputService()

	// Register first binding
	firstBinding := KeyBinding{
		Key:      tea.Key{Type: tea.KeyEnter},
		Action:   "confirm",
		Context:  "global",
		Priority: 1,
	}
	err1 := service.RegisterKeyBinding(ctx, firstBinding)
	if err1 != nil {
		t.Fatalf("Expected first binding registration to work, got error: %v", err1)
	}

	// Attempt to register conflicting binding - should fail with conflict error
	err2 := service.RegisterKeyBinding(ctx, conflictingBinding)
	if err2 == nil {
		t.Fatalf("Expected conflicting binding registration to fail with error")
	}
	if err2.Error() != ErrInputBindingConflict {
		t.Fatalf("Expected binding conflict error '%s', got '%v'", ErrInputBindingConflict, err2)
	}
}

func TestInputService_UnregisterKeyBinding(t *testing.T) {
	// Test key binding removal
	// This should fail before implementation

	ctx := context.Background()
	key := tea.Key{Type: tea.KeyEnter}

	// Attempt to test key binding removal - will fail due to missing implementation
	service := NewInputService()

	// Register binding first
	binding := KeyBinding{
		Key:     key,
		Action:  "confirm",
		Context: "global",
	}
	err1 := service.RegisterKeyBinding(ctx, binding)
	if err1 != nil {
		t.Fatalf("Expected binding registration to work, got error: %v", err1)
	}

	// Verify binding is registered before removal
	bindingsBefore := service.GetGlobalBindings()
	foundBefore := false
	for _, b := range bindingsBefore {
		if b.Action == binding.Action && b.Context == binding.Context {
			foundBefore = true
			break
		}
	}
	if !foundBefore {
		t.Fatalf("Expected binding to be registered before removal")
	}

	// Unregister binding
	err2 := service.UnregisterKeyBinding(ctx, key)
	if err2 != nil {
		t.Fatalf("Expected binding unregistration to work, got error: %v", err2)
	}

	// Verify binding is removed
	bindingsAfter := service.GetGlobalBindings()
	foundAfter := false
	for _, b := range bindingsAfter {
		if b.Action == binding.Action && b.Context == binding.Context {
			foundAfter = true
			break
		}
	}
	if foundAfter {
		t.Fatalf("Expected binding to be removed from global bindings list")
	}
}

func TestInputService_HierarchicalKeyHandling(t *testing.T) {
	// Test hierarchical key input routing
	// This should fail before implementation

	ctx := context.Background()

	// Attempt to test hierarchical input handling - will fail due to missing implementation
	service := NewInputService()

	// Create handlers with different priorities
	globalHandler := func(key tea.KeyMsg) tea.Cmd { return nil }
	contextHandler := func(key tea.KeyMsg) tea.Cmd { return nil }

	// Register handlers
	err1 := service.SubscribeToGlobalKeys(ctx, globalHandler)
	if err1 != nil {
		t.Fatalf("Expected global key subscription to work, got error: %v", err1)
	}

	err2 := service.SubscribeWithContextKeys(ctx, "file_browser", contextHandler)
	if err2 != nil {
		t.Fatalf("Expected context key subscription to work, got error: %v", err2)
	}

	// Test key routing priority - context handler should be called before global handler
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	handled := service.RouteKeyMessage(keyMsg, "file_browser")
	if !handled {
		t.Fatalf("Expected key message to be handled by hierarchical routing")
	}
}

func TestInputService_Performance(t *testing.T) {
	// Test input service performance meets <5ms target
	// This should fail before implementation

	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()

	// Attempt to test input service performance - will fail due to missing implementation
	service := NewInputService()

	// Register multiple bindings to create a realistic test scenario
	for i := 0; i < 100; i++ {
		binding := KeyBinding{
			Key:      tea.Key{Type: tea.KeyRunes, Runes: []rune{rune('a' + i % 26)}},
			Action:   fmt.Sprintf("action_%d", i),
			Context:  "test",
			Priority: i,
		}
		err := service.RegisterKeyBinding(ctx, binding)
		if err != nil {
			t.Fatalf("Expected binding registration to work, got error: %v", err)
		}
	}

	// Measure key processing time
	start := time.Now()
	for i := 0; i < 1000; i++ {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		handled := service.RouteKeyMessage(keyMsg, "test")
		if !handled {
			t.Fatalf("Expected key message to be handled during performance test")
		}
	}
	duration := time.Since(start)

	avgTime := duration / 1000
	if avgTime >= 5*time.Millisecond {
		t.Fatalf("Average key processing time %v is >= 5ms target", avgTime)
	}
}

// Input Service interface that must be implemented
// This defines the contract that tests expect
type InputService interface {
	SubscribeToGlobalKeys(ctx context.Context, handler func(tea.KeyMsg) tea.Cmd) error
	SubscribeWithContextKeys(ctx context.Context, context string, handler func(tea.KeyMsg) tea.Cmd) error
	RegisterKeyBinding(ctx context.Context, binding KeyBinding) error
	UnregisterKeyBinding(ctx context.Context, key tea.Key) error
	GetActiveHandlers() []func(tea.KeyMsg) tea.Cmd
	GetGlobalBindings() []KeyBinding
	RouteKeyMessage(keyMsg tea.KeyMsg, context string) bool
}

// NewInputService creates a new input service instance
// This function needs to be implemented
func NewInputService() InputService {
	// Import and use the actual implementation
	// This is the bridge between contract tests and implementation
	return &inputServiceAdapter{
		service: services.NewInputService(),
	}
}

// inputServiceAdapter adapts the actual implementation to the contract test interface
type inputServiceAdapter struct {
	service *services.InputService
}

func (a *inputServiceAdapter) SubscribeToGlobalKeys(ctx context.Context, handler func(tea.KeyMsg) tea.Cmd) error {
	return a.service.SubscribeToGlobalKeys(ctx, handler)
}

func (a *inputServiceAdapter) SubscribeWithContextKeys(ctx context.Context, context string, handler func(tea.KeyMsg) tea.Cmd) error {
	return a.service.SubscribeWithContextKeys(ctx, context, handler)
}

func (a *inputServiceAdapter) RegisterKeyBinding(ctx context.Context, binding KeyBinding) error {
	// Convert contract type to implementation type
	implBinding := services.KeyBinding{
		Key:      binding.Key,
		Action:   binding.Action,
		Context:  binding.Context,
		Priority: binding.Priority,
	}
	return a.service.RegisterKeyBinding(ctx, implBinding)
}

func (a *inputServiceAdapter) UnregisterKeyBinding(ctx context.Context, key tea.Key) error {
	return a.service.UnregisterKeyBinding(ctx, key)
}

func (a *inputServiceAdapter) GetActiveHandlers() []func(tea.KeyMsg) tea.Cmd {
	return a.service.GetActiveHandlers()
}

func (a *inputServiceAdapter) GetGlobalBindings() []KeyBinding {
	// Convert implementation types back to contract types
	implBindings := a.service.GetGlobalBindings()
	contractBindings := make([]KeyBinding, len(implBindings))
	for i, b := range implBindings {
		contractBindings[i] = KeyBinding{
			Key:      b.Key,
			Action:   b.Action,
			Context:  b.Context,
			Priority: b.Priority,
		}
	}
	return contractBindings
}

func (a *inputServiceAdapter) RouteKeyMessage(keyMsg tea.KeyMsg, context string) bool {
	return a.service.RouteKeyMessage(keyMsg, context)
}

// Test data types (will be defined in implementation)
type KeyBinding struct {
	Key      tea.Key
	Action   string
	Context  string
	Priority int
}

// Error constants for Input Service (will be defined in implementation)
const (
	ErrInputServiceNotActive = "INPUT_SERVICE_NOT_ACTIVE"
	ErrInputHandlerExists    = "INPUT_HANDLER_EXISTS"
	ErrInvalidInputBinding   = "INVALID_INPUT_BINDING"
	ErrInputBindingConflict  = "INPUT_BINDING_CONFLICT"
	ErrInputKeyNotFound      = "INPUT_KEY_NOT_FOUND"
)