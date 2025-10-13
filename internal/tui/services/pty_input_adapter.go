package services

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// PTYInputAdapter connects PTY input simulation with the InputService
// to enable realistic keyboard input testing through the actual InputService
// used in production. This provides a bridge between the PTY layer and
// the InputService for testing environments.
type PTYInputAdapter struct {
	// Core components
	ptyLayer     *pty.PTYLayer
	inputService *InputService

	// State management
	ctx      context.Context
	cancel   context.CancelFunc
	shutdown atomic.Bool
	active   atomic.Bool

	// PTY monitoring
	ptyReader *PTYReader

	// Event handling
	eventChan chan tea.KeyMsg
	errorChan chan error

	// Synchronization
	mu sync.RWMutex
	wg sync.WaitGroup

	// Configuration
	config PTYInputAdapterConfig

	// Statistics
	keyPressCount   atomic.Uint64
	errorCount      atomic.Uint64
	lastActivity    atomic.Value // stores time.Time
}

// PTYInputAdapterConfig contains configuration for the PTY input adapter
type PTYInputAdapterConfig struct {
	// Buffer sizes
	EventBufferSize   int
	ErrorBufferSize   int

	// Timing and timeouts
	ReadTimeout       time.Duration
	ShutdownTimeout   time.Duration
	ActivityTimeout   time.Duration

	// Feature flags
	EnableStatistics  bool
	EnableDebug       bool
	BufferInput       bool
	ForwardUnknownKeys bool

	// PTY monitoring
	MonitorPTYOutput  bool
	PTYReadInterval   time.Duration
}

// DefaultPTYInputAdapterConfig returns a default configuration
func DefaultPTYInputAdapterConfig() PTYInputAdapterConfig {
	return PTYInputAdapterConfig{
		EventBufferSize:   1000,
		ErrorBufferSize:   100,
		ReadTimeout:       100 * time.Millisecond,
		ShutdownTimeout:   5 * time.Second,
		ActivityTimeout:   30 * time.Second,
		EnableStatistics:  true,
		EnableDebug:       false,
		BufferInput:       true,
		ForwardUnknownKeys: false,
		MonitorPTYOutput:  false,
		PTYReadInterval:   10 * time.Millisecond,
	}
}

// PTYReader monitors PTY output for keyboard input simulation
type PTYReader struct {
	ptyLayer     *pty.PTYLayer
	adapter      *PTYInputAdapter
	ctx          context.Context
	cancel       context.CancelFunc
	active       atomic.Bool
}

