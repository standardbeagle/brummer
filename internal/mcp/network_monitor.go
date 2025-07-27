package mcp

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

// NetworkState represents the current network connectivity state
type NetworkState int

const (
	NetworkStateUnknown NetworkState = iota
	NetworkStateConnected
	NetworkStateDisconnected
	NetworkStateSuspicious // Potentially interrupted
)

func (ns NetworkState) String() string {
	switch ns {
	case NetworkStateConnected:
		return "connected"
	case NetworkStateDisconnected:
		return "disconnected"
	case NetworkStateSuspicious:
		return "suspicious"
	default:
		return "unknown"
	}
}

// SleepWakeEvent represents system sleep/wake events
type SleepWakeEvent struct {
	Type      string // "sleep", "wake", or "suspected_wake"
	Timestamp time.Time
	Reason    string // Description of why this event was detected
}

// NetworkEvent represents a network connectivity change
type NetworkEvent struct {
	State     NetworkState
	Timestamp time.Time
	Reason    string
	Interface string // Interface that changed (if applicable)
}

// NetworkMonitor monitors network connectivity and system sleep/wake cycles
type NetworkMonitor struct {
	// Event channels
	networkEvents   chan NetworkEvent
	sleepWakeEvents chan SleepWakeEvent

	// Current state
	currentState     NetworkState
	lastStateChange  time.Time
	interfaceStates  map[string]net.Interface
	lastConnectTest  time.Time
	consecutiveTests int

	// Configuration
	connectivityTestInterval time.Duration
	interfaceCheckInterval   time.Duration
	testEndpoints            []string
	connectivityTimeout      time.Duration

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Platform-specific sleep/wake detection
	sleepWakeDetector SleepWakeDetector
}

// SleepWakeDetector interface for platform-specific implementations
type SleepWakeDetector interface {
	Start(ctx context.Context, events chan<- SleepWakeEvent) error
	Stop() error
}

// NewNetworkMonitor creates a new network monitor
func NewNetworkMonitor() *NetworkMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	nm := &NetworkMonitor{
		networkEvents:            make(chan NetworkEvent, 10),
		sleepWakeEvents:          make(chan SleepWakeEvent, 10),
		currentState:             NetworkStateUnknown,
		interfaceStates:          make(map[string]net.Interface),
		connectivityTestInterval: 10 * time.Second,
		interfaceCheckInterval:   5 * time.Second,
		connectivityTimeout:      5 * time.Second,
		ctx:                      ctx,
		cancel:                   cancel,
		testEndpoints: []string{
			"1.1.1.1:53",        // Cloudflare DNS
			"8.8.8.8:53",        // Google DNS
			"208.67.222.222:53", // OpenDNS
		},
	}

	// Initialize platform-specific sleep/wake detector
	nm.sleepWakeDetector = newPlatformSleepWakeDetector()

	return nm
}

// Start begins network monitoring
func (nm *NetworkMonitor) Start() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Start connectivity monitoring
	nm.wg.Add(1)
	go nm.monitorConnectivity()

	// Start interface monitoring
	nm.wg.Add(1)
	go nm.monitorInterfaces()

	// Start sleep/wake monitoring
	nm.wg.Add(1)
	go nm.monitorSleepWake()

	// Start sleep/wake detector if available
	if nm.sleepWakeDetector != nil {
		if err := nm.sleepWakeDetector.Start(nm.ctx, nm.sleepWakeEvents); err != nil {
			debugLog("Failed to start platform sleep/wake detector: %v", err)
		}
	}

	// Initial state check
	nm.wg.Add(1)
	go nm.initialStateCheck()

	debugLog("Network monitor started with %d test endpoints", len(nm.testEndpoints))
	return nil
}

// Stop gracefully stops network monitoring
func (nm *NetworkMonitor) Stop() error {
	nm.cancel()

	// Stop sleep/wake detector
	if nm.sleepWakeDetector != nil {
		nm.sleepWakeDetector.Stop()
	}

	nm.wg.Wait()

	// Close channels
	close(nm.networkEvents)
	close(nm.sleepWakeEvents)

	debugLog("Network monitor stopped")
	return nil
}

