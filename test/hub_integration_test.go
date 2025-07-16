package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHubFullFlow tests the complete hub flow from discovery to tool proxying
func TestHubFullFlow(t *testing.T) {
	// Create temporary instances directory
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	require.NoError(t, os.MkdirAll(instancesDir, 0755))

	// Create discovery system
	disc, err := discovery.New(instancesDir)
	require.NoError(t, err)
	defer disc.Stop()

	// Create connection manager
	connMgr := mcp.NewConnectionManager()
	defer connMgr.Stop()

	// Create session manager
	sessionMgr := mcp.NewSessionManager()

	// Create health monitor
	healthMon := mcp.NewHealthMonitor(connMgr, &mcp.HealthMonitorConfig{
		PingInterval: 1 * time.Second,
		PingTimeout:  500 * time.Millisecond,
		MaxFailures:  2,
	})
	defer healthMon.Stop()

	// Start discovery
	disc.Start()

	// Set up discovery callback
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		for _, inst := range instances {
			connMgr.RegisterInstance(inst)
		}
	})

	// Create a mock instance MCP server
	mockInstance := createMockInstanceServer(t, "test-instance", 8888)
	defer mockInstance.Close()

	// Register the instance
	instance := &discovery.Instance{
		ID:        "test-instance",
		Name:      "Test Instance",
		Directory: "/test/dir",
		Port:      8888,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = os.Getpid()

	require.NoError(t, discovery.RegisterInstance(instancesDir, instance))

	// Wait for discovery
	time.Sleep(100 * time.Millisecond)

	// Verify instance is discovered
	connections := connMgr.ListInstances()
	require.Len(t, connections, 1)
	assert.Equal(t, "test-instance", connections[0].InstanceID)
	// State might be Discovered, Connecting, or Active depending on timing
	assert.Contains(t, []mcp.ConnectionState{mcp.StateDiscovered, mcp.StateConnecting, mcp.StateActive}, connections[0].State)

	// Wait for connection to be established
	time.Sleep(200 * time.Millisecond)

	// Check connection is active
	connections = connMgr.ListInstances()
	require.Len(t, connections, 1)
	assert.Equal(t, mcp.StateActive, connections[0].State)

	// Test session connection
	sessionID := "test-session"
	sessionMgr.CreateSession(sessionID, map[string]string{"client": "test"})

	err = connMgr.ConnectSession(sessionID, "test-instance")
	assert.NoError(t, err)

	// Test tool proxying
	client := connMgr.GetClient(sessionID)
	require.NotNil(t, client)

	// List tools
	ctx := context.Background()
	toolsData, err := client.ListTools(ctx)
	require.NoError(t, err)

	var toolsResp struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(toolsData, &toolsResp))
	assert.Len(t, toolsResp.Tools, 2)

	// Call a tool
	result, err := client.CallTool(ctx, "test/echo", map[string]interface{}{
		"message": "Hello, Hub!",
	})
	require.NoError(t, err)

	var toolResult map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &toolResult))
	assert.Equal(t, "Echo: Hello, Hub!", toolResult["response"])

	// Test health monitoring
	healthMon.Start()
	time.Sleep(1500 * time.Millisecond)

	status, err := healthMon.GetHealthStatus("test-instance")
	require.NoError(t, err)
	assert.True(t, status.IsHealthy)
	assert.Equal(t, 0, status.ConsecutiveFailures)

	// Test session disconnection
	err = connMgr.DisconnectSession(sessionID)
	assert.NoError(t, err)

	// Verify session is disconnected
	client = connMgr.GetClient(sessionID)
	assert.Nil(t, client)
}

// createMockInstanceServer creates a mock MCP server that acts like an instance
func createMockInstanceServer(t *testing.T, instanceID string, port int) *httptest.Server {
	mux := http.NewServeMux()

	// Handle MCP requests
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		method := request["method"].(string)
		id := request["id"]

		var result interface{}
		var rpcError interface{}

		switch method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "1.0",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
			}

		case "ping":
			result = map[string]interface{}{
				"pong": time.Now().Unix(),
			}

		case "tools/list":
			result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "test/echo",
						"description": "Echoes a message",
					},
					{
						"name":        "test/status",
						"description": "Returns instance status",
					},
				},
			}

		case "tools/call":
			params := request["params"].(map[string]interface{})
			toolName := params["name"].(string)
			var args map[string]interface{}
			if argInterface := params["arguments"]; argInterface != nil {
				args = argInterface.(map[string]interface{})
			}

			switch toolName {
			case "test/echo":
				message := ""
				if args != nil && args["message"] != nil {
					message = args["message"].(string)
				}
				result = map[string]interface{}{
					"response": fmt.Sprintf("Echo: %s", message),
				}
			case "test/status":
				result = map[string]interface{}{
					"instanceID": instanceID,
					"port":       port,
					"healthy":    true,
				}
			default:
				rpcError = map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
				}
			}

		case "resources/list":
			result = map[string]interface{}{
				"resources": []map[string]interface{}{
					{
						"uri":         "test://status",
						"name":        "Instance Status",
						"description": "Current instance status",
						"mimeType":    "application/json",
					},
				},
			}

		case "prompts/list":
			result = map[string]interface{}{
				"prompts": []map[string]interface{}{
					{
						"name":        "test_prompt",
						"description": "A test prompt",
					},
				},
			}

		default:
			rpcError = map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			}
		}

		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
		}

		if rpcError != nil {
			response["error"] = rpcError
		} else {
			response["result"] = result
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Start server on specific port
	server := httptest.NewUnstartedServer(mux)
	server.Listener.Close()
	server.Listener, _ = net.Listen("tcp", fmt.Sprintf(":%d", port))
	server.Start()

	return server
}

