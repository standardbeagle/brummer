package events

import (
	"testing"
	"time"
)

func TestAICoderEventTypes(t *testing.T) {
	// Test that AI coder event types are properly defined
	expectedTypes := []EventType{
		EventAICoderCreated, EventAICoderStarted, EventAICoderPaused,
		EventAICoderResumed, EventAICoderCompleted, EventAICoderFailed,
		EventAICoderStopped, EventAICoderDeleted, EventAICoderProgress,
		EventAICoderMilestone, EventAICoderOutput, EventAICoderFileCreated,
		EventAICoderFileModified, EventAICoderFileDeleted, EventAICoderWorkspaceSync,
		EventAICoderAPICall, EventAICoderAPIError, EventAICoderRateLimit,
		EventAICoderResourceUsage, EventAICoderResourceLimit,
	}

	for _, eventType := range expectedTypes {
		if string(eventType) == "" {
			t.Errorf("Event type is empty: %v", eventType)
		}
	}
}

func TestAICoderEventFactories(t *testing.T) {
	// Test lifecycle event factory
	lifecycleEvent := NewAICoderLifecycleEvent("test-id", "test-name", "test-type", "old", "new", "test reason")
	if lifecycleEvent.CoderID != "test-id" {
		t.Errorf("Expected coder ID 'test-id', got '%s'", lifecycleEvent.CoderID)
	}
	if lifecycleEvent.CoderName != "test-name" {
		t.Errorf("Expected coder name 'test-name', got '%s'", lifecycleEvent.CoderName)
	}
	if lifecycleEvent.PreviousStatus != "old" {
		t.Errorf("Expected previous status 'old', got '%s'", lifecycleEvent.PreviousStatus)
	}
	if lifecycleEvent.CurrentStatus != "new" {
		t.Errorf("Expected current status 'new', got '%s'", lifecycleEvent.CurrentStatus)
	}

	// Test progress event factory
	progressEvent := NewAICoderProgressEvent("test-id", "test-name", 0.5, "testing", "test progress")
	if progressEvent.Progress != 0.5 {
		t.Errorf("Expected progress 0.5, got %f", progressEvent.Progress)
	}
	if progressEvent.Stage != "testing" {
		t.Errorf("Expected stage 'testing', got '%s'", progressEvent.Stage)
	}
}

func TestAICoderEventAggregator(t *testing.T) {
	eventBus := NewEventBus()
	defer eventBus.Shutdown()

	aggregator := NewAICoderEventAggregator(eventBus, 100)

	// Test that aggregator was created properly
	if aggregator == nil {
		t.Fatal("Failed to create AI coder event aggregator")
	}

	// Test event emission
	eventBus.EmitAICoderEvent(EventAICoderCreated, "test-coder", "Test Coder", map[string]interface{}{
		"provider": "test-provider",
	})

	// Give event processing time
	time.Sleep(100 * time.Millisecond)

	// Check stats
	stats := aggregator.GetStats()
	if stats.TotalEvents == 0 {
		t.Error("Expected at least 1 event in stats, got 0")
	}

	// Test event filtering
	filter := AICoderEventFilter{
		CoderID: "test-coder",
	}
	events := aggregator.GetEvents(filter)

	if len(events) == 0 {
		t.Error("Expected at least 1 filtered event, got 0")
	}
}

func TestWorkspaceEventType(t *testing.T) {
	tests := []struct {
		operation string
		expected  EventType
	}{
		{"create", EventAICoderFileCreated},
		{"modify", EventAICoderFileModified},
		{"delete", EventAICoderFileDeleted},
		{"unknown", EventAICoderWorkspaceSync},
	}

	for _, test := range tests {
		result := getWorkspaceEventType(test.operation)
		if result != test.expected {
			t.Errorf("Expected %s for operation %s, got %s", test.expected, test.operation, result)
		}
	}
}

func TestAICoderEventFilter(t *testing.T) {
	event1 := AICoderEvent{
		Type:      "test-type",
		CoderID:   "coder-1",
		Timestamp: time.Now(),
	}

	event2 := AICoderEvent{
		Type:      "other-type",
		CoderID:   "coder-2",
		Timestamp: time.Now().Add(-time.Hour),
	}

	// Test coder ID filter
	filter := AICoderEventFilter{CoderID: "coder-1"}
	if !filter.matches(event1) {
		t.Error("Filter should match event1")
	}
	if filter.matches(event2) {
		t.Error("Filter should not match event2")
	}

	// Test event type filter
	filter = AICoderEventFilter{EventType: "test-type"}
	if !filter.matches(event1) {
		t.Error("Filter should match event1")
	}
	if filter.matches(event2) {
		t.Error("Filter should not match event2")
	}

	// Test time filter
	filter = AICoderEventFilter{Since: time.Now().Add(-30 * time.Minute)}
	if !filter.matches(event1) {
		t.Error("Filter should match event1 (recent)")
	}
	if filter.matches(event2) {
		t.Error("Filter should not match event2 (old)")
	}
}
