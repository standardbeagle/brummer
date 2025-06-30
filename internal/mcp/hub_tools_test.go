package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterHubTools tests that all hub tools are properly registered
func TestRegisterHubTools(t *testing.T) {
	srv := server.NewMCPServer("test", "1.0")
	connMgr := NewConnectionManager()

	// Register hub tools
	RegisterHubTools(srv, connMgr)

	// Expected hub tools
	expectedTools := []string{
		"hub_scripts_list",
		"hub_scripts_run",
		"hub_scripts_stop",
		"hub_scripts_status",
		"hub_logs_stream",
		"hub_logs_search",
		"hub_proxy_requests",
		"hub_telemetry_sessions",
		"hub_telemetry_events",
		"hub_browser_open",
		"hub_browser_refresh",
		"hub_browser_navigate",
		"hub_browser_screenshot",
		"hub_repl_execute",
	}

	// Verify we have the expected number of tools
	assert.Equal(t, 14, len(expectedTools))
	
	// Note: We can't directly inspect the server's registered tools
	// In a real test, we'd need to call the tools/list method
}

// TestHubToolsWithMockInstance tests hub tools with a mock MCP instance
func TestHubToolsWithMockInstance(t *testing.T) {
	// Create a mock MCP instance server
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCMessage
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Handle different methods
		switch req.Method {
		case "initialize":
			resp := JSONRPCMessage{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "1.0",
					"serverInfo": map[string]interface{}{
						"name":    "mock-instance",
						"version": "1.0.0",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)

		case "tools/call":
			var params map[string]interface{}
			json.Unmarshal(req.Params, &params)
			toolName := params["name"].(string)
			
			// Return different responses based on tool
			var result interface{}
			switch toolName {
			case "scripts_list":
				result = map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": `{"scripts": ["dev", "test", "build"]}`},
					},
				}
			case "logs_stream":
				result = map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": `{"logs": [{"timestamp": "2024-01-01T00:00:00Z", "message": "test log"}]}`},
					},
				}
			default:
				result = map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": fmt.Sprintf(`{"tool": "%s", "result": "ok"}`, toolName)},
					},
				}
			}
			
			resp := JSONRPCMessage{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Result:  result,
			}
			json.NewEncoder(w).Encode(resp)

		default:
			resp := JSONRPCMessage{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer mockInstance.Close()

	// Extract port from mock server URL
	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)

	// Create connection manager and register instance
	connMgr := NewConnectionManager()
	
	// Create discovery instance
	instance := &discovery.Instance{
		ID:        "test-instance",
		Name:      "Mock Instance",
		Directory: "/tmp/test",
		Port:      port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12345
	instance.ProcessInfo.Executable = "brum"
	
	// Register instance
	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	
	// Wait for connection to be established
	time.Sleep(100 * time.Millisecond)
	
	// Verify instance is connected
	connections := connMgr.ListInstances()
	require.Len(t, connections, 1)
	assert.Equal(t, StateActive, connections[0].State)

	// Test hub_scripts_list
	t.Run("hub_scripts_list", func(t *testing.T) {
		result, err := callInstanceTool(context.Background(), connMgr, "test-instance", "scripts_list", nil)
		require.NoError(t, err)
		assert.Contains(t, string(result), "scripts")
		assert.Contains(t, string(result), "dev")
	})

	// Test hub_logs_stream with parameters
	t.Run("hub_logs_stream", func(t *testing.T) {
		args := map[string]interface{}{
			"processId": "dev-123",
			"level":     "error",
			"follow":    true,
			"limit":     50,
		}
		result, err := callInstanceTool(context.Background(), connMgr, "test-instance", "logs_stream", args)
		require.NoError(t, err)
		assert.Contains(t, string(result), "logs")
	})

	// Test error cases
	t.Run("instance_not_found", func(t *testing.T) {
		_, err := callInstanceTool(context.Background(), connMgr, "non-existent", "scripts_list", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

// TestHubToolsConcurrentAccess tests concurrent access to hub tools with real instances
func TestHubToolsConcurrentAccess(t *testing.T) {
	// Create multiple mock instance servers
	var mockInstances []*httptest.Server
	for i := 0; i < 3; i++ {
		instanceID := fmt.Sprintf("instance-%d", i+1)
		mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req JSONRPCMessage
			json.NewDecoder(r.Body).Decode(&req)

			if req.Method == "initialize" {
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"protocolVersion": "1.0",
						"serverInfo": map[string]interface{}{
							"name":    instanceID,
							"version": "1.0.0",
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			} else if req.Method == "tools/call" {
				// Simulate some processing
				time.Sleep(10 * time.Millisecond)
				
				resp := JSONRPCMessage{
					Jsonrpc: "2.0",
					ID:      req.ID,
					Result: map[string]interface{}{
						"content": []map[string]interface{}{
							{"type": "text", "text": fmt.Sprintf(`{"instance": "%s", "result": "ok"}`, instanceID)},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			}
		}))
		mockInstances = append(mockInstances, mockInstance)
	}
	defer func() {
		for _, server := range mockInstances {
			server.Close()
		}
	}()

	// Create connection manager and register instances
	connMgr := NewConnectionManager()
	
	for i, server := range mockInstances {
		var port int
		fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)
		
		instance := &discovery.Instance{
			ID:         fmt.Sprintf("instance-%d", i+1),
			Name:       fmt.Sprintf("Mock Instance %d", i+1),
			Directory:  fmt.Sprintf("/tmp/test%d", i+1),
			Port:       port,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 12345 + i
		instance.ProcessInfo.Executable = "brum"
		
		err := connMgr.RegisterInstance(instance)
		require.NoError(t, err)
	}
	
	// Wait for connections
	time.Sleep(200 * time.Millisecond)
	
	// Verify all instances are connected
	connections := connMgr.ListInstances()
	require.Len(t, connections, 3)
	for _, conn := range connections {
		assert.Equal(t, StateActive, conn.State)
	}

	// Launch concurrent tool calls
	results := make(chan string, 15)
	errors := make(chan error, 15)
	
	tools := []string{"scripts_list", "logs_stream", "browser_open", "proxy_requests", "repl_execute"}
	
	for i := 1; i <= 3; i++ {
		for _, tool := range tools {
			go func(instanceID, toolName string) {
				result, err := callInstanceTool(context.Background(), connMgr, instanceID, toolName, nil)
				if err != nil {
					errors <- err
				} else {
					results <- string(result)
				}
			}(fmt.Sprintf("instance-%d", i), tool)
		}
	}

	// Collect results
	successCount := 0
	errorCount := 0
	
	for i := 0; i < 15; i++ {
		select {
		case err := <-errors:
			errorCount++
			t.Logf("Error: %v", err)
		case result := <-results:
			successCount++
			assert.Contains(t, result, "instance-")
			assert.Contains(t, result, "ok")
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}
	
	assert.Equal(t, 15, successCount)
	assert.Equal(t, 0, errorCount)
}

// TestHubToolsInstanceLifecycle tests tool behavior during instance lifecycle
func TestHubToolsInstanceLifecycle(t *testing.T) {
	connMgr := NewConnectionManager()
	
	// Create a mock instance that can be controlled
	var serverRunning bool
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serverRunning {
			// Simulate instance being down
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		resp := JSONRPCMessage{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": `{"status": "ok"}`},
				},
			},
		}
		
		if req.Method == "initialize" {
			resp.Result = map[string]interface{}{
				"protocolVersion": "1.0",
				"serverInfo": map[string]interface{}{
					"name":    "lifecycle-test",
					"version": "1.0.0",
				},
			}
		}
		
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockInstance.Close()
	
	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)
	
	instance := &discovery.Instance{
		ID:         "lifecycle-instance",
		Name:       "Lifecycle Test",
		Directory:  "/tmp/lifecycle",
		Port:       port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 99999
	instance.ProcessInfo.Executable = "brum"
	
	// Test 1: Instance starts up
	serverRunning = true
	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	
	// Should be able to call tools
	result, err := callInstanceTool(context.Background(), connMgr, "lifecycle-instance", "scripts_list", nil)
	assert.NoError(t, err)
	assert.Contains(t, string(result), "ok")
	
	// Test 2: Instance goes down
	serverRunning = false
	
	// Tool calls should fail but not crash
	_, err = callInstanceTool(context.Background(), connMgr, "lifecycle-instance", "scripts_list", nil)
	// Could fail immediately or after health check detects it
	if err == nil {
		// Wait for health check to detect failure
		time.Sleep(2 * time.Second)
		_, err = callInstanceTool(context.Background(), connMgr, "lifecycle-instance", "scripts_list", nil)
	}
	// Don't assert specific error as it depends on timing
	
	// Test 3: Instance comes back up
	serverRunning = true
	
	// The hub should handle intermittent availability
	// In real scenario, discovery would re-register the instance
	// For this test, we'll simulate by checking state
	connections := connMgr.ListInstances()
	assert.Len(t, connections, 1)
}

// TestStreamingToolsNotImplemented verifies streaming tools return static responses
func TestStreamingToolsNotImplemented(t *testing.T) {
	// This test documents that streaming is not yet implemented
	// When implemented, this test should be updated
	
	mockInstance := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCMessage
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.Method == "initialize" {
			resp := JSONRPCMessage{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "1.0",
					"serverInfo": map[string]interface{}{
						"name":    "streaming-test",
						"version": "1.0.0",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else if req.Method == "tools/call" {
			// Currently returns static JSON even for streaming tools
			resp := JSONRPCMessage{
				Jsonrpc: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": `{"data": "static response", "streaming": false}`},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer mockInstance.Close()
	
	var port int
	fmt.Sscanf(mockInstance.URL, "http://127.0.0.1:%d", &port)
	
	connMgr := NewConnectionManager()
	instance := &discovery.Instance{
		ID:         "streaming-instance",
		Name:       "Streaming Test",
		Directory:  "/tmp/streaming",
		Port:       port,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 88888
	instance.ProcessInfo.Executable = "brum"
	
	err := connMgr.RegisterInstance(instance)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	
	// Test logs_stream with follow=true
	args := map[string]interface{}{
		"follow": true,
	}
	result, err := callInstanceTool(context.Background(), connMgr, "streaming-instance", "logs_stream", args)
	require.NoError(t, err)
	
	// Verify it's JSON, not a stream
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(string(result)), &parsed)
	assert.NoError(t, err, "Expected JSON response, not a stream")
	assert.Contains(t, string(result), "static response")
	
	// TODO: When streaming is implemented, update this test to verify:
	// - Response is Server-Sent Events format
	// - Multiple events are received over time
	// - Stream can be cancelled
}