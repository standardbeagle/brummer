//go:build !windows

// Package tui provides PTY program termination handling for BubbleTea applications.
// This file contains comprehensive termination handling including graceful shutdown,
// signal handling, resource cleanup, and error recovery mechanisms.
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
)

// PTYTerminationHandler manages the termination lifecycle of PTY programs
type PTYTerminationHandler struct {
	mu sync.RWMutex

	// Core components
	ptyIntegration *PTYIntegration
	program        *tea.Program
	ctx            context.Context
	cancel         context.CancelFunc

	// Termination state
	state          TerminationState
	shutdownReason ShutdownReason
	shutdownStart  time.Time
	shutdownEnd    time.Time

	// Configuration
	config TerminationConfig

	// Signal handling
	sigChan        chan os.Signal
	signalHandlers map[syscall.Signal]SignalHandler

	// Termination callbacks
	terminationCallbacks []TerminationCallback
	cleanupCallbacks     []CleanupCallback

	// Error handling
	lastError     error
	errorHistory  []TerminationError
	maxErrorHistory int

	// Resource tracking
	resources     []TrackedResource
	resourceLock  sync.Mutex
}

// TerminationState represents the current termination state
type TerminationState int

const (
	StateRunning TerminationState = iota
	StateShuttingDown
	StateGracefulShutdown
	StateForcedShutdown
	StateTerminated
	StateError
)

// ShutdownReason represents why the program is shutting down
type ShutdownReason int

const (
	ReasonUserRequest ShutdownReason = iota
	ReasonSignal
	ReasonError
	ReasonTimeout
	ReasonResourceExhaustion
	ReasonParentTermination
)

// TerminationConfig contains configuration for termination handling
type TerminationConfig struct {
	// Timeouts
	GracefulShutdownTimeout time.Duration
	ForcedShutdownTimeout   time.Duration
	ResourceCleanupTimeout  time.Duration

	// Signal handling
	HandleSignals     bool
	PassthroughSignals bool
	HandledSignals    []syscall.Signal

	// Shutdown behavior
	EnableGracefulShutdown bool
	EnableForcedShutdown   bool
	ForceAfterTimeout      bool

	// Resource management
	TrackResources     bool
	CleanupOnExit      bool
	MaxResourceWait    time.Duration

	// Error handling
	RetryOnError       bool
	MaxRetryAttempts   int
	RetryDelay         time.Duration
	MaxErrorHistory    int

	// Debugging
	EnableDebug        bool
	LogShutdownSteps   bool
	CaptureStackTraces bool
}

// SignalHandler defines the interface for signal handlers
type SignalHandler func(sig syscall.Signal) error

// TerminationCallback represents a callback executed during termination
type TerminationCallback struct {
	Name     string
	Priority int
	Handler  func(ShutdownReason) error
}

// CleanupCallback represents a callback executed during cleanup
type CleanupCallback struct {
	Name     string
	Priority int
	Handler  func() error
}

// TrackedResource represents a resource that needs cleanup
type TrackedResource struct {
	Name       string
	Resource   interface{}
	Cleanup    func() error
	Registered time.Time
}

// TerminationError represents an error that occurred during termination
type TerminationError struct {
	Timestamp time.Time
	Error     error
	Context   string
	Signal    syscall.Signal
}

// DefaultTerminationConfig returns a default termination configuration
func DefaultTerminationConfig() TerminationConfig {
	return TerminationConfig{
		GracefulShutdownTimeout: 10 * time.Second,
		ForcedShutdownTimeout:   3 * time.Second,
		ResourceCleanupTimeout:  5 * time.Second,
		HandleSignals:           true,
		PassthroughSignals:      false,
		HandledSignals:          []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT},
		EnableGracefulShutdown:  true,
		EnableForcedShutdown:    true,
		ForceAfterTimeout:       true,
		TrackResources:          true,
		CleanupOnExit:           true,
		MaxResourceWait:         2 * time.Second,
		RetryOnError:            true,
		MaxRetryAttempts:        3,
		RetryDelay:              500 * time.Millisecond,
		MaxErrorHistory:         50,
		EnableDebug:             false,
		LogShutdownSteps:        true,
		CaptureStackTraces:      false,
	}
}

