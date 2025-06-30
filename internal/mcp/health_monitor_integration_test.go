package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthMonitorFailureDetection tests health monitor detects failures
func TestHealthMonitorFailureDetection(t *testing.T) {
	// Track callback invocations
	var unhealthyCalls, recoveryCalls, deadCalls int32
	var mu sync.Mutex
	
	// Create connection manager
	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Create controllable mock server
	var serverHealthy atomic.Bool
	serverHealthy.Store(true)
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serverHealthy.Load() {
			// Simulate unhealthy instance
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		
		// Healthy response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage("1"),
			Result:  map[string]interface{}{},
		})
	}))
	defer mockServer.Close()

	var port int
	fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port)

	// Register instance
	instance := &discovery.Instance{
		ID:        "health-test",
		Name:      "Health Test",
		Directory: "/test",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 40001
	instance.ProcessInfo.Executable = "brum"

	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	
	// Wait for connection
	time.Sleep(200 * time.Millisecond)

	// Create health monitor with short intervals
	config := &HealthMonitorConfig{
		PingInterval: 50 * time.Millisecond,
		PingTimeout:  25 * time.Millisecond,
		MaxFailures:  2, // Fail fast for testing
	}
	
	healthMon := NewHealthMonitor(connMgr, config)
	
	// Set callbacks
	healthMon.SetCallbacks(
		func(instanceID string, status *HealthStatus) {
			atomic.AddInt32(&unhealthyCalls, 1)
			mu.Lock()
			t.Logf("Instance %s became unhealthy: %v", instanceID, status.LastError)
			mu.Unlock()
		},
		func(instanceID string, status *HealthStatus) {
			atomic.AddInt32(&recoveryCalls, 1)
			mu.Lock()
			t.Logf("Instance %s recovered", instanceID)
			mu.Unlock()
		},
		func(instanceID string, status *HealthStatus) {
			atomic.AddInt32(&deadCalls, 1)
			mu.Lock()
			t.Logf("Instance %s marked as dead", instanceID)
			mu.Unlock()
		},
	)

	healthMon.Start()
	defer healthMon.Stop()

	// Test 1: Healthy instance
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&unhealthyCalls), "Should not be unhealthy yet")

	// Test 2: Instance becomes unhealthy
	serverHealthy.Store(false)
	time.Sleep(200 * time.Millisecond) // Wait for failures to accumulate

	assert.Equal(t, int32(1), atomic.LoadInt32(&unhealthyCalls), "Should be marked unhealthy")
	
	// Check connection state
	connections := connMgr.ListInstances()
	require.Len(t, connections, 1)
	assert.Equal(t, StateRetrying, connections[0].State, "Should be in retrying state")

	// Test 3: Instance recovers
	serverHealthy.Store(true)
	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&recoveryCalls), "Should have recovered")

	// Test 4: Instance fails completely
	serverHealthy.Store(false)
	time.Sleep(300 * time.Millisecond) // Wait for multiple failures

	assert.Greater(t, atomic.LoadInt32(&deadCalls), int32(0), "Should be marked dead")
}

// TestHealthMonitorConcurrentPings tests health checks don't overlap
func TestHealthMonitorConcurrentPings(t *testing.T) {
	var activePings int32
	var maxConcurrent int32
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track concurrent pings
		current := atomic.AddInt32(&activePings, 1)
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}
		
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage("1"),
			Result:  map[string]interface{}{},
		})
		
		atomic.AddInt32(&activePings, -1)
	}))
	defer mockServer.Close()

	var port int
	fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port)

	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Register multiple instances on same port (simulating shared server)
	for i := 0; i < 3; i++ {
		instance := &discovery.Instance{
			ID:        fmt.Sprintf("concurrent-%d", i),
			Name:      fmt.Sprintf("Concurrent %d", i),
			Directory: fmt.Sprintf("/test%d", i),
			Port:      port,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 40100 + i
		instance.ProcessInfo.Executable = "brum"

		err := connMgr.RegisterInstance(instance)
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// Fast ping interval to stress test
	config := &HealthMonitorConfig{
		PingInterval: 20 * time.Millisecond,
		PingTimeout:  200 * time.Millisecond,
		MaxFailures:  3,
	}
	
	healthMon := NewHealthMonitor(connMgr, config)
	healthMon.Start()
	
	// Let it run for a while
	time.Sleep(500 * time.Millisecond)
	
	healthMon.Stop()

	maxSeen := atomic.LoadInt32(&maxConcurrent)
	t.Logf("Maximum concurrent pings: %d", maxSeen)
	
	// Should handle multiple instances without excessive concurrency
	assert.LessOrEqual(t, maxSeen, int32(6), "Too many concurrent pings")
}

// TestHealthMonitorStateTransitions tests proper state transitions during health events
func TestHealthMonitorStateTransitions(t *testing.T) {
	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Track state changes
	var stateChanges []string
	var mu sync.Mutex
	
	// Controllable server
	var responseDelay atomic.Int64
	var shouldFail atomic.Bool
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delay := time.Duration(responseDelay.Load()) * time.Millisecond
		time.Sleep(delay)
		
		if shouldFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage("1"),
			Result:  map[string]interface{}{},
		})
	}))
	defer mockServer.Close()

	var port int
	fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "state-test",
		Name:      "State Test",
		Directory: "/test",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 40200
	instance.ProcessInfo.Executable = "brum"

	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Monitor state changes
	go func() {
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		
		var lastState ConnectionState
		for range ticker.C {
			connections := connMgr.ListInstances()
			if len(connections) > 0 {
				currentState := connections[0].State
				if currentState != lastState {
					mu.Lock()
					stateChanges = append(stateChanges, 
						fmt.Sprintf("%s->%s", lastState, currentState))
					mu.Unlock()
					lastState = currentState
				}
			}
		}
	}()

	config := &HealthMonitorConfig{
		PingInterval: 50 * time.Millisecond,
		PingTimeout:  30 * time.Millisecond,
		MaxFailures:  2,
	}
	
	healthMon := NewHealthMonitor(connMgr, config)
	healthMon.Start()

	// Scenario 1: Normal operation
	time.Sleep(100 * time.Millisecond)

	// Scenario 2: Slow response (timeout)
	responseDelay.Store(50) // Exceeds timeout
	time.Sleep(150 * time.Millisecond)

	// Scenario 3: Complete failure
	shouldFail.Store(true)
	time.Sleep(200 * time.Millisecond)

	// Scenario 4: Recovery
	shouldFail.Store(false)
	responseDelay.Store(0)
	time.Sleep(150 * time.Millisecond)

	healthMon.Stop()

	mu.Lock()
	t.Logf("State transitions: %v", stateChanges)
	mu.Unlock()

	// Verify we saw expected transitions
	assert.NotEmpty(t, stateChanges, "Should have state transitions")
}

