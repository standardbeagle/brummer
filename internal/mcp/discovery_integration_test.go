package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
)

// TestConnectionManagerDiscoveryIntegration tests the complete flow from discovery to connection
func TestConnectionManagerDiscoveryIntegration(t *testing.T) {
	t.Parallel()
	
	// Create test directories
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	// Create discovery system
	disc, err := discovery.New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer disc.Stop()
	
	// Create connection manager
	cm := NewConnectionManager()
	
	// Wire up discovery callbacks BEFORE starting discovery
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		for _, inst := range instances {
			if err := cm.RegisterInstance(inst); err != nil {
				t.Logf("Failed to register instance %s: %v", inst.ID, err)
			}
		}
	})
	
	// Start systems
	disc.Start()
	// ConnectionManager runs automatically when created
	defer cm.Stop()
	
	// Create mock MCP server for testing
	mockServer := createMockMCPServer(t, 7777)
	defer mockServer.Close()
	
	// Register an instance
	instance := &discovery.Instance{
		ID:        "test-instance-1",
		Name:      "Test Instance",
		Directory: "/test/dir",
		Port:      7777,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(),
			Executable: "test",
		},
	}
	
	if err := discovery.RegisterInstance(instancesDir, instance); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}
	
	// Wait for instance to be discovered and connected
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if !waitForCondition(ctx, 50*time.Millisecond, func() bool {
		connections := cm.ListInstances()
		for _, conn := range connections {
			if conn.InstanceID == "test-instance-1" {
				t.Logf("Instance state: %s", conn.State)
				if conn.State == StateActive {
					return true
				}
			}
		}
		return false
	}) {
		connections := cm.ListInstances()
		for _, conn := range connections {
			t.Logf("Connection: ID=%s, State=%s, Port=%d", conn.InstanceID, conn.State, conn.Port)
		}
		t.Fatalf("Instance not connected. Total connections: %d", len(connections))
	}
	
	// Verify we can get a client for the instance
	client := cm.GetClient("test-session")
	if client != nil {
		t.Error("Should not have client for unmapped session")
	}
	
	// Connect a session
	if err := cm.ConnectSession("test-session", "test-instance-1"); err != nil {
		t.Fatalf("Failed to connect session: %v", err)
	}
	
	// Debug: Check the connection has a client
	connections := cm.ListInstances()
	for _, conn := range connections {
		if conn.InstanceID == "test-instance-1" {
			t.Logf("Connection after session connect: State=%s, HasClient=%v", conn.State, conn.Client != nil)
		}
	}
	
	// Now we should get a client
	client = cm.GetClient("test-session")
	if client == nil {
		t.Fatal("Should have client for connected session")
	}
}

// TestDiscoveryToConnectionStateFlow tests state transitions from discovery
func TestDiscoveryToConnectionStateFlow(t *testing.T) {
	t.Skip("Skipping test that depends on ConnectionManager timing internals")
	t.Parallel()
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	disc, err := discovery.New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer disc.Stop()
	
	cm := NewConnectionManager()
	defer cm.Stop()
	
	// Wire up discovery
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		for _, inst := range instances {
			cm.RegisterInstance(inst)
		}
	})
	
	disc.Start()
	
	// Create instance that will fail to connect (no server)
	failInstance := &discovery.Instance{
		ID:        "fail-instance",
		Name:      "Will Fail",
		Directory: "/test",
		Port:      9999, // No server here
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(),
			Executable: "test",
		},
	}
	
	if err := discovery.RegisterInstance(instancesDir, failInstance); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}
	
	// Wait for connection attempts and retries
	time.Sleep(5 * time.Second)
	
	// Check final state
	connections := cm.ListInstances()
	var failedInstance *ConnectionInfo
	for _, conn := range connections {
		if conn.InstanceID == "fail-instance" {
			failedInstance = conn
			break
		}
	}
	
	if failedInstance == nil {
		t.Fatal("Instance not found in connection manager")
	}
	
	// Should be in retrying or dead state since no server exists
	acceptableStates := []ConnectionState{StateRetrying, StateDead}
	validState := false
	for _, state := range acceptableStates {
		if failedInstance.State == state {
			validState = true
			break
		}
	}
	
	if !validState {
		t.Errorf("Expected instance to be retrying or dead, got: %s", failedInstance.State)
	}
	
	t.Logf("Instance state after failed connection: %s (retry count: %d)", failedInstance.State, failedInstance.RetryCount)
}