// NewPTYTerminationHandler creates a new PTY termination handler
func NewPTYTerminationHandler(ptyIntegration *PTYIntegration, config TerminationConfig) *PTYTerminationHandler {
	ctx, cancel := context.WithCancel(context.Background())

	th := &PTYTerminationHandler{
		ptyIntegration:       ptyIntegration,
		ctx:                ctx,
		cancel:             cancel,
		state:              StateRunning,
		shutdownReason:     ReasonUserRequest,
		config:             config,
		sigChan:           make(chan os.Signal, 1),
		signalHandlers:     make(map[syscall.Signal]SignalHandler),
		terminationCallbacks: make([]TerminationCallback, 0),
		cleanupCallbacks:     make([]CleanupCallback, 0),
		resources:          make([]TrackedResource, 0),
		errorHistory:       make([]TerminationError, 0),
		maxErrorHistory:    config.MaxErrorHistory,
	}

	// Setup default signal handlers
	if config.HandleSignals {
		th.setupDefaultSignalHandlers()
	}

	return th
}

// SetProgram sets the BubbleTea program to be managed
func (th *PTYTerminationHandler) SetProgram(program *tea.Program) {
	th.mu.Lock()
	defer th.mu.Unlock()
	th.program = program
}

// Start begins termination handling
func (th *PTYTerminationHandler) Start() error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.state != StateRunning {
		return fmt.Errorf("termination handler not in running state")
	}

	// Setup signal handling if enabled
	if th.config.HandleSignals {
		if err := th.setupSignalHandling(); err != nil {
			return fmt.Errorf("failed to setup signal handling: %w", err)
		}
	}

	// Start signal monitoring through command pattern
	// NOTE: Signal monitoring via goroutines has been disabled to prevent deadlocks
	// To re-enable: Convert signal handling to use tea.Cmd pattern with proper
	// lifecycle management. Each signal should trigger a command that:
	// 1. Checks if shutdown is already in progress
	// 2. Coordinates with main update loop via messages
	// 3. Uses timeout-based callbacks instead of blocking
	if th.config.EnableDebug {
		fmt.Printf("Signal monitoring disabled for goroutine elimination\n")
	}

	if th.config.EnableDebug {
		fmt.Printf("PTY termination handler started\n")
	}

	return nil
}

// Shutdown initiates a graceful shutdown
func (th *PTYTerminationHandler) Shutdown(reason ShutdownReason) error {
	return th.shutdownWithTimeout(reason, th.config.GracefulShutdownTimeout)
}

// ForceShutdown initiates an immediate forced shutdown
func (th *PTYTerminationHandler) ForceShutdown(reason ShutdownReason) error {
	return th.shutdownWithTimeout(reason, th.config.ForcedShutdownTimeout)
}

