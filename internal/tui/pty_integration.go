// Package tui provides PTY integration for BubbleTea applications.
// This file contains PTY-based program startup and management functionality
// that enables realistic terminal simulation without requiring /dev/tty.
package tui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/test/tui/pty"
)

// PTYIntegration provides PTY-based program management for BubbleTea applications
type PTYIntegration struct {
	mu              sync.RWMutex
	ptyLayer        *pty.PTYLayerImpl
	ptyProgram      *pty.PTYProgram
	inputController *pty.PTYInputController
	renderer        *pty.PTYRenderer
	bridge          *pty.PTYBridge

	// Program state
	program    *tea.Program
	model      tea.Model
	options    []tea.ProgramOption
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	allocated  bool

	// Configuration
	config PTYIntegrationConfig

	// Event handling
	startupCallbacks []func()
	shutdownCallbacks []func()
	errorCallbacks    []func(error)

	// Terminal control
	initialSize pty.PTYSize
	currentSize pty.PTYSize
}

// PTYIntegrationConfig contains configuration for PTY integration
type PTYIntegrationConfig struct {
	// Terminal dimensions
	InitialCols uint16
	InitialRows uint16

	// Timeouts and delays
	AllocationTimeout time.Duration
	InputDelay        time.Duration
	ShutdownTimeout   time.Duration

	// Features
	EnableRenderer     bool
	EnableInputController bool
	EnableBridge       bool
	EnableAutoResize   bool

	// Debugging
	EnableDebug        bool
	CaptureInput       bool
	CaptureOutput      bool
	MaxInputHistory    int
	MaxOutputBuffer    int

	// Signal handling
	HandleSignals      bool
	PassthroughSignals bool
}

// DefaultPTYIntegrationConfig returns a default configuration for PTY integration
func DefaultPTYIntegrationConfig() PTYIntegrationConfig {
	return PTYIntegrationConfig{
		InitialCols:         80,
		InitialRows:         24,
		AllocationTimeout:   10 * time.Second,
		InputDelay:          50 * time.Millisecond,
		ShutdownTimeout:     5 * time.Second,
		EnableRenderer:      true,
		EnableInputController: true,
		EnableBridge:        true,
		EnableAutoResize:    false,
		EnableDebug:         false,
		CaptureInput:        true,
		CaptureOutput:       true,
		MaxInputHistory:     1000,
		MaxOutputBuffer:     4096,
		HandleSignals:       true,
		PassthroughSignals:  false,
	}
}

// NewPTYIntegration creates a new PTY integration instance
func NewPTYIntegration(config PTYIntegrationConfig) (*PTYIntegration, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ptyConfig := pty.PTYConfig{
		Size: pty.PTYSize{
			Rows: config.InitialRows,
			Cols: config.InitialCols,
		},
		Timeout:     config.AllocationTimeout,
		EnableDebug: config.EnableDebug,
	}

	ptyLayer, err := pty.NewPTYLayer(ptyConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create PTY layer: %w", err)
	}

	return &PTYIntegration{
		ctx:     ctx,
		cancel:  cancel,
		ptyLayer: ptyLayer,
		config:  config,
		initialSize: pty.PTYSize{
			Rows: config.InitialRows,
			Cols: config.InitialCols,
		},
		currentSize: pty.PTYSize{
			Rows: config.InitialRows,
			Cols: config.InitialCols,
		},
		startupCallbacks:  make([]func(), 0),
		shutdownCallbacks: make([]func(), 0),
		errorCallbacks:    make([]func(error), 0),
	}, nil
}

