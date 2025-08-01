package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestMCPServer(t *testing.T) *MCPServer {
	eventBus := events.NewEventBus()
	processMgr, err := process.NewManager("/tmp", eventBus, false)
	require.NoError(t, err)

	logStore := logs.NewStore(10000, nil)
	proxyServer := &proxy.Server{}

	server := NewMCPServer(7777, processMgr, logStore, proxyServer, eventBus)
	return server
}

func TestQueueSendTool(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Test queue_send tool
	args := json.RawMessage(`{
		"channel": "test-channel",
		"type": "test-message",
		"payload": {"data": "test"},
		"ttl": 3600
	}`)

	result, err := server.tools["queue_send"].Handler(args)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]map[string]interface{})
	assert.Contains(t, content[0]["text"].(string), "Message sent to channel 'test-channel'")

	msg := resultMap["message"].(*Message)
	assert.Equal(t, "test-channel", msg.Channel)
	assert.Equal(t, "test-message", msg.Type)
	assert.Equal(t, 3600, msg.TTL)
}

func TestQueueReceiveTool(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Send a message first
	_, err := server.messageQueue.Send("test-channel", "test", json.RawMessage(`{"data": "test"}`), 3600)
	require.NoError(t, err)

	// Test queue_receive tool
	args := json.RawMessage(`{
		"channel": "test-channel",
		"limit": 10
	}`)

	result, err := server.tools["queue_receive"].Handler(args)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]map[string]interface{})
	assert.Contains(t, content[0]["text"].(string), "Retrieved 1 messages from channel 'test-channel'")

	messages := resultMap["messages"].([]Message)
	assert.Len(t, messages, 1)
	assert.Equal(t, "test-channel", messages[0].Channel)
}

func TestQueueReceiveToolBlocking(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Test blocking receive with timeout
	args := json.RawMessage(`{
		"channel": "empty-channel",
		"blocking": true,
		"timeout": 1
	}`)

	start := time.Now()
	result, err := server.tools["queue_receive"].Handler(args)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.True(t, duration >= 1*time.Second)

	resultMap := result.(map[string]interface{})
	messages := resultMap["messages"].([]Message)
	assert.Len(t, messages, 0)
}

func TestQueueUnsubscribeTool(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Create a subscription first
	sub, err := server.messageQueue.Subscribe("test-channel")
	require.NoError(t, err)

	// Test queue_unsubscribe tool
	args := json.RawMessage(`{
		"subscription_id": "` + sub.ID + `"
	}`)

	result, err := server.tools["queue_unsubscribe"].Handler(args)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]map[string]interface{})
	assert.Contains(t, content[0]["text"].(string), "Unsubscribed from subscription ID: "+sub.ID)
}

func TestQueueListChannelsTool(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Create some channels
	_, err := server.messageQueue.Send("channel1", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)
	_, err = server.messageQueue.Send("channel2", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)

	// Test queue_list_channels tool
	args := json.RawMessage(`{}`)

	result, err := server.tools["queue_list_channels"].Handler(args)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]map[string]interface{})
	assert.Contains(t, content[0]["text"].(string), "Found 2 active channels")

	channels := resultMap["channels"].([]string)
	assert.Len(t, channels, 2)
	assert.Contains(t, channels, "channel1")
	assert.Contains(t, channels, "channel2")
}

func TestQueueStatsTool(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Create some data
	_, err := server.messageQueue.Send("channel1", "test", json.RawMessage(`{}`), 3600)
	require.NoError(t, err)
	sub, err := server.messageQueue.Subscribe("channel1")
	require.NoError(t, err)
	defer server.messageQueue.Unsubscribe(sub.ID)

	// Test queue_stats tool
	args := json.RawMessage(`{}`)

	result, err := server.tools["queue_stats"].Handler(args)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	content := resultMap["content"].([]map[string]interface{})
	assert.Contains(t, content[0]["text"].(string), "Queue Statistics")
	assert.Contains(t, content[0]["text"].(string), "Total Messages: 1")
	assert.Contains(t, content[0]["text"].(string), "Total Channels: 1")
	assert.Contains(t, content[0]["text"].(string), "Total Subscriptions: 1")

	stats := resultMap["stats"].(map[string]interface{})
	assert.Equal(t, int64(1), stats["total_messages"])
	assert.Equal(t, int64(1), stats["total_channels"])
	assert.Equal(t, int64(1), stats["total_subscriptions"])
}

func TestQueueToolsInvalidParameters(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	tests := []struct {
		name   string
		tool   string
		args   json.RawMessage
		errMsg string
	}{
		{
			name:   "queue_send missing channel",
			tool:   "queue_send",
			args:   json.RawMessage(`{"payload": {}}`),
			errMsg: "channel is required",
		},
		{
			name:   "queue_receive missing channel",
			tool:   "queue_receive",
			args:   json.RawMessage(`{}`),
			errMsg: "channel is required",
		},
		{
			name:   "queue_unsubscribe missing id",
			tool:   "queue_unsubscribe",
			args:   json.RawMessage(`{}`),
			errMsg: "subscription_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.tools[tt.tool].Handler(tt.args)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestWebSocketIntegration(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Simulate WebSocket message event
	msgData := map[string]interface{}{
		"channel": "ws-channel",
		"type":    "ws-message",
		"payload": map[string]interface{}{"test": "data"},
		"ttl":     float64(3600),
	}

	server.eventBus.Publish(events.Event{
		Type: events.EventType("queue.message"),
		Data: msgData,
	})

	// Give time for event processing
	time.Sleep(100 * time.Millisecond)

	// Check message was received
	messages, err := server.messageQueue.Receive("ws-channel", 10, false, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "ws-channel", messages[0].Channel)
	assert.Equal(t, "ws-message", messages[0].Type)
}

func TestToolSchemas(t *testing.T) {
	server := setupTestMCPServer(t)
	defer server.Stop()

	// Verify all queue tools have proper schemas
	queueTools := []string{
		"queue_send",
		"queue_receive",
		"queue_subscribe",
		"queue_unsubscribe",
		"queue_list_channels",
		"queue_stats",
	}

	for _, toolName := range queueTools {
		tool, exists := server.tools[toolName]
		assert.True(t, exists, "Tool %s should exist", toolName)
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)

		// Verify schema is valid JSON
		var schema map[string]interface{}
		err := json.Unmarshal(tool.InputSchema, &schema)
		assert.NoError(t, err, "Tool %s should have valid JSON schema", toolName)
		assert.Equal(t, "object", schema["type"])

		// All queue tools use standard handlers
		assert.NotNil(t, tool.Handler)
	}
}

