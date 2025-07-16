package mcp

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNetworkMonitorCreation(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	if nm == nil {
		t.Fatal("NetworkMonitor should not be nil")
	}

	if nm.currentState != NetworkStateUnknown {
		t.Errorf("Initial state should be Unknown, got %v", nm.currentState)
	}

	if len(nm.testEndpoints) == 0 {
		t.Error("Test endpoints should be configured")
	}
}

func TestNetworkMonitorLifecycle(t *testing.T) {
	nm := NewNetworkMonitor()

	// Test start
	err := nm.Start()
	if err != nil {
		t.Errorf("Start should not return error: %v", err)
	}

	// Give monitor time to initialize
	time.Sleep(100 * time.Millisecond)

	// Test state access
	state := nm.GetCurrentState()
	if state == NetworkStateUnknown {
		// State might still be unknown if connectivity test hasn't completed
		t.Logf("Network state is still unknown: %v", state)
	}

	// Test stop
	err = nm.Stop()
	if err != nil {
		t.Errorf("Stop should not return error: %v", err)
	}
}

func TestNetworkEventChannels(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	// Test that channels are available
	networkEvents := nm.NetworkEvents()
	if networkEvents == nil {
		t.Error("Network events channel should not be nil")
	}

	sleepWakeEvents := nm.SleepWakeEvents()
	if sleepWakeEvents == nil {
		t.Error("Sleep/wake events channel should not be nil")
	}

	err := nm.Start()
	if err != nil {
		t.Errorf("Start should not return error: %v", err)
	}

	// Wait for potential events (short timeout for testing)
	select {
	case event := <-networkEvents:
		t.Logf("Received network event: %+v", event)
	case <-time.After(2 * time.Second):
		t.Log("No network events received (this is normal for stable networks)")
	}
}