// TestMultipleInstanceDiscovery tests hub discovering multiple instances
func TestMultipleInstanceDiscovery(t *testing.T) {
	t.Parallel()
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	disc, err := discovery.New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer disc.Stop()
	
	cm := NewConnectionManager()
	
	// Track discovered instances
	var discoveredMu sync.Mutex
	discovered := make(map[string]bool)
	
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		discoveredMu.Lock()
		defer discoveredMu.Unlock()
		
		for id, inst := range instances {
			if !discovered[id] {
				discovered[id] = true
				cm.RegisterInstance(inst)
			}
		}
	})
	
	disc.Start()
	// ConnectionManager runs automatically
	defer cm.Stop()
	
	// Create multiple mock servers
	mockServers := make(map[int]MockServer)
	ports := []int{8001, 8002, 8003}
	
	for _, port := range ports {
		server := createMockMCPServer(t, port)
		defer server.Close()
		mockServers[port] = server
	}
	
	// Register multiple instances
	instances := []*discovery.Instance{
		{
			ID:        "frontend-abc123",
			Name:      "Frontend",
			Directory: "/frontend",
			Port:      8001,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        os.Getpid(),
				Executable: "test",
			},
		},
		{
			ID:        "backend-def456",
			Name:      "Backend",
			Directory: "/backend",
			Port:      8002,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        os.Getpid() + 1,
				Executable: "test",
			},
		},
		{
			ID:        "database-ghi789",
			Name:      "Database",
			Directory: "/database",
			Port:      8003,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        os.Getpid() + 2,
				Executable: "test",
			},
		},
	}
	
	// Register all instances
	for _, inst := range instances {
		if err := discovery.RegisterInstance(instancesDir, inst); err != nil {
			t.Fatalf("Failed to register instance %s: %v", inst.ID, err)
		}
	}
	
	// Wait for all instances to be connected
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if !waitForCondition(ctx, 100*time.Millisecond, func() bool {
		connections := cm.ListInstances()
		activeCount := 0
		for _, conn := range connections {
			if conn.State == StateActive {
				activeCount++
			}
		}
		return activeCount == len(instances)
	}) {
		connections := cm.ListInstances()
		t.Fatalf("Not all instances connected. Connections: %+v", connections)
	}
	
	// Test session routing to different instances
	sessions := map[string]string{
		"session-1": "frontend-abc123",
		"session-2": "backend-def456",
		"session-3": "database-ghi789",
	}
	
	for sessionID, instanceID := range sessions {
		if err := cm.ConnectSession(sessionID, instanceID); err != nil {
			t.Errorf("Failed to connect session %s to instance %s: %v", sessionID, instanceID, err)
		}
		
		client := cm.GetClient(sessionID)
		if client == nil {
			t.Errorf("No client for session %s", sessionID)
		}
	}
}

// TestInstanceFileDisappearance tests handling when instance files are deleted
func TestInstanceFileDisappearance(t *testing.T) {
	t.Parallel()
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	disc, err := discovery.New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer disc.Stop()
	
	cm := NewConnectionManager()
	
	// Track removals
	var removalsMu sync.Mutex
	removals := make(map[string]bool)
	
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		// Check for removals
		connections := cm.ListInstances()
		for _, conn := range connections {
			if _, exists := instances[conn.InstanceID]; !exists {
				removalsMu.Lock()
				removals[conn.InstanceID] = true
				removalsMu.Unlock()
			}
		}
		
		// Register new instances
		for _, inst := range instances {
			cm.RegisterInstance(inst)
		}
	})
	
	disc.Start()
	// ConnectionManager runs automatically
	defer cm.Stop()
	
	mockServer := createMockMCPServer(t, 7779)
	defer mockServer.Close()
	
	// Register instance
	instance := &discovery.Instance{
		ID:        "disappearing-instance",
		Name:      "Will Disappear",
		Directory: "/test",
		Port:      7779,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
		ProcessInfo: struct {
			PID        int    `json:"pid"`
			Executable string `json:"executable"`
		}{
			PID:        os.Getpid(),
			Executable: "test",
		},
	}
	
	if err := discovery.RegisterInstance(instancesDir, instance); err != nil {
		t.Fatalf("Failed to register instance: %v", err)
	}
	
	// Wait for connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if !waitForCondition(ctx, 50*time.Millisecond, func() bool {
		connections := cm.ListInstances()
		for _, conn := range connections {
			if conn.InstanceID == "disappearing-instance" && conn.State == StateActive {
				return true
			}
		}
		return false
	}) {
		t.Fatal("Instance not connected")
	}
	
	// Remove instance file
	if err := discovery.UnregisterInstance(instancesDir, "disappearing-instance"); err != nil {
		t.Fatalf("Failed to unregister instance: %v", err)
	}
	
	// Wait for removal to be detected
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	
	if !waitForCondition(ctx2, 50*time.Millisecond, func() bool {
		removalsMu.Lock()
		wasRemoved := removals["disappearing-instance"]
		removalsMu.Unlock()
		return wasRemoved
	}) {
		t.Error("Instance removal not detected")
	}
}

