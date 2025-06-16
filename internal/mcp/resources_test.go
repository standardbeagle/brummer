package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test resource registration
func TestRegisterResources(t *testing.T) {
	server := setupTestServer(t)
	
	// Verify resources are registered
	assert.Greater(t, len(server.resources), 0)
	
	// Check for expected resources
	expectedResources := []string{
		"logs://recent",
		"logs://errors",
		"telemetry://sessions",
		"telemetry://errors",
		"telemetry://console-errors",
		"proxy://requests",
		"proxy://mappings",
		"processes://active",
		"scripts://available",
	}
	
	for _, resourceURI := range expectedResources {
		_, exists := server.resources[resourceURI]
		assert.True(t, exists, "Resource %s should be registered", resourceURI)
	}
}

// Test resources/list
func TestResourcesList(t *testing.T) {
	server := setupTestServer(t)
	
	msg := makeJSONRPCRequest("resources/list", nil, 1)
	response := sendRequest(t, server, msg)
	
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)
	
	result := response.Result.(map[string]interface{})
	resources := result["resources"].([]interface{})
	
	// Check that we have resources registered
	assert.Greater(t, len(resources), 0)
	
	// Verify resource structure
	for _, resourceInterface := range resources {
		resource := resourceInterface.(map[string]interface{})
		assert.NotEmpty(t, resource["uri"])
		assert.NotEmpty(t, resource["name"])
		assert.NotEmpty(t, resource["description"])
		assert.NotEmpty(t, resource["mimeType"])
	}
}

// Test resources/read
func TestResourcesRead(t *testing.T) {
	server := setupTestServer(t)
	
	t.Run("read logs://recent", func(t *testing.T) {
		// Add some test logs
		server.logStore.Add("test-process", "test", "Test log 1", false)
		server.logStore.Add("test-process", "test", "Test log 2", false)
		
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "logs://recent",
		}, 1)
		
		response := sendRequest(t, server, msg)
		
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
		
		result := response.Result.(map[string]interface{})
		contents := result["contents"].([]interface{})
		
		// The contents is wrapped in a structure with uri, mimeType, and text
		assert.Len(t, contents, 1)
		
		contentWrapper := contents[0].(map[string]interface{})
		assert.Equal(t, "logs://recent", contentWrapper["uri"])
		assert.Equal(t, "application/json", contentWrapper["mimeType"])
		
		// Parse the text field which contains the actual JSON data
		textData := contentWrapper["text"].(string)
		var logs []interface{}
		err := json.Unmarshal([]byte(textData), &logs)
		require.NoError(t, err)
		
		assert.GreaterOrEqual(t, len(logs), 2)
		
		// Verify log structure
		for _, logInterface := range logs {
			logEntry := logInterface.(map[string]interface{})
			assert.NotEmpty(t, logEntry["id"])
			assert.NotEmpty(t, logEntry["processId"])
			assert.NotEmpty(t, logEntry["content"])
			assert.NotEmpty(t, logEntry["timestamp"])
			assert.Contains(t, logEntry, "isError")
		}
	})
	
	t.Run("read logs://errors", func(t *testing.T) {
		// Add an error log
		server.logStore.Add("test-process", "test", "Error: Something failed", true)
		
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "logs://errors",
		}, 1)
		
		response := sendRequest(t, server, msg)
		
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
		
		result := response.Result.(map[string]interface{})
		contents := result["contents"].([]interface{})
		
		// The contents is wrapped in a structure
		assert.Len(t, contents, 1)
		contentWrapper := contents[0].(map[string]interface{})
		
		// Parse the text field
		textData := contentWrapper["text"].(string)
		var logs []interface{}
		err := json.Unmarshal([]byte(textData), &logs)
		require.NoError(t, err)
		
		// All logs should be errors
		for _, logInterface := range logs {
			logEntry := logInterface.(map[string]interface{})
			assert.True(t, logEntry["isError"].(bool))
		}
	})
	
	t.Run("read processes://active", func(t *testing.T) {
		// Skip starting a process since it requires real commands
		
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "processes://active",
		}, 1)
		
		response := sendRequest(t, server, msg)
		
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
		
		result := response.Result.(map[string]interface{})
		contents := result["contents"].([]interface{})
		
		// The contents is wrapped in a structure
		assert.Len(t, contents, 1)
		contentWrapper := contents[0].(map[string]interface{})
		
		// Parse the text field
		textData := contentWrapper["text"].(string)
		var processes []interface{}
		err := json.Unmarshal([]byte(textData), &processes)
		require.NoError(t, err)
		
		// May be empty if no processes are running
		assert.NotNil(t, processes)
		
		// Verify process structure
		for _, procInterface := range processes {
			proc := procInterface.(map[string]interface{})
			assert.NotEmpty(t, proc["id"])
			assert.NotEmpty(t, proc["name"])
			assert.NotEmpty(t, proc["status"])
			assert.NotEmpty(t, proc["startTime"])
		}
	})
	
	t.Run("read scripts://available", func(t *testing.T) {
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "scripts://available",
		}, 1)
		
		response := sendRequest(t, server, msg)
		
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
		
		result := response.Result.(map[string]interface{})
		contents := result["contents"].([]interface{})
		
		// The contents is wrapped in a structure
		assert.Len(t, contents, 1)
		contentWrapper := contents[0].(map[string]interface{})
		
		// Parse the text field
		textData := contentWrapper["text"].(string)
		
		// The scripts resource returns an object, not an array
		var scriptsData map[string]interface{}
		err := json.Unmarshal([]byte(textData), &scriptsData)
		require.NoError(t, err)
		
		// Verify the scripts object structure
		if scripts, ok := scriptsData["scripts"]; ok {
			scriptsArray := scripts.([]interface{})
			for _, scriptInterface := range scriptsArray {
				script := scriptInterface.(map[string]interface{})
				assert.NotEmpty(t, script["name"])
				assert.NotEmpty(t, script["command"])
			}
		}
	})
	
	t.Run("read proxy://requests", func(t *testing.T) {
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "proxy://requests",
		}, 1)
		
		response := sendRequest(t, server, msg)
		
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Result)
		
		result := response.Result.(map[string]interface{})
		contents, ok := result["contents"].([]interface{})
		assert.True(t, ok, "contents should be an array")
		
		// The contents is wrapped in a structure
		assert.Len(t, contents, 1)
		contentWrapper := contents[0].(map[string]interface{})
		assert.Equal(t, "proxy://requests", contentWrapper["uri"])
	})
	
	t.Run("invalid resource URI", func(t *testing.T) {
		msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
			"uri": "invalid://resource",
		}, 1)
		
		response := sendRequest(t, server, msg)
		
		assert.NotNil(t, response.Error)
		assert.Equal(t, -32602, response.Error.Code)
		assert.Contains(t, response.Error.Message, "Resource not found")
	})
}