// Allocate allocates the PTY and initializes components
func (pi *PTYIntegration) Allocate() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if pi.allocated {
		return fmt.Errorf("PTY already allocated")
	}

	// Allocate the PTY layer
	if err := pi.ptyLayer.Allocate(); err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	// Create PTY program wrapper
	pi.ptyProgram = pty.NewPTYProgram(nil, pi.ptyLayer) // Model will be set later

	// Create input controller if enabled
	if pi.config.EnableInputController {
		pi.inputController = pty.NewPTYInputController(pi.ptyLayer)
	}

	// Create renderer if enabled
	if pi.config.EnableRenderer {
		pi.renderer = pty.NewPTYRenderer(pi.ptyLayer)
	}

	// Create bridge if enabled
	if pi.config.EnableBridge {
		pi.bridge = pty.NewPTYBridge(pi.ptyLayer)
	}

	pi.allocated = true

	// Log allocation if debug is enabled
	if pi.config.EnableDebug {
		fmt.Printf("PTY allocated: %dx%d\n", pi.config.InitialCols, pi.config.InitialRows)
	}

	return nil
}

// IsAllocated returns true if the PTY is allocated
func (pi *PTYIntegration) IsAllocated() bool {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.allocated
}

// SetModel sets the BubbleTea model for the program
func (pi *PTYIntegration) SetModel(model tea.Model) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	pi.model = model

	// Update PTY program with the model
	if pi.ptyProgram != nil {
		// This is a bit of a hack since the PTYProgram doesn't have a SetModel method
		// We'll need to recreate it or modify the PTYProgram to support model updates
		pi.ptyProgram = pty.NewPTYProgram(model, pi.ptyLayer)
	}

	return nil
}

// SetProgramOptions sets additional BubbleTea program options
func (pi *PTYIntegration) SetProgramOptions(options ...tea.ProgramOption) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	pi.options = append(pi.options, options...)
	return nil
}

// Start starts the PTY-integrated BubbleTea program
func (pi *PTYIntegration) Start() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if pi.running {
		return fmt.Errorf("program already running")
	}

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	if pi.model == nil {
		return fmt.Errorf("model not set")
	}

	// Setup signal handling if enabled
	if pi.config.HandleSignals {
		if err := pi.setupSignalHandling(); err != nil {
			return fmt.Errorf("failed to setup signal handling: %w", err)
		}
	}

	// Start components
	if pi.inputController != nil {
		pi.inputController.Start()
	}

	if pi.renderer != nil {
		pi.renderer.Start()
	}

	if pi.bridge != nil {
		pi.bridge.Start()
	}

	// Create the actual BubbleTea program with PTY integration
	ptyOptions := []tea.ProgramOption{
		tea.WithInput(pi.ptyLayer),
		tea.WithOutput(pi.ptyLayer),
		tea.WithoutSignalHandler(), // We handle signals ourselves
		tea.WithContext(pi.ctx),
	}

	// Add user-provided options
	ptyOptions = append(ptyOptions, pi.options...)

	pi.program = tea.NewProgram(pi.model, ptyOptions...)

	// Start the PTY program
	if err := pi.ptyProgram.Start(); err != nil {
		return fmt.Errorf("failed to start PTY program: %w", err)
	}

	pi.running = true

	// Execute startup callbacks
	for _, callback := range pi.startupCallbacks {
		go callback()
	}

	if pi.config.EnableDebug {
		fmt.Printf("PTY program started\n")
	}

	return nil
}

// Run starts the program and waits for it to complete
func (pi *PTYIntegration) Run() (tea.Model, error) {
	if err := pi.Start(); err != nil {
		return nil, err
	}

	defer pi.Stop()

	// Run the program in a goroutine
	done := make(chan tea.Model, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("program panic: %v", r)
			}
		}()

		// Run the PTY program to completion
		if _, ptyErr := pi.ptyProgram.Run(); ptyErr != nil {
			errChan <- fmt.Errorf("PTY program error: %w", ptyErr)
			return
		}

		// Get the final model
		finalModel := pi.ptyProgram.GetModel()
		done <- finalModel
	}()

	// Wait for completion or error
	select {
	case model := <-done:
		return model, nil
	case err := <-errChan:
		return nil, err
	case <-pi.ctx.Done():
		return nil, pi.ctx.Err()
	}
}