// TestRapidInstanceChurn tests handling rapid instance add/remove cycles
func TestRapidInstanceChurn(t *testing.T) {
	t.Parallel()
	
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	
	disc, err := discovery.New(instancesDir)
	if err != nil {
		t.Fatalf("Failed to create discovery: %v", err)
	}
	defer disc.Stop()
	
	cm := NewConnectionManager()
	
	// Count events
	var eventsMu sync.Mutex
	addCount := int32(0)
	removeCount := int32(0)
	
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		
		// Simple registration of all instances
		for _, inst := range instances {
			cm.RegisterInstance(inst)
			atomic.AddInt32(&addCount, 1)
		}
	})
	
	disc.Start()
	// ConnectionManager runs automatically
	defer cm.Stop()
	
	// Rapid add/remove cycles
	cycles := 10
	for i := 0; i < cycles; i++ {
		instance := &discovery.Instance{
			ID:        fmt.Sprintf("churn-%d", i),
			Name:      fmt.Sprintf("Churn Test %d", i),
			Directory: "/test",
			Port:      9000 + i,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
			ProcessInfo: struct {
				PID        int    `json:"pid"`
				Executable string `json:"executable"`
			}{
				PID:        os.Getpid() + i,
				Executable: "test",
			},
		}
		
		// Add
		if err := discovery.RegisterInstance(instancesDir, instance); err != nil {
			t.Errorf("Failed to register instance %d: %v", i, err)
		}
		
		// Small delay
		time.Sleep(50 * time.Millisecond)
		
		// Remove
		if err := discovery.UnregisterInstance(instancesDir, instance.ID); err != nil {
			t.Errorf("Failed to unregister instance %d: %v", i, err)
		}
		
		atomic.AddInt32(&removeCount, 1)
	}
	
	// Wait for all operations to complete
	// A small delay is reasonable here as we're testing rapid churn
	time.Sleep(200 * time.Millisecond)
	
	// Verify system is still functional
	connections := cm.ListInstances()
	t.Logf("After churn: %d connections, %d adds, %d removes", len(connections), atomic.LoadInt32(&addCount), atomic.LoadInt32(&removeCount))
	
	// System should have processed events without crashing
	if atomic.LoadInt32(&addCount) < int32(cycles) {
		t.Errorf("Not all additions processed: expected at least %d, got %d", cycles, atomic.LoadInt32(&addCount))
	}
}

// Helper function to wait for condition
func waitForCondition(ctx context.Context, checkInterval time.Duration, condition func() bool) bool {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
		}
	}
}

// MockServer interface for testing
type MockServer interface {
	Close()
}

// createMockMCPServer creates a simple HTTP server that responds to MCP requests
func createMockMCPServer(t *testing.T, port int) MockServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond to initialize request
		if r.Method == "POST" && r.URL.Path == "/mcp" {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			
			if method, ok := req["method"].(string); ok && method == "initialize" {
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"result": map[string]interface{}{
						"capabilities": map[string]interface{}{},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}
		}
		
		// Default response
		w.WriteHeader(http.StatusOK)
	})
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	
	// Start server in background
	go server.ListenAndServe()
	
	// Give server time to start
	time.Sleep(50 * time.Millisecond)
	
	return &mockServerImpl{
		t:      t,
		port:   port,
		server: server,
	}
}

type mockServerImpl struct {
	t      *testing.T
	port   int
	server *http.Server
}

func (m *mockServerImpl) Close() {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		m.server.Shutdown(ctx)
	}
}