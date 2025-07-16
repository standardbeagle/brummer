package mcp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/mcp"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestPhase1MCPServerImplementation tests basic MCP server functionality
func TestPhase1MCPServerImplementation(t *testing.T) {
	// Setup
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000)
	processMgr, err := process.NewManager(".", eventBus, false)
	if err != nil {
		t.Fatalf("Failed to create process manager: %v", err)
	}
	proxyServer := proxy.NewServer(0, eventBus) // Use random port

	// Create MCP server
	server := mcp.NewStreamableServer(0, processMgr, logStore, proxyServer, eventBus)

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			t.Logf("Server stopped: %v", err)
		}
	}()
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	port := server.GetPort()

	// Test 1: Initialize
	t.Run("Initialize", func(t *testing.T) {
		resp := callMCPMethod(t, port, "initialize", map[string]interface{}{
			"protocolVersion": "1.0",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0",
			},
		})

		// Verify response
		if resp["protocolVersion"] != "1.0" {
			t.Errorf("Expected protocol version 1.0, got %v", resp["protocolVersion"])
		}

		capabilities, ok := resp["capabilities"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing capabilities in response")
		}

		if capabilities["tools"] == nil {
			t.Error("Missing tools capability")
		}
	})

	// Test 2: List Tools
	t.Run("ListTools", func(t *testing.T) {
		resp := callMCPMethod(t, port, "tools/list", map[string]interface{}{})

		tools, ok := resp["tools"].([]interface{})
		if !ok {
			t.Fatal("Expected tools array in response")
		}

		// Verify essential tools exist
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolMap := tool.(map[string]interface{})
			name := toolMap["name"].(string)
			toolNames[name] = true
		}

		expectedTools := []string{
			"scripts/list",
			"scripts/run",
			"scripts/stop",
			"logs/search",
			"processes/list",
		}

		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("Missing expected tool: %s", expected)
			}
		}
	})

	// Test 3: List Resources
	t.Run("ListResources", func(t *testing.T) {
		resp := callMCPMethod(t, port, "resources/list", map[string]interface{}{})

		resources, ok := resp["resources"].([]interface{})
		if !ok {
			t.Fatal("Expected resources array in response")
		}

		// Verify essential resources exist
		resourceURIs := make(map[string]bool)
		for _, resource := range resources {
			resMap := resource.(map[string]interface{})
			uri := resMap["uri"].(string)
			resourceURIs[uri] = true
		}

		expectedResources := []string{
			"logs://recent",
			"logs://errors",
			"processes://active",
			"scripts://available",
		}

		for _, expected := range expectedResources {
			if !resourceURIs[expected] {
				t.Errorf("Missing expected resource: %s", expected)
			}
		}
	})

	// Test 4: Call Tool - processes/list
	t.Run("CallTool_ProcessesList", func(t *testing.T) {
		resp := callMCPMethod(t, port, "tools/call", map[string]interface{}{
			"name":      "processes/list",
			"arguments": map[string]interface{}{},
		})

		// Should return content array
		content, ok := resp["content"].([]interface{})
		if !ok {
			t.Fatal("Expected content array in response")
		}

		if len(content) == 0 {
			t.Error("Expected at least one content item")
		}
	})

	// Test 5: Read Resource
	t.Run("ReadResource", func(t *testing.T) {
		resp := callMCPMethod(t, port, "resources/read", map[string]interface{}{
			"uri": "logs://recent",
		})

		// Should return contents
		contents, ok := resp["contents"].([]interface{})
		if !ok {
			t.Fatal("Expected contents array in response")
		}

		// Even empty logs should return valid structure
		if len(contents) > 0 {
			firstItem := contents[0].(map[string]interface{})
			if firstItem["uri"] == nil || firstItem["text"] == nil {
				t.Error("Invalid resource content structure")
			}
		}
	})
}

