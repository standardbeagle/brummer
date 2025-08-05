package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/system"
	"github.com/standardbeagle/brummer/pkg/events"
	"github.com/stretchr/testify/assert"
)

func TestSystemMessages(t *testing.T) {
	// Create a model with all dependencies
	model := createTestModelWithDefaults()

	// Test sending system message through event controller
	t.Run("system_message_via_event", func(t *testing.T) {
		// Clear any existing messages by creating a fresh update channel
		updateChan := make(chan tea.Msg, 100)
		model.updateChan = updateChan
		
		// Also update the event controller's update channel to match
		model.eventController.updateChan = updateChan

		// Publish a system message event
		model.eventBus.Publish(events.Event{
			Type: events.EventType("system.message"),
			Data: map[string]interface{}{
				"level":   "success",
				"context": "MCP Server",
				"message": "✅ MCP server started on http://localhost:8080/mcp",
			},
		})

		// Give event time to propagate
		time.Sleep(50 * time.Millisecond)

		// Check if message was received
		select {
		case msg := <-updateChan:
			if sysMsg, ok := msg.(system.SystemMessageMsg); ok {
				assert.Equal(t, "success", sysMsg.Level, "Level should match")
				assert.Equal(t, "MCP Server", sysMsg.Context, "Context should match")
				assert.Equal(t, "✅ MCP server started on http://localhost:8080/mcp", sysMsg.Message, "Message should match")
			} else {
				t.Errorf("Expected system.SystemMessageMsg type, got %T", msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("No message received on update channel")
		}
	})

	// Test empty message handling
	t.Run("empty_message_ignored", func(t *testing.T) {
		// Clear channel
		updateChan := make(chan tea.Msg, 100)
		model.updateChan = updateChan
		
		// Also update the event controller's update channel to match
		model.eventController.updateChan = updateChan

		// Publish empty message
		model.eventBus.Publish(events.Event{
			Type: events.EventType("system.message"),
			Data: map[string]interface{}{
				"level":   "info",
				"context": "Test",
				"message": "", // Empty message
			},
		})

		// Give event time to propagate
		time.Sleep(50 * time.Millisecond)

		// Should not receive any message
		select {
		case <-updateChan:
			t.Error("Should not receive message for empty content")
		case <-time.After(50 * time.Millisecond):
			// Expected - no message sent
		}
	})
}

func TestSystemController(t *testing.T) {
	// Test system controller directly
	controller := system.NewController(100)

	t.Run("add_message", func(t *testing.T) {
		controller.AddMessage("info", "Test Context", "Test message")
		messages := controller.GetMessages()
		assert.Len(t, messages, 1, "Should have 1 message")
		assert.Equal(t, "info", messages[0].Level)
		assert.Equal(t, "Test Context", messages[0].Context)
		assert.Equal(t, "Test message", messages[0].Message)
	})

	t.Run("get_messages", func(t *testing.T) {
		// Clear messages
		controller = system.NewController(100)
		
		// Add some messages
		controller.AddMessage("info", "Test", "Message 1")
		controller.AddMessage("error", "Test", "Message 2")
		controller.AddMessage("warn", "Test", "Message 3")
		
		messages := controller.GetMessages()
		assert.Len(t, messages, 3, "Should have 3 messages")
		
		// Most recent should be first
		assert.Equal(t, "Message 3", messages[0].Message)
		assert.Equal(t, "Message 2", messages[1].Message)
		assert.Equal(t, "Message 1", messages[2].Message)
	})

	t.Run("message_limit", func(t *testing.T) {
		controller = system.NewController(10) // Small limit for testing
		
		// Add more than the limit
		for i := 0; i < 15; i++ {
			controller.AddMessage("info", "Test", fmt.Sprintf("Message %d", i))
		}
		
		messages := controller.GetMessages()
		assert.Equal(t, 10, len(messages), "Should limit messages to max")
		// Most recent should be first
		assert.Equal(t, "Message 14", messages[0].Message)
	})

	t.Run("clear_messages", func(t *testing.T) {
		// Add some messages
		controller.AddMessage("info", "Test", "Message 1")
		controller.AddMessage("error", "Test", "Message 2")
		
		// Clear them
		controller.Clear()
		
		messages := controller.GetMessages()
		assert.Empty(t, messages, "Should have no messages after clear")
		assert.False(t, controller.IsExpanded(), "Should not be expanded after clear")
	})
}

func TestSystemPanelRenderer(t *testing.T) {
	controller := system.NewController(100)
	renderer := system.NewPanelRenderer(controller)

	t.Run("render_empty", func(t *testing.T) {
		// RenderPanel should return empty when no messages
		output := renderer.RenderPanel()
		assert.Empty(t, output, "Should be empty when no messages")
	})

	t.Run("render_with_messages", func(t *testing.T) {
		// Set dimensions on controller
		controller.UpdateSize(80, 24, 3, 3)
		
		// Add some messages
		controller.AddMessage("info", "Test", "Info message")
		controller.AddMessage("error", "Test", "Error message")
		
		// Render panel
		output := renderer.RenderPanel()
		assert.NotEmpty(t, output, "Should render content when messages exist")
	})

	t.Run("handle_small_dimensions", func(t *testing.T) {
		// Test with very small dimensions
		assert.NotPanics(t, func() {
			controller.UpdateSize(1, 1, 0, 0)
			renderer.RenderPanel()
		}, "Should handle small dimensions without panic")
		
		// Test with zero dimensions
		assert.NotPanics(t, func() {
			controller.UpdateSize(0, 0, 0, 0)
			renderer.RenderPanel()
		}, "Should handle zero dimensions without panic")
	})
}