// Test resource subscriptions with updates
func TestResourceSubscriptionsWithUpdates(t *testing.T) {
	server := setupTestServer(t)
	sessionID := "update-test-123"
	
	// Set up update handler
	updateChan := make(chan ResourceUpdate, 10)
	server.registerResourceUpdateHandler(sessionID, updateChan)
	defer server.unregisterResourceUpdateHandler(sessionID)
	
	// Subscribe to logs://recent
	server.subscriptionsMu.Lock()
	server.subscriptions[sessionID] = map[string]bool{
		"logs://recent": true,
	}
	server.subscriptionsMu.Unlock()
	
	// Add a log which should trigger an update
	server.logStore.Add("test-process", "test", "New log entry", false)
	
	// Trigger the update notification
	server.notifyResourceUpdate("logs://recent", server.getRecentLogs(100))
	
	// Check for update
	select {
	case update := <-updateChan:
		assert.Equal(t, "logs://recent", update.URI)
		assert.NotNil(t, update.Contents)
		
		// Verify the update contains logs
		logs, ok := update.Contents.([]interface{})
		assert.True(t, ok, "Contents should be a log array")
		assert.Greater(t, len(logs), 0)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for resource update")
	}
}

// Test resource subscription and unsubscription flow
func TestResourceSubscriptionFlow(t *testing.T) {
	server := setupTestServer(t)
	sessionID := "flow-test-123"
	
	// Test subscribing to multiple resources
	resources := []string{"logs://recent", "logs://errors", "processes://active"}
	
	for _, uri := range resources {
		msg := makeJSONRPCRequest("resources/subscribe", map[string]interface{}{
			"uri": uri,
		}, 1)
		
		response := sendRequest(t, server, msg)
		assert.Nil(t, response.Error)
		
		// Verify subscription is recorded
		server.subscriptionsMu.RLock()
		subs := server.subscriptions[sessionID]
		server.subscriptionsMu.RUnlock()
		
		assert.True(t, subs[uri], "Should be subscribed to %s", uri)
	}
	
	// Verify all subscriptions are active
	server.subscriptionsMu.RLock()
	subs := server.subscriptions[sessionID]
	server.subscriptionsMu.RUnlock()
	
	assert.Len(t, subs, len(resources))
	
	// Unsubscribe from one resource
	unsubMsg := makeJSONRPCRequest("resources/unsubscribe", map[string]interface{}{
		"uri": "logs://errors",
	}, 2)
	
	response := sendRequest(t, server, unsubMsg)
	assert.Nil(t, response.Error)
	
	// Verify unsubscription
	server.subscriptionsMu.RLock()
	subs = server.subscriptions[sessionID]
	server.subscriptionsMu.RUnlock()
	
	assert.False(t, subs["logs://errors"])
	assert.True(t, subs["logs://recent"])
	assert.True(t, subs["processes://active"])
}

