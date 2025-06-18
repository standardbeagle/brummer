package mcp

import (
	"sync"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
)

// Test that MCP activity events are published
func TestMCPActivityTracking(t *testing.T) {
	server := setupTestServer(t)

	// Track events
	var mu sync.Mutex
	var activityEvents []events.Event
	var connectionEvents []events.Event

	server.eventBus.Subscribe(events.MCPActivity, func(e events.Event) {
		mu.Lock()
		activityEvents = append(activityEvents, e)
		mu.Unlock()
	})

	server.eventBus.Subscribe(events.MCPConnected, func(e events.Event) {
		mu.Lock()
		connectionEvents = append(connectionEvents, e)
		mu.Unlock()
	})

	server.eventBus.Subscribe(events.MCPDisconnected, func(e events.Event) {
		mu.Lock()
		connectionEvents = append(connectionEvents, e)
		mu.Unlock()
	})

	// Send a request
	msg := makeJSONRPCRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
	}, 1)

	response := sendRequest(t, server, msg)
	assert.Nil(t, response.Error)

	// Wait for events to be processed
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(activityEvents) >= 1
	}, 500*time.Millisecond, 10*time.Millisecond, "Expected 1 activity event")

	mu.Lock()
	activity := activityEvents[0]
	mu.Unlock()
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
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(activityEvents) >= 2
	}, 500*time.Millisecond, 10*time.Millisecond, "Expected 2 activity events")

	mu.Lock()
	activity2 := activityEvents[1]
	mu.Unlock()
	assert.Equal(t, "tools/list", activity2.Data["method"])
}

// Test SSE connection tracking
func TestMCPConnectionTracking(t *testing.T) {
	t.Skip("SSE connection tracking requires full HTTP context")
}