// Stop stops the PTY program and cleans up resources
func (pi *PTYIntegration) Stop() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if !pi.running {
		return nil
	}

	// Set a timeout for shutdown
	_, cancel := context.WithTimeout(context.Background(), pi.config.ShutdownTimeout)
	defer cancel()

	// Stop components in reverse order
	if pi.bridge != nil {
		pi.bridge.Stop()
	}

	if pi.renderer != nil {
		pi.renderer.Stop()
	}

	if pi.inputController != nil {
		pi.inputController.Stop()
	}

	// Stop the PTY program
	if pi.ptyProgram != nil {
		if err := pi.ptyProgram.Stop(); err != nil {
			if pi.config.EnableDebug {
				fmt.Printf("Warning: failed to stop PTY program: %v\n", err)
			}
		}
	}

	// Stop the BubbleTea program
	if pi.program != nil {
		pi.program.Quit()
		pi.program.ReleaseTerminal()
	}

	// Cancel context
	pi.cancel()

	pi.running = false

	// Execute shutdown callbacks
	for _, callback := range pi.shutdownCallbacks {
		go callback()
	}

	if pi.config.EnableDebug {
		fmt.Printf("PTY program stopped\n")
	}

	return nil
}

// IsRunning returns true if the program is running
func (pi *PTYIntegration) IsRunning() bool {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.running
}

// GetModel returns the current model
func (pi *PTYIntegration) GetModel() tea.Model {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	if pi.ptyProgram != nil {
		return pi.ptyProgram.GetModel()
	}
	return pi.model
}

// Resize changes the terminal size
func (pi *PTYIntegration) Resize(cols, rows uint16) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	newSize := pty.PTYSize{
		Rows: rows,
		Cols: cols,
	}

	if err := pi.ptyLayer.SetSize(newSize); err != nil {
		return fmt.Errorf("failed to resize PTY: %w", err)
	}

	pi.currentSize = newSize

	if pi.config.EnableDebug {
		fmt.Printf("PTY resized to %dx%d\n", cols, rows)
	}

	return nil
}

// GetSize returns the current terminal size
func (pi *PTYIntegration) GetSize() (cols, rows uint16) {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.currentSize.Cols, pi.currentSize.Rows
}

// SimulateInput simulates keyboard input through the PTY
func (pi *PTYIntegration) SimulateInput(input string) error {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	if !pi.running {
		return fmt.Errorf("program not running")
	}

	// Convert string to bytes and write to PTY
	if err := pi.ptyLayer.WriteInput([]byte(input)); err != nil {
		return fmt.Errorf("failed to write input to PTY: %w", err)
	}

	return nil
}

// SimulateKeyPress simulates a single key press
func (pi *PTYIntegration) SimulateKeyPress(key tea.KeyMsg) error {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	if !pi.running {
		return fmt.Errorf("program not running")
	}

	if pi.inputController != nil {
		return pi.inputController.SendKey(key)
	}

	// Fallback to direct PTY input
	return pi.ptyLayer.SimulateKeyPress(key)
}

// GetOutput returns captured output from the renderer
func (pi *PTYIntegration) GetOutput() string {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	if pi.renderer != nil {
		return pi.renderer.GetContent()
	}

	// Fallback to PTY layer output
	return pi.ptyLayer.GetContent()
}

// GetInputHistory returns input history if available
func (pi *PTYIntegration) GetInputHistory() []byte {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	return pi.ptyLayer.GetInputBuffer()
}

// ClearBuffers clears input and output buffers
func (pi *PTYIntegration) ClearBuffers() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if !pi.allocated {
		return fmt.Errorf("PTY not allocated")
	}

	pi.ptyLayer.ClearBuffers()

	if pi.renderer != nil {
		pi.renderer.ClearBuffer()
	}

	return nil
}

// AddStartupCallback adds a callback to be executed on startup
func (pi *PTYIntegration) AddStartupCallback(callback func()) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.startupCallbacks = append(pi.startupCallbacks, callback)
}

// AddShutdownCallback adds a callback to be executed on shutdown
func (pi *PTYIntegration) AddShutdownCallback(callback func()) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.shutdownCallbacks = append(pi.shutdownCallbacks, callback)
}

