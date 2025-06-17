package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/pkg/events"
)

func TestSystemMessages(t *testing.T) {
	// Create a minimal model for testing
	eventBus := events.NewEventBus()
	model := Model{
		eventBus:       eventBus,
		updateChan:     make(chan tea.Msg, 100),
		systemMessages: []SystemMessage{},
	}

	// Test adding system message directly
	model.addSystemMessage("info", "Test", "Test message")
	
	if len(model.systemMessages) != 1 {
		t.Errorf("Expected 1 system message, got %d", len(model.systemMessages))
	}
	
	if model.systemMessages[0].Message != "Test message" {
		t.Errorf("Expected 'Test message', got '%s'", model.systemMessages[0].Message)
	}

	// Test system.message event
	eventBus.Subscribe(events.EventType("system.message"), func(e events.Event) {
		level, _ := e.Data["level"].(string)
		context, _ := e.Data["context"].(string) 
		message, _ := e.Data["message"].(string)
		
		model.updateChan <- systemMessageMsg{
			level:   level,
			context: context,
			message: message,
		}
	})

	// Publish a system message event
	eventBus.Publish(events.Event{
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
	case msg := <-model.updateChan:
		if sysMsg, ok := msg.(systemMessageMsg); ok {
			if sysMsg.message != "✅ MCP server started on http://localhost:8080/mcp" {
				t.Errorf("Expected MCP message, got '%s'", sysMsg.message)
			}
		} else {
			t.Error("Expected systemMessageMsg type")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No message received on update channel")
	}
}

func TestSystemMessageLimit(t *testing.T) {
	model := Model{
		systemMessages: []SystemMessage{},
	}

	// Add 150 messages
	for i := 0; i < 150; i++ {
		model.addSystemMessage("info", "Test", fmt.Sprintf("Message %d", i))
	}

	// Should only keep 100
	if len(model.systemMessages) != 100 {
		t.Errorf("Expected 100 system messages, got %d", len(model.systemMessages))
	}

	// Most recent should be first
	if model.systemMessages[0].Message != "Message 149" {
		t.Errorf("Expected most recent message first, got '%s'", model.systemMessages[0].Message)
	}
}