// Test concurrent resource reads
func TestConcurrentResourceReads(t *testing.T) {
	server := setupTestServer(t)
	
	// Add some test data
	for i := 0; i < 10; i++ {
		server.logStore.Add("test", "test", fmt.Sprintf("Log %d", i), false)
	}
	
	numReads := 20
	results := make(chan error, numReads)
	
	// Perform concurrent reads
	for i := 0; i < numReads; i++ {
		go func(id int) {
			msg := makeJSONRPCRequest("resources/read", map[string]interface{}{
				"uri": "logs://recent",
			}, id)
			
			response := sendRequest(t, server, msg)
			if response.Error != nil {
				results <- fmt.Errorf("read %d failed: %v", id, response.Error)
			} else {
				results <- nil
			}
		}(i)
	}
	
	// Collect results
	for i := 0; i < numReads; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// Test resource read with different mime types
func TestResourceMimeTypes(t *testing.T) {
	server := setupTestServer(t)
	
	// Get resource list to check mime types
	msg := makeJSONRPCRequest("resources/list", nil, 1)
	response := sendRequest(t, server, msg)
	
	require.Nil(t, response.Error)
	
	result := response.Result.(map[string]interface{})
	resources := result["resources"].([]interface{})
	
	// Map of expected mime types
	expectedMimeTypes := map[string]string{
		"logs://recent":           "application/json",
		"logs://errors":          "application/json",
		"telemetry://sessions":   "application/json",
		"proxy://requests":       "application/json",
		"processes://active":     "application/json",
		"scripts://available":    "application/json",
	}
	
	for _, resourceInterface := range resources {
		resource := resourceInterface.(map[string]interface{})
		uri := resource["uri"].(string)
		mimeType := resource["mimeType"].(string)
		
		if expected, exists := expectedMimeTypes[uri]; exists {
			assert.Equal(t, expected, mimeType, "Resource %s should have mime type %s", uri, expected)
		}
	}
}

// Test invalid resource operations
func TestResourceErrors(t *testing.T) {
	server := setupTestServer(t)
	
	testCases := []struct {
		name        string
		method      string
		params      map[string]interface{}
		expectCode  int
		expectMsg   string
	}{
		{
			name:   "read without URI",
			method: "resources/read",
			params: map[string]interface{}{
				// missing "uri" field
			},
			expectCode: -32602,
			expectMsg:  "Invalid params",
		},
		{
			name:   "subscribe without URI",
			method: "resources/subscribe",
			params: map[string]interface{}{
				// missing "uri" field
			},
			expectCode: -32602,
			expectMsg:  "Invalid params",
		},
		{
			name:   "read non-existent resource",
			method: "resources/read",
			params: map[string]interface{}{
				"uri": "nonexistent://resource",
			},
			expectCode: -32602,
			expectMsg:  "Resource not found",
		},
		{
			name:   "subscribe to non-existent resource",
			method: "resources/subscribe",
			params: map[string]interface{}{
				"uri": "nonexistent://resource",
			},
			expectCode: -32602,
			expectMsg:  "Resource not found",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := makeJSONRPCRequest(tc.method, tc.params, 1)
			response := sendRequest(t, server, msg)
			
			assert.NotNil(t, response.Error)
			assert.Equal(t, tc.expectCode, response.Error.Code)
			assert.Contains(t, response.Error.Message, tc.expectMsg)
		})
	}
}