// AddErrorCallback adds a callback to be executed on errors
func (pi *PTYIntegration) AddErrorCallback(callback func(error)) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.errorCallbacks = append(pi.errorCallbacks, callback)
}

// setupSignalHandling sets up signal handling for the PTY program
func (pi *PTYIntegration) setupSignalHandling() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		select {
		case sig := <-sigChan:
			if pi.config.EnableDebug {
				fmt.Printf("Received signal: %v\n", sig)
			}

			// Handle signal based on configuration
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				go pi.Stop()
			case syscall.SIGQUIT:
				// Immediate shutdown
				pi.cancel()
			}

			// Passthrough to PTY if enabled
			if pi.config.PassthroughSignals {
				// Convert signal to PTY input if needed
				// This is platform-specific and would need implementation
			}

		case <-pi.ctx.Done():
			return
		}
	}()

	return nil
}

// Cleanup cleans up all PTY resources
func (pi *PTYIntegration) Cleanup() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	// Stop program if running
	if pi.running {
		pi.Stop() // Ignore errors during cleanup
	}

	// Cleanup PTY layer
	if pi.ptyLayer != nil {
		if err := pi.ptyLayer.Cleanup(); err != nil && pi.config.EnableDebug {
			fmt.Printf("Warning: PTY cleanup error: %v\n", err)
		}
	}

	pi.allocated = false
	pi.running = false

	return nil
}

// GetContext returns the PTY context
func (pi *PTYIntegration) GetContext() context.Context {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.ctx
}

// GetPTYLayer returns the underlying PTY layer for advanced operations
func (pi *PTYIntegration) GetPTYLayer() *pty.PTYLayerImpl {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.ptyLayer
}

// GetPTYProgram returns the PTY program wrapper
func (pi *PTYIntegration) GetPTYProgram() *pty.PTYProgram {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.ptyProgram
}

// GetRenderer returns the PTY renderer if enabled
func (pi *PTYIntegration) GetRenderer() *pty.PTYRenderer {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.renderer
}

// GetInputController returns the PTY input controller if enabled
func (pi *PTYIntegration) GetInputController() *pty.PTYInputController {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.inputController
}

// GetBridge returns the PTY bridge if enabled
func (pi *PTYIntegration) GetBridge() *pty.PTYBridge {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.bridge
}

// RunWithPTYIntegration is a convenience function to run a BubbleTea model with PTY integration
func RunWithPTYIntegration(model tea.Model, config PTYIntegrationConfig) (tea.Model, error) {
	// Create PTY integration
	ptyInt, err := NewPTYIntegration(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY integration: %w", err)
	}
	defer ptyInt.Cleanup()

	// Allocate PTY
	if err := ptyInt.Allocate(); err != nil {
		return nil, fmt.Errorf("failed to allocate PTY: %w", err)
	}

	// Set model
	if err := ptyInt.SetModel(model); err != nil {
		return nil, fmt.Errorf("failed to set model: %w", err)
	}

	// Run the program
	return ptyInt.Run()
}

// RunWithPTYIntegrationAndOptions is a convenience function that accepts both model and program options
func RunWithPTYIntegrationAndOptions(model tea.Model, config PTYIntegrationConfig, options ...tea.ProgramOption) (tea.Model, error) {
	// Create PTY integration
	ptyInt, err := NewPTYIntegration(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create PTY integration: %w", err)
	}
	defer ptyInt.Cleanup()

	// Allocate PTY
	if err := ptyInt.Allocate(); err != nil {
		return nil, fmt.Errorf("failed to allocate PTY: %w", err)
	}

	// Set model
	if err := ptyInt.SetModel(model); err != nil {
		return nil, fmt.Errorf("failed to set model: %w", err)
	}

	// Set program options
	if err := ptyInt.SetProgramOptions(options...); err != nil {
		return nil, fmt.Errorf("failed to set program options: %w", err)
	}

	// Run the program
	return ptyInt.Run()
}