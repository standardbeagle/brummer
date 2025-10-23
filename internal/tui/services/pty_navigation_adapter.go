//go:build !windows

package services

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// PTYNavigationAdapter connects PTY view switching with the NavigationService
// to enable realistic view switching testing through the actual NavigationService
// used in production. This provides a bridge between PTY input simulation and
// the NavigationService for comprehensive TUI testing.
type PTYNavigationAdapter struct {
	// Core components
	ptyLayer pty.PTYLayer
	navigationService *NavigationService
	eventBus         *events.EventBus

	// State management
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown atomic.Bool
	active   atomic.Bool

	// Navigation state tracking
	currentView     atomic.Value // stores messages.ViewType
	viewHistory     []messages.ViewType
	switchCount     atomic.Uint64
	lastSwitchTime  atomic.Value // stores time.Time
	switchTriggers  map[string]int // Track triggers for each view switch

	// PTY input handling
	inputAdapter *PTYInputAdapter

	// Event handling
	eventChan chan NavigationEvent
	errorChan chan error

	// Synchronization
	mu sync.RWMutex
	wg sync.WaitGroup

	// Configuration
	config PTYNavigationAdapterConfig

	// Statistics
	keyPressCount     atomic.Uint64
	errorCount        atomic.Uint64
	lastActivity      atomic.Value // stores time.Time
	numberKeyPresses  map[int]int    // Track number key presses
	tabPresses        atomic.Uint64 // Track tab key presses
}

// PTYNavigationAdapterConfig contains configuration for the PTY navigation adapter
type PTYNavigationAdapterConfig struct {
	// Buffer sizes
	EventBufferSize   int
	ErrorBufferSize   int

	// Timing and timeouts
	ReadTimeout       time.Duration
	ShutdownTimeout   time.Duration
	ActivityTimeout   time.Duration
	ViewSwitchTimeout time.Duration

	// Feature flags
	EnableStatistics    bool
	EnableDebug         bool
	TrackViewHistory    bool
	ValidateSwitches    bool
	RecordTriggers      bool

	// PTY monitoring
	MonitorPTYOutput    bool
	PTYReadInterval     time.Duration

	// Navigation testing
	MaxViewHistory      int
	ValidateNumberKeys  bool
	ValidateTabCycling  bool
}

// NavigationEvent represents a navigation event from the PTY layer
type NavigationEvent struct {
	Type      NavigationEventType
	View      messages.ViewType
	Trigger   string
	Timestamp time.Time
	Key       tea.KeyMsg
}

// NavigationEventType represents the type of navigation event
type NavigationEventType int

const (
	NavigationEventSwitch NavigationEventType = iota
	NavigationEventCycle
	NavigationEventNumberKey
	NavigationEventTabKey
	NavigationEventError
)

// DefaultPTYNavigationAdapterConfig returns a default configuration
func DefaultPTYNavigationAdapterConfig() PTYNavigationAdapterConfig {
	return PTYNavigationAdapterConfig{
		EventBufferSize:     1000,
		ErrorBufferSize:     100,
		ReadTimeout:         100 * time.Millisecond,
		ShutdownTimeout:     5 * time.Second,
		ActivityTimeout:     30 * time.Second,
		ViewSwitchTimeout:   1 * time.Second,
		EnableStatistics:    true,
		EnableDebug:         false,
		TrackViewHistory:    true,
		ValidateSwitches:    true,
		RecordTriggers:      true,
		MonitorPTYOutput:    false,
		PTYReadInterval:     10 * time.Millisecond,
		MaxViewHistory:      100,
		ValidateNumberKeys:  true,
		ValidateTabCycling:  true,
	}
}

