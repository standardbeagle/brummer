package integration

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/tui/bridge"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestEndToEndMessageFlow demonstrates the complete architecture:
// Service → ServiceBridge → Shell → Component → View
func TestEndToEndMessageFlow(t *testing.T) {
	// Setup services
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	defer logStore.Close()

	// Create components
	logsView := components.NewLogsView(logStore)
	shell := components.NewIntegratedShell()
	shell.SetLogsView(logsView)

	// Create ServiceBridge
	serviceBridge := bridge.NewServiceBridge(nil, logStore, nil, eventBus)

	// Create mock program to capture ServiceBridge messages
	msgChan := make(chan tea.Msg, 10)
	mockProg := &mockProgram{msgChan: msgChan}

	// Start the bridge
	serviceBridge.Start(mockProg)
	defer serviceBridge.Stop()

	// TEST 1: Add log to store (service action)
	logStore.Add("test-proc", "TestProcess", "Hello from service", false)

	// Wait for async processing
	time.Sleep(20 * time.Millisecond)

	// TEST 2: Forward all messages from ServiceBridge to shell
	messagesForwarded := 0
	timeout := time.After(100 * time.Millisecond)

drainLoop:
	for {
		select {
		case msg := <-msgChan:
			t.Logf("Received message from ServiceBridge: %T", msg)
			shell.Update(msg)
			messagesForwarded++
		case <-timeout:
			break drainLoop
		default:
			if messagesForwarded > 0 {
				break drainLoop
			}
			time.Sleep(5 * time.Millisecond)
		}
	}

	if messagesForwarded == 0 {
		t.Fatal("No messages were received from ServiceBridge")
	}

	// TEST 3: Verify component received update via shell routing
	view := logsView.View()
	if !strings.Contains(view, "Hello from service") {
		t.Error("LogsView should contain 'Hello from service'")
		t.Logf("View content (length=%d): %q", len(view), view)
		t.Logf("Messages forwarded: %d", messagesForwarded)
	}

	// TEST 4: Multiple logs
	logStore.Add("test-proc", "TestProcess", "Second log", false)
	logStore.Add("test-proc", "TestProcess", "Third log", true)

	time.Sleep(20 * time.Millisecond)

	// Process all messages
	for len(msgChan) > 0 {
		msg := <-msgChan
		shell.Update(msg)
	}

	view = logsView.View()
	if !strings.Contains(view, "Second log") {
		t.Error("LogsView should contain 'Second log'")
	}
	if !strings.Contains(view, "Third log") {
		t.Error("LogsView should contain 'Third log'")
	}
}

// TestEventBusIntegration tests EventBus → ServiceBridge → Component flow
func TestEventBusIntegration(t *testing.T) {
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(1000, eventBus)
	defer logStore.Close()

	logsView := components.NewLogsView(logStore)
	shell := components.NewIntegratedShell()
	shell.SetLogsView(logsView)

	serviceBridge := bridge.NewServiceBridge(nil, logStore, nil, eventBus)

	msgChan := make(chan tea.Msg, 10)
	mockProg := &mockProgram{msgChan: msgChan}

	serviceBridge.Start(mockProg)
	defer serviceBridge.Stop()

	// Publish process started event
	eventBus.Publish(events.Event{
		Type:      events.ProcessStarted,
		ProcessID: "dev-123",
		Data: map[string]interface{}{
			"name":   "dev",
			"script": "npm run dev",
		},
	})

	time.Sleep(20 * time.Millisecond)

	// Verify ProcessStartedMsg was received
	messagesReceived := 0
	for len(msgChan) > 0 {
		<-msgChan
		messagesReceived++
	}

	if messagesReceived == 0 {
		t.Error("Expected to receive ProcessStartedMsg")
	}
}

// TestHighVolumeLogFlow tests system under log pressure
func TestHighVolumeLogFlow(t *testing.T) {
	eventBus := events.NewEventBus()
	logStore := logs.NewStore(10000, eventBus)
	defer logStore.Close()

	logsView := components.NewLogsView(logStore)
	shell := components.NewIntegratedShell()
	shell.SetLogsView(logsView)

	serviceBridge := bridge.NewServiceBridge(nil, logStore, nil, eventBus)

	msgChan := make(chan tea.Msg, 1000)
	mockProg := &mockProgram{msgChan: msgChan}

	serviceBridge.Start(mockProg)
	defer serviceBridge.Stop()

	// Add 100 logs rapidly
	const numLogs = 100
	for i := 0; i < numLogs; i++ {
		logStore.Add("test", "Test", "Log message", false)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Process all messages
	messagesProcessed := 0
	for len(msgChan) > 0 {
		msg := <-msgChan
		shell.Update(msg)
		messagesProcessed++
	}

	if messagesProcessed < numLogs {
		t.Logf("Processed %d messages out of %d logs", messagesProcessed, numLogs)
	}

	// Verify logs appear in view
	view := logsView.View()
	if !strings.Contains(view, "Log message") {
		t.Error("LogsView should contain log messages")
	}
}

// mockProgram implements bridge.Program interface for testing
type mockProgram struct {
	msgChan chan tea.Msg
}

func (m *mockProgram) Send(msg tea.Msg) {
	select {
	case m.msgChan <- msg:
	default:
		// Channel full, drop message (shouldn't happen in tests)
	}
}
