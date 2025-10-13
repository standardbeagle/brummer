// Package tui provides PTY main wrapper for BubbleTea applications.
// This file contains the main wrapper that integrates all PTY components
// and provides a simple interface for running BubbleTea applications with PTY support.
package tui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// PTYMainWrapper is the main wrapper for PTY-enabled BubbleTea applications
type PTYMainWrapper struct {
	mu sync.RWMutex

	// Core components
	ptyIntegration      *PTYIntegration
	terminalEmulator    *PTYTerminalEmulator
	terminationHandler  *PTYTerminationHandler

	// Application state
	model              tea.Model
	program            *tea.Program
	ctx                context.Context
	cancel             context.CancelFunc
	running            bool
	started            bool

	// Configuration
	config             PTYMainWrapperConfig

	// Event handling
	startupCallbacks   []func() error
	shutdownCallbacks  []func() error
	errorCallbacks     []func(error)

	// Error tracking
	lastError          error
	errorCount         int

	// Performance monitoring
	startTime          time.Time
	launchTime         time.Time
	readyTime          time.Time
}

// PTYMainWrapperConfig contains configuration for the PTY main wrapper
type PTYMainWrapperConfig struct {
	// PTY configuration
	PTYIntegration     PTYIntegrationConfig
	TerminalEmulator   TerminalEmulatorConfig
	Termination       TerminationConfig

	// Application settings
	EnablePTY          bool
	FallbackToTTY      bool
	DetectTTY          bool

	// Performance settings
	StartupTimeout     time.Duration
	ReadyTimeout       time.Duration
	ShutdownTimeout    time.Duration

	// Features
	EnableTerminalEmulation bool
	EnableAdvancedHandling  bool
	EnablePerformanceMonitor bool

	// Debugging
	EnableDebug        bool
	LogStartupSteps    bool
	LogPerformance     bool
	TracePTYOperations bool

	// Platform-specific settings
	DisableOnWindows   bool // PTY support is limited on Windows
	ForcePTY           bool   // Force PTY even if TTY is available
}

// DefaultPTYMainWrapperConfig returns a default configuration
func DefaultPTYMainWrapperConfig() PTYMainWrapperConfig {
	return PTYMainWrapperConfig{
		PTYIntegration:   DefaultPTYIntegrationConfig(),
		TerminalEmulator: DefaultTerminalEmulatorConfig(),
		Termination:     DefaultTerminationConfig(),
		EnablePTY:       true,
		FallbackToTTY:   true,
		DetectTTY:       true,
		StartupTimeout:  30 * time.Second,
		ReadyTimeout:    10 * time.Second,
		ShutdownTimeout: 15 * time.Second,
		EnableTerminalEmulation: true,
		EnableAdvancedHandling:  true,
		EnablePerformanceMonitor: true,
		EnableDebug:     false,
		LogStartupSteps: false,
		LogPerformance:  false,
		TracePTYOperations: false,
		DisableOnWindows: true,
		ForcePTY:       false,
	}
}

// PTYMainWrapperResult contains the result of running a PTY-wrapped application
type PTYMainWrapperResult struct {
	Model              tea.Model
	Error              error
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
	WasPTYUsed         bool
	TerminationReason  ShutdownReason
	PerformanceStats   map[string]interface{}
}

// NewPTYMainWrapper creates a new PTY main wrapper
func NewPTYMainWrapper(config PTYMainWrapperConfig) *PTYMainWrapper {
	ctx, cancel := context.WithCancel(context.Background())

	wrapper := &PTYMainWrapper{
		ctx:                ctx,
		cancel:             cancel,
		config:             config,
		startupCallbacks:   make([]func() error, 0),
		shutdownCallbacks:  make([]func() error, 0),
		errorCallbacks:     make([]func(error), 0),
	}

	return wrapper
}