// NetworkEvents returns the channel for network connectivity events
func (nm *NetworkMonitor) NetworkEvents() <-chan NetworkEvent {
	return nm.networkEvents
}

// SleepWakeEvents returns the channel for sleep/wake events
func (nm *NetworkMonitor) SleepWakeEvents() <-chan SleepWakeEvent {
	return nm.sleepWakeEvents
}

// GetCurrentState returns the current network state
func (nm *NetworkMonitor) GetCurrentState() NetworkState {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.currentState
}

// monitorConnectivity periodically tests network connectivity
func (nm *NetworkMonitor) monitorConnectivity() {
	defer nm.wg.Done()

	ticker := time.NewTicker(nm.connectivityTestInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.checkConnectivity()
		}
	}
}

// monitorInterfaces monitors network interface changes
func (nm *NetworkMonitor) monitorInterfaces() {
	defer nm.wg.Done()

	ticker := time.NewTicker(nm.interfaceCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.checkInterfaces()
		}
	}
}

// monitorSleepWake handles sleep/wake events
func (nm *NetworkMonitor) monitorSleepWake() {
	defer nm.wg.Done()

	lastSeen := time.Now()
	checkInterval := 30 * time.Second

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return

		case <-ticker.C:
			now := time.Now()

			// Detect potential sleep by looking for time jumps
			// If more than 2x the check interval has passed, we might have slept
			expectedNext := lastSeen.Add(checkInterval)
			if now.After(expectedNext.Add(checkInterval)) {
				suspectedSleepDuration := now.Sub(expectedNext)

				// Only report if the gap is significant (more than 2 minutes)
				if suspectedSleepDuration > 2*time.Minute {
					select {
					case nm.sleepWakeEvents <- SleepWakeEvent{
						Type:      "suspected_wake",
						Timestamp: now,
						Reason:    fmt.Sprintf("Time jump detected: %v", suspectedSleepDuration),
					}:
					default:
					}
				}
			}

			lastSeen = now
		}
	}
}

// initialStateCheck performs an initial connectivity check
func (nm *NetworkMonitor) initialStateCheck() {
	defer nm.wg.Done()

	// Wait a moment for the system to stabilize
	select {
	case <-time.After(1 * time.Second):
		nm.checkConnectivity()
	case <-nm.ctx.Done():
		return
	}
}

// checkConnectivity tests network connectivity to multiple endpoints
func (nm *NetworkMonitor) checkConnectivity() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	connected := nm.testConnectivity()
	now := time.Now()

	previousState := nm.currentState
	var newState NetworkState

	if connected {
		nm.consecutiveTests = 0
		newState = NetworkStateConnected
	} else {
		nm.consecutiveTests++

		// Require multiple failures before declaring disconnected
		if nm.consecutiveTests >= 2 {
			newState = NetworkStateDisconnected
		} else {
			newState = NetworkStateSuspicious
		}
	}

	// Update state if changed
	if newState != previousState {
		nm.currentState = newState
		nm.lastStateChange = now

		reason := fmt.Sprintf("Connectivity test result: %v (consecutive failures: %d)",
			connected, nm.consecutiveTests)

		// Send network event
		select {
		case nm.networkEvents <- NetworkEvent{
			State:     newState,
			Timestamp: now,
			Reason:    reason,
		}:
		default:
			// Channel full, drop event
		}

		debugLog("Network state changed: %s -> %s (%s)",
			previousState, newState, reason)
	}

	nm.lastConnectTest = now
}

// testConnectivity tests connectivity to known endpoints
func (nm *NetworkMonitor) testConnectivity() bool {
	// Test multiple endpoints to be sure
	successCount := 0
	for _, endpoint := range nm.testEndpoints {
		if nm.testSingleEndpoint(endpoint) {
			successCount++
		}
	}

	// Consider connected if at least one endpoint is reachable
	return successCount > 0
}