// TestPhase2InstanceDiscovery tests instance discovery functionality
func TestPhase2InstanceDiscovery(t *testing.T) {
	// Create temporary instances directory
	tmpDir := t.TempDir()
	instancesDir := filepath.Join(tmpDir, "instances")
	os.MkdirAll(instancesDir, 0755)

	// Test 1: Register Instance
	t.Run("RegisterInstance", func(t *testing.T) {
		instance := &discovery.Instance{
			ID:        "test-instance-1",
			Name:      "Test Instance",
			Directory: "/test/dir",
			Port:      7778,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 12345
		instance.ProcessInfo.Executable = "/usr/bin/brum"

		err := discovery.RegisterInstance(instancesDir, instance)
		if err != nil {
			t.Fatalf("Failed to register instance: %v", err)
		}

		// Verify file was created
		instanceFile := filepath.Join(instancesDir, "test-instance-1.json")
		if _, err := os.Stat(instanceFile); os.IsNotExist(err) {
			t.Error("Instance file was not created")
		}

		// Verify permissions
		info, _ := os.Stat(instanceFile)
		mode := info.Mode()
		if mode.Perm() != 0600 {
			t.Errorf("Expected file permissions 0600, got %v", mode.Perm())
		}
	})

	// Test 2: Discovery System
	t.Run("DiscoverySystem", func(t *testing.T) {
		disc, err := discovery.New(instancesDir)
		if err != nil {
			t.Fatalf("Failed to create discovery system: %v", err)
		}
		defer disc.Stop()

		// Set up callback to track discovered instances
		discovered := make(chan *discovery.Instance, 1)
		disc.OnUpdate(func(instances map[string]*discovery.Instance) {
			for _, inst := range instances {
				select {
				case discovered <- inst:
				default:
				}
			}
		})

		// Start discovery
		disc.Start()
		time.Sleep(100 * time.Millisecond)

		// Register a new instance
		instance := &discovery.Instance{
			ID:        "test-instance-2",
			Name:      "Test Instance 2",
			Directory: "/test/dir2",
			Port:      7779,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 12346

		err = discovery.RegisterInstance(instancesDir, instance)
		if err != nil {
			t.Fatalf("Failed to register instance: %v", err)
		}

		// Wait for discovery
		select {
		case inst := <-discovered:
			if inst.ID != "test-instance-2" {
				t.Errorf("Expected instance ID test-instance-2, got %s", inst.ID)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for instance discovery")
		}
	})

	// Test 3: Stale Instance Cleanup
	t.Run("StaleInstanceCleanup", func(t *testing.T) {
		// Create a stale instance (old timestamp)
		staleInstance := &discovery.Instance{
			ID:        "stale-instance",
			Name:      "Stale Instance",
			Directory: "/test/stale",
			Port:      7780,
			StartedAt: time.Now().Add(-2 * time.Hour),
			LastPing:  time.Now().Add(-2 * time.Hour), // Very old
		}
		staleInstance.ProcessInfo.PID = 99999 // Non-existent PID

		err := discovery.RegisterInstance(instancesDir, staleInstance)
		if err != nil {
			t.Fatalf("Failed to register stale instance: %v", err)
		}

		disc, err := discovery.New(instancesDir)
		if err != nil {
			t.Fatalf("Failed to create discovery: %v", err)
		}
		defer disc.Stop()

		// Cleanup stale instances
		err = disc.CleanupStaleInstances()
		if err != nil {
			t.Errorf("Failed to cleanup stale instances: %v", err)
		}

		// Verify stale instance was removed
		staleFile := filepath.Join(instancesDir, "stale-instance.json")
		if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
			t.Error("Stale instance file was not removed")
		}
	})
}

// TestPhase3ConnectionManagement tests connection management
func TestPhase3ConnectionManagement(t *testing.T) {
	// This is already covered by connection_manager_test.go
	// Just verify the integration works

	t.Run("ConnectionManagerIntegration", func(t *testing.T) {
		connMgr := mcp.NewConnectionManager()
		defer connMgr.Stop()

		// Register a test instance
		instance := &discovery.Instance{
			ID:        "conn-test",
			Name:      "Connection Test",
			Directory: "/test",
			Port:      7781,
			StartedAt: time.Now(),
			LastPing:  time.Now(),
		}
		instance.ProcessInfo.PID = 12347

		err := connMgr.RegisterInstance(instance)
		if err != nil {
			t.Fatalf("Failed to register instance: %v", err)
		}

		// List instances
		instances := connMgr.ListInstances()
		if len(instances) != 1 {
			t.Errorf("Expected 1 instance, got %d", len(instances))
		}

		// Verify state
		if instances[0].State.String() != "connecting" && instances[0].State.String() != "retrying" {
			t.Errorf("Expected connecting or retrying state, got %s", instances[0].State)
		}
	})
}

// TestEndToEndScenario tests a complete user workflow
func TestEndToEndScenario(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create test directory with package.json
	testDir := t.TempDir()
	packageJSON := `{
		"name": "test-project",
		"scripts": {
			"dev": "echo 'Development server started' && sleep 1",
			"test": "echo 'Running tests' && exit 0"
		}
	}`
	err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(testDir)
	defer os.Chdir(oldDir)

	// Setup components
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000)
	processMgr, err := process.NewManager(testDir, eventBus, true)
	if err != nil {
		t.Fatalf("Failed to create process manager: %v", err)
	}
	defer processMgr.Cleanup()

	// Set up log callback
	processMgr.AddLogCallback(func(processID, line string, isError bool) {
		logStore.Add(processID, "test", line, isError)
	})

	// Create and start MCP server
	server := mcp.NewStreamableServer(0, processMgr, logStore, nil, eventBus)
	go server.Start()
	defer server.Stop()
	time.Sleep(100 * time.Millisecond)
	port := server.GetPort()

	// Test 1: List available scripts
	t.Run("ListScripts", func(t *testing.T) {
		resp := callMCPMethod(t, port, "tools/call", map[string]interface{}{
			"name":      "scripts/list",
			"arguments": map[string]interface{}{},
		})

		content := resp["content"].([]interface{})
		textContent := content[0].(map[string]interface{})["text"].(string)

		// Parse JSON response
		var scripts map[string]interface{}
		json.Unmarshal([]byte(textContent), &scripts)

		if scripts["dev"] == nil || scripts["test"] == nil {
			t.Error("Expected dev and test scripts to be listed")
		}
	})

	// Test 2: Run a script
	t.Run("RunScript", func(t *testing.T) {
		resp := callMCPMethod(t, port, "tools/call", map[string]interface{}{
			"name": "scripts/run",
			"arguments": map[string]interface{}{
				"script": "test",
			},
		})

		// Should start successfully
		content := resp["content"].([]interface{})
		textContent := content[0].(map[string]interface{})["text"].(string)

		if textContent == "" {
			t.Error("Expected process ID in response")
		}

		// Wait for process to complete
		time.Sleep(2 * time.Second)
	})

	// Test 3: Check logs
	t.Run("CheckLogs", func(t *testing.T) {
		resp := callMCPMethod(t, port, "resources/read", map[string]interface{}{
			"uri": "logs://recent",
		})

		contents := resp["contents"].([]interface{})
		if len(contents) == 0 {
			t.Error("Expected log entries")
		}

		// Verify we captured the output
		found := false
		for _, item := range contents {
			content := item.(map[string]interface{})
			text := content["text"].(string)
			if text == "Running tests" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Did not find expected log output")
		}
	})
}

// Helper function to call MCP methods
func callMCPMethod(t *testing.T, port int, method string, params interface{}) map[string]interface{} {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("http://localhost:%d/mcp", port),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["error"] != nil {
		t.Fatalf("RPC error: %v", result["error"])
	}

	return result["result"].(map[string]interface{})
}