// NewPTYInputAdapter creates a new PTY input adapter
func NewPTYInputAdapter(ptyLayer *pty.PTYLayer, inputService *InputService, config PTYInputAdapterConfig) (*PTYInputAdapter, error) {
	if ptyLayer == nil {
		return nil, fmt.Errorf("PTY layer is required")
	}
	if inputService == nil {
		return nil, fmt.Errorf("input service is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &PTYInputAdapter{
		ptyLayer:     ptyLayer,
		inputService: inputService,
		ctx:          ctx,
		cancel:       cancel,
		eventChan:    make(chan tea.KeyMsg, config.EventBufferSize),
		errorChan:    make(chan error, config.ErrorBufferSize),
		config:       config,
	}

	// Initialize last activity time
	adapter.lastActivity.Store(time.Time{})

	return adapter, nil
}

// Start begins the PTY input adapter and connects the PTY layer with InputService
func (a *PTYInputAdapter) Start() error {
	if a.active.Swap(true) {
		return fmt.Errorf("PTY input adapter is already active")
	}

	if !a.ptyLayer.IsAllocated() {
		return fmt.Errorf("PTY layer must be allocated before starting adapter")
	}

	if a.inputService == nil {
		return fmt.Errorf("input service is not available")
	}

	// Create PTY reader for monitoring input
	a.ptyReader = &PTYReader{
		ptyLayer: a.ptyLayer,
		adapter:  a,
	}

	// Start PTY reader
	if err := a.ptyReader.Start(); err != nil {
		a.active.Store(false)
		return fmt.Errorf("failed to start PTY reader: %w", err)
	}

	// Start input processing goroutine
	a.wg.Add(1)
	go a.inputProcessor()

	// Start activity monitoring if enabled
	if a.config.ActivityTimeout > 0 {
		a.wg.Add(1)
		go a.activityMonitor()
	}

	if a.config.EnableDebug {
		fmt.Printf("PTY input adapter started\n")
	}

	return nil
}

// Stop gracefully shuts down the PTY input adapter
func (a *PTYInputAdapter) Stop() error {
	if !a.active.Swap(false) {
		return nil // Already stopped
	}

	// Cancel context
	a.cancel()

	// Stop PTY reader
	if a.ptyReader != nil {
		a.ptyReader.Stop()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Close channels
		close(a.eventChan)
		close(a.errorChan)
	case <-time.After(a.config.ShutdownTimeout):
		return fmt.Errorf("timeout waiting for PTY input adapter to shutdown")
	}

	if a.config.EnableDebug {
		fmt.Printf("PTY input adapter stopped\n")
	}

	return nil
}

// IsActive returns whether the adapter is currently active
func (a *PTYInputAdapter) IsActive() bool {
	return a.active.Load()
}

// SimulateKeyPress simulates a key press through the PTY layer
func (a *PTYInputAdapter) SimulateKeyPress(key tea.KeyMsg) error {
	if !a.active.Load() {
		return fmt.Errorf("PTY input adapter is not active")
	}

	// Update activity time
	if a.config.EnableStatistics {
		a.lastActivity.Store(time.Now())
		a.keyPressCount.Add(1)
	}

	// Route key through InputService for realistic testing
	if a.inputService != nil && a.inputService.IsRunning() {
		// Try to route through InputService first
		// Use a generic context - the actual context doesn't matter for key routing
		handled, _ := a.inputService.RouteKeyMessage(key, "pty_test")

		if a.config.EnableDebug {
			fmt.Printf("PTY adapter routed key %v through InputService, handled: %v\n", key, handled)
		}

		// If InputService handled it, we're done
		if handled {
			return nil
		}
	}

	// If InputService didn't handle it, forward to PTY layer
	if err := a.ptyLayer.SimulateKeyPress(key); err != nil {
		a.errorCount.Add(1)
		return fmt.Errorf("failed to simulate key press through PTY: %w", err)
	}

	return nil
}

// SimulateKeySequence simulates a sequence of key presses
func (a *PTYInputAdapter) SimulateKeySequence(keys []tea.KeyMsg) error {
	if !a.active.Load() {
		return fmt.Errorf("PTY input adapter is not active")
	}

	for i, key := range keys {
		if err := a.SimulateKeyPress(key); err != nil {
			return fmt.Errorf("failed to simulate key at index %d: %w", i, err)
		}
	}

	return nil
}

// GetPTYLayer returns the underlying PTY layer
func (a *PTYInputAdapter) GetPTYLayer() *pty.PTYLayer {
	return a.ptyLayer
}

// GetInputService returns the underlying InputService
func (a *PTYInputAdapter) GetInputService() *InputService {
	return a.inputService
}

// GetStatistics returns statistics about the adapter's operation
func (a *PTYInputAdapter) GetStatistics() PTYInputAdapterStats {
	stats := PTYInputAdapterStats{
		Active:        a.active.Load(),
		KeyPressCount: a.keyPressCount.Load(),
		ErrorCount:    a.errorCount.Load(),
	}

	if lastActivity, ok := a.lastActivity.Load().(time.Time); ok {
		stats.LastActivity = lastActivity
	}

	return stats
}

// PTYInputAdapterStats contains statistics about the adapter's operation
type PTYInputAdapterStats struct {
	Active        bool
	KeyPressCount uint64
	ErrorCount    uint64
	LastActivity  time.Time
}

// inputProcessor processes incoming key events from the PTY reader
func (a *PTYInputAdapter) inputProcessor() {
	defer a.wg.Done()

	for {
		select {
		case <-a.ctx.Done():
			return

		case keyMsg := <-a.eventChan:
			if !a.active.Load() {
				continue
			}

			// Update statistics
			if a.config.EnableStatistics {
				a.lastActivity.Store(time.Now())
				a.keyPressCount.Add(1)
			}

			// Route key through InputService
			if a.inputService != nil && a.inputService.IsRunning() {
				handled, _ := a.inputService.RouteKeyMessage(keyMsg, "pty_input")

				if a.config.EnableDebug {
					fmt.Printf("PTY adapter processed key %v from PTY, handled: %v\n", keyMsg, handled)
				}

				// If InputService didn't handle it and forwarding is enabled, send to PTY
				if !handled && a.config.ForwardUnknownKeys {
					if err := a.ptyLayer.SimulateKeyPress(keyMsg); err != nil {
						a.errorCount.Add(1)
						select {
						case a.errorChan <- fmt.Errorf("failed to forward key to PTY: %w", err):
						default:
							// Error channel full, drop error
						}
					}
				}
			}

		case err := <-a.errorChan:
			if a.config.EnableDebug {
				fmt.Printf("PTY input adapter error: %v\n", err)
			}
			a.errorCount.Add(1)
		}
	}
}

// activityMonitor monitors for inactivity and handles timeouts
func (a *PTYInputAdapter) activityMonitor() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.ActivityTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return

		case <-ticker.C:
			if !a.active.Load() {
				continue
			}

			lastActivity, ok := a.lastActivity.Load().(time.Time)
			if !ok {
				continue
			}

			if time.Since(lastActivity) > a.config.ActivityTimeout {
				if a.config.EnableDebug {
					fmt.Printf("PTY input adapter detected inactivity, last activity: %v\n", lastActivity)
				}
				// Could trigger cleanup or reconnection logic here if needed
			}
		}
	}
}