// testSingleEndpoint tests connectivity to a single endpoint
func (nm *NetworkMonitor) testSingleEndpoint(endpoint string) bool {
	conn, err := net.DialTimeout("tcp", endpoint, nm.connectivityTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkInterfaces monitors network interface changes
func (nm *NetworkMonitor) checkInterfaces() {
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}

	nm.mu.Lock()
	defer nm.mu.Unlock()

	currentInterfaces := make(map[string]net.Interface)
	for _, iface := range interfaces {
		currentInterfaces[iface.Name] = iface
	}

	// Check for interface changes
	for name, currentIface := range currentInterfaces {
		if previousIface, exists := nm.interfaceStates[name]; exists {
			// Check if interface state changed
			if nm.interfaceStateChanged(previousIface, currentIface) {
				select {
				case nm.networkEvents <- NetworkEvent{
					State:     NetworkStateSuspicious, // Interface change suggests potential connectivity change
					Timestamp: time.Now(),
					Reason:    fmt.Sprintf("Interface %s state changed", name),
					Interface: name,
				}:
				default:
				}
			}
		}
	}

	// Check for removed interfaces
	for name := range nm.interfaceStates {
		if _, exists := currentInterfaces[name]; !exists {
			select {
			case nm.networkEvents <- NetworkEvent{
				State:     NetworkStateSuspicious,
				Timestamp: time.Now(),
				Reason:    fmt.Sprintf("Interface %s removed", name),
				Interface: name,
			}:
			default:
			}
		}
	}

	nm.interfaceStates = currentInterfaces
}

// interfaceStateChanged checks if an interface's state has changed significantly
func (nm *NetworkMonitor) interfaceStateChanged(old, new net.Interface) bool {
	// Check flags for significant changes
	return old.Flags != new.Flags
}

// GetStats returns network monitoring statistics
func (nm *NetworkMonitor) GetStats() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	return map[string]interface{}{
		"currentState":     nm.currentState.String(),
		"lastStateChange":  nm.lastStateChange,
		"lastConnectTest":  nm.lastConnectTest,
		"consecutiveTests": nm.consecutiveTests,
		"interfaceCount":   len(nm.interfaceStates),
		"testEndpoints":    nm.testEndpoints,
		"platform":         runtime.GOOS,
	}
}

// Platform-specific sleep/wake detection implementations

// newPlatformSleepWakeDetector creates a platform-specific sleep/wake detector
func newPlatformSleepWakeDetector() SleepWakeDetector {
	switch runtime.GOOS {
	case "darwin":
		return &macOSSleepWakeDetector{}
	case "windows":
		return &windowsSleepWakeDetector{}
	case "linux":
		return &linuxSleepWakeDetector{}
	default:
		return &genericSleepWakeDetector{}
	}
}

// Generic sleep/wake detector (fallback for unsupported platforms)
type genericSleepWakeDetector struct{}

func (d *genericSleepWakeDetector) Start(ctx context.Context, events chan<- SleepWakeEvent) error {
	// Generic implementation using time-based detection only
	// This is handled by the main monitorSleepWake function
	return nil
}

func (d *genericSleepWakeDetector) Stop() error {
	return nil
}

// macOS-specific sleep/wake detector
type macOSSleepWakeDetector struct{}

func (d *macOSSleepWakeDetector) Start(ctx context.Context, events chan<- SleepWakeEvent) error {
	// macOS implementation would use IOKit notifications
	// For now, fall back to generic time-based detection
	debugLog("macOS sleep/wake detection not fully implemented, using generic detection")
	return nil
}

func (d *macOSSleepWakeDetector) Stop() error {
	return nil
}

// Windows-specific sleep/wake detector
type windowsSleepWakeDetector struct{}

func (d *windowsSleepWakeDetector) Start(ctx context.Context, events chan<- SleepWakeEvent) error {
	// Windows implementation would use WM_POWERBROADCAST messages
	// For now, fall back to generic time-based detection
	debugLog("Windows sleep/wake detection not fully implemented, using generic detection")
	return nil
}

func (d *windowsSleepWakeDetector) Stop() error {
	return nil
}

// Linux-specific sleep/wake detector
type linuxSleepWakeDetector struct{}

func (d *linuxSleepWakeDetector) Start(ctx context.Context, events chan<- SleepWakeEvent) error {
	// Linux implementation would monitor D-Bus signals from systemd/logind
	// For now, fall back to generic time-based detection
	debugLog("Linux sleep/wake detection not fully implemented, using generic detection")
	return nil
}

func (d *linuxSleepWakeDetector) Stop() error {
	return nil
}