// TestMultipleInstances tests hub handling of multiple instances
func TestMultipleInstances(t *testing.T) {
	// Create temporary instances directory
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	require.NoError(t, os.MkdirAll(instancesDir, 0755))

	// Create systems
	disc, err := discovery.New(instancesDir)
	require.NoError(t, err)
	defer disc.Stop()

	connMgr := mcp.NewConnectionManager()
	defer connMgr.Stop()

	sessionMgr := mcp.NewSessionManager()

	// Start discovery
	disc.Start()
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		for _, inst := range instances {
			connMgr.RegisterInstance(inst)
		}
	})

	// Create multiple mock instances
	instances := []struct {
		id   string
		port int
	}{
		{"instance-1", 8881},
		{"instance-2", 8882},
		{"instance-3", 8883},
	}

	servers := make([]*httptest.Server, 0)
	for _, inst := range instances {
		server := createMockInstanceServer(t, inst.id, inst.port)
		servers = append(servers, server)
		defer server.Close()

		// Register instance
		instance := &discovery.Instance{
			ID:        inst.id,
			Name:      fmt.Sprintf("Instance %s", inst.id),
			Directory: fmt.Sprintf("/test/%s", inst.id),
			Port:      inst.port,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = os.Getpid()

		require.NoError(t, discovery.RegisterInstance(instancesDir, instance))
	}

	// Wait for discovery and connection
	time.Sleep(500 * time.Millisecond)

	// Verify all instances are connected
	connections := connMgr.ListInstances()
	assert.Len(t, connections, 3)

	activeCount := 0
	for _, conn := range connections {
		if conn.State == mcp.StateActive {
			activeCount++
		}
	}
	assert.Equal(t, 3, activeCount)

	// Test session switching between instances
	sessionID := "test-session"
	sessionMgr.CreateSession(sessionID, nil)

	// Connect to first instance
	err = connMgr.ConnectSession(sessionID, "instance-1")
	require.NoError(t, err)

	client := connMgr.GetClient(sessionID)
	require.NotNil(t, client)

	// Call tool on first instance
	result, err := client.CallTool(context.Background(), "test/status", nil)
	require.NoError(t, err)

	var status map[string]interface{}
	json.Unmarshal(result, &status)
	assert.Equal(t, "instance-1", status["instanceID"])

	// Switch to second instance
	err = connMgr.ConnectSession(sessionID, "instance-2")
	require.NoError(t, err)

	// Call tool on second instance
	result, err = client.CallTool(context.Background(), "test/status", nil)
	require.NoError(t, err)

	json.Unmarshal(result, &status)
	assert.Equal(t, "instance-2", status["instanceID"])
}

// TestInstanceFailureAndRecovery tests handling of instance failures
func TestInstanceFailureAndRecovery(t *testing.T) {
	// Create systems
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	require.NoError(t, os.MkdirAll(instancesDir, 0755))

	disc, err := discovery.New(instancesDir)
	require.NoError(t, err)
	defer disc.Stop()

	connMgr := mcp.NewConnectionManager()
	defer connMgr.Stop()

	healthMon := mcp.NewHealthMonitor(connMgr, &mcp.HealthMonitorConfig{
		PingInterval: 500 * time.Millisecond,
		PingTimeout:  200 * time.Millisecond,
		MaxFailures:  2,
	})
	defer healthMon.Stop()

	// Track health events
	var unhealthyCount, recoveredCount, deadCount int
	healthMon.SetCallbacks(
		func(id string, status *mcp.HealthStatus) { unhealthyCount++ },
		func(id string, status *mcp.HealthStatus) { recoveredCount++ },
		func(id string, status *mcp.HealthStatus) { deadCount++ },
	)

	disc.Start()
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		for _, inst := range instances {
			connMgr.RegisterInstance(inst)
		}
	})

	// Create a mock instance that can be stopped
	var serverStopped bool
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if serverStopped {
			// Simulate network failure
			time.Sleep(300 * time.Millisecond)
			return
		}

		// Normal response
		var request map[string]interface{}
		json.NewDecoder(r.Body).Decode(&request)

		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result": map[string]interface{}{
				"pong": time.Now().Unix(),
			},
		}

		json.NewEncoder(w).Encode(response)
	})

	server := httptest.NewUnstartedServer(mux)
	server.Listener.Close()
	server.Listener, _ = net.Listen("tcp", ":8890")
	server.Start()
	defer server.Close()

	// Register instance
	instance := &discovery.Instance{
		ID:        "failing-instance",
		Name:      "Failing Instance",
		Directory: "/test/failing",
		Port:      8890,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = os.Getpid()

	require.NoError(t, discovery.RegisterInstance(instancesDir, instance))

	// Wait for connection
	time.Sleep(300 * time.Millisecond)

	// Start health monitoring
	healthMon.Start()

	// Verify healthy
	time.Sleep(600 * time.Millisecond)
	status, err := healthMon.GetHealthStatus("failing-instance")
	require.NoError(t, err)
	assert.True(t, status.IsHealthy)

	// Simulate failure
	serverStopped = true

	// Wait for unhealthy state
	time.Sleep(1500 * time.Millisecond)

	status, err = healthMon.GetHealthStatus("failing-instance")
	require.NoError(t, err)
	assert.False(t, status.IsHealthy)
	assert.GreaterOrEqual(t, unhealthyCount, 1)

	// Recover
	serverStopped = false

	// Wait for recovery
	time.Sleep(1000 * time.Millisecond)

	status, err = healthMon.GetHealthStatus("failing-instance")
	require.NoError(t, err)
	assert.True(t, status.IsHealthy)
	assert.GreaterOrEqual(t, recoveredCount, 1)
}
