package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// Error constants for Input Service
const (
	ErrInputServiceNotActive = "INPUT_SERVICE_NOT_ACTIVE"
	ErrInputHandlerExists    = "INPUT_HANDLER_EXISTS"
	ErrInvalidInputBinding   = "INVALID_INPUT_BINDING"
	ErrInputBindingConflict  = "INPUT_BINDING_CONFLICT"
	ErrInputKeyNotFound      = "INPUT_KEY_NOT_FOUND"
)

// KeyBinding represents a keyboard binding configuration
type KeyBinding struct {
	Key      tea.Key
	Action   string
	Context  string
	Priority int
}

// InputHandler represents an input handler with metadata
type InputHandler struct {
	ID        string
	Handler   func(tea.KeyMsg) tea.Cmd
	Context   string
	Priority  int
	CreatedAt time.Time
}

// InputService manages keyboard input, key bindings, and hierarchical routing
type InputService struct {
	// Event handling
	eventBus *events.EventBus

	// Storage for lock-free concurrent access
	keyBindings sync.Map // map[string]*KeyBinding - string representation of tea.Key
	handlers    sync.Map // map[string]*InputHandler - handler ID

	// Handler management
	handlerCounter atomic.Uint64

	// PTY support for testing
	ptyLayer      *pty.PTYLayer
	ptyEnabled    atomic.Bool
	ptyMode       atomic.Value // stores string: "test", "production", "disabled"

	// Lifecycle management
	shutdown atomic.Bool
	shutdownCh chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex // For non-atomic operations

	// Quit functionality (double-press protection)
	lastQuitPress atomic.Value // stores time.Time

	// Statistics
	ptyKeyCount   atomic.Uint64
	totalKeyCount atomic.Uint64
}

// NewInputService creates a new input service instance
func NewInputService() *InputService {
	service := &InputService{
		eventBus:    events.NewEventBus(),
		shutdownCh:  make(chan struct{}),
		lastQuitPress: atomic.Value{},
	}

	// Initialize lastQuitPress with zero time
	service.lastQuitPress.Store(time.Time{})

	// Initialize PTY mode as disabled
	service.ptyMode.Store("disabled")

	return service
}

// NewInputServiceWithPTY creates a new input service instance with PTY support
func NewInputServiceWithPTY(ptyLayer *pty.PTYLayer) *InputService {
	service := &InputService{
		eventBus:    events.NewEventBus(),
		shutdownCh:  make(chan struct{}),
		lastQuitPress: atomic.Value{},
		ptyLayer:    ptyLayer,
	}

	// Initialize lastQuitPress with zero time
	service.lastQuitPress.Store(time.Time{})

	// Enable PTY mode if PTY layer is provided
	if ptyLayer != nil {
		service.ptyEnabled.Store(true)
		service.ptyMode.Store("test")
	} else {
		service.ptyMode.Store("disabled")
	}

	return service
}