// shutdownWithTimeout performs shutdown with timeout
func (th *PTYTerminationHandler) shutdownWithTimeout(reason ShutdownReason, timeout time.Duration) error {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.state == StateTerminated || th.state == StateShuttingDown {
		return nil // Already shutting down or terminated
	}

	th.shutdownReason = reason
	th.shutdownStart = time.Now()
	th.state = StateShuttingDown

	if th.config.LogShutdownSteps {
		fmt.Printf("Starting shutdown (reason: %v, timeout: %v)\n", reason, timeout)
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(th.ctx, timeout)
	defer cancel()

	// Execute termination callbacks in priority order
	if err := th.executeTerminationCallbacks(shutdownCtx, reason); err != nil {
		th.recordError(fmt.Errorf("termination callback error: %w", err), "termination_callbacks")
		if th.config.ForceAfterTimeout {
			return th.forceTerminate()
		}
	}

	// Stop the PTY integration
	if th.ptyIntegration != nil {
		if err := th.ptyIntegration.Stop(); err != nil {
			th.recordError(fmt.Errorf("PTY integration stop error: %w", err), "pty_integration_stop")
		}
	}

	// Stop the BubbleTea program
	if th.program != nil {
		th.program.Quit()
		th.program.ReleaseTerminal()
	}

	// Cleanup resources
	if err := th.cleanupResources(shutdownCtx); err != nil {
		th.recordError(fmt.Errorf("resource cleanup error: %w", err), "resource_cleanup")
	}

	th.shutdownEnd = time.Now()
	th.state = StateTerminated

	// Cancel the main context
	th.cancel()

	// Execute cleanup callbacks
	th.executeCleanupCallbacks()

	if th.config.LogShutdownSteps {
		duration := th.shutdownEnd.Sub(th.shutdownStart)
		fmt.Printf("Shutdown completed in %v\n", duration)
	}

	return nil
}

// forceTerminate performs immediate forced termination
func (th *PTYTerminationHandler) forceTerminate() error {
	th.state = StateForcedShutdown

	if th.config.EnableDebug {
		fmt.Printf("Performing forced termination\n")
	}

	// Cancel context immediately
	th.cancel()

	// Force stop PTY integration
	if th.ptyIntegration != nil {
		// Use immediate cleanup
		_ = th.ptyIntegration.Cleanup()
	}

	// Force stop BubbleTea program
	if th.program != nil {
		th.program.Quit()
	}

	th.state = StateTerminated
	th.shutdownEnd = time.Now()

	return nil
}

// setupDefaultSignalHandlers sets up default signal handlers
func (th *PTYTerminationHandler) setupDefaultSignalHandlers() {
	// SIGINT (Ctrl+C) - graceful shutdown
	th.signalHandlers[syscall.SIGINT] = func(sig syscall.Signal) error {
		return th.Shutdown(ReasonSignal)
	}

	// SIGTERM - graceful shutdown
	th.signalHandlers[syscall.SIGTERM] = func(sig syscall.Signal) error {
		return th.Shutdown(ReasonSignal)
	}

	// SIGQUIT - forced shutdown
	th.signalHandlers[syscall.SIGQUIT] = func(sig syscall.Signal) error {
		return th.ForceShutdown(ReasonSignal)
	}
}

// setupSignalHandling sets up signal handling
func (th *PTYTerminationHandler) setupSignalHandling() error {
	if len(th.config.HandledSignals) == 0 {
		return nil
	}

	// Convert syscall.Signal to os.Signal for signal.Notify
	osSignals := make([]os.Signal, len(th.config.HandledSignals))
	for i, sig := range th.config.HandledSignals {
		osSignals[i] = sig
	}

	// Create signal channel with buffer for multiple signals
	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan, osSignals...)

	// Replace the internal signal channel
	th.sigChan = sigChan

	return nil
}

// monitorSignals monitors and handles incoming signals
// NOTE: Disabled to avoid goroutines and deadlocks
// IMPLEMENTATION REQUIRED:
// 1. Replace goroutine-based signal monitoring with tea.Cmd pattern
// 2. Signal handlers should return commands that send messages to main update loop
// 3. Main update loop processes signals and triggers appropriate shutdown
// 4. Use timeout-based coordination to prevent blocking
func (th *PTYTerminationHandler) monitorSignals() {
	if th.config.EnableDebug {
		fmt.Printf("Signal monitoring disabled for goroutine elimination\n")
	}
}

// executeTerminationCallbacks executes termination callbacks in priority order
func (th *PTYTerminationHandler) executeTerminationCallbacks(ctx context.Context, reason ShutdownReason) error {
	// Sort callbacks by priority (higher priority first)
	callbacks := make([]TerminationCallback, len(th.terminationCallbacks))
	copy(callbacks, th.terminationCallbacks)

	// Simple bubble sort by priority (descending)
	for i := 0; i < len(callbacks); i++ {
		for j := i + 1; j < len(callbacks); j++ {
			if callbacks[i].Priority < callbacks[j].Priority {
				callbacks[i], callbacks[j] = callbacks[j], callbacks[i]
			}
		}
	}

	// Execute callbacks with timeout
	for _, callback := range callbacks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if th.config.LogShutdownSteps {
				fmt.Printf("Executing termination callback: %s (priority: %d)\n", callback.Name, callback.Priority)
			}

			if err := callback.Handler(reason); err != nil {
				return fmt.Errorf("termination callback '%s' failed: %w", callback.Name, err)
			}
		}
	}

	return nil
}