// PTYReader methods

// Start begins monitoring PTY for input
func (r *PTYReader) Start() error {
	if r.active.Swap(true) {
		return fmt.Errorf("PTY reader is already active")
	}

	ctx, cancel := context.WithCancel(r.adapter.ctx)
	r.ctx = ctx
	r.cancel = cancel

	// Start PTY monitoring goroutine
	go r.monitorPTYInput()

	return nil
}

// Stop stops PTY monitoring
func (r *PTYReader) Stop() {
	if !r.active.Swap(false) {
		return
	}

	if r.cancel != nil {
		r.cancel()
	}
}

// monitorPTYInput monitors the PTY for keyboard input
func (r *PTYReader) monitorPTYInput() {
	ticker := time.NewTicker(r.adapter.config.PTYReadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return

		case <-ticker.C:
			if !r.active.Load() {
				continue
			}

			// Try to read from PTY
			output, err := r.ptyLayer.ReadOutput()
			if err != nil {
				select {
				case r.adapter.errorChan <- fmt.Errorf("PTY read error: %w", err):
				default:
					// Error channel full, drop error
				}
				continue
			}

			// Process output for potential key sequences
			if len(output) > 0 {
				keys := r.parsePTYOutput(output)
				for _, key := range keys {
					select {
					case r.adapter.eventChan <- key:
					default:
						// Event channel full, drop key
						r.adapter.errorCount.Add(1)
					}
				}
			}
		}
	}
}

// parsePTYOutput attempts to parse PTY output into key messages
// This is a simplified implementation - in practice, you'd want more sophisticated
// terminal escape sequence parsing
func (r *PTYReader) parsePTYOutput(output []byte) []tea.KeyMsg {
	// This is a simplified parser - real terminal input would need
	// more sophisticated escape sequence handling
	var keys []tea.KeyMsg

	for _, b := range output {
		var keyMsg tea.KeyMsg

		switch b {
		case '\r':
			keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
		case '\n':
			keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
		case '\t':
			keyMsg = tea.KeyMsg{Type: tea.KeyTab}
		case '\x7f':
			keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
		case '\x03':
			keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
		case '\x04':
			keyMsg = tea.KeyMsg{Type: tea.KeyCtrlD}
		case '\x1a':
			keyMsg = tea.KeyMsg{Type: tea.KeyCtrlZ}
		case '\x1b':
			// Could be escape or start of escape sequence
			keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
		default:
			// Regular character
			if b >= 32 && b <= 126 {
				keyMsg = tea.KeyMsg{
					Type: tea.KeyRunes,
					Runes: []rune{rune(b)},
				}
			} else {
				// Unknown byte, skip unless forwarding is enabled
				continue
			}
		}

		keys = append(keys, keyMsg)
	}

	return keys
}