package bridge

import (
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// MockProgram captures messages sent via program.Send()
// Thread-safe implementation to match ServiceBridge's concurrent usage
type MockProgram struct {
	mu       sync.Mutex
	messages []tea.Msg
	sendFunc func(tea.Msg)
}

func (m *MockProgram) Send(msg tea.Msg) {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	if m.sendFunc != nil {
		m.sendFunc(msg)
	}
	m.mu.Unlock()
}

func (m *MockProgram) GetMessages() []tea.Msg {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	result := make([]tea.Msg, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *MockProgram) ClearMessages() {
	m.mu.Lock()
	m.messages = []tea.Msg{}
	m.mu.Unlock()
}

// Test that ServiceBridge converts LogStore.Add() to LogAddedMsg
func TestServiceBridge_LogStoreObserver(t *testing.T) {
	// Create services
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	// Create bridge
	bridge := NewServiceBridge(nil, logStore, nil, nil)

	// Create mock program
	mockProg := &MockProgram{}
	bridge.Start(mockProg)
	defer bridge.Stop()

	// Add a log entry (should trigger observer)
	logStore.Add("test-proc", "TestProc", "Hello World", false)

	// Small delay to ensure async observer fires
	time.Sleep(10 * time.Millisecond)

	// Verify message was sent
	msgs := mockProg.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Check message type and content
	logMsg, ok := msgs[0].(messages.LogAddedMsg)
	if !ok {
		t.Fatalf("Expected LogAddedMsg, got %T", msgs[0])
	}

	if logMsg.Entry.Content != "Hello World" {
		t.Errorf("Expected content 'Hello World', got '%s'", logMsg.Entry.Content)
	}
	if logMsg.Entry.ProcessID != "test-proc" {
		t.Errorf("Expected ProcessID 'test-proc', got '%s'", logMsg.Entry.ProcessID)
	}
	if logMsg.Entry.ProcessName != "TestProc" {
		t.Errorf("Expected ProcessName 'TestProc', got '%s'", logMsg.Entry.ProcessName)
	}
}

// Test that multiple log additions generate multiple messages
func TestServiceBridge_MultipleLogEntries(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	bridge := NewServiceBridge(nil, logStore, nil, nil)
	mockProg := &MockProgram{}
	bridge.Start(mockProg)
	defer bridge.Stop()

	// Add multiple entries
	logStore.Add("proc1", "Process1", "Log 1", false)
	logStore.Add("proc2", "Process2", "Log 2", true)
	logStore.Add("proc1", "Process1", "Log 3", false)

	time.Sleep(10 * time.Millisecond)

	msgs := mockProg.GetMessages()
	if len(msgs) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(msgs))
	}

	// Verify each message
	for i, msg := range msgs {
		logMsg, ok := msg.(messages.LogAddedMsg)
		if !ok {
			t.Errorf("Message %d: expected LogAddedMsg, got %T", i, msg)
			continue
		}

		switch i {
		case 0:
			if logMsg.Entry.Content != "Log 1" {
				t.Errorf("Message 0: expected 'Log 1', got '%s'", logMsg.Entry.Content)
			}
		case 1:
			if logMsg.Entry.Content != "Log 2" {
				t.Errorf("Message 1: expected 'Log 2', got '%s'", logMsg.Entry.Content)
			}
			if !logMsg.Entry.IsError {
				t.Error("Message 1: expected IsError=true")
			}
		case 2:
			if logMsg.Entry.Content != "Log 3" {
				t.Errorf("Message 2: expected 'Log 3', got '%s'", logMsg.Entry.Content)
			}
		}
	}
}

// Test EventBus ProcessExited (maps to ProcessStoppedMsg)
func TestServiceBridge_ProcessExited(t *testing.T) {
	eventBus := events.NewEventBus()

	bridge := NewServiceBridge(nil, nil, nil, eventBus)
	mockProg := &MockProgram{}
	bridge.Start(mockProg)
	defer bridge.Stop()

	eventBus.Publish(events.Event{
		Type:      events.ProcessExited,
		ProcessID: "dev-123",
		Data: map[string]interface{}{
			"name":      "dev",
			"exit_code": 0,
		},
	})

	time.Sleep(10 * time.Millisecond)

	msgs := mockProg.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	stopMsg, ok := msgs[0].(messages.ProcessStoppedMsg)
	if !ok {
		t.Fatalf("Expected ProcessStoppedMsg, got %T", msgs[0])
	}

	if stopMsg.ProcessID != "dev-123" {
		t.Errorf("Expected ProcessID 'dev-123', got '%s'", stopMsg.ProcessID)
	}
	if stopMsg.ExitCode != 0 {
		t.Errorf("Expected ExitCode 0, got %d", stopMsg.ExitCode)
	}
}