func TestNetworkStateStrings(t *testing.T) {
	tests := []struct {
		state    NetworkState
		expected string
	}{
		{NetworkStateUnknown, "unknown"},
		{NetworkStateConnected, "connected"},
		{NetworkStateDisconnected, "disconnected"},
		{NetworkStateSuspicious, "suspicious"},
	}

	for _, test := range tests {
		result := test.state.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestSleepWakeEvent(t *testing.T) {
	event := SleepWakeEvent{
		Type:      "wake",
		Timestamp: time.Now(),
		Reason:    "Test event",
	}

	if event.Type != "wake" {
		t.Errorf("Expected type 'wake', got %s", event.Type)
	}

	if event.Reason != "Test event" {
		t.Errorf("Expected reason 'Test event', got %s", event.Reason)
	}
}

func TestNetworkEvent(t *testing.T) {
	event := NetworkEvent{
		State:     NetworkStateConnected,
		Timestamp: time.Now(),
		Reason:    "Test connectivity",
		Interface: "eth0",
	}

	if event.State != NetworkStateConnected {
		t.Errorf("Expected state Connected, got %v", event.State)
	}

	if event.Interface != "eth0" {
		t.Errorf("Expected interface 'eth0', got %s", event.Interface)
	}
}

func TestNetworkMonitorStats(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	stats := nm.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	expectedKeys := []string{
		"currentState", "lastStateChange", "lastConnectTest",
		"consecutiveTests", "interfaceCount", "testEndpoints", "platform",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key %s", key)
		}
	}

	// Test that current state is reported correctly
	if stats["currentState"] != nm.GetCurrentState().String() {
		t.Errorf("Stats current state should match GetCurrentState()")
	}
}

func TestConnectionManagerNetworkIntegration(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Test that network monitor is properly initialized
	if cm.networkMonitor == nil {
		t.Error("Connection manager should have network monitor")
	}

	// Test that network monitor is accessible
	nm := cm.GetNetworkMonitor()
	if nm == nil {
		t.Error("GetNetworkMonitor should return network monitor")
	}

	if nm != cm.networkMonitor {
		t.Error("GetNetworkMonitor should return the same instance")
	}
}

func TestNetworkEventHandling(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Test sleep/wake event handling
	event := SleepWakeEvent{
		Type:      "suspected_wake",
		Timestamp: time.Now(),
		Reason:    "Test wake event",
	}

	// This should not panic
	cm.handleSleepWakeEvent(event)

	// Test network event handling
	networkEvent := NetworkEvent{
		State:     NetworkStateConnected,
		Timestamp: time.Now(),
		Reason:    "Test network event",
	}

	// This should not panic
	cm.handleNetworkEvent(networkEvent)
}

func TestNetworkConnectivityTesting(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	// Test individual endpoint connectivity
	// Use a reliable endpoint that should be reachable
	connected := nm.testSingleEndpoint("1.1.1.1:53")
	
	// This test might fail in environments without internet access
	// so we'll just log the result rather than failing
	t.Logf("Connectivity test to 1.1.1.1:53 result: %v", connected)

	// Test full connectivity check
	fullConnected := nm.testConnectivity()
	t.Logf("Full connectivity test result: %v", fullConnected)
}

func TestInterfaceChangeDetection(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	// Test interface state change detection
	// We'll test with net.Interface structs with different flags

	// This is more of a unit test to ensure the logic works
	iface1 := net.Interface{Name: "eth0", Flags: net.FlagUp}
	iface2 := net.Interface{Name: "eth0", Flags: net.FlagUp | net.FlagRunning}

	// The interface state should be considered changed
	changed := nm.interfaceStateChanged(iface1, iface2)
	if !changed {
		t.Error("Interface state change should be detected when flags differ")
	}

	// Same interface should not be considered changed
	unchanged := nm.interfaceStateChanged(iface1, iface1)
	if unchanged {
		t.Error("Same interface should not be considered changed")
	}
}

func TestSleepWakeDetection(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	// Test platform-specific detector creation
	detector := newPlatformSleepWakeDetector()
	if detector == nil {
		t.Error("Platform sleep/wake detector should not be nil")
	}

	// Test generic detector
	genericDetector := &genericSleepWakeDetector{}
	err := genericDetector.Start(context.Background(), make(chan SleepWakeEvent, 1))
	if err != nil {
		t.Errorf("Generic detector start should not return error: %v", err)
	}

	err = genericDetector.Stop()
	if err != nil {
		t.Errorf("Generic detector stop should not return error: %v", err)
	}
}

func TestNetworkMonitorWithRealConnectivity(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	// Set shorter intervals for testing
	nm.connectivityTestInterval = 1 * time.Second
	nm.interfaceCheckInterval = 500 * time.Millisecond

	err := nm.Start()
	if err != nil {
		t.Errorf("Start should not return error: %v", err)
	}

	// Monitor for events for a short period
	timeout := time.After(3 * time.Second)
	eventReceived := false

	for !eventReceived {
		select {
		case event := <-nm.NetworkEvents():
			t.Logf("Received network event: %+v", event)
			eventReceived = true

		case event := <-nm.SleepWakeEvents():
			t.Logf("Received sleep/wake event: %+v", event)
			eventReceived = true

		case <-timeout:
			t.Log("No events received within timeout (this is normal for stable systems)")
			return
		}
	}
}

func TestConcurrentNetworkMonitoring(t *testing.T) {
	nm := NewNetworkMonitor()
	defer nm.Stop()

	err := nm.Start()
	if err != nil {
		t.Errorf("Start should not return error: %v", err)
	}

	// Test concurrent access to state
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				state := nm.GetCurrentState()
				stats := nm.GetStats()
				
				// Just verify we can access these without panicking
				_ = state
				_ = stats
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNetworkMonitorRecoveryScenarios(t *testing.T) {
	cm := NewConnectionManager()
	defer cm.Stop()

	// Simulate network recovery scenarios

	// Test reconnection after wake
	cm.reconnectAllInstances("Test reconnection")

	// Test marking connections suspect
	cm.markAllConnectionsSuspect("Test suspect marking")

	// Test connection validation
	cm.validateAllConnections("Test validation")

	// These should complete without panicking
	t.Log("Network recovery scenarios completed successfully")
}