// NewPTYNavigationAdapter creates a new PTY navigation adapter
func NewPTYNavigationAdapter(
	ptyLayer pty.PTYLayer,
	navigationService *NavigationService,
	eventBus *events.EventBus,
	config PTYNavigationAdapterConfig,
) (*PTYNavigationAdapter, error) {
	if ptyLayer == nil {
		return nil, fmt.Errorf("PTY layer is required")
	}
	if navigationService == nil {
		return nil, fmt.Errorf("navigation service is required")
	}
	if eventBus == nil {
		return nil, fmt.Errorf("event bus is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &PTYNavigationAdapter{
		ptyLayer:          ptyLayer,
		navigationService: navigationService,
		eventBus:          eventBus,
		ctx:               ctx,
		cancel:            cancel,
		eventChan:         make(chan NavigationEvent, config.EventBufferSize),
		errorChan:         make(chan error, config.ErrorBufferSize),
		config:            config,
		switchTriggers:    make(map[string]int),
		numberKeyPresses:  make(map[int]int),
		viewHistory:       make([]messages.ViewType, 0, config.MaxViewHistory),
	}

	// Initialize atomic values
	adapter.lastActivity.Store(time.Time{})
	adapter.lastSwitchTime.Store(time.Time{})

	// Set initial view from navigation service
	if currentView, err := navigationService.GetCurrentView(); err == nil {
		adapter.currentView.Store(currentView)
		if config.TrackViewHistory {
			adapter.viewHistory = append(adapter.viewHistory, currentView)
		}
	}

	return adapter, nil
}

// Start begins the PTY navigation adapter and connects PTY input with NavigationService
func (a *PTYNavigationAdapter) Start() error {
	if a.active.Swap(true) {
		return fmt.Errorf("PTY navigation adapter is already active")
	}

	if !a.ptyLayer.IsAllocated() {
		return fmt.Errorf("PTY layer must be allocated before starting adapter")
	}

	// Create input adapter for PTY input handling
	inputConfig := DefaultPTYInputAdapterConfig()
	inputConfig.EnableDebug = a.config.EnableDebug
	inputConfig.ActivityTimeout = a.config.ActivityTimeout

	inputAdapter, err := NewPTYInputAdapter(a.ptyLayer, nil, inputConfig)
	if err != nil {
		a.active.Store(false)
		return fmt.Errorf("failed to create PTY input adapter: %w", err)
	}
	a.inputAdapter = inputAdapter

	// Start input adapter
	if err := a.inputAdapter.Start(); err != nil {
		a.active.Store(false)
		return fmt.Errorf("failed to start PTY input adapter: %w", err)
	}

	// Subscribe to navigation service events
	a.subscribeToNavigationEvents()

	// Start event processing through command pattern
	// TODO: Replace with proper command pattern for event processing
	// For now, disable event processing to avoid goroutines
	if a.config.EnableDebug {
		fmt.Printf("Event processing disabled for goroutine elimination\n")
	}

	// Start activity monitoring through command pattern
	// TODO: Replace with proper command pattern for activity monitoring
	// For now, disable activity monitoring to avoid goroutines
	if a.config.ActivityTimeout > 0 && a.config.EnableDebug {
		fmt.Printf("Activity monitoring disabled for goroutine elimination\n")
	}

	if a.config.EnableDebug {
		fmt.Printf("PTY navigation adapter started\n")
	}

	return nil
}

// Stop gracefully shuts down the PTY navigation adapter
func (a *PTYNavigationAdapter) Stop() error {
	if !a.active.Swap(false) {
		return nil // Already stopped
	}

	// Cancel context
	a.cancel()

	// Stop input adapter
	if a.inputAdapter != nil {
		if err := a.inputAdapter.Stop(); err != nil && a.config.EnableDebug {
			fmt.Printf("Warning: failed to stop PTY input adapter: %v\n", err)
		}
	}

	// Wait for operations to finish through command pattern
	// TODO: Replace with proper command pattern for shutdown coordination
	// For now, perform immediate shutdown to avoid goroutines
	if a.config.EnableDebug {
		fmt.Printf("Immediate shutdown for goroutine elimination\n")
	}

	// Close channels immediately
	close(a.eventChan)
	close(a.errorChan)

	if a.config.EnableDebug {
		fmt.Printf("PTY navigation adapter stopped\n")
	}

	return nil
}

// IsActive returns whether the adapter is currently active
func (a *PTYNavigationAdapter) IsActive() bool {
	return a.active.Load()
}

// SimulateKeyPress simulates a key press and processes it through the navigation system
func (a *PTYNavigationAdapter) SimulateKeyPress(key tea.KeyMsg) error {
	if !a.active.Load() {
		return fmt.Errorf("PTY navigation adapter is not active")
	}

	// Update activity time
	if a.config.EnableStatistics {
		a.lastActivity.Store(time.Now())
		a.keyPressCount.Add(1)
	}

	// Process the key through navigation logic
	if err := a.processNavigationKey(key); err != nil {
		a.errorCount.Add(1)
		return fmt.Errorf("failed to process navigation key: %w", err)
	}

	// Also send to PTY layer for realistic input simulation
	if a.inputAdapter != nil {
		if err := a.inputAdapter.SimulateKeyPress(key); err != nil && a.config.EnableDebug {
			fmt.Printf("Warning: failed to forward key to PTY: %v\n", err)
		}
	}

	return nil
}

// SimulateKeySequence simulates a sequence of key presses
func (a *PTYNavigationAdapter) SimulateKeySequence(keys []tea.KeyMsg) error {
	if !a.active.Load() {
		return fmt.Errorf("PTY navigation adapter is not active")
	}

	for i, key := range keys {
		if err := a.SimulateKeyPress(key); err != nil {
			return fmt.Errorf("failed to simulate key at index %d: %w", i, err)
		}
		// Small delay between keys for realistic simulation
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// GetCurrentView returns the current view from the navigation service
func (a *PTYNavigationAdapter) GetCurrentView() messages.ViewType {
	if view := a.currentView.Load(); view != nil {
		return view.(messages.ViewType)
	}
	return ""
}

// GetViewHistory returns the history of view switches
func (a *PTYNavigationAdapter) GetViewHistory() []messages.ViewType {
	a.mu.RLock()
	defer a.mu.RUnlock()

	history := make([]messages.ViewType, len(a.viewHistory))
	copy(history, a.viewHistory)
	return history
}

// GetSwitchCount returns the number of view switches recorded
func (a *PTYNavigationAdapter) GetSwitchCount() uint64 {
	return a.switchCount.Load()
}

// GetTabPresses returns the number of tab presses recorded
func (a *PTYNavigationAdapter) GetTabPresses() uint64 {
	return a.tabPresses.Load()
}

// GetNumberKeyPresses returns the count of presses for each number key
func (a *PTYNavigationAdapter) GetNumberKeyPresses() map[int]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	presses := make(map[int]int)
	for k, v := range a.numberKeyPresses {
		presses[k] = v
	}
	return presses
}

// GetLastSwitchTime returns the timestamp of the last view switch
func (a *PTYNavigationAdapter) GetLastSwitchTime() time.Time {
	if t := a.lastSwitchTime.Load(); t != nil {
		return t.(time.Time)
	}
	return time.Time{}
}

// WaitForViewSwitch waits for a specific number of view switches
func (a *PTYNavigationAdapter) WaitForViewSwitch(count uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if a.GetSwitchCount() >= count {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %d view switches (got %d)", count, a.GetSwitchCount())
}

// GetStatistics returns statistics about the adapter's operation
func (a *PTYNavigationAdapter) GetStatistics() PTYNavigationAdapterStats {
	stats := PTYNavigationAdapterStats{
		Active:          a.active.Load(),
		KeyPressCount:   a.keyPressCount.Load(),
		ErrorCount:      a.errorCount.Load(),
		SwitchCount:     a.switchCount.Load(),
		TabPresses:      a.tabPresses.Load(),
	}

	if lastActivity, ok := a.lastActivity.Load().(time.Time); ok {
		stats.LastActivity = lastActivity
	}

	if lastSwitchTime, ok := a.lastSwitchTime.Load().(time.Time); ok {
		stats.LastSwitchTime = lastSwitchTime
	}

	a.mu.RLock()
	stats.ViewHistoryLength = len(a.viewHistory)
	stats.NumberKeyPresses = make(map[int]int)
	for k, v := range a.numberKeyPresses {
		stats.NumberKeyPresses[k] = v
	}
	stats.SwitchTriggers = make(map[string]int)
	for k, v := range a.switchTriggers {
		stats.SwitchTriggers[k] = v
	}
	a.mu.RUnlock()

	return stats
}

// PTYNavigationAdapterStats contains statistics about the adapter's operation
type PTYNavigationAdapterStats struct {
	Active            bool
	KeyPressCount     uint64
	ErrorCount        uint64
	SwitchCount       uint64
	TabPresses        uint64
	LastActivity      time.Time
	LastSwitchTime    time.Time
	ViewHistoryLength int
	NumberKeyPresses  map[int]int
	SwitchTriggers    map[string]int
}

// processNavigationKey processes a key press for navigation
func (a *PTYNavigationAdapter) processNavigationKey(key tea.KeyMsg) error {
	var trigger string
	var targetView messages.ViewType
	var err error

	// Handle number keys (1-5)
	if key.Type == tea.KeyRunes && len(key.Runes) == 1 {
		r := key.Runes[0]
		if r >= '1' && r <= '5' {
			numKey := int(r - '0')
			a.mu.Lock()
			a.numberKeyPresses[numKey]++
			a.mu.Unlock()

			targetView, err = a.navigationService.GetViewForNumberKey(numKey)
			if err != nil {
				if a.config.ValidateNumberKeys {
					return fmt.Errorf("invalid number key %d: %w", numKey, err)
				}
				// If validation is disabled, ignore invalid number keys
				return nil
			}
			trigger = "number_key"

			// Switch view through navigation service
			if err := a.navigationService.SwitchToView(targetView, trigger); err != nil {
				return fmt.Errorf("failed to switch to view %s: %w", targetView, err)
			}

			a.recordNavigationEvent(NavigationEventNumberKey, targetView, trigger, key)
			return nil
		}
	}

	// Handle tab key for view cycling
	if key.Type == tea.KeyTab {
		a.tabPresses.Add(1)
		trigger = "cycle"

		// Cycle view through navigation service
		if err := a.navigationService.CycleToNextView(); err != nil {
			return fmt.Errorf("failed to cycle to next view: %w", err)
		}

		// Get the new view after cycling
		if currentView, err := a.navigationService.GetCurrentView(); err == nil {
			a.recordNavigationEvent(NavigationEventTabKey, currentView, trigger, key)
		}
		return nil
	}

	// Handle other navigation keys as needed
	// For now, we only process number keys and tab

	return nil
}

// recordNavigationEvent records a navigation event and updates state
func (a *PTYNavigationAdapter) recordNavigationEvent(eventType NavigationEventType, view messages.ViewType, trigger string, key tea.KeyMsg) {
	// Update current view
	a.currentView.Store(view)

	// Update switch count and time
	a.switchCount.Add(1)
	a.lastSwitchTime.Store(time.Now())

	// Update view history
	if a.config.TrackViewHistory {
		a.mu.Lock()
		a.viewHistory = append(a.viewHistory, view)
		// Limit history size
		if len(a.viewHistory) > a.config.MaxViewHistory {
			a.viewHistory = a.viewHistory[1:]
		}
		a.mu.Unlock()
	}

	// Record trigger
	if a.config.RecordTriggers && trigger != "" {
		a.mu.Lock()
		a.switchTriggers[trigger]++
		a.mu.Unlock()
	}

	// Publish navigation event
	select {
	case a.eventChan <- NavigationEvent{
		Type:      eventType,
		View:      view,
		Trigger:   trigger,
		Timestamp: time.Now(),
		Key:       key,
	}:
	default:
		// Event channel full, drop event
		a.errorCount.Add(1)
	}
}

// subscribeToNavigationEvents subscribes to navigation service events
func (a *PTYNavigationAdapter) subscribeToNavigationEvents() {
	// Subscribe to view changed events
	if a.eventBus != nil {
		a.eventBus.Subscribe("navigation.view.changed", func(event events.Event) {
			if !a.active.Load() {
				return
			}

			// Extract view changed message
			if msgData, ok := event.Data["message"]; ok {
				if msg, ok := msgData.(messages.ViewChangedMsg); ok {
					a.currentView.Store(msg.ToView)
					a.lastSwitchTime.Store(msg.Time)

					// Update view history
					if a.config.TrackViewHistory {
						a.mu.Lock()
						a.viewHistory = append(a.viewHistory, msg.ToView)
						if len(a.viewHistory) > a.config.MaxViewHistory {
							a.viewHistory = a.viewHistory[1:]
						}
						a.mu.Unlock()
					}

					// Record trigger
					if a.config.RecordTriggers && msg.Trigger != "" {
						a.mu.Lock()
						a.switchTriggers[msg.Trigger]++
						a.mu.Unlock()
					}

					a.switchCount.Add(1)
				}
			}
		})
	}
}

// eventProcessor processes navigation events
// TODO: Replace with proper command pattern for event processing
// For now, disable event processing to avoid goroutines
func (a *PTYNavigationAdapter) eventProcessor() {
	if a.config.EnableDebug {
		fmt.Printf("Event processing disabled for goroutine elimination\n")
	}
}

// activityMonitor monitors for inactivity and handles timeouts
// TODO: Replace with proper command pattern for activity monitoring
// For now, disable activity monitoring to avoid goroutines
func (a *PTYNavigationAdapter) activityMonitor() {
	if a.config.EnableDebug {
		fmt.Printf("Activity monitoring disabled for goroutine elimination\n")
	}
}

// GetPTYLayer returns the underlying PTY layer
func (a *PTYNavigationAdapter) GetPTYLayer() pty.PTYLayer {
	return a.ptyLayer
}

// GetNavigationService returns the underlying NavigationService
func (a *PTYNavigationAdapter) GetNavigationService() *NavigationService {
	return a.navigationService
}

// GetEventBus returns the underlying event bus
func (a *PTYNavigationAdapter) GetEventBus() *events.EventBus {
	return a.eventBus
}