// SubscribeToGlobalKeys registers a global keyboard handler
func (s *InputService) SubscribeToGlobalKeys(ctx context.Context, handler func(tea.KeyMsg) tea.Cmd) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	if handler == nil {
		return errors.New(ErrInvalidInputBinding)
	}

	handlerID := fmt.Sprintf("global_%d", s.handlerCounter.Add(1))

	inputHandler := &InputHandler{
		ID:        handlerID,
		Handler:   handler,
		Context:   "global",
		Priority:  1000, // Global handlers have lowest priority
		CreatedAt: time.Now(),
	}

	// Store handler
	s.handlers.Store(handlerID, inputHandler)

	// Publish registration event
	s.eventBus.Publish(events.Event{
		Type: "input.handler.registered",
		Data: map[string]interface{}{
			"handler_id": handlerID,
			"context":    "global",
			"priority":   1000,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// SubscribeWithContextKeys registers a context-specific keyboard handler
func (s *InputService) SubscribeWithContextKeys(ctx context.Context, context string, handler func(tea.KeyMsg) tea.Cmd) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	if handler == nil {
		return errors.New(ErrInvalidInputBinding)
	}

	if context == "" {
		return errors.New(ErrInvalidInputBinding)
	}

	handlerID := fmt.Sprintf("context_%s_%d", context, s.handlerCounter.Add(1))

	inputHandler := &InputHandler{
		ID:        handlerID,
		Handler:   handler,
		Context:   context,
		Priority:  100, // Context handlers have higher priority than global
		CreatedAt: time.Now(),
	}

	// Store handler
	s.handlers.Store(handlerID, inputHandler)

	// Publish registration event
	s.eventBus.Publish(events.Event{
		Type: "input.handler.registered",
		Data: map[string]interface{}{
			"handler_id": handlerID,
			"context":    context,
			"priority":   100,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// RegisterKeyBinding registers a key binding with conflict detection
func (s *InputService) RegisterKeyBinding(ctx context.Context, binding KeyBinding) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	// Validate binding
	if binding.Action == "" || binding.Context == "" {
		return errors.New(ErrInvalidInputBinding)
	}

	keyMsg := tea.KeyMsg{Type: binding.Key.Type, Runes: binding.Key.Runes}
	keyStr := s.keyToString(keyMsg)

	// Store binding - multiple bindings per key are allowed, prioritized by context and priority
	// Use a slice to store multiple bindings per key
	var bindings []*KeyBinding
	if existing, found := s.keyBindings.Load(keyStr); found {
		bindings = existing.([]*KeyBinding)
	}

	// Check for conflicts with same key and context (regardless of action or priority)
	for _, existingBinding := range bindings {
		if existingBinding.Context == binding.Context {
			return errors.New(ErrInputBindingConflict)
		}
	}

	// Allow multiple bindings per key and context - use priority for resolution
	bindings = append(bindings, &binding)
	s.keyBindings.Store(keyStr, bindings)

	// Publish binding change event
	s.eventBus.Publish(events.Event{
		Type: "input.binding.changed",
		Data: map[string]interface{}{
			"action":  "added",
			"key":     keyStr,
			"binding": binding,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterKeyBinding removes a key binding
func (s *InputService) UnregisterKeyBinding(ctx context.Context, key tea.Key) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	keyMsg := tea.KeyMsg{Type: key.Type, Runes: key.Runes}
	keyStr := s.keyToString(keyMsg)

	bindingList, found := s.keyBindings.Load(keyStr)
	if !found {
		return errors.New(ErrInputKeyNotFound)
	}

	bindings := bindingList.([]*KeyBinding)
	if len(bindings) == 0 {
		return errors.New(ErrInputKeyNotFound)
	}

	// Remove the binding that matches the key
	// For simplicity, we'll remove all bindings for this key
	s.keyBindings.Delete(keyStr)

	// Publish binding change event
	s.eventBus.Publish(events.Event{
		Type: "input.binding.changed",
		Data: map[string]interface{}{
			"action": "removed",
			"key":    keyStr,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// GetActiveHandlers returns the list of active handlers
func (s *InputService) GetActiveHandlers() []func(tea.KeyMsg) tea.Cmd {
	var handlers []func(tea.KeyMsg) tea.Cmd

	s.handlers.Range(func(key, value interface{}) bool {
		handler := value.(*InputHandler)
		handlers = append(handlers, handler.Handler)
		return true
	})

	return handlers
}

// GetGlobalBindings returns the list of global key bindings
func (s *InputService) GetGlobalBindings() []KeyBinding {
	var bindings []KeyBinding

	s.keyBindings.Range(func(key, value interface{}) bool {
		bindingList := value.([]*KeyBinding)
		for _, binding := range bindingList {
			bindings = append(bindings, *binding)
		}
		return true
	})

	return bindings
}

// RouteKeyMessage routes a key message to appropriate handlers using hierarchical logic
func (s *InputService) RouteKeyMessage(keyMsg tea.KeyMsg, context string) (bool, tea.Cmd) {
	if s.shutdown.Load() {
		return false, nil
	}

	handled := false
	var resultCmd tea.Cmd
	keyStr := s.keyToString(keyMsg)

	// Publish key pressed event
	s.eventBus.Publish(events.Event{
		Type: "input.key.pressed",
		Data: map[string]interface{}{
			"key":     keyStr,
			"context": context,
			"handled": false,
		},
		Timestamp: time.Now(),
	})

	// Hierarchical routing: context-specific handlers first, then global handlers
	var priorityHandlers []*InputHandler

	// Collect handlers based on context
	s.handlers.Range(func(key, value interface{}) bool {
		handler := value.(*InputHandler)
		if handler.Context == context || handler.Context == "global" {
			priorityHandlers = append(priorityHandlers, handler)
		}
		return true
	})

	// Sort by priority (lower number = higher priority)
	// Context handlers (priority 100) should come before global (priority 1000)
	for i := 0; i < len(priorityHandlers); i++ {
		for j := i + 1; j < len(priorityHandlers); j++ {
			if priorityHandlers[i].Priority > priorityHandlers[j].Priority {
				priorityHandlers[i], priorityHandlers[j] = priorityHandlers[j], priorityHandlers[i]
			}
		}
	}

	// Try handlers in priority order
	for _, handler := range priorityHandlers {
		// Call the handler and capture the returned command
		cmd := handler.Handler(keyMsg)
		handled = true
		resultCmd = cmd  // Capture the command to return
		break
	}

	// If no handler handled it, check for key bindings
	if !handled {
		if bindingList, found := s.keyBindings.Load(keyStr); found {
			bindings := bindingList.([]*KeyBinding)
			// Find the best binding for the current context (highest priority)
			var bestBinding *KeyBinding
			for _, keyBinding := range bindings {
				if keyBinding.Context == context || keyBinding.Context == "global" {
					if bestBinding == nil || keyBinding.Priority < bestBinding.Priority {
						bestBinding = keyBinding
					}
				}
			}
			if bestBinding != nil {
				handled = true
			}
		}
	}

	// Handle quit functionality (double-press protection)
	if !handled && (keyMsg.Type == tea.KeyCtrlC || keyMsg.Type == tea.KeyEsc) {
		now := time.Now()
		lastPress, _ := s.lastQuitPress.Load().(time.Time)

		if now.Sub(lastPress) < 500*time.Millisecond {
			// Double press detected - handle quit
			handled = true
			resultCmd = tea.Quit  // Return quit command
		} else {
			// First press - store timestamp
			s.lastQuitPress.Store(now)
		}
	}

	// Update event with handled status
	s.eventBus.Publish(events.Event{
		Type: "input.key.pressed",
		Data: map[string]interface{}{
			"key":     keyStr,
			"context": context,
			"handled": handled,
		},
		Timestamp: time.Now(),
	})

	return handled, resultCmd
}

// Start initializes the service and begins event handling
func (s *InputService) Start(ctx context.Context) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	// Initialize any background tasks here if needed
	s.wg.Add(1)
	go s.backgroundWorker()

	return nil
}

// Shutdown gracefully shuts down the service
func (s *InputService) Shutdown(ctx context.Context) error {
	if !s.shutdown.Swap(true) {
		return nil // Already shutdown
	}

	// Close shutdown channel
	close(s.shutdownCh)

	// Wait for background tasks
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Cleanup event bus
		s.eventBus.Shutdown()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// backgroundWorker handles any background tasks
func (s *InputService) backgroundWorker() {
	defer s.wg.Done()

	for {
		select {
		case <-s.shutdownCh:
			return
		case <-time.After(1 * time.Second):
			// Periodic cleanup tasks could go here
			// For now, just continue
		}
	}
}

// keyToString converts a tea.Key or tea.KeyMsg to a string representation for map keys
func (s *InputService) keyToString(keyMsg tea.KeyMsg) string {
	if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 {
		return string(keyMsg.Runes)
	}
	return keyMsg.Type.String()
}

// EventBus returns the event bus for external subscriptions
func (s *InputService) EventBus() *events.EventBus {
	return s.eventBus
}

// GetServiceName returns the service name for ServiceBridge integration
func (s *InputService) GetServiceName() string {
	return "input"
}

// GetServiceStatus returns the current status of the input service
func (s *InputService) GetServiceStatus() ServiceStatus {
	status := "active"

	// Build detailed status information
	var details string
	if s.ptyEnabled.Load() {
		stats := s.GetPTYStatistics()
		details = fmt.Sprintf("handlers: %d, bindings: %d, PTY: %s (%s), PTY keys: %d/%d",
			s.getHandlerCount(), s.getBindingCount(), stats.PTYMode,
			map[bool]string{true: "active", false: "inactive"}[stats.PTYLayerActive],
			stats.PTYKeyCount, stats.TotalKeyCount)
	} else {
		details = fmt.Sprintf("handlers: %d, bindings: %d", s.getHandlerCount(), s.getBindingCount())
	}

	if s.shutdown.Load() {
		status = "shutting down"
	}

	return ServiceStatus{
		Service: s.GetServiceName(),
		Status:  status,
		Details: details,
	}
}

// SubscribeToEvents subscribes the input service to relevant system events
func (s *InputService) SubscribeToEvents() {
	if s.eventBus == nil {
		return
	}

	// Input service doesn't need to subscribe to external events
	// It only publishes events based on user input
}

// Stop gracefully shuts down the input service
func (s *InputService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}

// getHandlerCount returns the current number of registered handlers
func (s *InputService) getHandlerCount() int {
	count := 0
	s.handlers.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetHandlerCount returns the current number of registered handlers (exported method)
func (s *InputService) GetHandlerCount() int {
	return s.getHandlerCount()
}

// IsRunning returns whether the input service is currently active
func (s *InputService) IsRunning() bool {
	return !s.shutdown.Load()
}

// getBindingCount returns the current number of registered bindings
func (s *InputService) getBindingCount() int {
	count := 0
	s.keyBindings.Range(func(key, value interface{}) bool {
		bindings := value.([]*KeyBinding)
		count += len(bindings)
		return true
	})
	return count
}

// PTY-specific methods

// EnablePTY enables PTY mode for testing with the specified PTY layer
func (s *InputService) EnablePTY(ptyLayer *pty.PTYLayer) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	if ptyLayer == nil {
		return errors.New("PTY layer cannot be nil")
	}

	if !ptyLayer.IsAllocated() {
		return errors.New("PTY layer must be allocated before enabling")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ptyLayer = ptyLayer
	s.ptyEnabled.Store(true)
	s.ptyMode.Store("test")

	// Publish PTY enabled event
	s.eventBus.Publish(events.Event{
		Type: "input.pty.enabled",
		Data: map[string]interface{}{
			"pty_enabled": true,
			"mode":        "test",
		},
		Timestamp: time.Now(),
	})

	return nil
}

// DisablePTY disables PTY mode
func (s *InputService) DisablePTY() error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.ptyLayer = nil
	s.ptyEnabled.Store(false)
	s.ptyMode.Store("disabled")

	// Publish PTY disabled event
	s.eventBus.Publish(events.Event{
		Type: "input.pty.disabled",
		Data: map[string]interface{}{
			"pty_enabled": false,
			"mode":        "disabled",
		},
		Timestamp: time.Now(),
	})

	return nil
}

// IsPTYEnabled returns whether PTY mode is enabled
func (s *InputService) IsPTYEnabled() bool {
	return s.ptyEnabled.Load()
}

// GetPTYMode returns the current PTY mode
func (s *InputService) GetPTYMode() string {
	if mode, ok := s.ptyMode.Load().(string); ok {
		return mode
	}
	return "disabled"
}

// SetPTYMode sets the PTY mode ("test", "production", or "disabled")
func (s *InputService) SetPTYMode(mode string) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	validModes := map[string]bool{
		"test":       true,
		"production": true,
		"disabled":   true,
	}

	if !validModes[mode] {
		return fmt.Errorf("invalid PTY mode: %s, must be one of: test, production, disabled", mode)
	}

	s.ptyMode.Store(mode)

	// Update PTY enabled state based on mode
	s.ptyEnabled.Store(mode != "disabled")

	// Publish PTY mode changed event
	s.eventBus.Publish(events.Event{
		Type: "input.pty.mode_changed",
		Data: map[string]interface{}{
			"old_mode": s.GetPTYMode(),
			"new_mode": mode,
		},
		Timestamp: time.Now(),
	})

	return nil
}

// RouteKeyMessagePTY routes a key message specifically from PTY input
func (s *InputService) RouteKeyMessagePTY(keyMsg tea.KeyMsg, context string) (bool, tea.Cmd) {
	if s.shutdown.Load() {
		return false, nil
	}

	// Update statistics
	s.totalKeyCount.Add(1)
	if s.ptyEnabled.Load() {
		s.ptyKeyCount.Add(1)
	}

	// Add PTY context information
	ptyContext := context
	if ptyContext == "" {
		ptyContext = "pty_input"
	}

	// Route through the standard key message router
	return s.RouteKeyMessage(keyMsg, ptyContext)
}

// SimulatePTYKeyPress simulates a key press through the PTY layer
func (s *InputService) SimulatePTYKeyPress(key tea.KeyMsg) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	if !s.ptyEnabled.Load() {
		return errors.New("PTY mode is not enabled")
	}

	if s.ptyLayer == nil {
		return errors.New("PTY layer is not available")
	}

	// Route through PTY-aware router
	handled, _ := s.RouteKeyMessagePTY(key, "pty_simulation")

	// If not handled by InputService, send to PTY layer
	if !handled {
		if err := s.ptyLayer.SimulateKeyPress(key); err != nil {
			return fmt.Errorf("failed to send key to PTY layer: %w", err)
		}
	}

	return nil
}

// GetPTYLayer returns the underlying PTY layer if available
func (s *InputService) GetPTYLayer() *pty.PTYLayer {
	return s.ptyLayer
}

// GetPTYStatistics returns PTY-related statistics
func (s *InputService) GetPTYStatistics() PTYStatistics {
	return PTYStatistics{
		PTYEnabled:     s.ptyEnabled.Load(),
		PTYMode:        s.GetPTYMode(),
		PTYKeyCount:    s.ptyKeyCount.Load(),
		TotalKeyCount:  s.totalKeyCount.Load(),
		PTYLayerActive: s.ptyLayer != nil && s.ptyLayer.IsAllocated(),
	}
}

// PTYStatistics contains PTY-related statistics
type PTYStatistics struct {
	PTYEnabled     bool
	PTYMode        string
	PTYKeyCount    uint64
	TotalKeyCount  uint64
	PTYLayerActive bool
}

// ValidatePTYInput validates that PTY input can be properly routed through InputService
func (s *InputService) ValidatePTYInput(testKeys []tea.KeyMsg) error {
	if s.shutdown.Load() {
		return errors.New(ErrInputServiceNotActive)
	}

	if !s.ptyEnabled.Load() {
		return errors.New("PTY mode is not enabled")
	}

	if s.ptyLayer == nil {
		return errors.New("PTY layer is not available")
	}

	if !s.ptyLayer.IsAllocated() {
		return errors.New("PTY layer is not allocated")
	}

	// Test each key through the PTY routing path
	for i, key := range testKeys {
		if err := s.SimulatePTYKeyPress(key); err != nil {
			return fmt.Errorf("failed to validate PTY input for key at index %d (%v): %w", i, key, err)
		}
	}

	return nil
}

// CreateTestKeySequence creates a test key sequence for PTY validation
func CreateTestKeySequence() []tea.KeyMsg {
	return []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'t'}},
		{Type: tea.KeyRunes, Runes: []rune{'e'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
		{Type: tea.KeyRunes, Runes: []rune{'t'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyTab},
		{Type: tea.KeyEsc},
		{Type: tea.KeyCtrlC},
	}
}