// ShouldUsePTY determines if PTY should be used based on configuration and environment
func (wrapper *PTYMainWrapper) ShouldUsePTY() bool {
	// Check if PTY is disabled
	if !wrapper.config.EnablePTY {
		if wrapper.config.EnableDebug {
			fmt.Printf("PTY disabled in configuration\n")
		}
		return false
	}

	// Check platform restrictions
	if wrapper.config.DisableOnWindows && runtime.GOOS == "windows" {
		if wrapper.config.EnableDebug {
			fmt.Printf("PTY disabled on Windows\n")
		}
		return false
	}

	// Force PTY if configured
	if wrapper.config.ForcePTY {
		if wrapper.config.EnableDebug {
			fmt.Printf("PTY forced by configuration\n")
		}
		return true
	}

	// Detect TTY availability if configured
	if wrapper.config.DetectTTY {
		if wrapper.isTTYAvailable() {
			if wrapper.config.EnableDebug {
				fmt.Printf("TTY available, not using PTY\n")
			}
			return false
		}
	}

	return true
}

// isTTYAvailable checks if a real TTY is available
func (wrapper *PTYMainWrapper) isTTYAvailable() bool {
	// Check if stdin is a terminal
	stdinStat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	stdoutStat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if both stdin and stdout are character devices
	isStdinTTY := (stdinStat.Mode() & os.ModeCharDevice) != 0
	isStdoutTTY := (stdoutStat.Mode() & os.ModeCharDevice) != 0

	return isStdinTTY && isStdoutTTY
}

// Initialize initializes the PTY components
func (wrapper *PTYMainWrapper) Initialize() error {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	if wrapper.started {
		return fmt.Errorf("wrapper already initialized")
	}

	usePTY := wrapper.ShouldUsePTY()

	if wrapper.config.LogStartupSteps {
		fmt.Printf("Initializing PTY main wrapper (PTY: %v)\n", usePTY)
	}

	wrapper.startTime = time.Now()

	if usePTY {
		// Initialize PTY integration
		ptyIntegration, err := NewPTYIntegration(wrapper.config.PTYIntegration)
		if err != nil {
			return fmt.Errorf("failed to create PTY integration: %w", err)
		}
		wrapper.ptyIntegration = ptyIntegration

		// Allocate the PTY - this is critical for it to work
		if err := ptyIntegration.Allocate(); err != nil {
			return fmt.Errorf("failed to allocate PTY: %w", err)
		}

		// Initialize terminal emulator if enabled
		if wrapper.config.EnableTerminalEmulation {
			terminalEmulator := NewPTYTerminalEmulator(
				ptyIntegration.GetPTYLayer(),
				wrapper.config.TerminalEmulator,
			)
			terminalEmulator.SetPTYIntegration(ptyIntegration)
			wrapper.terminalEmulator = terminalEmulator
		}

		// Initialize termination handler
		terminationHandler := NewPTYTerminationHandler(ptyIntegration, wrapper.config.Termination)
		wrapper.terminationHandler = terminationHandler

		// Add default callbacks
		wrapper.addDefaultCallbacks()

		if wrapper.config.LogStartupSteps {
			fmt.Printf("PTY components initialized and allocated\n")
		}
	}

	wrapper.started = true
	wrapper.launchTime = time.Now()

	if wrapper.config.LogPerformance {
		fmt.Printf("Initialization completed in %v\n", wrapper.launchTime.Sub(wrapper.startTime))
	}

	return nil
}

// addDefaultCallbacks adds default startup and shutdown callbacks
func (wrapper *PTYMainWrapper) addDefaultCallbacks() {
	// Add startup callback to set up the program
	if wrapper.terminationHandler != nil {
		wrapper.terminationHandler.AddTerminationCallback("setup_program", 100, func(reason ShutdownReason) error {
			if wrapper.program != nil {
				wrapper.program.Quit()
				wrapper.program.ReleaseTerminal()
			}
			return nil
		})
	}

	// Add cleanup callback for PTY resources
	if wrapper.terminationHandler != nil {
		wrapper.terminationHandler.AddCleanupCallback("pty_cleanup", 100, func() error {
			if wrapper.ptyIntegration != nil {
				return wrapper.ptyIntegration.Cleanup()
			}
			return nil
		})
	}
}

