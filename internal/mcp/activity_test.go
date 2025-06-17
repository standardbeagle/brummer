package mcp

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
)

// Test that MCP activity events are published
func TestMCPActivityTracking(t *testing.T) {
	server := setupTestServer(t)
	
	// Track events
	var activityEvents []events.Event
	var connectionEvents []events.Event
	
	server.eventBus.Subscribe(events.MCPActivity, func(e events.Event) {
		activityEvents = append(activityEvents, e)
	})
	
	server.eventBus.Subscribe(events.MCPConnected, func(e events.Event) {
		connectionEvents = append(connectionEvents, e)
	})
	
	server.eventBus.Subscribe(events.MCPDisconnected, func(e events.Event) {
		connectionEvents = append(connectionEvents, e)
	})
	
	// Send a request
	msg := makeJSONRPCRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
	}, 1)
	
	response := sendRequest(t, server, msg)
	assert.Nil(t, response.Error)
	
	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
	
	// Check activity event was published
	assert.Len(t, activityEvents, 1)
	
	activity := activityEvents[0]
	assert.Equal(t, events.MCPActivity, activity.Type)
	assert.Equal(t, "initialize", activity.Data["method"])
	assert.NotEmpty(t, activity.Data["sessionId"])
	assert.NotEmpty(t, activity.Data["duration"])
	assert.NotEmpty(t, activity.Data["response"])
	
	// Send another request
	msg2 := makeJSONRPCRequest("tools/list", nil, 2)
	response2 := sendRequest(t, server, msg2)
	assert.Nil(t, response2.Error)
	
	// Wait for events
	time.Sleep(100 * time.Millisecond)
	
	// Should have 2 activity events now
	assert.Len(t, activityEvents, 2)
	
	activity2 := activityEvents[1]
	assert.Equal(t, "tools/list", activity2.Data["method"])
}

// Test SSE connection tracking
func TestMCPConnectionTracking(t *testing.T) {
	t.Skip("SSE connection tracking requires full HTTP context")
}