// executeCleanupCallbacks executes cleanup callbacks
func (th *PTYTerminationHandler) executeCleanupCallbacks() {
	// Sort callbacks by priority (higher priority first)
	callbacks := make([]CleanupCallback, len(th.cleanupCallbacks))
	copy(callbacks, th.cleanupCallbacks)

	// Simple bubble sort by priority (descending)
	for i := 0; i < len(callbacks); i++ {
		for j := i + 1; j < len(callbacks); j++ {
			if callbacks[i].Priority < callbacks[j].Priority {
				callbacks[i], callbacks[j] = callbacks[j], callbacks[i]
			}
		}
	}

	// Execute callbacks (no timeout for cleanup)
	for _, callback := range callbacks {
		if th.config.LogShutdownSteps {
			fmt.Printf("Executing cleanup callback: %s (priority: %d)\n", callback.Name, callback.Priority)
		}

		if err := callback.Handler(); err != nil && th.config.EnableDebug {
			fmt.Printf("Cleanup callback '%s' failed: %v\n", callback.Name, err)
		}
	}
}

// cleanupResources cleans up tracked resources
func (th *PTYTerminationHandler) cleanupResources(ctx context.Context) error {
	if !th.config.TrackResources {
		return nil
	}

	th.resourceLock.Lock()
	defer th.resourceLock.Unlock()

	for _, resource := range th.resources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if th.config.LogShutdownSteps {
				fmt.Printf("Cleaning up resource: %s\n", resource.Name)
			}

			if err := resource.Cleanup(); err != nil {
				th.recordError(fmt.Errorf("resource cleanup error for '%s': %w", resource.Name, err), "resource_cleanup")
			}
		}
	}

	// Clear resources
	th.resources = th.resources[:0]

	return nil
}

// AddTerminationCallback adds a termination callback
func (th *PTYTerminationHandler) AddTerminationCallback(name string, priority int, handler func(ShutdownReason) error) {
	th.mu.Lock()
	defer th.mu.Unlock()

	th.terminationCallbacks = append(th.terminationCallbacks, TerminationCallback{
		Name:     name,
		Priority: priority,
		Handler:  handler,
	})
}

// AddCleanupCallback adds a cleanup callback
func (th *PTYTerminationHandler) AddCleanupCallback(name string, priority int, handler func() error) {
	th.mu.Lock()
	defer th.mu.Unlock()

	th.cleanupCallbacks = append(th.cleanupCallbacks, CleanupCallback{
		Name:     name,
		Priority: priority,
		Handler:  handler,
	})
}

// TrackResource tracks a resource that needs cleanup
func (th *PTYTerminationHandler) TrackResource(name string, resource interface{}, cleanup func() error) {
	if !th.config.TrackResources {
		return
	}

	th.resourceLock.Lock()
	defer th.resourceLock.Unlock()

	th.resources = append(th.resources, TrackedResource{
		Name:       name,
		Resource:   resource,
		Cleanup:    cleanup,
		Registered: time.Now(),
	})

	if th.config.EnableDebug {
		fmt.Printf("Tracking resource: %s (total: %d)\n", name, len(th.resources))
	}
}

// UntrackResource removes a resource from tracking
func (th *PTYTerminationHandler) UntrackResource(name string) {
	th.resourceLock.Lock()
	defer th.resourceLock.Unlock()

	for i, resource := range th.resources {
		if resource.Name == name {
			th.resources = append(th.resources[:i], th.resources[i+1:]...)
			if th.config.EnableDebug {
				fmt.Printf("Untracked resource: %s (remaining: %d)\n", name, len(th.resources))
			}
			return
		}
	}
}

