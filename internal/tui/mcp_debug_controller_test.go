package tui

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPDebugController_RaceSafety(t *testing.T) {
	// Create controller in debug mode
	controller := NewMCPDebugController(true)
	require.NotNil(t, controller)
	assert.True(t, controller.IsDebugMode())

	// Test concurrent access to connections and activities
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Start multiple goroutines that read and write concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sessionId := fmt.Sprintf("session-%d", id)

			for j := 0; j < numOperations; j++ {
				// Write operations
				if j%3 == 0 {
					// Add connection
					controller.HandleConnection(mcpConnectionMsg{
						sessionId:      sessionId,
						clientInfo:     fmt.Sprintf("Test Client %d", id),
						connected:      true,
						connectedAt:    time.Now(),
						connectionType: "WebSocket",
						method:         "GET",
					})
				} else if j%3 == 1 {
					// Add activity
					controller.HandleActivity(mcpActivityMsg{
						sessionId: sessionId,
						activity: MCPActivity{
							Method:    fmt.Sprintf("method-%d", j),
							Params:    fmt.Sprintf(`{"test": %d}`, j),
							Response:  fmt.Sprintf(`{"result": %d}`, j),
							Error:     "",
							Timestamp: time.Now(),
							Duration:  time.Duration(j) * time.Millisecond,
						},
					})
				} else {
					// Read operations
					controller.UpdateConnectionsList()
					controller.UpdateActivityView()
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify data integrity
	controller.mu.RLock()
	defer controller.mu.RUnlock()

	assert.GreaterOrEqual(t, len(controller.connections), 1, "Should have at least one connection")
	assert.GreaterOrEqual(t, len(controller.activities), 1, "Should have at least one activity list")
}

func TestMCPDebugController_ConnectionHandling(t *testing.T) {
	controller := NewMCPDebugController(true)

	// Test connection creation
	msg := mcpConnectionMsg{
		sessionId:      "test-session-1",
		clientInfo:     "Claude Code Test Client",
		connected:      true,
		connectedAt:    time.Now(),
		connectionType: "WebSocket",
		method:         "GET",
	}

	controller.HandleConnection(msg)

	// Verify connection was created
	controller.mu.RLock()
	conn, exists := controller.connections["test-session-1"]
	controller.mu.RUnlock()

	assert.True(t, exists, "Connection should exist")
	assert.Equal(t, "Claude Code", conn.clientName, "Should detect Claude Code client")
	assert.True(t, conn.isConnected, "Connection should be marked as connected")
	assert.Equal(t, "WebSocket", conn.connectionType)

	// Test disconnection
	disconnectMsg := mcpConnectionMsg{
		sessionId: "test-session-1",
		connected: false,
	}

	controller.HandleConnection(disconnectMsg)

	// Verify connection was marked as disconnected
	controller.mu.RLock()
	conn, exists = controller.connections["test-session-1"]
	controller.mu.RUnlock()

	assert.True(t, exists, "Connection should still exist")
	assert.False(t, conn.isConnected, "Connection should be marked as disconnected")
}

func TestMCPDebugController_ActivityTracking(t *testing.T) {
	controller := NewMCPDebugController(true)

	sessionId := "test-session-2"

	// Add some activities
	for i := 0; i < 5; i++ {
		controller.HandleActivity(mcpActivityMsg{
			sessionId: sessionId,
			activity: MCPActivity{
				Method:    fmt.Sprintf("test.method%d", i),
				Params:    fmt.Sprintf(`{"index": %d}`, i),
				Response:  fmt.Sprintf(`{"success": true, "index": %d}`, i),
				Timestamp: time.Now(),
				Duration:  time.Duration(i*10) * time.Millisecond,
			},
		})
	}

	// Verify activities were tracked
	controller.mu.RLock()
	activities, exists := controller.activities[sessionId]
	controller.mu.RUnlock()

	assert.True(t, exists, "Activities should exist for session")
	assert.Len(t, activities, 5, "Should have 5 activities")

	// Verify connection was auto-created for HTTP-only session
	controller.mu.RLock()
	conn, exists := controller.connections[sessionId]
	controller.mu.RUnlock()

	assert.True(t, exists, "Connection should be auto-created")
	assert.Equal(t, "HTTP Client", conn.clientName)
	assert.False(t, conn.isConnected, "HTTP-only sessions should not be marked as connected")
}

func TestMCPDebugController_ActivityLimit(t *testing.T) {
	controller := NewMCPDebugController(true)

	sessionId := "test-session-3"

	// Add more than maxActivities (100)
	for i := 0; i < 110; i++ {
		controller.HandleActivity(mcpActivityMsg{
			sessionId: sessionId,
			activity: MCPActivity{
				Method:    fmt.Sprintf("test.method%d", i),
				Timestamp: time.Now(),
				Duration:  time.Millisecond,
			},
		})
	}

	// Verify activities are limited
	controller.mu.RLock()
	activities := controller.activities[sessionId]
	controller.mu.RUnlock()

	assert.Len(t, activities, 100, "Activities should be limited to 100")

	// Verify oldest activities were removed (first activity should be index 10)
	assert.Equal(t, "test.method10", activities[0].Method, "Oldest activities should be removed")
}

func TestMCPDebugController_UpdateConnectionsList(t *testing.T) {
	controller := NewMCPDebugController(true)

	// Add multiple connections with different timestamps
	now := time.Now()
	for i := 0; i < 3; i++ {
		controller.HandleConnection(mcpConnectionMsg{
			sessionId:      fmt.Sprintf("session-%d", i),
			clientInfo:     fmt.Sprintf("Client %d", i),
			connected:      true,
			connectedAt:    now.Add(time.Duration(-i) * time.Hour),
			connectionType: "WebSocket",
		})
	}

	// Update connections list
	controller.UpdateConnectionsList()

	// Verify list was updated and sorted
	items := controller.connectionsList.Items()
	assert.Len(t, items, 3, "Should have 3 items in list")

	// Verify sorting (newest first)
	for i := 0; i < len(items)-1; i++ {
		item1 := items[i].(mcpConnectionItem)
		item2 := items[i+1].(mcpConnectionItem)
		assert.True(t, item1.connectedAt.After(item2.connectedAt),
			"Items should be sorted by connection time (newest first)")
	}
}

func TestMCPDebugController_NonDebugMode(t *testing.T) {
	// Create controller in non-debug mode
	controller := NewMCPDebugController(false)
	assert.False(t, controller.IsDebugMode())

	// All operations should be no-ops
	controller.HandleConnection(mcpConnectionMsg{
		sessionId: "test",
		connected: true,
	})

	controller.HandleActivity(mcpActivityMsg{
		sessionId: "test",
		activity:  MCPActivity{},
	})

	controller.UpdateConnectionsList()
	controller.UpdateActivityView()

	// Verify no data was stored
	assert.Nil(t, controller.connections)
	assert.Nil(t, controller.activities)

	// Render should return empty string
	result := controller.Render(100, 50, 5, 3)
	assert.Empty(t, result)
}

func TestMCPDebugController_ClientNameDetection(t *testing.T) {
	controller := NewMCPDebugController(true)

	tests := []struct {
		clientInfo   string
		expectedName string
	}{
		{"Claude Code/1.0", "Claude Code"},
		{"VS Code MCP Extension", "VS Code MCP"},
		{"Some Random Client", "Some Random Client"},
		{"Very Long Client Name That Should Be Truncated Because It Is Too Long", "Very Long Client ..."},
		{"", "Unknown Client"},
	}

	for i, tt := range tests {
		controller.HandleConnection(mcpConnectionMsg{
			sessionId:   fmt.Sprintf("session-%d", i),
			clientInfo:  tt.clientInfo,
			connected:   true,
			connectedAt: time.Now(),
		})

		controller.mu.RLock()
		conn := controller.connections[fmt.Sprintf("session-%d", i)]
		controller.mu.RUnlock()

		assert.Equal(t, tt.expectedName, conn.clientName,
			"Client name should be detected/formatted correctly for: %s", tt.clientInfo)
	}
}