// TestHealthMonitorMemoryUsage tests for memory leaks during long operation
func TestHealthMonitorMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage("1"),
			Result:  map[string]interface{}{},
		})
	}))
	defer mockServer.Close()

	var port int
	fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port)

	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Register a few instances
	for i := 0; i < 5; i++ {
		instance := &discovery.Instance{
			ID:        fmt.Sprintf("memory-%d", i),
			Name:      fmt.Sprintf("Memory %d", i),
			Directory: fmt.Sprintf("/mem%d", i),
			Port:      port,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 50000 + i
		instance.ProcessInfo.Executable = "brum"

		err := connMgr.RegisterInstance(instance)
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	config := &HealthMonitorConfig{
		PingInterval: 10 * time.Millisecond, // Very frequent
		PingTimeout:  5 * time.Millisecond,
		MaxFailures:  3,
	}
	
	healthMon := NewHealthMonitor(connMgr, config)
	
	// Track status count
	initialStatuses := len(healthMon.healthStatuses)
	
	healthMon.Start()

	// Run for a while
	time.Sleep(500 * time.Millisecond)

	// Stop and check
	healthMon.Stop()

	finalStatuses := len(healthMon.healthStatuses)
	assert.Equal(t, initialStatuses, finalStatuses, "Health statuses should not grow")
}

// TestHealthMonitorIntermittentAvailability tests realistic scenario
func TestHealthMonitorIntermittentAvailability(t *testing.T) {
	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Simulate intermittent network issues
	var failureRate atomic.Int32
	failureRate.Store(0)
	
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Randomly fail based on rate
		if time.Now().UnixNano()%100 < int64(failureRate.Load()) {
			// Simulate various failures
			switch time.Now().UnixNano() % 3 {
			case 0:
				w.WriteHeader(http.StatusServiceUnavailable)
			case 1:
				time.Sleep(100 * time.Millisecond) // Timeout
			case 2:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		
		// Success
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage("1"),
			Result:  map[string]interface{}{},
		})
	}))
	defer mockServer.Close()

	var port int
	fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port)

	instance := &discovery.Instance{
		ID:        "intermittent",
		Name:      "Intermittent Test",
		Directory: "/test",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 60000
	instance.ProcessInfo.Executable = "brum"

	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	config := &HealthMonitorConfig{
		PingInterval: 50 * time.Millisecond,
		PingTimeout:  40 * time.Millisecond,
		MaxFailures:  3,
	}
	
	var stateHistory []ConnectionState
	var mu sync.Mutex
	
	healthMon := NewHealthMonitor(connMgr, config)
	healthMon.Start()

	// Monitor states
	go func() {
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		
		for range ticker.C {
			connections := connMgr.ListInstances()
			if len(connections) > 0 {
				mu.Lock()
				stateHistory = append(stateHistory, connections[0].State)
				mu.Unlock()
			}
		}
	}()

	// Simulate varying network conditions
	scenarios := []struct {
		duration    time.Duration
		failureRate int32
		description string
	}{
		{200 * time.Millisecond, 0, "Stable network"},
		{200 * time.Millisecond, 30, "30% failure rate"},
		{200 * time.Millisecond, 70, "70% failure rate"},
		{200 * time.Millisecond, 0, "Recovery"},
	}

	for _, scenario := range scenarios {
		t.Logf("Testing: %s", scenario.description)
		failureRate.Store(scenario.failureRate)
		time.Sleep(scenario.duration)
	}

	healthMon.Stop()

	// Analyze behavior
	mu.Lock()
	defer mu.Unlock()
	
	activeCount := 0
	retryingCount := 0
	for _, state := range stateHistory {
		switch state {
		case StateActive:
			activeCount++
		case StateRetrying:
			retryingCount++
		}
	}

	t.Logf("State distribution: Active=%d, Retrying=%d, Total=%d", 
		activeCount, retryingCount, len(stateHistory))

	// Should have experienced both states
	assert.Greater(t, activeCount, 0, "Should have been active sometimes")
	assert.Greater(t, retryingCount, 0, "Should have retried sometimes")
}