// AddSignalHandler adds a custom signal handler
func (th *PTYTerminationHandler) AddSignalHandler(sig syscall.Signal, handler SignalHandler) {
	th.mu.Lock()
	defer th.mu.Unlock()

	th.signalHandlers[sig] = handler
}

// recordError records an error that occurred during termination
func (th *PTYTerminationHandler) recordError(err error, context string) {
	th.lastError = err

	errorRecord := TerminationError{
		Timestamp: time.Now(),
		Error:     err,
		Context:   context,
	}

	th.errorHistory = append(th.errorHistory, errorRecord)
	if len(th.errorHistory) > th.maxErrorHistory {
		th.errorHistory = th.errorHistory[1:]
	}

	if th.config.EnableDebug {
		fmt.Printf("Termination error recorded: %s - %v\n", context, err)
	}
}

// GetState returns the current termination state
func (th *PTYTerminationHandler) GetState() TerminationState {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.state
}

// GetShutdownReason returns the shutdown reason
func (th *PTYTerminationHandler) GetShutdownReason() ShutdownReason {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.shutdownReason
}

// GetLastError returns the last error that occurred
func (th *PTYTerminationHandler) GetLastError() error {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.lastError
}

// GetErrorHistory returns the error history
func (th *PTYTerminationHandler) GetErrorHistory() []TerminationError {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return append([]TerminationError{}, th.errorHistory...)
}

// GetTrackedResources returns tracked resources
func (th *PTYTerminationHandler) GetTrackedResources() []TrackedResource {
	th.resourceLock.Lock()
	defer th.resourceLock.Unlock()
	return append([]TrackedResource{}, th.resources...)
}

// GetShutdownStats returns shutdown statistics
func (th *PTYTerminationHandler) GetShutdownStats() map[string]interface{} {
	th.mu.RLock()
	defer th.mu.RUnlock()

	stats := map[string]interface{}{
		"state":             th.state.String(),
		"shutdown_reason":   th.shutdownReason.String(),
		"tracked_resources": len(th.resources),
		"error_count":       len(th.errorHistory),
		"callbacks": map[string]int{
			"termination": len(th.terminationCallbacks),
			"cleanup":     len(th.cleanupCallbacks),
		},
	}

	if !th.shutdownStart.IsZero() {
		if !th.shutdownEnd.IsZero() {
			stats["shutdown_duration"] = th.shutdownEnd.Sub(th.shutdownStart).String()
		} else {
			stats["shutdown_duration"] = time.Since(th.shutdownStart).String()
		}
	}

	if th.lastError != nil {
		stats["last_error"] = th.lastError.Error()
	}

	return stats
}

// IsShuttingDown returns true if the handler is shutting down
func (th *PTYTerminationHandler) IsShuttingDown() bool {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.state == StateShuttingDown || th.state == StateGracefulShutdown || th.state == StateForcedShutdown
}

// IsTerminated returns true if the handler has terminated
func (th *PTYTerminationHandler) IsTerminated() bool {
	th.mu.RLock()
	defer th.mu.RUnlock()
	return th.state == StateTerminated
}

// String methods for debugging
func (ts TerminationState) String() string {
	switch ts {
	case StateRunning:
		return "running"
	case StateShuttingDown:
		return "shutting_down"
	case StateGracefulShutdown:
		return "graceful_shutdown"
	case StateForcedShutdown:
		return "forced_shutdown"
	case StateTerminated:
		return "terminated"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

func (sr ShutdownReason) String() string {
	switch sr {
	case ReasonUserRequest:
		return "user_request"
	case ReasonSignal:
		return "signal"
	case ReasonError:
		return "error"
	case ReasonTimeout:
		return "timeout"
	case ReasonResourceExhaustion:
		return "resource_exhaustion"
	case ReasonParentTermination:
		return "parent_termination"
	default:
		return "unknown"
	}
}