// Test EventBus ProcessStarted conversion
func TestServiceBridge_ProcessStarted(t *testing.T) {
	eventBus := events.NewEventBus()

	bridge := NewServiceBridge(nil, nil, nil, eventBus)
	mockProg := &MockProgram{}
	bridge.Start(mockProg)
	defer bridge.Stop()

	eventBus.Publish(events.Event{
		Type:      events.ProcessStarted,
		ProcessID: "dev-123",
		Data: map[string]interface{}{
			"name":   "dev",
			"script": "npm run dev",
		},
	})

	time.Sleep(10 * time.Millisecond)

	msgs := mockProg.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	startMsg, ok := msgs[0].(messages.ProcessStartedMsg)
	if !ok {
		t.Fatalf("Expected ProcessStartedMsg, got %T", msgs[0])
	}

	if startMsg.ProcessID != "dev-123" {
		t.Errorf("Expected ProcessID 'dev-123', got '%s'", startMsg.ProcessID)
	}
	if startMsg.Name != "dev" {
		t.Errorf("Expected Name 'dev', got '%s'", startMsg.Name)
	}
	if startMsg.Script != "npm run dev" {
		t.Errorf("Expected Script 'npm run dev', got '%s'", startMsg.Script)
	}
}

// Test EventBus URL detection conversion
func TestServiceBridge_URLDetected(t *testing.T) {
	eventBus := events.NewEventBus()

	bridge := NewServiceBridge(nil, nil, nil, eventBus)
	mockProg := &MockProgram{}
	bridge.Start(mockProg)
	defer bridge.Stop()

	eventBus.Publish(events.Event{
		Type:      events.URLDetected,
		ProcessID: "dev-123",
		Data: map[string]interface{}{
			"url":         "http://localhost:3000",
			"type":        "http",
			"has_proxy":   true,
			"proxy_port":  8080,
			"proxy_url":   "http://localhost:8080",
			"proxy_label": "Dev Server",
		},
	})

	time.Sleep(10 * time.Millisecond)

	msgs := mockProg.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	urlMsg, ok := msgs[0].(messages.URLDetectedMsg)
	if !ok {
		t.Fatalf("Expected URLDetectedMsg, got %T", msgs[0])
	}

	if urlMsg.URL != "http://localhost:3000" {
		t.Errorf("Expected URL 'http://localhost:3000', got '%s'", urlMsg.URL)
	}
	if !urlMsg.HasProxy {
		t.Error("Expected HasProxy=true")
	}
	if urlMsg.ProxyPort != 8080 {
		t.Errorf("Expected ProxyPort 8080, got %d", urlMsg.ProxyPort)
	}
}

// Test Stop() cleanup
func TestServiceBridge_Stop(t *testing.T) {
	logStore := logs.NewStore(1000, nil)
	defer logStore.Close()

	eventBus := events.NewEventBus()

	bridge := NewServiceBridge(nil, logStore, nil, eventBus)
	mockProg := &MockProgram{}
	bridge.Start(mockProg)

	// Add a log before stopping
	logStore.Add("test", "Test", "Before stop", false)
	time.Sleep(10 * time.Millisecond)

	initialCount := len(mockProg.GetMessages())
	if initialCount != 1 {
		t.Fatalf("Expected 1 initial message, got %d", initialCount)
	}

	// Stop the bridge
	bridge.Stop()

	// Add a log after stopping (should not generate message)
	logStore.Add("test", "Test", "After stop", false)
	time.Sleep(10 * time.Millisecond)

	finalCount := len(mockProg.GetMessages())
	if finalCount != initialCount {
		t.Errorf("Expected no new messages after Stop(), got %d messages (had %d)", finalCount, initialCount)
	}
}

// Test concurrent log additions (stress test)
func TestServiceBridge_ConcurrentLogAdditions(t *testing.T) {
	logStore := logs.NewStore(10000, nil)
	defer logStore.Close()

	bridge := NewServiceBridge(nil, logStore, nil, nil)
	mockProg := &MockProgram{}
	bridge.Start(mockProg)
	defer bridge.Stop()

	// Add 100 logs concurrently
	const numLogs = 100
	done := make(chan struct{})

	go func() {
		for i := 0; i < numLogs; i++ {
			go logStore.Add("test", "Test", "Concurrent log", false)
		}
		close(done)
	}()

	<-done
	time.Sleep(50 * time.Millisecond)

	msgs := mockProg.GetMessages()
	if len(msgs) != numLogs {
		t.Errorf("Expected %d messages, got %d", numLogs, len(msgs))
	}
}