// SetModel sets the BubbleTea model to be run
func (wrapper *PTYMainWrapper) SetModel(model tea.Model) error {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	if !wrapper.started {
		return fmt.Errorf("wrapper not initialized")
	}

	wrapper.model = model

	// Set model in PTY integration if available
	if wrapper.ptyIntegration != nil {
		if err := wrapper.ptyIntegration.SetModel(model); err != nil {
			return fmt.Errorf("failed to set model in PTY integration: %w", err)
		}
	}

	return nil
}

// SetProgramOptions sets additional BubbleTea program options
func (wrapper *PTYMainWrapper) SetProgramOptions(options ...tea.ProgramOption) error {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	if !wrapper.started {
		return fmt.Errorf("wrapper not initialized")
	}

	// Set options in PTY integration if available
	if wrapper.ptyIntegration != nil {
		if err := wrapper.ptyIntegration.SetProgramOptions(options...); err != nil {
			return fmt.Errorf("failed to set program options in PTY integration: %w", err)
		}
	}

	return nil
}

// Run runs the BubbleTea application with PTY integration
func (wrapper *PTYMainWrapper) Run() PTYMainWrapperResult {
	startTime := time.Now()
	result := PTYMainWrapperResult{
		StartTime: startTime,
		WasPTYUsed: wrapper.ShouldUsePTY(),
		PerformanceStats: make(map[string]interface{}),
	}

	// Initialize if not already done
	if !wrapper.started {
		if err := wrapper.Initialize(); err != nil {
			result.Error = fmt.Errorf("initialization failed: %w", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
	}

	// Execute startup callbacks
	if err := wrapper.executeStartupCallbacks(); err != nil {
		result.Error = fmt.Errorf("startup callback failed: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	wrapper.mu.Lock()
	wrapper.running = true
	wrapper.mu.Unlock()

	defer func() {
		wrapper.mu.Lock()
		wrapper.running = false
		wrapper.mu.Unlock()
	}()

	// Run with PTY or fallback to regular BubbleTea
	if wrapper.ShouldUsePTY() && wrapper.ptyIntegration != nil {
		result = wrapper.runWithPTY(result)
	} else {
		result = wrapper.runWithoutPTY(result)
	}

	// Execute shutdown callbacks
	if err := wrapper.executeShutdownCallbacks(); err != nil && result.Error == nil {
		result.Error = fmt.Errorf("shutdown callback failed: %w", err)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Collect performance stats
	wrapper.collectPerformanceStats(&result)

	if wrapper.config.LogPerformance {
		fmt.Printf("Application ran for %v\n", result.Duration)
	}

	return result
}

// runWithPTY runs the application with PTY integration
func (wrapper *PTYMainWrapper) runWithPTY(result PTYMainWrapperResult) PTYMainWrapperResult {
	if wrapper.config.LogStartupSteps {
		fmt.Printf("Running with PTY integration\n")
	}

	wrapper.readyTime = time.Now()

	// Start termination handler
	if wrapper.terminationHandler != nil {
		if err := wrapper.terminationHandler.Start(); err != nil {
			result.Error = fmt.Errorf("failed to start termination handler: %w", err)
			return result
		}
		wrapper.terminationHandler.SetProgram(wrapper.program)
	}

	// Run the PTY program
	model, err := wrapper.ptyIntegration.Run()
	if err != nil {
		result.Error = fmt.Errorf("PTY program failed: %w", err)
		if wrapper.terminationHandler != nil {
			result.TerminationReason = wrapper.terminationHandler.GetShutdownReason()
		}
		return result
	}

	result.Model = model
	return result
}

// runWithoutPTY runs the application without PTY integration
func (wrapper *PTYMainWrapper) runWithoutPTY(result PTYMainWrapperResult) PTYMainWrapperResult {
	if wrapper.config.LogStartupSteps {
		fmt.Printf("Running without PTY integration\n")
	}

	wrapper.readyTime = time.Now()

	// Create standard BubbleTea program
	options := []tea.ProgramOption{
		tea.WithAltScreen(),
	}

	// Add signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run program in goroutine for signal handling
	done := make(chan tea.Model, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("program panic: %v", r)
			}
		}()

		program := tea.NewProgram(wrapper.model, options...)
		wrapper.program = program

		finalModel, err := program.Run()
		if err != nil {
			errChan <- err
			return
		}
		done <- finalModel
	}()

	// Wait for completion, error, or signal
	select {
	case model := <-done:
		result.Model = model
		result.TerminationReason = ReasonUserRequest

	case err := <-errChan:
		result.Error = err
		result.TerminationReason = ReasonError

	case sig := <-sigChan:
		// Handle signal
		if wrapper.program != nil {
			wrapper.program.Quit()
		}

		// Wait for graceful shutdown
		select {
		case model := <-done:
			result.Model = model
		case <-time.After(wrapper.config.ShutdownTimeout):
			result.Error = fmt.Errorf("shutdown timeout after signal %v", sig)
		}

		switch sig {
		case syscall.SIGINT:
			result.TerminationReason = ReasonSignal
		case syscall.SIGTERM:
			result.TerminationReason = ReasonSignal
		default:
			result.TerminationReason = ReasonParentTermination
		}
	}

	return result
}

// executeStartupCallbacks executes startup callbacks
func (wrapper *PTYMainWrapper) executeStartupCallbacks() error {
	for i, callback := range wrapper.startupCallbacks {
		if wrapper.config.LogStartupSteps {
			fmt.Printf("Executing startup callback %d\n", i)
		}

		if err := callback(); err != nil {
			wrapper.recordError(err)
			return fmt.Errorf("startup callback %d failed: %w", i, err)
		}
	}
	return nil
}

// executeShutdownCallbacks executes shutdown callbacks
func (wrapper *PTYMainWrapper) executeShutdownCallbacks() error {
	for i, callback := range wrapper.shutdownCallbacks {
		if wrapper.config.LogStartupSteps {
			fmt.Printf("Executing shutdown callback %d\n", i)
		}

		if err := callback(); err != nil {
			wrapper.recordError(err)
			// Continue executing other callbacks even if one fails
		}
	}
	return nil
}

// collectPerformanceStats collects performance statistics
func (wrapper *PTYMainWrapper) collectPerformanceStats(result *PTYMainWrapperResult) {
	stats := result.PerformanceStats

	stats["was_pty_used"] = result.WasPTYUsed
	stats["total_duration"] = result.Duration.String()

	if !wrapper.launchTime.IsZero() {
		stats["launch_duration"] = wrapper.launchTime.Sub(wrapper.startTime).String()
	}

	if !wrapper.readyTime.IsZero() {
		stats["ready_duration"] = wrapper.readyTime.Sub(wrapper.startTime).String()
	}

	stats["error_count"] = wrapper.errorCount

	if wrapper.lastError != nil {
		stats["last_error"] = wrapper.lastError.Error()
	}

	// Collect PTY-specific stats
	if wrapper.ptyIntegration != nil {
		stats["pty_allocated"] = wrapper.ptyIntegration.IsAllocated()
		stats["pty_running"] = wrapper.ptyIntegration.IsRunning()
	}

	// Collect terminal emulator stats
	if wrapper.terminalEmulator != nil {
		terminalStats := wrapper.terminalEmulator.GetPerformanceStats()
		for k, v := range terminalStats {
			stats["terminal_"+k] = v
		}
	}

	// Collect termination handler stats
	if wrapper.terminationHandler != nil {
		terminationStats := wrapper.terminationHandler.GetShutdownStats()
		for k, v := range terminationStats {
			stats["termination_"+k] = v
		}
	}
}

// recordError records an error
func (wrapper *PTYMainWrapper) recordError(err error) {
	wrapper.lastError = err
	wrapper.errorCount++

	// Execute error callbacks
	for _, callback := range wrapper.errorCallbacks {
		go callback(err)
	}
}

// AddStartupCallback adds a startup callback
func (wrapper *PTYMainWrapper) AddStartupCallback(callback func() error) {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()
	wrapper.startupCallbacks = append(wrapper.startupCallbacks, callback)
}

// AddShutdownCallback adds a shutdown callback
func (wrapper *PTYMainWrapper) AddShutdownCallback(callback func() error) {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()
	wrapper.shutdownCallbacks = append(wrapper.shutdownCallbacks, callback)
}

// AddErrorCallback adds an error callback
func (wrapper *PTYMainWrapper) AddErrorCallback(callback func(error)) {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()
	wrapper.errorCallbacks = append(wrapper.errorCallbacks, callback)
}

// GetPTYIntegration returns the PTY integration instance
func (wrapper *PTYMainWrapper) GetPTYIntegration() *PTYIntegration {
	wrapper.mu.RLock()
	defer wrapper.mu.RUnlock()
	return wrapper.ptyIntegration
}

// GetTerminalEmulator returns the terminal emulator instance
func (wrapper *PTYMainWrapper) GetTerminalEmulator() *PTYTerminalEmulator {
	wrapper.mu.RLock()
	defer wrapper.mu.RUnlock()
	return wrapper.terminalEmulator
}

// GetTerminationHandler returns the termination handler instance
func (wrapper *PTYMainWrapper) GetTerminationHandler() *PTYTerminationHandler {
	wrapper.mu.RLock()
	defer wrapper.mu.RUnlock()
	return wrapper.terminationHandler
}

// IsRunning returns true if the application is running
func (wrapper *PTYMainWrapper) IsRunning() bool {
	wrapper.mu.RLock()
	defer wrapper.mu.RUnlock()
	return wrapper.running
}

// GetLastError returns the last error that occurred
func (wrapper *PTYMainWrapper) GetLastError() error {
	wrapper.mu.RLock()
	defer wrapper.mu.RUnlock()
	return wrapper.lastError
}

// Shutdown initiates a graceful shutdown
func (wrapper *PTYMainWrapper) Shutdown() error {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	if !wrapper.running {
		return nil
	}

	if wrapper.terminationHandler != nil {
		return wrapper.terminationHandler.Shutdown(ReasonUserRequest)
	}

	if wrapper.program != nil {
		wrapper.program.Quit()
	}

	wrapper.cancel()
	return nil
}

// Cleanup cleans up all resources
func (wrapper *PTYMainWrapper) Cleanup() error {
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	// Cancel context
	wrapper.cancel()

	// Cleanup PTY integration
	if wrapper.ptyIntegration != nil {
		if err := wrapper.ptyIntegration.Cleanup(); err != nil {
			return err
		}
	}

	wrapper.running = false
	wrapper.started = false

	return nil
}

// RunWithPTY is a convenience function to run a BubbleTea model with PTY wrapper
func RunWithPTY(model tea.Model, config PTYMainWrapperConfig) PTYMainWrapperResult {
	wrapper := NewPTYMainWrapper(config)
	defer wrapper.Cleanup()

	// Initialize the wrapper first before setting model
	if err := wrapper.Initialize(); err != nil {
		return PTYMainWrapperResult{
			Error:     fmt.Errorf("failed to initialize PTY wrapper: %w", err),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	if err := wrapper.SetModel(model); err != nil {
		return PTYMainWrapperResult{
			Error:     fmt.Errorf("failed to set model: %w", err),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	return wrapper.Run()
}

// RunWithPTYAndOptions is a convenience function that accepts both model and program options
func RunWithPTYAndOptions(model tea.Model, config PTYMainWrapperConfig, options ...tea.ProgramOption) PTYMainWrapperResult {
	wrapper := NewPTYMainWrapper(config)
	defer wrapper.Cleanup()

	// Initialize the wrapper first before setting model and options
	if err := wrapper.Initialize(); err != nil {
		return PTYMainWrapperResult{
			Error:     fmt.Errorf("failed to initialize PTY wrapper: %w", err),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	if err := wrapper.SetModel(model); err != nil {
		return PTYMainWrapperResult{
			Error:     fmt.Errorf("failed to set model: %w", err),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	if err := wrapper.SetProgramOptions(options...); err != nil {
		return PTYMainWrapperResult{
			Error:     fmt.Errorf("failed to set program options: %w", err),
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	return wrapper.Run()
}

// AutoRunWithPTY automatically detects whether to use PTY and runs the application
func AutoRunWithPTY(model tea.Model, options ...tea.ProgramOption) PTYMainWrapperResult {
	config := DefaultPTYMainWrapperConfig()
	return RunWithPTYAndOptions(model, config, options...)
}