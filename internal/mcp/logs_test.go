package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test logs with manually added entries
func TestLogsSearchWithManualEntries(t *testing.T) {
	server := setupTestServer(t)

	// Manually add some log entries
	server.logStore.Add("test-process-1", "test-app", "Starting application", false)
	server.logStore.Add("test-process-1", "test-app", "Hello from test", false)
	server.logStore.Add("test-process-1", "test-app", "Error: Connection failed", true)
	server.logStore.Add("test-process-2", "other-app", "Other process log", false)

	// Wait a bit to ensure logs are stored
	time.Sleep(10 * time.Millisecond)

	t.Run("search for specific text", func(t *testing.T) {
		msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
			"name": "logs_search",
			"arguments": map[string]interface{}{
				"query": "Hello",
			},
		}, 1)

		response := sendRequest(t, server, msg)
		assert.Nil(t, response.Error)

		logs, ok := response.Result.([]interface{})
		assert.True(t, ok, "Result should be an array")
		assert.Equal(t, 1, len(logs), "Should find one log with 'Hello'")

		log := logs[0].(map[string]interface{})
		assert.Equal(t, "Hello from test", log["message"])
		assert.Equal(t, "test-process-1", log["processId"])
	})

	t.Run("search by process ID", func(t *testing.T) {
		msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
			"name": "logs_search",
			"arguments": map[string]interface{}{
				"query":     "log", // Match "log" in content
				"processId": "test-process-2",
			},
		}, 2)

		response := sendRequest(t, server, msg)
		assert.Nil(t, response.Error)

		logs, ok := response.Result.([]interface{})
		assert.True(t, ok, "Result should be an array")
		assert.Equal(t, 1, len(logs), "Should find one log for process 2")

		log := logs[0].(map[string]interface{})
		assert.Equal(t, "test-process-2", log["processId"])
	})

	t.Run("search for errors only", func(t *testing.T) {
		msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
			"name": "logs_search",
			"arguments": map[string]interface{}{
				"query": "Error",
				"level": "error",
			},
		}, 3)

		response := sendRequest(t, server, msg)
		assert.Nil(t, response.Error)

		logs, ok := response.Result.([]interface{})
		assert.True(t, ok, "Result should be an array")
		assert.Equal(t, 1, len(logs), "Should find one error log")

		log := logs[0].(map[string]interface{})
		assert.True(t, log["isError"].(bool))
		assert.Contains(t, log["message"], "Error")
	})
}

// Test log streaming tool structure
func TestLogsStreamTool(t *testing.T) {
	server := setupTestServer(t)

	// Verify the tool exists and is marked as streaming
	tool, exists := server.tools["logs_stream"]
	assert.True(t, exists, "logs_stream tool should exist")
	assert.True(t, tool.Streaming, "logs_stream should be marked as streaming")

	// Test that it can be called (even if streaming isn't fully implemented in tests)
	msg := makeJSONRPCRequest("tools/call", map[string]interface{}{
		"name": "logs_stream",
		"arguments": map[string]interface{}{
			"follow": false, // Don't follow, just get historical
			"limit":  10,
		},
	}, 1)

	response := sendRequest(t, server, msg)
	assert.Nil(t, response.Error, "Should not error when called")
}
