package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	
	"github.com/standardbeagle/brummer/internal/discovery"
)

func TestHealthMonitorUpdatesActivity(t *testing.T) {
	// Create a mock MCP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond to ping requests
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))
	defer mockServer.Close()
	
	// Extract port from mock server URL
	// Parse URL to get port
	port := 0
	if _, err := fmt.Sscanf(mockServer.URL, "http://127.0.0.1:%d", &port); err != nil {
		t.Fatalf("Failed to parse mock server port: %v", err)
	}
	
	// Create connection manager
	connMgr := NewConnectionManager()
	defer connMgr.Stop()
	
	// Create mock instance pointing to our test server
	instance := &discovery.Instance{
		ID:   "test-instance",
		Name: "test",
		Port: port,
		Directory: "/tmp/test",
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        1234,
			Executable: "brum",
		},
	}
	
	// Register instance
	if err := connMgr.RegisterInstance(instance); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}
	
	// Wait for connection to be established
	time.Sleep(500 * time.Millisecond)
	
	// Get initial state
	connections := connMgr.ListInstances()
	var initialActivity time.Time
	var found bool
	for _, conn := range connections {
		if conn.InstanceID == instance.ID {
			initialActivity = conn.LastActivity
			found = true
			t.Logf("Initial state: %s, activity: %v", conn.State, initialActivity)
			break
		}
	}
	
	if !found {
		t.Fatal("Instance not found after registration")
	}
	
	// Create health monitor with short intervals
	config := &HealthMonitorConfig{
		PingInterval: 100 * time.Millisecond,
		PingTimeout:  50 * time.Millisecond,
		MaxFailures:  3,
	}
	healthMon := NewHealthMonitor(connMgr, config)
	
	// Start health monitor
	healthMon.Start()
	defer healthMon.Stop()
	
	// Wait for at least two ping cycles
	time.Sleep(300 * time.Millisecond)
	
	// Check that activity was updated
	connections = connMgr.ListInstances()
	var updatedActivity time.Time
	found = false
	for _, conn := range connections {
		if conn.InstanceID == instance.ID {
			updatedActivity = conn.LastActivity
			found = true
			t.Logf("Updated state: %s, activity: %v", conn.State, updatedActivity)
			
			// Also verify state remains active
			if conn.State != StateActive {
				t.Errorf("Instance state changed to %v, expected active", conn.State)
			}
			break
		}
	}
	
	if !found {
		t.Fatal("Instance not found after health check")
	}
	
	if !updatedActivity.After(initialActivity) {
		t.Errorf("Activity time not updated: initial=%v, updated=%v", 
			initialActivity, updatedActivity)
	}
}